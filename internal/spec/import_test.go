package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fakeStore struct {
	bodies []string
}

func (f *fakeStore) insertBody(body string) error {
	f.bodies = append(f.bodies, body)
	return nil
}

func TestImportInsertsBody(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "specs", "design.md"), "# Spec\nbody text\n")

	st := &fakeStore{}
	if err := runImport(root, "design.md", st); err != nil {
		t.Fatal(err)
	}
	if len(st.bodies) != 1 || st.bodies[0] != "# Spec\nbody text\n" {
		t.Fatalf("bodies %q", st.bodies)
	}
}

func TestImportNestedPath(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "specs", "subcommands", "migrate", "cmd.md"), "import me\n")

	st := &fakeStore{}
	if err := runImport(root, "subcommands/migrate/cmd.md", st); err != nil {
		t.Fatal(err)
	}
	if len(st.bodies) != 1 || st.bodies[0] != "import me\n" {
		t.Fatalf("bodies %q", st.bodies)
	}
}

func TestImportFailsWhenMissing(t *testing.T) {
	root := newMetadata(t)
	err := runImport(root, "design.md", &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "spec file .4sight/specs/design.md does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestImportFailsWhenDirectory(t *testing.T) {
	root := newMetadata(t)
	if err := os.MkdirAll(filepath.Join(root, "specs", "design.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := runImport(root, "design.md", &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestImportRejectsAbsolutePath(t *testing.T) {
	root := newMetadata(t)
	err := runImport(root, "/tmp/design.md", &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "relative to specs dir") {
		t.Fatalf("got %v", err)
	}
}

func TestImportRejectsTraversal(t *testing.T) {
	root := newMetadata(t)
	err := runImport(root, "../secrets.md", &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "relative to specs dir") {
		t.Fatalf("got %v", err)
	}
}

func TestImportRejectsEmptyPath(t *testing.T) {
	root := newMetadata(t)
	err := runImport(root, "  ", &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "missing spec file path") {
		t.Fatalf("got %v", err)
	}
}

func TestFindMetadataRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".4sight")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findMetadataRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("root %q want %q", got, want)
	}
}

func TestImportIntegration(t *testing.T) {
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not installed")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, ".4sight")
	dbDir := filepath.Join(root, "db")
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "specs", "cmd.md"), "quote ' and\nnewline\n")

	cmd := exec.Command("dolt", "init")
	cmd.Dir = dbDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}
	store := newDoltStore(dbDir)
	if err := store.run("sql", "-q", "CREATE DATABASE spec"); err != nil {
		t.Fatal(err)
	}
	if err := store.run("sql", "-q", "CREATE TABLE spec.bodies (body TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}

	if err := Import(dir, "cmd.md"); err != nil {
		t.Fatal(err)
	}
	rows, err := store.queryCSV("SELECT body FROM spec.bodies")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0][0] != "quote ' and\nnewline\n" {
		t.Fatalf("rows %v", rows)
	}
}

func newMetadata(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".4sight")
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "db"), 0o755); err != nil {
		t.Fatal(err)
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
