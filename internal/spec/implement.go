package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func ImplementPrompt(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runImplementPrompt(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runImplementPrompt(metadataRoot string, st store, stdout io.Writer) error {
	docs, err := st.listLatestBodies()
	if err != nil {
		return fmt.Errorf("list spec bodies: %w", err)
	}
	var codeDocs []specDoc
	for _, doc := range docs {
		if isCodeSpecName(doc.Name) {
			codeDocs = append(codeDocs, doc)
		}
	}
	slices.SortFunc(codeDocs, func(a, b specDoc) int {
		if c := strings.Compare(a.Dir, b.Dir); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})

	schemas, err := listSchemaSpecs(filepath.Join(metadataRoot, "schemas"))
	if err != nil {
		return err
	}

	projectRoot := filepath.Dir(metadataRoot)
	var b strings.Builder
	b.WriteString("Read these design and command specification documents from the spec database to understand the application behavior:\n")
	for _, doc := range codeDocs {
		fmt.Fprintf(&b, "\n----- SPEC %s -----\n", displaySpecPath(doc.Dir, doc.Name))
		b.WriteString(doc.Body)
		if !strings.HasSuffix(doc.Body, "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("----- END SPEC -----\n")
	}
	b.WriteByte('\n')
	b.WriteString("Read these database schema specifications:\n")
	for _, schema := range schemas {
		fmt.Fprintf(&b, "- %s\n", relFromProject(projectRoot, schema))
	}
	b.WriteByte('\n')
	b.WriteString("Review the application code specification and update the code to match the spec.\n")
	b.WriteString("Do not update the schema.\n")
	b.WriteString("If the schema needs a spec change or a migration to be compatible with the code spec, stop and tell the user the gap (schema spec or migration, and what kinds of changes are needed).\n")
	_, err = io.WriteString(stdout, b.String())
	return err
}

func listSchemaSpecs(schemasDir string) ([]string, error) {
	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read schemas dir %s: %w", schemasDir, err)
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		paths = append(paths, filepath.Join(schemasDir, entry.Name()))
	}
	slices.Sort(paths)
	return paths, nil
}

func relFromProject(projectRoot, path string) string {
	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
