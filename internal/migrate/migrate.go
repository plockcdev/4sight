package migrate

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	initialID            = "00000000"
	initialMigrationFile = "00000000-00000000.sql"
	migrationsDB         = "4sight_migration"
	currentName          = "current"
	submittedName        = "submitted"
)

var migrationIDRe = regexp.MustCompile(`^\d{8}$`)

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
		if !strings.HasSuffix(entry.Name(), ".sql") && !strings.HasSuffix(entry.Name(), ".delete") {
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
		stem, ok = strings.CutSuffix(name, ".delete")
	}
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
	tmp := filepath.Join(dir, ".4sight-current-new")
	if err := os.Symlink(filename, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
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
