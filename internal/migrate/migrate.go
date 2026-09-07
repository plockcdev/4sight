package migrate

import (
	"cmp"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	initialID            = "00000000"
	initialMigrationFile = "00000000-00000000.sql"
	migrationsDB         = "4sight_migration"
	currentName          = "current"
	submittedName        = "submitted"
)

var migrationIDRe = regexp.MustCompile(`^\d{8}$`)
var migrationSuffixRe = regexp.MustCompile(`^\d{4}$`)

type store interface {
	databaseExists(name string) (bool, error)
	applyFile(path string) error
	trackingTableExists(table string) (bool, error)
	insertApplied(table, id string, at time.Time) error
}

type database struct {
	dbType string
	name   string
}

func (d database) key() string {
	return d.dbType + "_" + d.name
}

type migrationFile struct {
	name     string
	id       string
	parentID string
	path     string
}

func Up(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runUp(root, newDoltStore(filepath.Join(root, "db")), time.Now, os.Stdout)
}

func runUp(metadataRoot string, st store, now func() time.Time, stdout io.Writer) error {
	dbs, err := listSchemaDatabases(filepath.Join(metadataRoot, "schemas"))
	if err != nil {
		return err
	}
	for _, db := range dbs {
		applied, err := migrateDatabase(metadataRoot, db, st, now)
		if err != nil {
			return err
		}
		for _, id := range applied {
			if _, err := fmt.Fprintf(stdout, "applied %s %s\n", db.key(), id); err != nil {
				return err
			}
		}
	}
	return nil
}

func migrateDatabase(metadataRoot string, db database, st store, now func() time.Time) ([]string, error) {
	dir := filepath.Join(metadataRoot, "migrations", db.key())
	if _, err := os.Stat(filepath.Join(dir, initialMigrationFile)); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s: missing %s in %s", db.key(), initialMigrationFile, dir)
		}
		return nil, fmt.Errorf("%s: check %s in %s: %w", db.key(), initialMigrationFile, dir, err)
	}

	byID, byParent, err := listMigrationFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", db.key(), dir, err)
	}

	submittedName, err := resolveLink(dir, submittedName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s: missing submitted symlink to an existing file in %s", db.key(), dir)
		}
		return nil, fmt.Errorf("%s: submitted in %s: %w", db.key(), dir, err)
	}
	submitted, ok := fileByName(byID, submittedName)
	if !ok {
		return nil, fmt.Errorf("%s: submitted %s is not a migration file in %s", db.key(), submittedName, dir)
	}

	currentNameResolved, currentErr := resolveLink(dir, currentName)
	var current *migrationFile
	switch {
	case currentErr == nil:
		var ok bool
		current, ok = fileByName(byID, currentNameResolved)
		if !ok {
			return nil, fmt.Errorf("%s: current %s is not a migration file in %s", db.key(), currentNameResolved, dir)
		}
	case os.IsNotExist(currentErr):
		// uninitialized until the database-exists check below
	default:
		return nil, fmt.Errorf("%s: current in %s: %w", db.key(), dir, currentErr)
	}

	if err := validateSubmittedToCurrent(byID, byParent, submitted, current); err != nil {
		return nil, fmt.Errorf("%s: %s: %w", db.key(), dir, err)
	}

	if current == nil {
		exists, err := st.databaseExists(db.name)
		if err != nil {
			return nil, fmt.Errorf("%s: check database %s (dir %s): %w", db.key(), db.name, dir, err)
		}
		if exists {
			return nil, fmt.Errorf("%s: database %s exists but current symlink is missing in %s", db.key(), db.name, dir)
		}
	}

	if current != nil && current.name == submitted.name {
		return nil, nil
	}

	next, err := startingMigration(byID, byParent, current)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", db.key(), dir, err)
	}

	var applied []string
	for {
		if err := st.applyFile(next.path); err != nil {
			return applied, fmt.Errorf("%s: apply %s: %w", db.key(), next.path, err)
		}
		if err := updateCurrent(dir, next.name); err != nil {
			return applied, fmt.Errorf("%s: update current in %s: %w", db.key(), dir, err)
		}
		table := db.name
		ok, err := st.trackingTableExists(table)
		if err != nil {
			return applied, fmt.Errorf("%s: check tracking table %s.%s (dir %s): %w", db.key(), migrationsDB, table, dir, err)
		}
		if !ok {
			return applied, fmt.Errorf("%s: 4sight_migration table %s is missing (dir %s)", db.key(), table, dir)
		}
		if err := st.insertApplied(table, next.id, now().UTC().Truncate(time.Second)); err != nil {
			return applied, fmt.Errorf("%s: record %s in %s.%s (dir %s): %w", db.key(), next.id, migrationsDB, table, dir, err)
		}
		applied = append(applied, next.id)
		if next.name == submitted.name {
			return applied, nil
		}
		next, err = childOf(byParent, next.id)
		if err != nil {
			return applied, fmt.Errorf("%s: %s: %w", db.key(), dir, err)
		}
	}
}

