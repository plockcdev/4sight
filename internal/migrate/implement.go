package migrate

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func ImplementPrompt(cwd, schemaRel string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runImplementPrompt(root, schemaRel, os.Stdout, randomMigrationSuffix)
}

func runImplementPrompt(metadataRoot, schemaRel string, stdout io.Writer, suffix func() (string, error)) error {
	schemaRel = strings.TrimSpace(schemaRel)
	if schemaRel == "" {
		return fmt.Errorf("missing schema spec path")
	}
	if filepath.IsAbs(schemaRel) {
		return fmt.Errorf("schema spec path must be relative to schemas dir, not %q", schemaRel)
	}
	clean := filepath.Clean(schemaRel)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("schema spec path must be relative to schemas dir, not %q", schemaRel)
	}
	if !strings.HasSuffix(clean, ".md") {
		clean += ".md"
	}

	schemasDir := filepath.Join(metadataRoot, "schemas")
	schemaPath := filepath.Join(schemasDir, clean)
	if _, err := os.Stat(schemaPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("schema spec %s does not exist", relFromMetadata(metadataRoot, schemaPath))
		}
		return fmt.Errorf("schema spec %s: %w", relFromMetadata(metadataRoot, schemaPath), err)
	}

	db, err := databaseFromSchemaName(filepath.Base(clean))
	if err != nil {
		return fmt.Errorf("schema spec %s: %w", relFromMetadata(metadataRoot, schemaPath), err)
	}

	migDir := filepath.Join(metadataRoot, "migrations", db.key())
	parentFileName, err := resolveLink(migDir, submittedName)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: missing submitted symlink to an existing file in %s", db.key(), migDir)
		}
		return fmt.Errorf("%s: submitted in %s: %w", db.key(), migDir, err)
	}
	submitted, err := parseMigrationName(parentFileName)
	if err != nil {
		return fmt.Errorf("%s: submitted %s in %s: %w", db.key(), parentFileName, migDir, err)
	}

	suffixPart, err := suffix()
	if err != nil {
		return fmt.Errorf("%s: generate migration id: %w", db.key(), err)
	}
	nextID, err := nextMigrationID(submitted.id, suffixPart)
	if err != nil {
		return fmt.Errorf("%s: %w", db.key(), err)
	}
	migrationName := fmt.Sprintf("%s-%s.sql", nextID, submitted.id)
	targetPath := filepath.Join(migDir, migrationName)

	specDocs, err := listSpecDocs(metadataRoot)
	if err != nil {
		return err
	}

	projectRoot := filepath.Dir(metadataRoot)
	schemaDisplay := relFromProject(projectRoot, schemaPath)
	parentDisplay := relFromProject(projectRoot, filepath.Join(migDir, parentFileName))
	targetDisplay := relFromProject(projectRoot, targetPath)
	submittedSymlinkDisplay := relFromProject(projectRoot, filepath.Join(migDir, submittedName))

	if _, err := fmt.Fprintln(stdout, "Generate a SQL migration file for the database schema."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Read these design and command specification documents to understand what the schema must provide:"); err != nil {
		return err
	}
	for _, doc := range specDocs {
		if _, err := fmt.Fprintf(stdout, "- %s\n", relFromProject(projectRoot, doc)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "The parent migration is the file the submitted symlink currently points to:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "- %s\n", parentDisplay); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Read the schema specification:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "- %s\n", schemaDisplay); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Generate the migration file at:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "- %s\n", targetDisplay); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "After writing the migration file, move submitted to point to the new migration:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "- %s -> %s\n", submittedSymlinkDisplay, migrationName); err != nil {
		return err
	}
	return nil
}

func databaseFromSchemaName(name string) (database, error) {
	stem, ok := strings.CutSuffix(name, ".md")
	if !ok {
		return database{}, fmt.Errorf("expected <db_type>_<db_name>.md")
	}
	dbType, dbName, ok := strings.Cut(stem, "_")
	if !ok || dbType == "" || dbName == "" {
		return database{}, fmt.Errorf("expected <db_type>_<db_name>.md")
	}
	return database{dbType: dbType, name: dbName}, nil
}

func nextMigrationID(submittedID, suffix string) (string, error) {
	if len(submittedID) != 8 || !migrationIDRe.MatchString(submittedID) {
		return "", fmt.Errorf("invalid submitted migration id %q", submittedID)
	}
	prefix, err := strconv.ParseUint(submittedID[:4], 10, 16)
	if err != nil {
		return "", fmt.Errorf("invalid submitted migration id %q", submittedID)
	}
	if prefix >= 9999 {
		return "", fmt.Errorf("migration id prefix overflow at %s", submittedID)
	}
	if len(suffix) != 4 || !migrationSuffixRe.MatchString(suffix) {
		return "", fmt.Errorf("invalid migration id suffix %q", suffix)
	}
	return fmt.Sprintf("%04d%s", prefix+1, suffix), nil
}

func randomMigrationSuffix() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}

func listSpecDocs(metadataRoot string) ([]string, error) {
	var docs []string
	err := filepath.WalkDir(metadataRoot, func(path string, d os.DirEntry, err error) error {
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
		return nil, fmt.Errorf("list spec docs under %s: %w", metadataRoot, err)
	}
	slices.Sort(docs)
	return docs, nil
}

func relFromMetadata(metadataRoot, path string) string {
	rel, err := filepath.Rel(metadataRoot, path)
	if err != nil {
		return path
	}
	return filepath.Join(".4sight", rel)
}

func relFromProject(projectRoot, path string) string {
	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
