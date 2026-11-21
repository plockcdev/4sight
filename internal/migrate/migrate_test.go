package migrate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDoltStoreRoundTrip(t *testing.T) {
	store, _ := newTestDolt(t)
	if err := store.run("sql", "-q", "CREATE DATABASE 4sight_migration"); err != nil {
		t.Fatal(err)
	}
	if err := store.run("sql", "-q", "CREATE TABLE 4sight_migration.spec (id CHAR(8) NOT NULL PRIMARY KEY, time DATETIME NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	exists, err := store.databaseExists("4sight_migration")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("database missing")
	}
	ok, err := store.trackingTableExists("spec")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("table missing")
	}
	at := time.Date(2026, 9, 6, 16, 13, 9, 0, time.UTC)
	if err := store.insertApplied("spec", "00000000", at); err != nil {
		t.Fatal(err)
	}
	rows, err := store.queryCSV("SELECT id, time FROM 4sight_migration.spec")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0][0] != "00000000" || rows[0][1] != "2026-09-06 16:13:09" {
		t.Fatalf("rows %v", rows)
	}
}

func newTestDolt(t *testing.T) (*doltStore, string) {
	t.Helper()
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not installed")
	}
	dir := t.TempDir()
	cmd := exec.Command("dolt", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}
	return newDoltStore(dir), dir
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func assertLink(t *testing.T, link, want string) {
	t.Helper()
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("link %s: got %s want %s", link, got, want)
	}
}
