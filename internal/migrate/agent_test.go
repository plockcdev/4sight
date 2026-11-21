package migrate

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentUpCreatesTrackingBeforeApplication(t *testing.T) {
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt unavailable")
	}
	root := t.TempDir()
	meta := filepath.Join(root, ".4sight")
	db := filepath.Join(meta, "db")
	dir := filepath.Join(meta, "migrations", "dolt_example")
	for _, p := range []string{db, dir, filepath.Join(meta, "schemas")} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("dolt", "init", "--name", "test", "--email", "test@example.com")
	cmd.Dir = db
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %v: %s", err, out)
	}
	st := newDoltStore(db)
	if err := st.run("sql", "-q", "CREATE DATABASE 4sight_migration;"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(meta, "schemas", "dolt_example.md"), "example\n")
	writeFile(t, filepath.Join(dir, "00000000-00000000.sql"), "CREATE DATABASE example;\n")
	writeFile(t, filepath.Join(dir, "00010001-00000000.sql"), "CREATE TABLE example.things (id INT PRIMARY KEY);\n")
	mustSymlink(t, "00010001-00000000.sql", filepath.Join(dir, "submitted"))
	if err := applyApplicationChain(meta, database{dbType: "dolt", name: "example"}, st); err != nil {
		t.Fatal(err)
	}
	rows, err := st.queryCSV("SELECT id FROM 4sight_migration.example ORDER BY id")
	if err != nil || len(rows) != 2 {
		t.Fatalf("history %v: %v", rows, err)
	}
	assertLink(t, filepath.Join(dir, "current"), "00010001-00000000.sql")
	if err := applyApplicationChain(meta, database{dbType: "dolt", name: "example"}, st); err != nil {
		t.Fatal(err)
	}
	rogue := filepath.Join(meta, "migrations", "dolt_rogue")
	if err := os.MkdirAll(rogue, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(rogue, "00000000-00000000.sql"), "CREATE DATABASE rogue;\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(rogue, "submitted"))
	if err := st.run("sql", "-q", "CREATE DATABASE rogue"); err != nil {
		t.Fatal(err)
	}
	if err := precheckApplication(meta, database{dbType: "dolt", name: "rogue"}, st, ""); err == nil {
		t.Fatal("existing untracked database must be rejected")
	}
	retry := filepath.Join(meta, "migrations", "dolt_retry")
	if err := os.MkdirAll(retry, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(retry, "00000000-00000000.sql"), "CREATE DATABASE retry;\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(retry, "submitted"))
	if err := st.run("sql", "-q", "CREATE TABLE 4sight_migration.retry (id CHAR(8) NOT NULL PRIMARY KEY, time DATETIME NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(db, ".4sight-app-dolt_retry.intent"), "create tracking table\n")
	if err := applyApplicationChain(meta, database{dbType: "dolt", name: "retry"}, st); err != nil {
		t.Fatalf("retry interrupted tracking creation: %v", err)
	}
}

func TestRetirementAndCollectionRecovery(t *testing.T) {
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt unavailable")
	}
	root := t.TempDir()
	meta := filepath.Join(root, ".4sight")
	dbpath := filepath.Join(meta, "db")
	dir := filepath.Join(meta, "migrations", "dolt_example")
	for _, p := range []string{dbpath, dir} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("dolt", "init", "--name", "test", "--email", "test@example.com")
	cmd.Dir = dbpath
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %v: %s", err, out)
	}
	st := newDoltStore(dbpath)
	if err := st.run("sql", "-q", `CREATE DATABASE 4sight_migration; CREATE TABLE 4sight_migration.database_deletions (id BIGINT AUTO_INCREMENT PRIMARY KEY, database_type VARCHAR(32) NOT NULL, original_name VARCHAR(64) NOT NULL, trash_name VARCHAR(64) NOT NULL UNIQUE, migration_id CHAR(8) NOT NULL, trashed_at DATETIME, garbage_collected_at DATETIME, state VARCHAR(32) NOT NULL, physical_device BIGINT UNSIGNED NOT NULL, physical_inode BIGINT UNSIGNED NOT NULL, UNIQUE KEY (database_type, original_name, migration_id));`); err != nil {
		t.Fatal(err)
	}
	if err := st.run("sql", "-q", "CREATE DATABASE spec; CREATE TABLE spec.dirs (id INT PRIMARY KEY, path VARCHAR(255)); CREATE TABLE spec.filepaths (id INT PRIMARY KEY, dir_id INT, name VARCHAR(255)); CREATE TABLE spec.file_changes (id INT PRIMARY KEY, filepath_id INT); CREATE TABLE spec.bodies (file_change_id INT, state VARCHAR(32), deleted BOOL);"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "00000000-00000000.sql"), "CREATE DATABASE example;\n")
	writeFile(t, filepath.Join(dir, "00010001-00000000.sql"), "CREATE TABLE example.things (id INT PRIMARY KEY); INSERT INTO example.things VALUES (7);\n")
	mustSymlink(t, "00010001-00000000.sql", filepath.Join(dir, "submitted"))
	app := database{dbType: "dolt", name: "example"}
	if err := applyApplicationChain(meta, app, st); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "00020002-00010001.delete"), "")
	if err := os.Remove(filepath.Join(dir, "submitted")); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, "00020002-00010001.delete", filepath.Join(dir, "submitted"))
	if err := retireDatabase(meta, app, st); err != nil {
		t.Fatal(err)
	}
	input, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString("yes\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	if err := GarbageCollect(root, "", input, io.Discard); err == nil {
		t.Fatal("noninteractive input must cancel collection")
	}
	_ = input.Close()
	records, err := collectionRecords(st)
	if err != nil || len(records) != 1 {
		t.Fatalf("records %v: %v", records, err)
	}
	rows, err := st.queryCSV("SELECT id FROM `" + records[0].trash + "`.things")
	if err != nil || len(rows) != 1 || rows[0][0] != "7" {
		t.Fatalf("trashed data %v: %v", rows, err)
	}
	// Simulate interruption after the move but before tracking and current-link
	// completion. An unrelated schema discovery error must not block recovery.
	if err := st.run("sql", "-q", "UPDATE 4sight_migration.database_deletions SET state='retiring'; DELETE FROM 4sight_migration.example WHERE id='00020002'"); err != nil {
		t.Fatal(err)
	}
	if err := updateCurrent(dir, "00010001-00000000.sql"); err != nil {
		t.Fatal(err)
	}
	if err := st.run("sql", "-q", "DELETE FROM 4sight_migration.example WHERE id='00010001'"); err != nil {
		t.Fatal(err)
	}
	if err := AgentUp(root); err == nil || !strings.Contains(err.Error(), "history is incomplete") {
		t.Fatalf("expected incomplete retirement history error: %v", err)
	}
	assertLink(t, filepath.Join(dir, "current"), "00010001-00000000.sql")
	if err := st.run("sql", "-q", "INSERT INTO 4sight_migration.example (id,time) VALUES ('00010001',NOW())"); err != nil {
		t.Fatal(err)
	}
	if err := AgentUp(root); err == nil {
		t.Fatal("expected unrelated schema discovery failure")
	}
	assertLink(t, filepath.Join(dir, "current"), "00020002-00010001.delete")
	rows, err = st.queryCSV("SELECT id FROM 4sight_migration.example WHERE id='00020002'")
	if err != nil || len(rows) != 1 {
		t.Fatalf("retirement was not reconciled: %v %v", rows, err)
	}
	if err := retireDatabase(meta, app, st); err != nil {
		t.Fatal(err)
	}
	if err := st.run("sql", "-q", "UPDATE 4sight_migration.database_deletions SET state='collecting'"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dbpath, records[0].trash)); err != nil {
		t.Fatal(err)
	}
	records, err = collectionRecords(st)
	if err != nil || len(records) != 1 {
		t.Fatalf("collect intent %v: %v", records, err)
	}
	if err := collectOne(meta, st, records[0]); err != nil {
		t.Fatal(err)
	}
	records, err = collectionRecords(st)
	if err != nil || len(records) != 0 {
		t.Fatalf("collected records %v: %v", records, err)
	}
}
