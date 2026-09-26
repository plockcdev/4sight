package spec

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeSpecBody(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{in: "", want: ""},
		{in: "\n\n", want: ""},
		{in: "   \n\t\n", want: ""},
		{in: "foo", want: "foo\n"},
		{in: "foo\n", want: "foo\n"},
		{in: "foo \n", want: "foo\n"},
		{in: "foo\t\n\n", want: "foo\n"},
		{in: "foo\r\nbar\r\n", want: "foo\nbar\n"},
		{in: "foo\n\nbar\n\n", want: "foo\n\nbar\n"},
		{in: "  foo  \n", want: "  foo\n"},
		{in: "foo\n   \n", want: "foo\n"},
	}
	for _, tc := range cases {
		if got := normalizeSpecBody(tc.in); got != tc.want {
			t.Fatalf("normalize %q = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestChangeIDSequenceMatchesSpec(t *testing.T) {
	checks := []struct{ id, next string }{
		{"ch-000", "ch-001"},
		{"ch-001", "ch-002"},
		{"ch-998", "ch-999"},
		{"ch-999", "ch-a00"},
		{"ch-a00", "ch-a01"},
		{"ch-a99", "ch-b00"},
		{"ch-z98", "ch-z99"},
		{"ch-z99", "ch-za0"},
		{"ch-za0", "ch-za1"},
		{"ch-za9", "ch-zaa"},
		{"ch-zaz", "ch-zb0"},
		{"ch-zzy", "ch-zzz"},
	}
	for _, tc := range checks {
		got, err := nextChangeID([]string{tc.id})
		if err != nil {
			t.Fatalf("next %s: %v", tc.id, err)
		}
		if got != tc.next {
			t.Fatalf("next %s = %s, want %s", tc.id, got, tc.next)
		}
	}
	if _, err := nextChangeID([]string{"ch-zzz"}); err == nil {
		t.Fatal("expected exhaustion after ch-zzz")
	}
	got, err := nextChangeID([]string{"ch-000", "ch-004", "ch-001"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ch-005" {
		t.Fatalf("next max = %s", got)
	}
}

func TestChangeIDRoundTrip(t *testing.T) {
	prev := ""
	for rank := range changeIDSpace {
		id, err := formatChangeID(rank)
		if err != nil {
			t.Fatal(err)
		}
		got, err := changeIDRank(id)
		if err != nil || got != rank {
			t.Fatalf("rank %d id %s got %d err %v", rank, id, got, err)
		}
		if prev != "" {
			next, err := nextChangeID([]string{prev})
			if err != nil || next != id {
				t.Fatalf("after %s got %s want %s err %v", prev, next, id, err)
			}
		}
		prev = id
	}
	if prev != "ch-zzz" {
		t.Fatalf("last id %s", prev)
	}
}

func TestChangeImportWritesYAML(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "spec", "cmd.md"), "updated \n\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "schema v2\n")
	writeFile(t, filepath.Join(root, "coding.md"), "keep \n\n")

	st := &fakeChangeStore{
		workspace: "ch-001",
		ids:       []string{"ch-000", "ch-001"},
		bodies: []specDoc{
			{Dir: "", Name: "coding.md", Body: "keep\n"},
			{Dir: "subcommands/spec", Name: "cmd.md", Body: "original\n"},
			{Dir: "subcommands/migrate", Name: "design.md", Body: "removed\n"},
		},
	}
	var stdout, stderr bytes.Buffer
	if err := runChangeImport(root, st, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr %q", stderr.String())
	}
	want := "" +
		"id: ch-002\n" +
		"file_changes:\n" +
		"  - path: schemas/dolt_spec.md\n" +
		"    summary: +1/-0\n" +
		"  - path: subcommands/migrate/design.md\n" +
		"    summary: +0/-1\n" +
		"  - path: subcommands/spec/cmd.md\n" +
		"    summary: +1/-1\n"
	if stdout.String() != want {
		t.Fatalf("stdout\n%s\nwant\n%s", stdout.String(), want)
	}
	if st.appliedID != "ch-002" || st.appliedParent != "ch-001" {
		t.Fatalf("applied id %s parent %s", st.appliedID, st.appliedParent)
	}
	if len(st.applied) != 3 {
		t.Fatalf("applied %#v", st.applied)
	}
	byPath := map[string]plannedChange{}
	for _, file := range st.applied {
		byPath[file.Path] = file
	}
	if byPath["subcommands/spec/cmd.md"].Body != "updated\n" {
		t.Fatalf("updated body %q", byPath["subcommands/spec/cmd.md"].Body)
	}
	if !strings.Contains(byPath["subcommands/spec/cmd.md"].Diff, "-original") ||
		!strings.Contains(byPath["subcommands/spec/cmd.md"].Diff, "+updated") {
		t.Fatalf("diff %q", byPath["subcommands/spec/cmd.md"].Diff)
	}
	if byPath["subcommands/migrate/design.md"].Body != "" {
		t.Fatalf("deleted body %q", byPath["subcommands/migrate/design.md"].Body)
	}
	if byPath["schemas/dolt_spec.md"].Body != "schema v2\n" {
		t.Fatalf("schema body %q", byPath["schemas/dolt_spec.md"].Body)
	}
}

func TestChangeImportNoChanges(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "coding.md"), "keep \n\n")
	st := &fakeChangeStore{
		workspace: "ch-000",
		ids:       []string{"ch-000"},
		bodies:    []specDoc{{Dir: "", Name: "coding.md", Body: "keep\n"}},
	}
	var stdout, stderr bytes.Buffer
	if err := runChangeImport(root, st, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout %q", stdout.String())
	}
	if stderr.String() != "no changes found\n" {
		t.Fatalf("stderr %q", stderr.String())
	}
	if st.appliedID != "" {
		t.Fatalf("applied %s", st.appliedID)
	}
}

