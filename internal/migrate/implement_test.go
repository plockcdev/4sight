package migrate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImplementPrompt(t *testing.T) {
	root := newMetadata(t, []string{"dolt_spec.md"})
	mig := filepath.Join(root, "migrations", "dolt_spec")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))

	writeFile(t, filepath.Join(root, "subcommands", "migrate", "design.md"), "design\n")
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "subcommands", "up", "cmd.md"), "cmd\n")

	var out bytes.Buffer
	if err := runImplementPrompt(root, "dolt_spec.md", &out, func() (string, error) {
		return "1234", nil
	}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Read these design and command specification documents") {
		t.Fatalf("missing spec docs section: %q", got)
	}
	if !strings.Contains(got, ".4sight/subcommands/migrate/design.md") {
		t.Fatalf("missing design doc: %q", got)
	}
	if !strings.Contains(got, ".4sight/subcommands/migrate/subcommands/up/cmd.md") {
		t.Fatalf("missing cmd doc: %q", got)
	}
	if !strings.Contains(got, ".4sight/schemas/dolt_spec.md") {
		t.Fatalf("missing schema path: %q", got)
	}
	if !strings.Contains(got, ".4sight/migrations/dolt_spec/00011234-00000000.sql") {
		t.Fatalf("missing target migration path: %q", got)
	}
	if !strings.Contains(got, ".4sight/migrations/dolt_spec/00000000-00000000.sql") {
		t.Fatalf("missing parent migration path: %q", got)
	}
	if !strings.Contains(got, ".4sight/migrations/dolt_spec/submitted -> 00011234-00000000.sql") {
		t.Fatalf("missing submitted update instruction: %q", got)
	}
}

func TestImplementPromptAddsMDSuffix(t *testing.T) {
	root := newMetadata(t, []string{"dolt_spec.md"})
	mig := filepath.Join(root, "migrations", "dolt_spec")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))

	var out bytes.Buffer
	if err := runImplementPrompt(root, "dolt_spec", &out, func() (string, error) {
		return "5678", nil
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), ".4sight/schemas/dolt_spec.md") {
		t.Fatalf("stdout %q", out.String())
	}
}

func TestImplementPromptUsesSubmittedParent(t *testing.T) {
	root := newMetadata(t, []string{"dolt_4sight_migration.md"})
	mig := filepath.Join(root, "migrations", "dolt_4sight_migration")
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	writeFile(t, filepath.Join(mig, "00016920-00000000.sql"), "-- next\n")
	mustSymlink(t, "00016920-00000000.sql", filepath.Join(mig, "submitted"))

	var out bytes.Buffer
	if err := runImplementPrompt(root, "dolt_4sight_migration.md", &out, func() (string, error) {
		return "9999", nil
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), ".4sight/migrations/dolt_4sight_migration/00029999-00016920.sql") {
		t.Fatalf("stdout %q", out.String())
	}
}

func TestImplementPromptRejectsMissingSchema(t *testing.T) {
	root := newMetadata(t, []string{"dolt_spec.md"})
	err := runImplementPrompt(root, "dolt_missing.md", ioDiscard(), func() (string, error) {
		return "0001", nil
	})
	if err == nil || !strings.Contains(err.Error(), "schema spec .4sight/schemas/dolt_missing.md does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestImplementPromptRejectsAbsolutePath(t *testing.T) {
	root := newMetadata(t, []string{"dolt_spec.md"})
	err := runImplementPrompt(root, "/tmp/dolt_spec.md", ioDiscard(), func() (string, error) {
		return "0001", nil
	})
	if err == nil || !strings.Contains(err.Error(), "relative to schemas dir") {
		t.Fatalf("got %v", err)
	}
}

func TestImplementPromptRejectsTraversal(t *testing.T) {
	root := newMetadata(t, []string{"dolt_spec.md"})
	err := runImplementPrompt(root, "../secrets.md", ioDiscard(), func() (string, error) {
		return "0001", nil
	})
	if err == nil || !strings.Contains(err.Error(), "relative to schemas dir") {
		t.Fatalf("got %v", err)
	}
}

func TestImplementPromptFailsWhenSubmittedMissing(t *testing.T) {
	root := newMetadata(t, []string{"dolt_spec.md"})
	err := runImplementPrompt(root, "dolt_spec.md", ioDiscard(), func() (string, error) {
		return "0001", nil
	})
	if err == nil || !strings.Contains(err.Error(), "missing submitted") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Join(root, "migrations", "dolt_spec")) {
		t.Fatalf("error missing dir: %v", err)
	}
}

func TestNextMigrationID(t *testing.T) {
	got, err := nextMigrationID("00016920", "4321")
	if err != nil {
		t.Fatal(err)
	}
	if got != "00024321" {
		t.Fatalf("got %q", got)
	}
}

func TestDatabaseFromSchemaName(t *testing.T) {
	db, err := databaseFromSchemaName("dolt_spec.md")
	if err != nil {
		t.Fatal(err)
	}
	if db.key() != "dolt_spec" {
		t.Fatalf("key %q", db.key())
	}
}

func TestListSpecDocs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "design.md"), "d\n")
	writeFile(t, filepath.Join(root, "subcommands", "migrate", "subcommands", "up", "cmd.md"), "c\n")
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "s\n")

	docs, err := listSpecDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("docs %v", docs)
	}
	if !strings.HasSuffix(docs[0], "design.md") {
		t.Fatalf("docs %v", docs)
	}
	if !strings.HasSuffix(docs[1], "cmd.md") {
		t.Fatalf("docs %v", docs)
	}
}

func TestImplementPromptFindsMetadataRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".4sight")
	if err := os.MkdirAll(filepath.Join(root, "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "schemas", "dolt_spec.md"), "spec\n")
	mig := filepath.Join(root, "migrations", "dolt_spec")
	if err := os.MkdirAll(mig, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(mig, "00000000-00000000.sql"), "-- init\n")
	mustSymlink(t, "00000000-00000000.sql", filepath.Join(mig, "submitted"))

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

	var out bytes.Buffer
	if err := runImplementPrompt(root, "dolt_spec.md", &out, func() (string, error) {
		return "1111", nil
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "00011111-00000000.sql") {
		t.Fatalf("stdout %q", out.String())
	}
}
