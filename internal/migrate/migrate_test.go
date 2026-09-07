package migrate

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	dbs     map[string]bool
	tables  map[string]bool
	applied []string
	rows    []string
	onApply func(path string) error
}

func (f *fakeStore) databaseExists(name string) (bool, error) {
	return f.dbs[name], nil
}

func (f *fakeStore) applyFile(path string) error {
	f.applied = append(f.applied, filepath.Base(path))
	if f.onApply != nil {
		return f.onApply(path)
	}
	return nil
}

func (f *fakeStore) trackingTableExists(table string) (bool, error) {
	return f.tables[table], nil
}

func (f *fakeStore) insertApplied(table, id string, at time.Time) error {
	f.rows = append(f.rows, table+" "+id+" "+at.Format("2006-01-02 15:04:05"))
	return nil
}

func TestUpAppliesInitialMigration(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "CREATE DATABASE 4sight_migration;\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))

	st := &fakeStore{
		dbs:    map[string]bool{},
		tables: map[string]bool{},
	}
	st.onApply = func(string) error {
		st.dbs["4sight_migration"] = true
		st.tables["4sight_migration"] = true
		return nil
	}
	fixed := time.Date(2026, 9, 6, 16, 13, 9, 123, time.UTC)
	var out bytes.Buffer
	if err := runUp(root, st, func() time.Time { return fixed }, &out); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.applied, []string{"00000000-00000000.sql"}) {
		t.Fatalf("applied %v", st.applied)
	}
	if !slices.Equal(st.rows, []string{"4sight_migration 00000000 2026-09-06 16:13:09"}) {
		t.Fatalf("rows %v", st.rows)
	}
	if got := strings.TrimSpace(out.String()); got != "applied dolt_4sight_migration 00000000" {
		t.Fatalf("stdout %q", out.String())
	}
	assertLink(t, filepath.Join(mig, "current"), "00000000-00000000.sql")
}

func TestUpAppliesChainFromCurrent(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	writeFile(t, filepath.Join(mig, "00010001-00000000.sql"), "-- next\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "current"))
	mustSymlink(t, "00010001-00000000.sql", filepath.Join(mig, "submitted"))

	st := &fakeStore{
		dbs:    map[string]bool{"4sight_migration": true},
		tables: map[string]bool{"4sight_migration": true},
	}
	if err := runUp(root, st, func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }, ioDiscard()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.applied, []string{"00010001-00000000.sql"}) {
		t.Fatalf("applied %v", st.applied)
	}
	assertLink(t, filepath.Join(mig, "current"), "00010001-00000000.sql")
}

func TestUpNoopsWhenCurrentMatchesSubmitted(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "current"))
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))

	st := &fakeStore{
		dbs:    map[string]bool{"4sight_migration": true},
		tables: map[string]bool{"4sight_migration": true},
	}
	if err := runUp(root, st, time.Now, ioDiscard()); err != nil {
		t.Fatal(err)
	}
	if len(st.applied) != 0 {
		t.Fatalf("applied %v", st.applied)
	}
}

func TestUpFailsWhenInitialSQLMissing(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	err := runUp(root, &fakeStore{}, time.Now, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "missing 00000000-00000000.sql") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Join(root, "migrations", "dolt_4sight_migration")) {
		t.Fatalf("error missing dir: %v", err)
	}
}

func TestUpFailsWhenSubmittedMissing(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	err := runUp(root, &fakeStore{}, time.Now, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "missing submitted") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), mig) {
		t.Fatalf("error missing dir: %v", err)
	}
}

func TestUpFailsWhenSubmittedDangling(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	mustSymlink(t, "missing.sql", filepath.Join(mig, "submitted"))
	err := runUp(root, &fakeStore{}, time.Now, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "existing file") {
		t.Fatalf("got %v", err)
	}
}

func TestUpFailsWhenNoPathFromSubmittedToCurrent(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	writeFile(t, filepath.Join(mig, "00010001-00000000.sql"), "-- a\n")
	mustSymlink(t, "00010001-00000000.sql", filepath.Join(mig, "current"))
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))
	err := runUp(root, &fakeStore{dbs: map[string]bool{"4sight_migration": true}}, time.Now, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "no path through parent ids") {
		t.Fatalf("got %v", err)
	}
}

