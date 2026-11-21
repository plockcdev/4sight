package init

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"foursight.dev/4sight/internal/spec"
	"time"
)

func TestUpgradePriorInternalDatabase(t *testing.T) {
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt is unavailable")
	}
	root := filepath.Join(t.TempDir(), "application")
	database := filepath.Join(root, ".4sight", "db")
	if err := os.MkdirAll(database, 0755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", root}, {"-C", root, "remote", "add", "origin", "https://example.com/application.git"}} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, output)
		}
	}
	cmd := exec.Command("dolt", "init", "--name", "test", "--email", "test@example.com")
	cmd.Dir = database
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %v: %s", err, output)
	}
	old := fstest.MapFS{}
	err := fs.WalkDir(embedded, "migrations", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Base(path) == "00022367-00016920.sql" {
			return nil
		}
		data, err := embedded.ReadFile(path)
		if err == nil {
			old[path] = &fstest.MapFile{Data: data}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := bootstrapDatabases(database, old, time.Now); err != nil {
		t.Fatal(err)
	}
	store := doltBootstrap{dataDir: database}
	if err := store.run("sql", "-q", "CREATE DATABASE application_data; CREATE TABLE application_data.items (id INT PRIMARY KEY); INSERT INTO application_data.items VALUES (42); CREATE TABLE 4sight_migration.application_data (id CHAR(8) NOT NULL PRIMARY KEY, time DATETIME NOT NULL); INSERT INTO 4sight_migration.application_data VALUES ('00000000', NOW());"); err != nil {
		t.Fatal(err)
	}
	if err := preflightDatabase(database); err == nil {
		t.Fatal("expected old history to require upgrade")
	}
	if err := Upgrade(root); err != nil {
		t.Fatal(err)
	}
	if err := preflightDatabase(database); err != nil {
		t.Fatal(err)
	}
	data, err := store.queryCSV("SELECT id FROM application_data.items")
	if err != nil || len(data) != 1 || data[0][0] != "42" {
		t.Fatalf("application data lost: %v %v", data, err)
	}
	history, err := store.queryCSV("SELECT id FROM 4sight_migration.application_data")
	if err != nil || len(history) != 1 || history[0][0] != "00000000" {
		t.Fatalf("application history lost: %v %v", history, err)
	}
	rows, err := (doltBootstrap{dataDir: database}).queryCSV("SELECT id FROM 4sight_migration.4sight_migration WHERE id = '00022367'")
	if err != nil || len(rows) != 1 {
		t.Fatalf("migration record: %v %v", rows, err)
	}
	if err := Upgrade(root); err != nil {
		t.Fatalf("repeat upgrade: %v", err)
	}
}

func TestSelfMigrationRequiresMatchingSubmittedSources(t *testing.T) {
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt is unavailable")
	}
	root := filepath.Join(t.TempDir(), "4sight")
	db := filepath.Join(root, ".4sight", "db")
	if err := os.MkdirAll(db, 0755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", root}, {"-C", root, "remote", "add", "origin", "https://example.com/4sight.git"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, out)
		}
	}
	cmd := exec.Command("dolt", "init", "--name", "test", "--email", "test@example.com")
	cmd.Dir = db
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %v: %s", err, out)
	}
	old := fstest.MapFS{}
	if err := fs.WalkDir(embedded, "migrations", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Base(path) == "00022367-00016920.sql" {
			return nil
		}
		data, err := embedded.ReadFile(path)
		if err == nil {
			old[path] = &fstest.MapFile{Data: data}
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapDatabases(db, old, time.Now); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(root, ".4sight")
	if err := os.WriteFile(filepath.Join(metadata, "design.md"), []byte("test spec\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := spec.ChangeImport(root); err != nil {
		t.Fatal(err)
	}
	if err := spec.ChangeSubmit(root); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSelf(root); err == nil {
		t.Fatal("missing source chains must fail")
	}
	rows, err := (doltBootstrap{dataDir: db}).queryCSV("SELECT id FROM 4sight_migration.4sight_migration WHERE id = '00022367'")
	if err != nil || len(rows) != 0 {
		t.Fatalf("migration applied despite invalid sources: %v %v", rows, err)
	}
	if err := fs.WalkDir(embedded, "migrations", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(metadata, path)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := embedded.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ name, current, submitted string }{
		{"4sight_migration", "00016920-00000000.sql", "00022367-00016920.sql"},
		{"spec", "00113758-00101071.sql", "00113758-00101071.sql"},
	} {
		dir := filepath.Join(metadata, "migrations", "dolt_"+item.name)
		if err := os.Symlink(item.current, filepath.Join(dir, "current")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(item.submitted, filepath.Join(dir, "submitted")); err != nil {
			t.Fatal(err)
		}
	}
	if err := MigrateSelf(root); err != nil {
		t.Fatal(err)
	}
	if err := preflightDatabase(db); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(metadata, "migrations", "dolt_4sight_migration", "current"))
	if err != nil || target != "00022367-00016920.sql" {
		t.Fatalf("current link: %q %v", target, err)
	}
}

func TestInitPreservesDirtyWorkspaceAndRejectsExisting(t *testing.T) {
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt unavailable")
	}
	root := filepath.Join(t.TempDir(), "application")
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("uncommitted\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := run(root, embedded, io.Discard); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "unrelated.txt"))
	if err != nil || string(body) != "uncommitted\n" {
		t.Fatalf("workspace changed: %q %v", body, err)
	}
	cmd := exec.Command("git", "-C", root, "diff", "--cached", "--name-only")
	out, err := cmd.Output()
	if err != nil || len(out) != 0 {
		t.Fatalf("initialization staged files: %q %v", out, err)
	}
	if err := run(root, embedded, io.Discard); err == nil || !strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("second init: %v", err)
	}
}
