package spec

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncDiffShowsDifferences(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "spec", "design.md"), "disk spec\n")
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "design.md"), "only disk\n")
	writeFile(t, filepath.Join(root, "coding.md"), "same\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "disk schema\n")

	st := &fakeStore{specs: []importedSpec{
		{dir: "subcommands/spec", name: "design.md", body: "db spec\n"},
		{dir: "subcommands/spec/subcommands/import", name: "cmd.md", body: "only db\n"},
		{dir: "", name: "coding.md", body: "same\n"},
		{dir: "schemas", name: "dolt_spec.md", body: "db schema\n"},
	}}

	var out bytes.Buffer
	if err := runSyncDiff(root, st, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "--- .4sight/subcommands/spec/design.md") {
		t.Fatalf("missing mismatch diff: %q", got)
	}
	if !strings.Contains(got, "-db spec") || !strings.Contains(got, "+disk spec") {
		t.Fatalf("missing spec body diff: %q", got)
	}
	if !strings.Contains(got, "--- .4sight/subcommands/migrate/design.md") {
		t.Fatalf("missing disk-only diff: %q", got)
	}
	if !strings.Contains(got, "+only disk") {
		t.Fatalf("missing disk-only addition: %q", got)
	}
	if !strings.Contains(got, "--- .4sight/subcommands/spec/subcommands/import/cmd.md") {
		t.Fatalf("missing db-only diff: %q", got)
	}
	if !strings.Contains(got, "-only db") {
		t.Fatalf("missing db-only removal: %q", got)
	}
	if !strings.Contains(got, "--- .4sight/schemas/dolt_spec.md") {
		t.Fatalf("missing schema diff: %q", got)
	}
	if strings.Contains(got, "coding.md") {
		t.Fatalf("listed matching file: %q", got)
	}
}

func TestSyncDiffAllMatch(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "design.md"), "top\n")
	st := &fakeStore{specs: []importedSpec{
		{dir: "", name: "design.md", body: "top\n"},
	}}

	var out bytes.Buffer
	if err := runSyncDiff(root, st, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" {
		t.Fatalf("stdout %q", out.String())
	}
}
