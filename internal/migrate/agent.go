package migrate

import (
	"fmt"
	intentpkg "foursight.dev/4sight/internal/intent"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentUp applies prepared SQL chains to ordinary application databases. The
// caller holds the project lock and validates the submitted specification.
func AgentUp(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	st := newDoltStore(filepath.Join(root, "db"))
	if err := reconcilePendingRetirements(root, st); err != nil {
		return err
	}
	retired, err := st.queryCSV("SELECT p.name FROM spec.bodies b JOIN spec.file_changes f ON f.id=b.file_change_id JOIN spec.filepaths p ON p.id=f.filepath_id JOIN spec.dirs d ON d.id=p.dir_id LEFT JOIN spec.bodies s ON s.file_id=b.file_id AND s.state='submitted' LEFT JOIN spec.file_changes sf ON sf.id=s.file_change_id LEFT JOIN spec.filepaths sp ON sp.id=sf.filepath_id LEFT JOIN spec.dirs sd ON sd.id=sp.dir_id WHERE b.state='implemented' AND b.deleted=0 AND d.path='schemas' AND (s.file_id IS NULL OR s.deleted=1 OR sd.path<>'schemas' OR sp.name<>p.name)")
	if err != nil {
		return fmt.Errorf("inspect submitted schema retirements: %w", err)
	}
	var retiring []database
	for _, row := range retired {
		stem, ok := strings.CutSuffix(row[0], ".md")
		kind, name, valid := strings.Cut(stem, "_")
		if !ok || !valid || kind != "dolt" || name == "" || name == "spec" || name == "4sight_migration" {
			return fmt.Errorf("invalid or internal retired schema %s", row[0])
		}
		if err := ident(name); err != nil {
			return err
		}
		retiring = append(retiring, database{dbType: kind, name: name})
	}
	dbs, err := listSchemaDatabases(filepath.Join(root, "schemas"))
	if err != nil {
		return err
	}
	names := map[string]bool{}
	for _, db := range dbs {
		key := strings.ToLower(db.name)
		if (key == "spec" || key == "4sight_migration") && (db.dbType != "dolt" || db.name != key) {
			return fmt.Errorf("database name %q collides with an internal tracking table", db.name)
		}
		if names[key] {
			return fmt.Errorf("managed database name %q collides with another tracking table", db.name)
		}
		names[key] = true
	}
	for _, db := range dbs {
		if strings.EqualFold(db.name, "4sight_migration") || strings.EqualFold(db.name, "spec") {
			continue
		}
		if err := precheckApplication(root, db, st, ""); err != nil {
			return err
		}
	}
	for _, db := range retiring {
		exists, err := st.databaseExists(db.name)
		if err != nil {
			return err
		}
		if !exists {
			tracked, err := st.trackingTableExists(db.name)
			if err != nil {
				return err
			}
			if tracked {
				rows, err := st.queryCSV("SELECT trash_name FROM 4sight_migration.database_deletions WHERE database_type='" + db.dbType + "' AND original_name='" + db.name + "'")
				if err != nil || len(rows) == 0 {
					return fmt.Errorf("%s: absent database has tracking without a retirement record; reconcile before proceeding", db.key())
				}
			}
			continue
		}
		dir := filepath.Join(root, "migrations", db.key())
		name, err := resolveLink(dir, "submitted")
		if err != nil {
			return err
		}
		files, children, err := listMigrationFiles(dir)
		if err != nil {
			return err
		}
		item, ok := fileByName(files, name)
		if !ok || !strings.HasSuffix(name, ".delete") || len(children[item.id]) != 0 {
			return fmt.Errorf("%s: missing final retirement entry", db.key())
		}
		parent := files[item.parentID]
		if parent == nil {
			return fmt.Errorf("%s: missing retirement parent", db.key())
		}
		if err := precheckApplication(root, db, st, parent.name); err != nil {
			return err
		}
	}
	for _, db := range retiring {
		if err := retireDatabase(root, db, st); err != nil {
			return err
		}
	}
	for _, db := range dbs {
		if strings.EqualFold(db.name, "4sight_migration") || strings.EqualFold(db.name, "spec") {
			continue
		}
		if db.dbType != "dolt" {
			return fmt.Errorf("unsupported database type %s", db.dbType)
		}
		if err := applyApplicationChain(root, db, st); err != nil {
			return err
		}
	}
	return nil
}

func precheckApplication(root string, db database, st *doltStore, target string) error {
	if db.dbType != "dolt" {
		return fmt.Errorf("unsupported database type %s", db.dbType)
	}
	if err := ident(db.name); err != nil {
		return err
	}
	dir := filepath.Join(root, "migrations", db.key())
	files, children, err := listMigrationFiles(dir)
	if err != nil {
		return err
	}
	name, err := resolveLink(dir, "submitted")
	if err != nil {
		return err
	}
	if target != "" {
		name = target
	}
	end, ok := fileByName(files, name)
	if !ok || strings.HasSuffix(name, ".delete") {
		return fmt.Errorf("%s: invalid live submitted chain", db.key())
	}
	if err := validateSubmittedToCurrent(files, children, end, nil); err != nil {
		return err
	}
	if first := files[initialID]; first == nil || first.name != initialMigrationFile {
		return fmt.Errorf("%s: missing valid initial migration", db.key())
	}
	var chain []*migrationFile
	for item := end; ; item = files[item.parentID] {
		chain = append(chain, item)
		if item.id == initialID {
			break
		}
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	exists, err := st.databaseExists(db.name)
	if err != nil {
		return err
	}
	tracked, err := st.trackingTableExists(db.name)
	if err != nil {
		return err
	}
	intentPath := filepath.Join(root, "db", ".4sight-app-"+db.key()+".intent")
	intent, intentErr := os.ReadFile(intentPath)
	if intentErr != nil && !os.IsNotExist(intentErr) {
		return intentErr
	}
	if exists && !tracked {
		return fmt.Errorf("%s: existing database has no trustworthy tracking; restore or reconcile its history", db.key())
	}
	if !tracked {
		if len(intent) > 0 {
			return fmt.Errorf("%s: interrupted operation needs reconciliation", db.key())
		}
		if _, err := os.Lstat(filepath.Join(dir, "current")); !os.IsNotExist(err) {
			return fmt.Errorf("%s: unexpected current link for new database", db.key())
		}
		return nil
	}
	rows, err := st.queryCSV("SELECT id, time FROM 4sight_migration.`" + db.name + "`")
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for _, row := range rows {
		if len(row) != 2 {
			return fmt.Errorf("%s: malformed migration history", db.key())
		}
		if _, err := time.Parse("2006-01-02 15:04:05", row[1]); err != nil {
			return fmt.Errorf("%s: malformed migration timestamp for %s", db.key(), row[0])
		}
		present[row[0]] = true
	}
	applied := 0
	for _, item := range chain {
		if !present[item.id] {
			break
		}
		delete(present, item.id)
		applied++
	}
	if len(present) > 0 || (exists && applied == 0) {
		return fmt.Errorf("%s: migration history is incomplete or divergent", db.key())
	}
	if applied > 0 {
		link, err := resolveLink(dir, "current")
		if err != nil {
			if !os.IsNotExist(err) || len(intent) == 0 {
				return fmt.Errorf("%s: unsafe current link: %w", db.key(), err)
			}
		} else {
			found := false
			for _, item := range chain[:applied] {
				if item.name == link {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("%s: current link is not in recorded history", db.key())
			}
		}
	}
	if !exists {
		if intentErr != nil || (string(intent) != "create tracking table\n" && strings.TrimSpace(string(intent)) != initialID) {
			return fmt.Errorf("%s: tracking exists without its database; reconcile before proceeding", db.key())
		}
	} else if len(intent) > 0 && (applied == 0 || strings.TrimSpace(string(intent)) != chain[applied-1].id) {
		return fmt.Errorf("%s: interrupted operation needs reconciliation", db.key())
	}
	return nil
}

func applyApplicationChain(root string, db database, st *doltStore) error {
	return applyApplicationTo(root, db, st, "")
}

func applyApplicationTo(root string, db database, st *doltStore, target string) error {
	if err := ident(db.name); err != nil {
		return err
	}
	dir := filepath.Join(root, "migrations", db.key())
	files, children, err := listMigrationFiles(dir)
	if err != nil {
		return err
	}
	submittedName, err := resolveLink(dir, "submitted")
	if err != nil {
		return err
	}
	if target != "" {
		submittedName = target
	}
	submitted, ok := fileByName(files, submittedName)
	if !ok {
		return fmt.Errorf("%s: invalid submitted migration", db.key())
	}
	if err := validateSubmittedToCurrent(files, children, submitted, nil); err != nil {
		return err
	}
	if first := files[initialID]; first == nil || first.name != initialMigrationFile {
		return fmt.Errorf("%s: missing valid initial migration", db.key())
	}
	var chain []*migrationFile
	for item := submitted; ; item = files[item.parentID] {
		chain = append(chain, item)
		if item.id == initialID {
			break
		}
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	exists, err := st.databaseExists(db.name)
	if err != nil {
		return err
	}
	tracked, err := st.trackingTableExists(db.name)
	if err != nil {
		return err
	}
	intent := filepath.Join(root, "db", ".4sight-app-"+db.key()+".intent")
	intentBody, intentErr := os.ReadFile(intent)
	if intentErr != nil && !os.IsNotExist(intentErr) {
		return intentErr
	}
	if exists && !tracked {
		return fmt.Errorf("%s: database exists without trustworthy tracking; restore or reconcile its history", db.key())
	}
	if !exists && tracked && string(intentBody) != "create tracking table\n" && strings.TrimSpace(string(intentBody)) != initialID {
		return fmt.Errorf("%s: tracking exists but database is absent; reconcile the lifecycle", db.key())
	}
	if !exists && tracked && (string(intentBody) == "create tracking table\n" || strings.TrimSpace(string(intentBody)) == initialID) {
		rows, err := st.queryCSV("SHOW CREATE TABLE 4sight_migration.`" + db.name + "`")
		if err != nil || len(rows) != 1 || len(rows[0]) < 2 || !strings.Contains(rows[0][1], "`id` char(8) NOT NULL") || !strings.Contains(rows[0][1], "`time` datetime NOT NULL") || !strings.Contains(rows[0][1], "PRIMARY KEY (`id`)") {
			return fmt.Errorf("%s: interrupted tracking table creation requires repair", db.key())
		}
		columns, err := st.queryCSV("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='4sight_migration' AND table_name='" + db.name + "'")
		if err != nil || len(columns) != 1 || len(columns[0]) != 1 || columns[0][0] != "2" {
			return fmt.Errorf("%s: interrupted tracking table definition differs", db.key())
		}
		rows, err = st.queryCSV("SELECT id FROM 4sight_migration.`" + db.name + "`")
		if err != nil || len(rows) != 0 {
			return fmt.Errorf("%s: interrupted tracking table has conflicting history", db.key())
		}
		intentBody = nil
	}
	if !tracked {
		if err := intentpkg.Write(intent, []byte("create tracking table\n")); err != nil {
			return err
		}
		if err := ident(db.name); err != nil {
			return err
		}
		query := fmt.Sprintf("CREATE TABLE 4sight_migration.`%s` (id CHAR(8) NOT NULL PRIMARY KEY, time DATETIME NOT NULL)", db.name)
		if err := st.run("sql", "-q", query); err != nil {
			return fmt.Errorf("create tracking table: %w; reconcile %s before retry", err, intent)
		}
	}
	rows, err := st.queryCSV("SELECT id, time FROM 4sight_migration.`" + db.name + "`")
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for _, row := range rows {
		if len(row) != 2 {
			return fmt.Errorf("%s: malformed migration history", db.key())
		}
		if _, err := time.Parse("2006-01-02 15:04:05", row[1]); err != nil {
			return fmt.Errorf("%s: malformed migration timestamp for %s", db.key(), row[0])
		}
		present[row[0]] = true
	}
	applied := 0
	for _, item := range chain {
		if !present[item.id] {
			break
		}
		delete(present, item.id)
		applied++
	}
	if len(present) > 0 || (exists && applied == 0) {
		return fmt.Errorf("%s: migration history is incomplete or divergent; reconcile before retrying", db.key())
	}
	if len(intentBody) > 0 {
		if applied == 0 || strings.TrimSpace(string(intentBody)) != chain[applied-1].id {
			return fmt.Errorf("%s: interrupted migration intent %s requires reconciliation", db.key(), intent)
		}
	}
	if applied > 0 {
		link, err := resolveLink(dir, "current")
		if err != nil {
			if os.IsNotExist(err) && len(intentBody) > 0 {
				if err := updateCurrent(dir, chain[applied-1].name); err != nil {
					return err
				}
				link = chain[applied-1].name
			} else {
				return fmt.Errorf("%s: current link needs reconciliation: %w", db.key(), err)
			}
		}
		if link != chain[applied-1].name {
			found := false
			for _, item := range chain[:applied] {
				if item.name == link {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("%s: unsafe current link %s", db.key(), link)
			}
			if err := updateCurrent(dir, chain[applied-1].name); err != nil {
				return err
			}
		}
	} else if _, err := os.Lstat(filepath.Join(dir, "current")); !os.IsNotExist(err) {
		return fmt.Errorf("%s: unexpected current link", db.key())
	}
	if len(intentBody) > 0 {
		if err := intentpkg.Remove(intent); err != nil {
			return err
		}
	}
	for _, item := range chain[applied:] {
		if strings.HasSuffix(item.name, ".delete") {
			return fmt.Errorf("%s: retirement requires the retirement workflow", db.key())
		}
		if err := intentpkg.Write(intent, []byte(item.id+"\n")); err != nil {
			return err
		}
		if err := st.applyFile(item.path); err != nil {
			return fmt.Errorf("%s: apply %s: %w; reconcile %s", db.key(), item.id, err, intent)
		}
		if err := st.insertApplied(db.name, item.id, time.Now().UTC().Truncate(time.Second)); err != nil {
			return fmt.Errorf("%s: record %s: %w; reconcile %s", db.key(), item.id, err, intent)
		}
		if err := updateCurrent(dir, item.name); err != nil {
			return err
		}
		if err := intentpkg.Remove(intent); err != nil {
			return err
		}
		fmt.Printf("applied %s %s\n", db.key(), item.id)
	}
	return nil
}
