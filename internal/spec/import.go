package spec

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Import(cwd, specRel string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runImport(root, specRel, newDoltStore(filepath.Join(root, "db")))
}

type store interface {
	insertBody(body string) error
}

func runImport(metadataRoot, specRel string, st store) error {
	specRel = strings.TrimSpace(specRel)
	if specRel == "" {
		return fmt.Errorf("missing spec file path")
	}
	if filepath.IsAbs(specRel) {
		return fmt.Errorf("spec file path must be relative to specs dir, not %q", specRel)
	}
	clean := filepath.Clean(specRel)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("spec file path must be relative to specs dir, not %q", specRel)
	}

	specPath := filepath.Join(metadataRoot, "specs", clean)
	info, err := os.Stat(specPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("spec file %s does not exist", relFromMetadata(metadataRoot, specPath))
		}
		return fmt.Errorf("spec file %s: %w", relFromMetadata(metadataRoot, specPath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("spec file %s does not exist", relFromMetadata(metadataRoot, specPath))
	}

	body, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("spec file %s: %w", relFromMetadata(metadataRoot, specPath), err)
	}
	if err := st.insertBody(string(body)); err != nil {
		return fmt.Errorf("insert spec body: %w", err)
	}
	return nil
}

func findMetadataRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	for dir := abs; ; {
		candidate := filepath.Join(dir, ".4sight")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .4sight directory found from %s", start)
		}
		dir = parent
	}
}

func relFromMetadata(metadataRoot, path string) string {
	rel, err := filepath.Rel(metadataRoot, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(filepath.Join(".4sight", rel))
}

func sqlBodyLiteral(body string) string {
	return "CONVERT(UNHEX('" + hex.EncodeToString([]byte(body)) + "') USING utf8mb4)"
}