func startingMigration(byID map[string]*migrationFile, byParent map[string][]*migrationFile, current *migrationFile) (*migrationFile, error) {
	if current == nil {
		next, ok := byID[initialID]
		if !ok {
			return nil, fmt.Errorf("missing %s", initialMigrationFile)
		}
		return next, nil
	}
	return childOf(byParent, current.id)
}

func childOf(byParent map[string][]*migrationFile, parentID string) (*migrationFile, error) {
	kids := byParent[parentID]
	switch len(kids) {
	case 0:
		return nil, fmt.Errorf("no migration with parent id %s", parentID)
	case 1:
		return kids[0], nil
	default:
		return nil, fmt.Errorf("multiple migrations have parent id %s", parentID)
	}
}

func validateSubmittedToCurrent(byID map[string]*migrationFile, byParent map[string][]*migrationFile, submitted, current *migrationFile) error {
	targetID := initialID
	if current != nil {
		targetID = current.id
	}
	seen := make(map[string]struct{})
	at := submitted
	for {
		if at.id == targetID {
			return nil
		}
		if _, ok := seen[at.id]; ok {
			return fmt.Errorf("cycle in parent chain at %s", at.id)
		}
		seen[at.id] = struct{}{}
		if at.parentID == "" || at.parentID == at.id {
			return fmt.Errorf("no path through parent ids from submitted %s to current", submitted.id)
		}
		if len(byParent[at.parentID]) > 1 {
			return fmt.Errorf("multiple migrations have parent id %s", at.parentID)
		}
		parent, ok := byID[at.parentID]
		if !ok {
			return fmt.Errorf("no path through parent ids from submitted %s to current", submitted.id)
		}
		at = parent
	}
}

func listSchemaDatabases(schemasDir string) ([]database, error) {
	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return nil, fmt.Errorf("read schemas dir %s: %w", schemasDir, err)
	}
	var found []database
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		stem, ok := strings.CutSuffix(name, ".md")
		if !ok {
			return nil, fmt.Errorf("%s: expected <db_type>_<db_name>.md", filepath.Join(schemasDir, name))
		}
		dbType, dbName, ok := strings.Cut(stem, "_")
		if !ok || dbType == "" || dbName == "" {
			return nil, fmt.Errorf("%s: expected <db_type>_<db_name>.md", filepath.Join(schemasDir, name))
		}
		found = append(found, database{dbType: dbType, name: dbName})
	}
	return orderDatabases(found), nil
}

func orderDatabases(found []database) []database {
	var first []database
	var rest []database
	for _, db := range found {
		if db.dbType == "dolt" && db.name == migrationsDB {
			first = append(first, db)
			continue
		}
		rest = append(rest, db)
	}
	slices.SortFunc(rest, func(a, b database) int {
		return cmp.Compare(a.key(), b.key())
	})
	if len(first) == 0 {
		first = append(first, database{dbType: "dolt", name: migrationsDB})
	}
	return append(first, rest...)
}

func listMigrationFiles(dir string) (map[string]*migrationFile, map[string][]*migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	byID := make(map[string]*migrationFile)
	byParent := make(map[string][]*migrationFile)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == currentName || entry.Name() == submittedName {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		file, err := parseMigrationName(entry.Name())
		if err != nil {
			return nil, nil, err
		}
		file.path = filepath.Join(dir, entry.Name())
		if _, exists := byID[file.id]; exists {
			return nil, nil, fmt.Errorf("duplicate migration id %s", file.id)
		}
		byID[file.id] = file
		if file.parentID != "" && file.parentID != file.id {
			byParent[file.parentID] = append(byParent[file.parentID], file)
		}
	}
	return byID, byParent, nil
}

func parseMigrationName(name string) (*migrationFile, error) {
	stem, ok := strings.CutSuffix(name, ".sql")
	if !ok {
		return nil, fmt.Errorf("invalid migration filename %q", name)
	}
	id, parent, ok := strings.Cut(stem, "-")
	if !ok || !migrationIDRe.MatchString(id) || !migrationIDRe.MatchString(parent) {
		return nil, fmt.Errorf("invalid migration filename %q", name)
	}
	return &migrationFile{name: name, id: id, parentID: parent}, nil
}

func resolveLink(dir, name string) (string, error) {
	link := filepath.Join(dir, name)
	info, err := os.Lstat(link)
	if err != nil {
		return "", err
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return "", fmt.Errorf("%s is not a symlink", name)
	}
	target, err := os.Readlink(link)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(target) {
		target = filepath.Clean(target)
	} else {
		target = filepath.Clean(filepath.Join(dir, target))
	}
	if filepath.Dir(target) != filepath.Clean(dir) {
		return "", fmt.Errorf("%s must point to a file in the migration directory", name)
	}
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s symlink does not point to an existing file", name)
		}
		return "", err
	}
	return filepath.Base(target), nil
}

func updateCurrent(dir, filename string) error {
	link := filepath.Join(dir, currentName)
	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(filename, link)
}

func fileByName(byID map[string]*migrationFile, name string) (*migrationFile, bool) {
	for _, file := range byID {
		if file.name == name {
			return file, true
		}
	}
	return nil, false
}

func findMetadataRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	for dir := abs; ; {
		candidate := filepath.Join(dir, ".4sight")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .4sight directory found from %s", start)
		}
		dir = parent
	}
}