func TestUpFailsWhenParentIsAmbiguous(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	writeFile(t, filepath.Join(mig, "00010001-00000000.sql"), "-- a\n")
	writeFile(t, filepath.Join(mig, "00020002-00000000.sql"), "-- b\n")
	mustSymlink(t, "00010001-00000000.sql", filepath.Join(mig, "submitted"))
	err := runUp(root, &fakeStore{}, time.Now, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "multiple migrations have parent id 00000000") {
		t.Fatalf("got %v", err)
	}
}

func TestUpFailsWhenDatabaseExistsWithoutCurrent(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))
	err := runUp(root, &fakeStore{dbs: map[string]bool{"4sight_migration": true}}, time.Now, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "exists but current symlink is missing") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), mig) {
		t.Fatalf("error missing dir: %v", err)
	}
}

func TestUpFailsWhenTrackingTableMissing(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))
	err := runUp(root, &fakeStore{tables: map[string]bool{}}, time.Now, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "4sight_migration table 4sight_migration is missing") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), mig) {
		t.Fatalf("error missing dir: %v", err)
	}
}

func TestUpMigratesFourSightFirst(t *testing.T) {
	root := newMetadata(t, []string{"dolt_aaa.md", "dolt_4sight_migration.md"})
	for _, key := range []string{"dolt_4sight_migration", "dolt_aaa"} {
		mig := filepath.Join(root, "migrations", key)
		if err := os.MkdirAll(mig, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
		mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))
	}
	st := &fakeStore{tables: map[string]bool{
		"4sight_migration": true,
		"aaa":              true,
	}}
	if err := runUp(root, st, func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }, ioDiscard()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.applied, []string{"00000000-00000000.sql", "00000000-00000000.sql"}) {
		t.Fatalf("applied %v", st.applied)
	}
	if !slices.Equal(st.rows, []string{
		"4sight_migration 00000000 2026-01-01 00:00:00",
		"aaa 00000000 2026-01-01 00:00:00",
	}) {
		t.Fatalf("rows %v", st.rows)
	}
}

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

func TestUpIntegration(t *testing.T) {
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not installed")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, ".4sight")
	dbDir := filepath.Join(root, "db")
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	if err := os.MkdirAll(filepath.Join(root, "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(mig, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "schemas", "dolt_4sight_migration.md"), "tracking\n")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), `CREATE DATABASE 4sight_migration;

CREATE TABLE 4sight_migration.4sight_migration (
  id CHAR(8) NOT NULL PRIMARY KEY,
  time DATETIME NOT NULL
);
`)
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))
	cmd := exec.Command("dolt", "init")
	cmd.Dir = dbDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}

	if err := Up(dir); err != nil {
		t.Fatal(err)
	}
	assertLink(t, filepath.Join(mig, "current"), "00000000-00000000.sql")

	store := newDoltStore(dbDir)
	exists, err := store.databaseExists("4sight_migration")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("4sight_migration database missing")
	}
	ok, err := store.trackingTableExists("4sight_migration")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("tracking table missing")
	}
	rows, err := store.queryCSV("SELECT id FROM 4sight_migration.4sight_migration")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0][0] != "00000000" {
		t.Fatalf("rows %v", rows)
	}
	if err := Up(dir); err != nil {
		t.Fatal(err)
	}
	rows, err = store.queryCSV("SELECT id FROM 4sight_migration.4sight_migration")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("second up inserted extra rows: %v", rows)
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

func newMetadata(t *testing.T, schemas []string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".4sight")
	if err := os.MkdirAll(filepath.Join(root, "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "db"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range schemas {
		writeFile(t, filepath.Join(root, "schemas", name), "spec\n")
		key, _ := strings.CutSuffix(name, ".md")
		if err := os.MkdirAll(filepath.Join(root, "migrations", key), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
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

func ioDiscard() *bytes.Buffer {
	return &bytes.Buffer{}
}
