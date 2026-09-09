package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportWritesBodyToDisk(t *testing.T) {
	root := newMetadata(t)
	st := &fakeStore{specs: []importedSpec{
		{dir: "subcommands/spec", name: "design.md", body: "from db\n"},
	}}

	var out bytes.Buffer
	if err := runExport(root, []string{"spec", "design.md"}, st, &out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "subcommands", "spec", "design.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "from db\n" {
		t.Fatalf("file %q", got)
	}
	if !strings.Contains(out.String(), "wrote .4sight/subcommands/spec/design.md") {
		t.Fatalf("stdout %q", out.String())
	}
}

func TestExportOverwritesExistingFile(t *testing.T) {
	root := newMetadata(t)
	path := filepath.Join(root, "coding.md")
	writeFile(t, path, "old disk\n")
	st := &fakeStore{specs: []importedSpec{
		{dir: "", name: "coding.md", body: "new db\n"},
	}}

	var out bytes.Buffer
	if err := runExport(root, []string{"coding.md"}, st, &out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new db\n" {
		t.Fatalf("file %q", got)
	}
}

func TestExportFailsWhenMissingBody(t *testing.T) {
	root := newMetadata(t)
	err := runExport(root, []string{"spec", "cmd.md"}, &fakeStore{}, ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "spec body .4sight/subcommands/spec/cmd.md is not in the database") {
		t.Fatalf("got %v", err)
	}
}

func ioDiscard() *bytes.Buffer {
	return &bytes.Buffer{}
}
