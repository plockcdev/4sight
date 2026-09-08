package spec

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImplementPrompt(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "design.md"), "top\n")
	writeFile(t, filepath.Join(root, "cmd.md"), "top cmd\n")
	writeFile(t, filepath.Join(root, "coding.md"), "ignore\n")
	writeFile(t, filepath.Join(root, "main", "design.md"), "main\n")
	writeFile(t, filepath.Join(root, "subcommands", "spec", "design.md"), "spec\n")
	writeFile(t, filepath.Join(root, "subcommands", "spec", "subcommands", "implement", "subcommands", "prompt", "cmd.md"), "prompt\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "schema\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_4sight_migration.md"), "mig\n")
	writeFile(t, filepath.Join(root, "schemas", "notes.txt"), "skip\n")

	var out bytes.Buffer
	if err := runImplementPrompt(root, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Read these design and command specification documents to understand the application behavior:") {
		t.Fatalf("missing spec docs section: %q", got)
	}
	for _, want := range []string{
		".4sight/design.md",
		".4sight/cmd.md",
		".4sight/main/design.md",
		".4sight/subcommands/spec/design.md",
		".4sight/subcommands/spec/subcommands/implement/subcommands/prompt/cmd.md",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s: %q", want, got)
		}
	}
	if strings.Contains(got, "coding.md") {
		t.Fatalf("listed coding.md: %q", got)
	}
	if !strings.Contains(got, "Read these database schema specifications:") {
		t.Fatalf("missing schemas section: %q", got)
	}
	if !strings.Contains(got, ".4sight/schemas/dolt_4sight_migration.md") {
		t.Fatalf("missing migration schema: %q", got)
	}
	if !strings.Contains(got, ".4sight/schemas/dolt_spec.md") {
		t.Fatalf("missing spec schema: %q", got)
	}
	if strings.Contains(got, "notes.txt") {
		t.Fatalf("listed non-md schema: %q", got)
	}
	if !strings.Contains(got, "Review the application code specification and update the code to match the spec.") {
		t.Fatalf("missing review instruction: %q", got)
	}
	if !strings.Contains(got, "Do not update the schema.") {
		t.Fatalf("missing schema restriction: %q", got)
	}
	if !strings.Contains(got, "If the schema needs a spec change or a migration to be compatible with the code spec, stop and tell the user the gap") {
		t.Fatalf("missing schema gap instruction: %q", got)
	}
}

func TestImplementPromptAllowsMissingSchemasDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".4sight")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "subcommands", "spec", "design.md"), "spec\n")

	var out bytes.Buffer
	if err := runImplementPrompt(root, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, ".4sight/subcommands/spec/design.md") {
		t.Fatalf("stdout %q", got)
	}
	if strings.Contains(got, ".4sight/schemas/") {
		t.Fatalf("unexpected schema path: %q", got)
	}
}

func TestImplementPromptFindsMetadataRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".4sight")
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "design.md"), "d\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "s\n")

	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	oldOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = ImplementPrompt(sub)
	_ = w.Close()
	os.Stdout = oldOut
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, ".4sight/subcommands/migrate/design.md") {
		t.Fatalf("stdout %q", got)
	}
	if !strings.Contains(got, ".4sight/schemas/dolt_spec.md") {
		t.Fatalf("stdout %q", got)
	}
}

func TestListCommandSpecDocsSorted(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "z", "cmd.md"), "z\n")
	writeFile(t, filepath.Join(root, "subcommands", "a", "design.md"), "a\n")

	docs, err := listCommandSpecDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("docs %v", docs)
	}
	if !strings.HasSuffix(docs[0], filepath.Join("a", "design.md")) {
		t.Fatalf("docs %v", docs)
	}
	if !strings.HasSuffix(docs[1], filepath.Join("z", "cmd.md")) {
		t.Fatalf("docs %v", docs)
	}
}
