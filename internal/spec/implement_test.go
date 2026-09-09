package spec

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestImplementPromptUsesDatabaseBodies(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "spec", "design.md"), "disk stale\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "schema\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_4sight_migration.md"), "mig\n")
	writeFile(t, filepath.Join(root, "schemas", "notes.txt"), "skip\n")

	st := &fakeStore{specs: []importedSpec{
		{dir: "subcommands/spec", name: "design.md", body: "spec design from db\n"},
		{dir: "subcommands/spec/subcommands/implement/subcommands/prompt", name: "cmd.md", body: "prompt from db\n"},
		{dir: "", name: "coding.md", body: "coding from db\n"},
		{dir: "schemas", name: "dolt_spec.md", body: "should not appear as code spec\n"},
	}}

	var out bytes.Buffer
	if err := runImplementPrompt(root, st, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Read these design and command specification documents from the spec database") {
		t.Fatalf("missing spec docs section: %q", got)
	}
	for _, want := range []string{
		"----- SPEC .4sight/coding.md -----",
		"coding from db",
		"----- SPEC .4sight/subcommands/spec/design.md -----",
		"spec design from db",
		"----- SPEC .4sight/subcommands/spec/subcommands/implement/subcommands/prompt/cmd.md -----",
		"prompt from db",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, "disk stale") {
		t.Fatalf("used on-disk spec instead of db: %q", got)
	}
	if strings.Contains(got, "should not appear as code spec") {
		t.Fatalf("included schema body as code spec: %q", got)
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
	root := newMetadata(t)
	st := &fakeStore{specs: []importedSpec{
		{dir: "subcommands/spec", name: "design.md", body: "spec\n"},
	}}

	var out bytes.Buffer
	if err := runImplementPrompt(root, st, &out); err != nil {
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

func TestImplementPromptOmitsDiskOnlySpecs(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "design.md"), "only on disk\n")
	st := &fakeStore{}

	var out bytes.Buffer
	if err := runImplementPrompt(root, st, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "only on disk") || strings.Contains(out.String(), "migrate/design.md") {
		t.Fatalf("listed disk-only spec: %q", out.String())
	}
}