func TestChangeImportDolt(t *testing.T) {
	root := newMetadata(t)
	store := setupChangeImportDB(t, root)
	writeFile(t, filepath.Join(root, "coding.md"), "keep \n")
	writeFile(t, filepath.Join(root, "subcommands", "spec", "cmd.md"), "updated\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "schema v2\n")
	seedWorkspaceBody(t, store, "", "coding.md", "keep\n")
	seedWorkspaceBody(t, store, "subcommands/spec", "cmd.md", "original\n")
	seedWorkspaceBody(t, store, "subcommands/migrate", "design.md", "removed\n")

	var stdout, stderr bytes.Buffer
	if err := runChangeImport(root, store, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "id: ch-001\n") {
		t.Fatalf("stdout %s", stdout.String())
	}

	workspace, err := store.workspaceChangeID()
	if err != nil {
		t.Fatal(err)
	}
	if workspace != "ch-001" {
		t.Fatalf("workspace %s", workspace)
	}
	prompts, err := store.queryCSV("SELECT id, prompt FROM spec.changes ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 2 || prompts[1][0] != "ch-001" || prompts[1][1] != "" {
		t.Fatalf("prompts %v", prompts)
	}
	closure, err := store.queryCSV("SELECT parent_id, child_id, depth FROM spec.changes_closure WHERE child_id = 'ch-001' ORDER BY depth")
	if err != nil {
		t.Fatal(err)
	}
	if len(closure) != 2 || closure[0][0] != "ch-001" || closure[0][2] != "0" || closure[1][0] != "ch-000" || closure[1][2] != "1" {
		t.Fatalf("closure %v", closure)
	}

	bodies, err := store.queryCSV(
		"SELECT d.path, f.name, b.body, b.summary, b.state, b.ieee_crc32, fc.change_id " +
			"FROM spec.bodies b " +
			"JOIN spec.files f ON f.id = b.file_id " +
			"JOIN spec.dirs d ON d.id = f.dir_id " +
			"JOIN spec.file_changes fc ON fc.id = b.file_change_id " +
			"ORDER BY d.path, f.name, b.state",
	)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, row := range bodies {
		got[specRelPath(row[0], row[1])] = row
	}
	if row := got["coding.md"]; row[2] != "keep\n" || row[6] != "ch-000" {
		t.Fatalf("coding row %v", row)
	}
	if row := got["subcommands/spec/cmd.md"]; row[2] != "updated\n" || row[3] != "" || row[4] != "workspace" || row[6] != "ch-001" {
		t.Fatalf("updated row %v", row)
	}
	if got["subcommands/spec/cmd.md"][5] != fmt.Sprintf("%d", crc32.ChecksumIEEE([]byte("updated\n"))) {
		t.Fatalf("crc %v", got["subcommands/spec/cmd.md"])
	}
	if row := got["subcommands/migrate/design.md"]; row[2] != "" || row[6] != "ch-001" || row[3] != "" {
		t.Fatalf("deleted row %v", row)
	}
	if row := got["schemas/dolt_spec.md"]; row[2] != "schema v2\n" || row[6] != "ch-001" {
		t.Fatalf("schema row %v", row)
	}
	if len(got) != 4 {
		t.Fatalf("bodies %v", got)
	}

	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "schema v3\n")
	stdout.Reset()
	stderr.Reset()
	if err := runChangeImport(root, store, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "id: ch-002\n") || !strings.Contains(stdout.String(), "path: schemas/dolt_spec.md\n") {
		t.Fatalf("second stdout %s", stdout.String())
	}
	workspace, err = store.workspaceChangeID()
	if err != nil {
		t.Fatal(err)
	}
	if workspace != "ch-002" {
		t.Fatalf("workspace %s", workspace)
	}
	depth, err := store.queryCSV("SELECT depth FROM spec.changes_closure WHERE parent_id = 'ch-000' AND child_id = 'ch-002'")
	if err != nil {
		t.Fatal(err)
	}
	if len(depth) != 1 || depth[0][0] != "2" {
		t.Fatalf("depth %v", depth)
	}
	schemaBodies, err := store.queryCSV(
		"SELECT b.body, fc.change_id FROM spec.bodies b " +
			"JOIN spec.file_changes fc ON fc.id = b.file_change_id " +
			"JOIN spec.files f ON f.id = b.file_id " +
			"JOIN spec.dirs d ON d.id = f.dir_id " +
			"WHERE d.path = 'schemas' AND f.name = 'dolt_spec.md' AND b.state = 'workspace'",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(schemaBodies) != 1 || schemaBodies[0][0] != "schema v3\n" || schemaBodies[0][1] != "ch-002" {
		t.Fatalf("schema bodies %v", schemaBodies)
	}

	stdout.Reset()
	stderr.Reset()
	if err := runChangeImport(root, store, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 || stderr.String() != "no changes found\n" {
		t.Fatalf("stdout %q stderr %q", stdout.String(), stderr.String())
	}
	workspace, err = store.workspaceChangeID()
	if err != nil {
		t.Fatal(err)
	}
	if workspace != "ch-002" {
		t.Fatalf("workspace after empty import %s", workspace)
	}
}

type fakeChangeStore struct {
	bodies        []specDoc
	workspace     string
	ids           []string
	appliedID     string
	appliedParent string
	applied       []plannedChange
	listErr       error
}

func (f *fakeChangeStore) listWorkspaceBodies() ([]specDoc, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.bodies, nil
}

func (f *fakeChangeStore) workspaceChangeID() (string, error) {
	return f.workspace, nil
}

func (f *fakeChangeStore) changeIDs() ([]string, error) {
	return f.ids, nil
}

func (f *fakeChangeStore) applyChangeImport(id, parent string, files []plannedChange) error {
	f.appliedID = id
	f.appliedParent = parent
	f.applied = files
	return nil
}

func setupChangeImportDB(t *testing.T, root string) *doltStore {
	t.Helper()
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not installed")
	}
	dbDir := filepath.Join(root, "db")
	cmd := exec.Command("dolt", "init")
	cmd.Dir = dbDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}
	store := newDoltStore(dbDir)
	for _, stmt := range []string{
		"CREATE DATABASE spec",
		`CREATE TABLE spec.changes (
  id CHAR(6) NOT NULL,
  prompt TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT changes_id_chk CHECK (regexp_like(id, '^ch-[0-9a-z]{3}$'))
)`,
		"INSERT INTO spec.changes (id, prompt) VALUES ('ch-000', '')",
		`CREATE TABLE spec.changes_closure (
  parent_id CHAR(6) NOT NULL,
  child_id CHAR(6) NOT NULL,
  depth INT UNSIGNED NOT NULL,
  PRIMARY KEY (parent_id, child_id),
  CONSTRAINT fk_changes_closure_parent_id FOREIGN KEY (parent_id) REFERENCES spec.changes (id),
  CONSTRAINT fk_changes_closure_child_id FOREIGN KEY (child_id) REFERENCES spec.changes (id)
)`,
		"INSERT INTO spec.changes_closure (parent_id, child_id, depth) VALUES ('ch-000', 'ch-000', 0)",
		`CREATE TABLE spec.current_changes (
  singleton TINYINT NOT NULL DEFAULT 1,
  implemented_change_id CHAR(6) NOT NULL,
  submitted_change_id CHAR(6) NOT NULL,
  workspace_change_id CHAR(6) NOT NULL,
  PRIMARY KEY (singleton),
  CONSTRAINT current_changes_singleton_chk CHECK (singleton = 1),
  CONSTRAINT fk_current_changes_implemented_change_id FOREIGN KEY (implemented_change_id) REFERENCES spec.changes (id),
  CONSTRAINT fk_current_changes_submitted_change_id FOREIGN KEY (submitted_change_id) REFERENCES spec.changes (id),
  CONSTRAINT fk_current_changes_workspace_change_id FOREIGN KEY (workspace_change_id) REFERENCES spec.changes (id)
)`,
		"INSERT INTO spec.current_changes (implemented_change_id, submitted_change_id, workspace_change_id) VALUES ('ch-000', 'ch-000', 'ch-000')",
		`CREATE TABLE spec.dirs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  path VARCHAR(255) NOT NULL,
  UNIQUE KEY path (path),
  CONSTRAINT dirs_path_chk CHECK (path = '' OR regexp_like(path, '^[a-z0-9]([a-z0-9.-]*(/[a-z0-9.-]+)*)?$'))
)`,
		`CREATE TABLE spec.files (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  dir_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  UNIQUE KEY dir_name (dir_id, name),
  CONSTRAINT fk_files_dir_id FOREIGN KEY (dir_id) REFERENCES spec.dirs (id),
  CONSTRAINT files_name_chk CHECK (regexp_like(name, '^[a-z0-9][a-z0-9._-]*$'))
)`,
		`CREATE TABLE spec.file_changes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  change_id CHAR(6) NOT NULL,
  file_id BIGINT UNSIGNED NOT NULL,
  diff TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY file_changes_change_file_uniq (change_id, file_id),
  CONSTRAINT fk_file_changes_change_id FOREIGN KEY (change_id) REFERENCES spec.changes (id),
  CONSTRAINT fk_file_changes_file_id FOREIGN KEY (file_id) REFERENCES spec.files (id)
)`,
		`CREATE TABLE spec.bodies (
  body TEXT NOT NULL,
  file_id BIGINT UNSIGNED NOT NULL,
  file_change_id BIGINT UNSIGNED NOT NULL,
  state VARCHAR(32) NOT NULL,
  ieee_crc32 INT UNSIGNED NOT NULL,
  summary VARCHAR(65) NOT NULL,
  UNIQUE KEY bodies_file_state_uniq (file_id, state),
  CONSTRAINT fk_bodies_file_id FOREIGN KEY (file_id) REFERENCES spec.files (id),
  CONSTRAINT fk_bodies_file_change_id FOREIGN KEY (file_change_id) REFERENCES spec.file_changes (id),
  CONSTRAINT bodies_state_chk CHECK (state IN ('implemented', 'submitted', 'workspace'))
)`,
	} {
		if err := store.run("sql", "-q", stmt); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

func seedWorkspaceBody(t *testing.T, store *doltStore, dir, name, body string) {
	t.Helper()
	dirID, err := store.ensureDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := store.ensureFile(dirID, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.run("sql", "-q", fmt.Sprintf(
		"INSERT INTO spec.file_changes (change_id, file_id, diff) VALUES ('ch-000', %d, %s)",
		fileID, sqlBodyLiteral(unifiedDiff("", body)),
	)); err != nil {
		t.Fatal(err)
	}
	if err := store.run("sql", "-q", fmt.Sprintf(
		"INSERT INTO spec.bodies (body, file_id, file_change_id, state, ieee_crc32, summary) "+
			"SELECT %s, %d, fc.id, 'workspace', %d, 'seed' FROM spec.file_changes fc "+
			"WHERE fc.change_id = 'ch-000' AND fc.file_id = %d",
		sqlBodyLiteral(body), fileID, crc32.ChecksumIEEE([]byte(body)), fileID,
	)); err != nil {
		t.Fatal(err)
	}
}
