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
	return runImplementPrompt(root, os.Stdout)
}

func runImplementPrompt(metadataRoot string, stdout io.Writer) error {
	specDocs, err := listCommandSpecDocs(metadataRoot)
	if err != nil {
		return err
	}
	schemas, err := listSchemaSpecs(filepath.Join(metadataRoot, "schemas"))
	if err != nil {
		return err
	}

	projectRoot := filepath.Dir(metadataRoot)
	var b strings.Builder
	b.WriteString("Read these design and command specification documents to understand the application behavior:\n")
	for _, doc := range specDocs {
		fmt.Fprintf(&b, "- %s\n", relFromProject(projectRoot, doc))
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

func listCommandSpecDocs(metadataRoot string) ([]string, error) {
	var docs []string
	for _, name := range []string{"design.md", "cmd.md"} {
		path := filepath.Join(metadataRoot, name)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("list spec docs under %s: %w", metadataRoot, err)
		}
		if !info.IsDir() {
			docs = append(docs, path)
		}
	}
	for _, dir := range []string{"main", "subcommands"} {
		base := filepath.Join(metadataRoot, dir)
		info, err := os.Stat(base)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("list spec docs under %s: %w", base, err)
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch d.Name() {
			case "design.md", "cmd.md":
				docs = append(docs, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("list spec docs under %s: %w", base, err)
		}
	}
	slices.Sort(docs)
	return docs, nil
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
