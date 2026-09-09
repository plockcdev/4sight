package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type importedSpec struct {
	dir, name, body, diff string
}

type fakeStore struct {
	specs []importedSpec
}

func (f *fakeStore) latestBody(dirPath, fileName string) (string, bool, error) {
	for _, v := range slices.Backward(f.specs) {
		if v.dir == dirPath && v.name == fileName {
			return v.body, true, nil
		}
	}
	return "", false, nil
}

func (f *fakeStore) insertSpec(dirPath, fileName, body, diff string) error {
	f.specs = append(f.specs, importedSpec{dir: dirPath, name: fileName, body: body, diff: diff})
	return nil
}

func (f *fakeStore) listLatestBodies() ([]specDoc, error) {
	latest := make(map[string]specDoc)
	var order []string
	for _, s := range f.specs {
		key := s.dir + "\x00" + s.name
		if _, ok := latest[key]; !ok {
			order = append(order, key)
		}
		latest[key] = specDoc{Dir: s.dir, Name: s.name, Body: s.body}
	}
	docs := make([]specDoc, 0, len(order))
	for _, key := range order {
		docs = append(docs, latest[key])
	}
	return docs, nil
}

func TestImportInsertsBody(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "cmd.md"), "# Spec\nbody text\n")

	st := &fakeStore{}
	if err := runImport(root, []string{"migrate", "cmd.md"}, st); err != nil {
		t.Fatal(err)
	}
	if len(st.specs) != 1 {
		t.Fatalf("specs %v", st.specs)
	}
	got := st.specs[0]
	if got.dir != "subcommands/migrate" || got.name != "cmd.md" || got.body != "# Spec\nbody text\n" {
		t.Fatalf("got %+v", got)
	}
	if !strings.Contains(got.diff, "+# Spec") {
		t.Fatalf("diff %q", got.diff)
	}
}

func TestImportNestedCommandPath(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "myprog", "subcommands", "foo", "subcommands", "bar", "cmd.md"), "import me\n")

	st := &fakeStore{}
	if err := runImport(root, []string{"myprog", "foo", "bar", "cmd.md"}, st); err != nil {
		t.Fatal(err)
	}
	if len(st.specs) != 1 || st.specs[0].dir != "subcommands/myprog/subcommands/foo/subcommands/bar" {
		t.Fatalf("specs %v", st.specs)
	}
}

func TestImportMainCommandPath(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "main", "design.md"), "main design\n")

	st := &fakeStore{}
	if err := runImport(root, []string{"main", "design.md"}, st); err != nil {
		t.Fatal(err)
	}
	if len(st.specs) != 1 || st.specs[0].dir != "main" || st.specs[0].body != "main design\n" {
		t.Fatalf("specs %v", st.specs)
	}
}

func TestImportReimportDiffsAgainstPriorBody(t *testing.T) {
	root := newMetadata(t)
	path := filepath.Join(root, "subcommands", "spec", "cmd.md")
	writeFile(t, path, "first\n")
	st := &fakeStore{}
	if err := runImport(root, []string{"spec", "cmd.md"}, st); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, "second\n")
	if err := runImport(root, []string{"spec", "cmd.md"}, st); err != nil {
		t.Fatal(err)
	}
	if len(st.specs) != 2 {
		t.Fatalf("specs %v", st.specs)
	}
	if !strings.Contains(st.specs[1].diff, "-first") || !strings.Contains(st.specs[1].diff, "+second") {
		t.Fatalf("diff %q", st.specs[1].diff)
	}
}

