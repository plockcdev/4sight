package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplicationLifecycleGuidanceInPrompts(t *testing.T) {
	source, err := os.ReadFile("../../.4sight/subcommands/spec/subcommands/change/subcommands/author/assets/application-database-lifecycle.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != applicationDatabaseLifecycle {
		t.Fatal("embedded lifecycle guidance differs from source")
	}
	root := newMetadata(t)
	store := setupChangeImportDB(t, root)
	var out, stderr bytes.Buffer
	if err := runChangeRequest(store, "Add a database", &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runChangeAuthorPrompt(root, store, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{applicationDatabaseLifecycle, "different migration design", "explicitly excludes migration support"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("author prompt missing %q", want)
		}
	}
	writeFile(t, filepath.Join(root, "coding.md"), "changed\n")
	out.Reset()
	if err := runChangeImport(root, store, &out, &stderr); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runChangeReviewPrompt(root, store, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{applicationDatabaseLifecycle, "established alternative", "explicit exclusion of migration support"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("review prompt missing %q", want)
		}
	}
}
