package migrate

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"foursight.dev/4sight/internal/intent"
)

type physicalID struct{ device, inode uint64 }

// Reconcile only previously recorded intents before normal migration checks.
// New retirements still wait until the remaining databases pass preflight.
func reconcilePendingRetirements(root string, st *doltStore) error {
	rows, err := st.queryCSV("SELECT database_type, original_name, migration_id FROM 4sight_migration.database_deletions WHERE state='retiring' ORDER BY id")
	if err != nil {
		return fmt.Errorf("inspect incomplete retirements: %w", err)
	}
	for _, row := range rows {
		if len(row) != 3 || row[0] != "dolt" || ident(row[1]) != nil || !migrationIDRe.MatchString(row[2]) || strings.EqualFold(row[1], "spec") || strings.EqualFold(row[1], "4sight_migration") {
			return fmt.Errorf("invalid incomplete retirement record; reconcile it before proceeding")
		}
		db := database{dbType: row[0], name: row[1]}
		tracked, err := st.trackingTableExists(db.name)
		if err != nil {
			return err
		}
		if !tracked {
			return fmt.Errorf("%s: incomplete retirement has no tracking table; reconcile it before proceeding", db.key())
		}
		dir := filepath.Join(root, "migrations", db.key())
		target, err := resolveLink(dir, "submitted")
		if err != nil {
			return fmt.Errorf("%s: incomplete retirement submitted link: %w", db.key(), err)
		}
		entry, err := parseMigrationName(target)
		if err != nil || !strings.HasSuffix(target, ".delete") || entry.id != row[2] {
			return fmt.Errorf("%s: incomplete retirement does not match its submitted entry; reconcile it before proceeding", db.key())
		}
		if err := retireDatabase(root, db, st); err != nil {
			return err
		}
	}
	return nil
}

func identifyDirectory(path string) (physicalID, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return physicalID{}, false, nil
	}
	if err != nil {
		return physicalID{}, false, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return physicalID{}, true, fmt.Errorf("unsafe database path %s", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return physicalID{}, true, fmt.Errorf("physical identity unavailable for %s", path)
	}
	return physicalID{uint64(stat.Dev), uint64(stat.Ino)}, true, nil
}

