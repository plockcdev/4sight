package spec

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var specTokenRe = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)

func Import(cwd string, loc []string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runImport(root, loc, newDoltStore(filepath.Join(root, "db")))
}

type store interface {
	latestBody(dirPath, fileName string) (string, bool, error)
	insertSpec(dirPath, fileName, body, diff string) error
}

func runImport(metadataRoot string, loc []string, st store) error {
	dirRel, name, err := resolveSpecLocation(loc)
	if err != nil {
		return err
	}

	specPath := filepath.Join(metadataRoot, filepath.FromSlash(dirRel), name)
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

	bodyBytes, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("spec file %s: %w", relFromMetadata(metadataRoot, specPath), err)
	}
	body := string(bodyBytes)

	prior, _, err := st.latestBody(dirRel, name)
	if err != nil {
		return fmt.Errorf("lookup prior spec body: %w", err)
	}
	if err := st.insertSpec(dirRel, name, body, unifiedDiff(prior, body)); err != nil {
		return fmt.Errorf("insert spec body: %w", err)
	}
	return nil
}

func resolveSpecLocation(loc []string) (dirRel, name string, err error) {
	if len(loc) == 0 {
		return "", "", fmt.Errorf("missing spec file location")
	}
	for _, part := range loc {
		if strings.TrimSpace(part) == "" || part != strings.TrimSpace(part) {
			return "", "", fmt.Errorf("spec file location %q is invalid", strings.Join(loc, " "))
		}
		if !specTokenRe.MatchString(part) {
			return "", "", fmt.Errorf("spec file location token %q is invalid", part)
		}
	}
	name = loc[len(loc)-1]
	cmdPath := loc[:len(loc)-1]
	if len(cmdPath) == 0 {
		return "", name, nil
	}
	var parts []string
	if cmdPath[0] == "main" {
		parts = append(parts, "main")
		cmdPath = cmdPath[1:]
	}
	for _, seg := range cmdPath {
		parts = append(parts, "subcommands", seg)
	}
	return strings.Join(parts, "/"), name, nil
}

func unifiedDiff(oldText, newText string) string {
	if oldText == newText {
		return ""
	}
	oldLines := diffLines(oldText)
	newLines := diffLines(newText)
	var b strings.Builder
	b.WriteString("--- a\n+++ b\n")
	fmt.Fprintf(&b, "@@ -%s +%s @@\n", hunkSpan(oldLines), hunkSpan(newLines))
	for _, line := range oldLines {
		b.WriteString("-")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	for _, line := range newLines {
		b.WriteString("+")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func diffLines(s string) []string {
	if s == "" {
		return nil
	}
	s, _ = strings.CutSuffix(s, "\n")
	return strings.Split(s, "\n")
}

func hunkSpan(lines []string) string {
	if len(lines) == 0 {
		return "0,0"
	}
	return fmt.Sprintf("1,%d", len(lines))
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

func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
