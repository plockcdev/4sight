package spec

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncCheckSections(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "design.md"), "disk migrate\n")
	writeFile(t, filepath.Join(root, "subcommands", "spec", "design.md"), "disk spec\n")
	writeFile(t, filepath.Join(root, "coding.md"), "same\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "disk schema\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_4sight_migration.md"), "disk mig\n")

	st := &fakeStore{specs: []importedSpec{
		{dir: "subcommands/spec", name: "design.md", body: "db spec\n"},
		{dir: "subcommands/spec/subcommands/import", name: "cmd.md", body: "only db\n"},
		{dir: "", name: "coding.md", body: "same\n"},
		{dir: "schemas", name: "dolt_spec.md", body: "db schema\n"},
	}}

	var out bytes.Buffer
	if err := runSyncCheck(root, st, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Import (overwrite database from disk):") {
		t.Fatalf("missing import section: %q", got)
	}
	if !strings.Contains(got, "Export (overwrite disk from database):") {
		t.Fatalf("missing export section: %q", got)
	}
	if !strings.Contains(got, "4sight spec import migrate design.md") {
		t.Fatalf("missing disk-only import: %q", got)
	}
	if !strings.Contains(got, "4sight spec import spec design.md") {
		t.Fatalf("missing mismatch import: %q", got)
	}
	if !strings.Contains(got, "4sight spec export spec design.md") {
		t.Fatalf("missing mismatch export: %q", got)
	}
	if !strings.Contains(got, "4sight spec export spec import cmd.md") {
		t.Fatalf("missing db-only export: %q", got)
	}
	if !strings.Contains(got, "4sight migrate import dolt_spec.md") {
		t.Fatalf("missing schema mismatch import: %q", got)
	}
	if !strings.Contains(got, "4sight migrate import dolt_4sight_migration.md") {
		t.Fatalf("missing schema disk-only import: %q", got)
	}
	if strings.Contains(got, "coding.md") {
		t.Fatalf("listed matching file: %q", got)
	}
	for line := range strings.SplitSeq(got, "\n") {
		if strings.Contains(line, "spec export") && strings.Contains(line, "dolt_spec") {
			t.Fatalf("used spec export for schema: %q", got)
		}
	}
}

func TestSyncCheckAllMatch(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "design.md"), "top\n")
	st := &fakeStore{specs: []importedSpec{
		{dir: "", name: "design.md", body: "top\n"},
	}}

	var out bytes.Buffer
	if err := runSyncCheck(root, st, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Count(got, "(none)") != 2 {
		t.Fatalf("stdout %q", got)
	}
	if strings.Contains(got, "4sight spec") {
		t.Fatalf("unexpected command: %q", got)
	}
}

func TestLocatorFromDirName(t *testing.T) {
	cases := []struct {
		dir, name string
		want      []string
	}{
		{"", "coding.md", []string{"coding.md"}},
		{"main", "design.md", []string{"main", "design.md"}},
		{"subcommands/spec/subcommands/implement/subcommands/prompt", "cmd.md", []string{"spec", "implement", "prompt", "cmd.md"}},
	}
	for _, tc := range cases {
		got := locatorFromDirName(tc.dir, tc.name)
		if strings.Join(got, " ") != strings.Join(tc.want, " ") {
			t.Fatalf("dir %q name %q got %v want %v", tc.dir, tc.name, got, tc.want)
		}
		dir, name, err := resolveSpecLocation(got)
		if err != nil {
			t.Fatal(err)
		}
		if dir != tc.dir || name != tc.name {
			t.Fatalf("round trip dir %q name %q", dir, name)
		}
	}
}
