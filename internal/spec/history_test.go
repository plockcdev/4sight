package spec

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryAndActiveChanges(t *testing.T) {
	root := newMetadata(t)
	store := setupChangeImportDB(t, root)
	writeFile(t, filepath.Join(root, "coding.md"), "first\n")
	var out, stderr bytes.Buffer
	if err := runChangeImport(root, store, &out, &stderr); err != nil {
		t.Fatal(err)
	}
	id, err := store.workspaceChangeID()
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err = runChangeList(store, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), id) || strings.Contains(out.String(), "ch-000") {
		t.Fatalf("active list: %s", out.String())
	}
	out.Reset()
	if err = listChanges(store, &out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ch-000") || !strings.Contains(out.String(), id) {
		t.Fatalf("history: %s", out.String())
	}
	out.Reset()
	if err = fileHistory(store, &out, "coding.md"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), id) || !strings.Contains(out.String(), "no review yet") {
		t.Fatalf("file history: %s", out.String())
	}
	if err = store.run("sql", "-q", "UPDATE spec.current_changes SET implemented_change_id="+sqlString(id)+", submitted_change_id="+sqlString(id)); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err = runChangeList(store, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "no active changes\n" {
		t.Fatalf("empty list: %s", out.String())
	}
}

func TestHistoryPath(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".4sight")
	for _, tc := range []struct{ cwd, file, want string }{
		{project, ".4sight/coding.md", "coding.md"},
		{root, "coding.md", "coding.md"},
		{filepath.Join(root, "subcommands"), "../deleted/file.md", "deleted/file.md"},
		{project, filepath.Join(root, "missing", "..", "coding.md"), "coding.md"},
	} {
		got, err := historyPath(root, tc.cwd, tc.file)
		if err != nil || got != tc.want {
			t.Errorf("historyPath(%q, %q) = %q, %v; want %q", tc.cwd, tc.file, got, err, tc.want)
		}
	}
	for _, file := range []string{"outside.md", ".4sight/../outside.md", ".4sight-other/file.md"} {
		if _, err := historyPath(root, project, file); err == nil || !strings.Contains(err.Error(), "outside .4sight") {
			t.Errorf("historyPath(%q): expected outside error, got %v", file, err)
		}
	}
}

func TestWrapHistory(t *testing.T) {
	lines := wrapHistory(strings.Repeat("a", 90)+"\n\nlast words", 78)
	if len(lines) != 4 || lines[2] != "" {
		t.Fatalf("lines: %#v", lines)
	}
	for _, line := range lines {
		if len([]rune(line)) > 78 {
			t.Fatalf("line too long: %q", line)
		}
	}
}