func retireDatabase(root string, db database, st *doltStore) error {
	if err := ident(db.name); err != nil {
		return err
	}
	exists, err := st.databaseExists(db.name)
	if err != nil {
		return err
	}
	tracked, err := st.trackingTableExists(db.name)
	if err != nil {
		return err
	}
	if !exists && !tracked {
		fmt.Printf("%s: no retirement needed; database was never created\n", db.key())
		return nil
	}
	dir := filepath.Join(root, "migrations", db.key())
	files, children, err := listMigrationFiles(dir)
	if err != nil {
		return err
	}
	target, err := resolveLink(dir, "submitted")
	if err != nil {
		return err
	}
	item, ok := fileByName(files, target)
	if !ok || !strings.HasSuffix(target, ".delete") {
		return fmt.Errorf("%s: submitted retirement entry is missing", db.key())
	}
	info, err := os.Lstat(item.path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s: retirement entry must be a regular file", db.key())
	}
	if len(children[item.id]) != 0 {
		return fmt.Errorf("%s: retirement is not final", db.key())
	}
	if err := validateSubmittedToCurrent(files, children, item, nil); err != nil {
		return err
	}
	if first := files[initialID]; first == nil || first.name != initialMigrationFile {
		return fmt.Errorf("%s: missing valid initial migration", db.key())
	}
	parent := files[item.parentID]
	if parent == nil || strings.HasSuffix(parent.name, ".delete") {
		return fmt.Errorf("%s: invalid retirement parent", db.key())
	}
	query := fmt.Sprintf("SELECT trash_name,physical_device,physical_inode,state FROM 4sight_migration.database_deletions WHERE database_type='%s' AND original_name='%s' AND migration_id='%s'", db.dbType, db.name, item.id)
	rows, err := st.queryCSV(query)
	if err != nil {
		return err
	}
	if len(rows) > 1 {
		return fmt.Errorf("%s: ambiguous retirement records", db.key())
	}
	if len(rows) == 1 {
		if len(rows[0]) < 4 {
			return fmt.Errorf("%s: malformed retirement record", db.key())
		}
		if err := validateRetirementHistory(db, st, files, item, rows[0][3] != "retiring"); err != nil {
			return err
		}
	}
	original := filepath.Join(root, "db", db.name)
	if len(rows) == 0 {
		_, exists, err := identifyDirectory(original)
		if err != nil {
			return err
		}
		if !exists {
			tracked, err := st.trackingTableExists(db.name)
			if err != nil {
				return err
			}
			if tracked {
				return fmt.Errorf("%s: database is absent with retained tracking but no retirement record; reconcile it", db.key())
			}
			fmt.Printf("%s: no retirement needed; database was never created\n", db.key())
			return nil
		}
		if err := applyApplicationTo(root, db, st, parent.name); err != nil {
			return err
		}
		identity, exists, err := identifyDirectory(original)
		if err != nil || !exists {
			return fmt.Errorf("%s: database identity unavailable: %v", db.key(), err)
		}
		var id [8]byte
		if _, err := rand.Read(id[:]); err != nil {
			return err
		}
		trash := "trashed_db_" + hex.EncodeToString(id[:]) + "_" + db.name
		if len(trash) > 64 {
			return fmt.Errorf("%s: trash name exceeds backend limit", db.key())
		}
		if _, exists, err := identifyDirectory(filepath.Join(root, "db", trash)); err != nil || exists {
			return fmt.Errorf("%s: trash destination is unavailable", db.key())
		}
		insert := fmt.Sprintf("INSERT INTO 4sight_migration.database_deletions (database_type,original_name,trash_name,migration_id,state,physical_device,physical_inode) VALUES ('%s','%s','%s','%s','retiring',%d,%d)", db.dbType, db.name, trash, item.id, identity.device, identity.inode)
		if err := st.run("sql", "-q", insert); err != nil {
			return fmt.Errorf("persist retirement intent: %w", err)
		}
		rows, err = st.queryCSV(query)
		if err != nil || len(rows) != 1 {
			return fmt.Errorf("read retirement intent for %s: %v", db.key(), err)
		}
	}
	row := rows[0]
	if len(row) < 4 {
		return fmt.Errorf("%s: malformed retirement record", db.key())
	}
	trash := row[0]
	if err := ident(trash); err != nil || !strings.HasPrefix(trash, "trashed_db_") {
		return fmt.Errorf("%s: unsafe recorded trash name", db.key())
	}
	device, e1 := strconv.ParseUint(row[1], 10, 64)
	inode, e2 := strconv.ParseUint(row[2], 10, 64)
	if e1 != nil || e2 != nil {
		return fmt.Errorf("%s: malformed physical identity", db.key())
	}
	if row[3] == "collected" || row[3] == "collecting" {
		return nil
	}
	if row[3] != "retiring" && row[3] != "trashed" {
		return fmt.Errorf("%s: retirement state %q needs reconciliation", db.key(), row[3])
	}
	trashPath := filepath.Join(root, "db", trash)
	origID, origExists, err := identifyDirectory(original)
	if err != nil {
		return err
	}
	trashID, trashExists, err := identifyDirectory(trashPath)
	if err != nil {
		return err
	}
	if origExists == trashExists {
		return fmt.Errorf("%s: original and trash existence are inconsistent; reconcile manually", db.key())
	}
	expected := physicalID{device, inode}
	if origExists {
		if origID != expected {
			return fmt.Errorf("%s: original physical identity changed", db.key())
		}
		if err := os.Rename(original, trashPath); err != nil {
			return fmt.Errorf("%s: move to trash: %w", db.key(), err)
		}
		if err := intent.SyncDirectory(filepath.Dir(trashPath)); err != nil {
			return fmt.Errorf("%s: synchronize trash move: %w", db.key(), err)
		}
		trashID, trashExists, err = identifyDirectory(trashPath)
		if err != nil || !trashExists {
			return fmt.Errorf("%s: verify move: %v", db.key(), err)
		}
	}
	if trashID != expected {
		return fmt.Errorf("%s: trash physical identity changed", db.key())
	}
	update := fmt.Sprintf("UPDATE 4sight_migration.database_deletions SET trashed_at=COALESCE(trashed_at,NOW()) WHERE trash_name='%s'", trash)
	if err := st.run("sql", "-q", update); err != nil {
		return err
	}
	rows, err = st.queryCSV("SELECT id FROM 4sight_migration.`" + db.name + "` WHERE id='" + item.id + "'")
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		if err := st.insertApplied(db.name, item.id, time.Now().UTC().Truncate(time.Second)); err != nil {
			return err
		}
	}
	current, err := resolveLink(dir, "current")
	if err != nil {
		return err
	}
	if current != item.name {
		if current != parent.name {
			return fmt.Errorf("%s: unsafe current link %s", db.key(), current)
		}
		if err := updateCurrent(dir, item.name); err != nil {
			return err
		}
	}
	if err := st.run("sql", "-q", fmt.Sprintf("UPDATE 4sight_migration.database_deletions SET state='trashed' WHERE trash_name='%s'", trash)); err != nil {
		return err
	}
	fmt.Printf("retired %s as %s\n", db.key(), trash)
	return nil
}

func validateRetirementHistory(db database, st *doltStore, files map[string]*migrationFile, retirement *migrationFile, completed bool) error {
	var chain []*migrationFile
	for item := retirement; ; item = files[item.parentID] {
		if item == nil {
			return fmt.Errorf("%s: incomplete retirement chain", db.key())
		}
		chain = append(chain, item)
		if item.id == initialID {
			break
		}
		if len(chain) > len(files) {
			return fmt.Errorf("%s: cyclic retirement chain", db.key())
		}
	}
	rows, err := st.queryCSV("SELECT id, time FROM 4sight_migration.`" + db.name + "`")
	if err != nil {
		return fmt.Errorf("%s: read retirement history: %w", db.key(), err)
	}
	present := make(map[string]bool, len(rows))
	for _, row := range rows {
		if len(row) != 2 {
			return fmt.Errorf("%s: malformed retirement history", db.key())
		}
		if _, err := time.Parse("2006-01-02 15:04:05", row[1]); err != nil {
			return fmt.Errorf("%s: invalid retirement history timestamp", db.key())
		}
		present[row[0]] = true
	}
	// Every preceding migration must already be recorded. The retirement ID
	// itself may be absent only while reconciliation is in progress.
	if completed && !present[retirement.id] {
		return fmt.Errorf("%s: completed retirement is missing its migration record", db.key())
	}
	delete(present, retirement.id)
	for _, item := range chain[1:] {
		if !present[item.id] {
			return fmt.Errorf("%s: retirement history is incomplete at %s", db.key(), item.id)
		}
		delete(present, item.id)
	}
	if len(present) != 0 {
		return fmt.Errorf("%s: retirement history contains an unknown migration", db.key())
	}
	return nil
}