func TestImportFailsWhenMissing(t *testing.T) {
	root := newMetadata(t)
	err := runImport(root, []string{"migrate", "cmd.md"}, &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "spec file .4sight/subcommands/migrate/cmd.md does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestImportFailsWhenDirectory(t *testing.T) {
	root := newMetadata(t)
	if err := os.MkdirAll(filepath.Join(root, "subcommands", "migrate", "cmd.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := runImport(root, []string{"migrate", "cmd.md"}, &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestImportTopLevelRootDir(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "design.md"), "top\n")
	st := &fakeStore{}
	if err := runImport(root, []string{"design.md"}, st); err != nil {
		t.Fatal(err)
	}
	if len(st.specs) != 1 || st.specs[0].dir != "" || st.specs[0].name != "design.md" || st.specs[0].body != "top\n" {
		t.Fatalf("specs %v", st.specs)
	}
}

func TestImportRejectsInvalidToken(t *testing.T) {
	root := newMetadata(t)
	err := runImport(root, []string{"../secrets.md"}, &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("got %v", err)
	}
}

func TestImportRejectsEmptyLocation(t *testing.T) {
	root := newMetadata(t)
	err := runImport(root, nil, &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "missing spec file location") {
		t.Fatalf("got %v", err)
	}
}

func TestImportSchemaInsertsBody(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "# Spec\nschema body\n")

	st := &fakeStore{}
	if err := runImportSchema(root, "dolt_spec.md", st); err != nil {
		t.Fatal(err)
	}
	if len(st.specs) != 1 {
		t.Fatalf("specs %v", st.specs)
	}
	got := st.specs[0]
	if got.dir != "schemas" || got.name != "dolt_spec.md" || got.body != "# Spec\nschema body\n" {
		t.Fatalf("got %+v", got)
	}
}

func TestImportSchemaAddsMDSuffix(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "schema\n")
	st := &fakeStore{}
	if err := runImportSchema(root, "dolt_spec", st); err != nil {
		t.Fatal(err)
	}
	if len(st.specs) != 1 || st.specs[0].name != "dolt_spec.md" {
		t.Fatalf("specs %v", st.specs)
	}
}

func TestImportSchemaFailsWhenMissing(t *testing.T) {
	root := newMetadata(t)
	err := runImportSchema(root, "dolt_missing.md", &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "schema spec .4sight/schemas/dolt_missing.md does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestImportSchemaRejectsTraversal(t *testing.T) {
	root := newMetadata(t)
	err := runImportSchema(root, "../secrets.md", &fakeStore{})
	if err == nil || !strings.Contains(err.Error(), "relative to schemas dir") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveSpecLocation(t *testing.T) {
	dir, name, err := resolveSpecLocation([]string{"myprog", "foo", "bar", "cmd.md"})
	if err != nil {
		t.Fatal(err)
	}
	if dir != "subcommands/myprog/subcommands/foo/subcommands/bar" || name != "cmd.md" {
		t.Fatalf("dir %q name %q", dir, name)
	}
}

func TestUnifiedDiffEmptyPrior(t *testing.T) {
	got := unifiedDiff("", "a\nb\n")
	if !strings.Contains(got, "@@ -0,0 +1,2 @@") || !strings.Contains(got, "+a") {
		t.Fatalf("diff %q", got)
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
	writeFile(t, filepath.Join(root, "subcommands", "spec", "cmd.md"), "quote ' and\nnewline\n")
	writeFile(t, filepath.Join(root, "design.md"), "top body\n")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("dolt", "init")
	cmd.Dir = dbDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}
	store := newDoltStore(dbDir)
	for _, stmt := range []string{
		"CREATE DATABASE spec",
		"CREATE TABLE spec.bodies (body TEXT NOT NULL)",
		`CREATE TABLE spec.dirs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  path VARCHAR(255) NOT NULL,
  UNIQUE KEY path (path),
  CHECK (path = '' OR path REGEXP '^[a-z0-9]([a-z0-9.-]*(/[a-z0-9.-]+)*)?$')
)`,
		`CREATE TABLE spec.files (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  dir_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  UNIQUE KEY dir_name (dir_id, name),
  CONSTRAINT fk_files_dir_id FOREIGN KEY (dir_id) REFERENCES spec.dirs (id),
  CHECK (name REGEXP '^[a-z0-9.-]+$')
)`,
		`CREATE TABLE spec.file_changes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  file_id BIGINT UNSIGNED NOT NULL,
  diff TEXT NOT NULL,
  CONSTRAINT fk_file_changes_file_id FOREIGN KEY (file_id) REFERENCES spec.files (id)
)`,
		`ALTER TABLE spec.bodies
  ADD COLUMN file_change_id BIGINT UNSIGNED NOT NULL,
  ADD CONSTRAINT fk_bodies_file_change_id FOREIGN KEY (file_change_id) REFERENCES spec.file_changes (id)`,
	} {
		if err := store.run("sql", "-q", stmt); err != nil {
			t.Fatal(err)
		}
	}

	if err := Import(dir, []string{"spec", "cmd.md"}); err != nil {
		t.Fatal(err)
	}
	if err := Import(dir, []string{"design.md"}); err != nil {
		t.Fatal(err)
	}
	rows, err := store.queryCSV("SELECT body FROM spec.bodies ORDER BY file_change_id")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0][0] != "quote ' and\nnewline\n" || rows[1][0] != "top body\n" {
		t.Fatalf("rows %v", rows)
	}
	dirs, err := store.queryCSV("SELECT path FROM spec.dirs ORDER BY path")
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 2 || dirs[0][0] != "" || dirs[1][0] != "subcommands/spec" {
		t.Fatalf("dirs %v", dirs)
	}
}

func newMetadata(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".4sight")
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
