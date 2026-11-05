package spec

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type importedSpec struct {
	dir, name, body, diff string
}

type fakeStore struct {
	specs []importedSpec
}

func (f *fakeStore) latestBody(dirPath, fileName string) (string, bool, error) {
	for _, v := range slices.Backward(f.specs) {
		if v.dir == dirPath && v.name == fileName {
			return v.body, true, nil
		}
	}
	return "", false, nil
}

func (f *fakeStore) insertSpec(dirPath, fileName, body, diff string) error {
	f.specs = append(f.specs, importedSpec{dir: dirPath, name: fileName, body: body, diff: diff})
	return nil
}

func (f *fakeStore) listWorkspaceBodies() ([]specDoc, error) {
	return f.listLatestBodies()
}

func (f *fakeStore) listLatestBodies() ([]specDoc, error) {
	latest := make(map[string]specDoc)
	var order []string
	for _, s := range f.specs {
		key := s.dir + "\x00" + s.name
		if _, ok := latest[key]; !ok {
			order = append(order, key)
		}
		latest[key] = specDoc{Dir: s.dir, Name: s.name, Body: s.body}
	}
	docs := make([]specDoc, 0, len(order))
	for _, key := range order {
		docs = append(docs, latest[key])
	}
	return docs, nil
}

func TestUnifiedDiffEmptyPrior(t *testing.T) {
	got := unifiedDiff("", "a\nb\n")
	if !strings.Contains(got, "@@ -0,0 +1,2 @@") || !strings.Contains(got, "+a") {
		t.Fatalf("diff %q", got)
	}
}

func TestUnifiedDiffPreservesUnchangedLines(t *testing.T) {
	old := "# title\n\nunchanged line\n\nold line\n\nfooter\n"
	new := "# title\n\nunchanged line\n\nnew line\n\nfooter\n"
	got := unifiedDiff(old, new)
	if strings.Contains(got, "-unchanged line") || strings.Contains(got, "+unchanged line") {
		t.Fatalf("marked unchanged line as changed: %q", got)
	}
	if !strings.Contains(got, " unchanged line\n") {
		t.Fatalf("missing unchanged context: %q", got)
	}
	if !strings.Contains(got, "-old line\n") || !strings.Contains(got, "+new line\n") {
		t.Fatalf("missing changed lines: %q", got)
	}
	if strings.Count(got, "@@ ") != 1 {
		t.Fatalf("expected one hunk, got %q", got)
	}
}

func TestFindMetadataRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".4sight")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findMetadataRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("root %q want %q", got, want)
	}
}

func newMetadata(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".4sight")
	if err := os.MkdirAll(filepath.Join(root, "db"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
