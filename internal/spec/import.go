package spec

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

func ImportSchema(cwd, schemaRel string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runImportSchema(root, schemaRel, newDoltStore(filepath.Join(root, "db")))
}

func runImportSchema(metadataRoot, schemaRel string, st store) error {
	name, err := resolveSchemaFileName(schemaRel)
	if err != nil {
		return err
	}
	specPath := filepath.Join(metadataRoot, "schemas", name)
	info, err := os.Stat(specPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("schema spec %s does not exist", relFromMetadata(metadataRoot, specPath))
		}
		return fmt.Errorf("schema spec %s: %w", relFromMetadata(metadataRoot, specPath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("schema spec %s does not exist", relFromMetadata(metadataRoot, specPath))
	}
	bodyBytes, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("schema spec %s: %w", relFromMetadata(metadataRoot, specPath), err)
	}
	body := string(bodyBytes)
	prior, _, err := st.latestBody("schemas", name)
	if err != nil {
		return fmt.Errorf("lookup prior spec body: %w", err)
	}
	if err := st.insertSpec("schemas", name, body, unifiedDiff(prior, body)); err != nil {
		return fmt.Errorf("insert spec body: %w", err)
	}
	return nil
}

func resolveSchemaFileName(schemaRel string) (string, error) {
	schemaRel = strings.TrimSpace(schemaRel)
	if schemaRel == "" {
		return "", fmt.Errorf("missing schema spec path")
	}
	if filepath.IsAbs(schemaRel) {
		return "", fmt.Errorf("schema spec path must be relative to schemas dir, not %q", schemaRel)
	}
	clean := filepath.Clean(schemaRel)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("schema spec path must be relative to schemas dir, not %q", schemaRel)
	}
	if filepath.Base(clean) != clean {
		return "", fmt.Errorf("schema spec path must be a file in the schemas dir, not %q", schemaRel)
	}
	if !strings.HasSuffix(clean, ".md") {
		clean += ".md"
	}
	if !schemaFileRe.MatchString(clean) {
		return "", fmt.Errorf("schema spec path token %q is invalid", clean)
	}
	return clean, nil
}

var schemaFileRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*\.md$`)

type specDoc struct {
	Dir  string
	Name string
	Body string
}

type store interface {
	latestBody(dirPath, fileName string) (string, bool, error)
	insertSpec(dirPath, fileName, body, diff string) error
	listLatestBodies() ([]specDoc, error)
}

func isCodeSpecName(name string) bool {
	switch name {
	case "design.md", "cmd.md", "coding.md":
		return true
	}
	return false
}

func displaySpecPath(dirRel, name string) string {
	if dirRel == "" {
		return ".4sight/" + name
	}
	return ".4sight/" + dirRel + "/" + name
}

func locatorFromDirName(dir, name string) []string {
	if dir == "" {
		return []string{name}
	}
	parts := strings.Split(dir, "/")
	var loc []string
	i := 0
	if parts[0] == "main" {
		loc = append(loc, "main")
		i = 1
	}
	for i < len(parts) {
		if parts[i] == "subcommands" && i+1 < len(parts) {
			loc = append(loc, parts[i+1])
			i += 2
			continue
		}
		loc = append(loc, parts[i])
		i++
	}
	return append(loc, name)
}

func writeSpecFile(metadataRoot, dirRel, name, body string) (string, error) {
	path := filepath.Join(metadataRoot, filepath.FromSlash(dirRel), name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create spec dir for %s: %w", relFromMetadata(metadataRoot, path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write spec file %s: %w", relFromMetadata(metadataRoot, path), err)
	}
	return path, nil
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
	return unifiedDiffPaths("a", "b", oldText, newText)
}

func unifiedDiffPaths(oldPath, newPath, oldText, newText string) string {
	if oldText == newText {
		return ""
	}
	ops := lineDiffOps(diffLines(oldText), diffLines(newText))
	hunks := diffHunks(ops, 3)
	if len(hunks) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ %s\n", oldPath, newPath)
	for _, hunk := range hunks {
		writeDiffHunk(&b, ops, hunk)
	}
	return b.String()
}

type diffOp struct {
	tag  byte
	text string
}

func lineDiffOps(oldLines, newLines []string) []diffOp {
	m, n := len(oldLines), len(newLines)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var rev []diffOp
	for i, j := m, n; i > 0 || j > 0; {
		switch {
		case i > 0 && j > 0 && oldLines[i-1] == newLines[j-1]:
			rev = append(rev, diffOp{tag: ' ', text: oldLines[i-1]})
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			rev = append(rev, diffOp{tag: '+', text: newLines[j-1]})
			j--
		default:
			rev = append(rev, diffOp{tag: '-', text: oldLines[i-1]})
			i--
		}
	}
	ops := make([]diffOp, len(rev))
	for i, op := range rev {
		ops[len(rev)-1-i] = op
	}
	return ops
}

type diffHunk struct {
	start, end int
}

func diffHunks(ops []diffOp, context int) []diffHunk {
	if len(ops) == 0 {
		return nil
	}
	changed := make([]bool, len(ops))
	for i, op := range ops {
		changed[i] = op.tag != ' '
	}
	var ranges [][2]int
	for i := range ops {
		if !changed[i] {
			continue
		}
		start := max(0, i-context)
		end := min(len(ops), i+context+1)
		ranges = append(ranges, [2]int{start, end})
	}
	if len(ranges) == 0 {
		return nil
	}
	slices.SortFunc(ranges, func(a, b [2]int) int {
		return a[0] - b[0]
	})
	merged := [][2]int{ranges[0]}
	for _, r := range ranges[1:] {
		last := &merged[len(merged)-1]
		if r[0] <= last[1] {
			last[1] = max(last[1], r[1])
			continue
		}
		merged = append(merged, r)
	}
	hunks := make([]diffHunk, len(merged))
	for i, r := range merged {
		hunks[i] = diffHunk{start: r[0], end: r[1]}
	}
	return hunks
}

func writeDiffHunk(b *strings.Builder, ops []diffOp, hunk diffHunk) {
	oldStart, oldCount := diffSideSpan(ops, hunk, '-', ' ')
	newStart, newCount := diffSideSpan(ops, hunk, '+', ' ')
	fmt.Fprintf(b, "@@ -%s +%s @@\n", diffRange(oldStart, oldCount), diffRange(newStart, newCount))
	for _, op := range ops[hunk.start:hunk.end] {
		b.WriteByte(op.tag)
		b.WriteString(op.text)
		b.WriteByte('\n')
	}
}

func diffSideSpan(ops []diffOp, hunk diffHunk, removeTag, keepTag byte) (start, count int) {
	start = 1
	for _, op := range ops[:hunk.start] {
		if op.tag == keepTag || op.tag == removeTag {
			start++
		}
	}
	for _, op := range ops[hunk.start:hunk.end] {
		if op.tag == keepTag || op.tag == removeTag {
			count++
		}
	}
	return start, count
}

func diffRange(start, count int) string {
	if count == 0 {
		return "0,0"
	}
	return fmt.Sprintf("%d,%d", start, count)
}

func diffLines(s string) []string {
	if s == "" {
		return nil
	}
	s, _ = strings.CutSuffix(s, "\n")
	return strings.Split(s, "\n")
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
