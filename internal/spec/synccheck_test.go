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
	writeFile(t, filepath.Join(root, "subcommands", "spec", "design.md"), "disk spec \n\n")
	writeFile(t, filepath.Join(root, "coding.md"), "same\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "disk schema\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_4sight_migration.md"), "disk mig\n")

	st := &fakeStore{specs: []importedSpec{
		{dir: "subcommands/spec", name: "design.md", body: "db spec\n"},
		{dir: "subcommands/spec/subcommands/import", name: "cmd.md", body: "only db\n"},
		{dir: "", name: "coding.md", body: "same \n"},
		{dir: "schemas", name: "dolt_spec.md", body: "db schema\n"},
	}}

	var out bytes.Buffer
	if err := runSyncCheck(root, st, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	importSec, exportSec := splitSyncCheckOutput(t, got)
	if importSec != "4sight spec change import\n" {
		t.Fatalf("import section %q", importSec)
	}
	if strings.Contains(importSec, "migrate import") || strings.Contains(importSec, "spec import") {
		t.Fatalf("per-file import command: %q", got)
	}
	if !strings.Contains(exportSec, "4sight spec export spec design.md\n") {
		t.Fatalf("missing mismatch export: %q", got)
	}
	if !strings.Contains(exportSec, "4sight spec export spec import cmd.md\n") {
		t.Fatalf("missing db-only export: %q", got)
	}
	if strings.Contains(exportSec, "migrate design.md") {
		t.Fatalf("exported disk-only file: %q", got)
	}
	if strings.Contains(got, "coding.md") {
		t.Fatalf("listed matching file: %q", got)
	}
	for line := range strings.SplitSeq(got, "\n") {
		if strings.Contains(line, "spec export") && strings.Contains(line, "dolt_") {
			t.Fatalf("used spec export for schema: %q", got)
		}
	}
	if strings.Contains(got, "Summary:") {
		t.Fatalf("unexpected summary: %q", got)
	}
}

func TestSyncCheckAllMatch(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "design.md"), "top \n\n")
	st := &fakeStore{specs: []importedSpec{
		{dir: "", name: "design.md", body: "top\n"},
	}}

	var out bytes.Buffer
	if err := runSyncCheck(root, st, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	importSec, exportSec := splitSyncCheckOutput(t, got)
	if importSec != "(none)\n" || exportSec != "(none)\n" {
		t.Fatalf("stdout %q", got)
	}
	if strings.Contains(got, "4sight spec") {
		t.Fatalf("unexpected command: %q", got)
	}
}

func TestSyncCheckSchemaOnlyDifferenceImports(t *testing.T) {
	root := newMetadata(t)
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "disk\n")
	st := &fakeStore{specs: []importedSpec{
		{dir: "schemas", name: "dolt_spec.md", body: "db\n"},
	}}

	var out bytes.Buffer
	if err := runSyncCheck(root, st, &out); err != nil {
		t.Fatal(err)
	}
	importSec, exportSec := splitSyncCheckOutput(t, out.String())
	if importSec != "4sight spec change import\n" {
		t.Fatalf("import %q", importSec)
	}
	if exportSec != "(none)\n" {
		t.Fatalf("export %q", exportSec)
	}
}

func splitSyncCheckOutput(t *testing.T, got string) (importSec, exportSec string) {
	t.Helper()
	const (
		hImport = "Import (overwrite database from disk):\n"
		hExport = "Export (overwrite disk from database):\n"
	)
	afterImport, ok := strings.CutPrefix(got, hImport)
	if !ok {
		t.Fatalf("missing import section: %q", got)
	}
	importSec, exportSec, ok = strings.Cut(afterImport, "\n"+hExport)
	if !ok {
		t.Fatalf("missing export section: %q", got)
	}
	return importSec, exportSec
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
