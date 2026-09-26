package spec

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

func ChangeImport(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeImport(root, newDoltStore(filepath.Join(root, "db")), os.Stdout, os.Stderr)
}

type plannedChange struct {
	Dir     string
	Name    string
	Path    string
	Body    string
	Diff    string
	Summary string
}

type changeImportStore interface {
	listWorkspaceBodies() ([]specDoc, error)
	workspaceChangeID() (string, error)
	changeIDs() ([]string, error)
	applyChangeImport(id, parent string, files []plannedChange) error
}

func runChangeImport(metadataRoot string, st changeImportStore, stdout, stderr io.Writer) error {
	disk, err := listOnDiskCodeSpecs(metadataRoot)
	if err != nil {
		return err
	}
	bodies, err := st.listWorkspaceBodies()
	if err != nil {
		return fmt.Errorf("list workspace bodies: %w", err)
	}
	planned, err := planChangeImport(metadataRoot, disk, bodies)
	if err != nil {
		return err
	}
	if len(planned) == 0 {
		_, err := fmt.Fprintln(stderr, "no changes found")
		return err
	}
	parent, err := st.workspaceChangeID()
	if err != nil {
		return fmt.Errorf("lookup workspace change: %w", err)
	}
	ids, err := st.changeIDs()
	if err != nil {
		return fmt.Errorf("list changes: %w", err)
	}
	id, err := nextChangeID(ids)
	if err != nil {
		return fmt.Errorf("next change id: %w", err)
	}
	if err := st.applyChangeImport(id, parent, planned); err != nil {
		return fmt.Errorf("record change import: %w", err)
	}
	return writeChangeImportYAML(stdout, id, planned)
}

func planChangeImport(metadataRoot string, disk map[specKey]string, bodies []specDoc) ([]plannedChange, error) {
	dbBodies := make(map[specKey]string, len(bodies))
	for _, doc := range bodies {
		dbBodies[specKey{dir: doc.Dir, name: doc.Name}] = doc.Body
	}
	keys := make(map[specKey]bool, len(disk)+len(dbBodies))
	for key := range disk {
		keys[key] = true
	}
	for key := range dbBodies {
		keys[key] = true
	}

	var planned []plannedChange
	for key := range keys {
		rawDisk, onDisk, err := diskSpec(metadataRoot, disk, key)
		if err != nil {
			return nil, err
		}
		oldText := ""
		if body, ok := dbBodies[key]; ok {
			oldText = normalizeSpecBody(body)
		}
		newText := ""
		if onDisk {
			newText = normalizeSpecBody(rawDisk)
		}
		if oldText == newText {
			continue
		}
		path := specRelPath(key.dir, key.name)
		diff := unifiedDiffPaths(path, path, oldText, newText)
		if diff == "" {
			continue
		}
		add, del := lineChangeCounts(oldText, newText)
		planned = append(planned, plannedChange{
			Dir:     key.dir,
			Name:    key.name,
			Path:    path,
			Body:    newText,
			Diff:    diff,
			Summary: fmt.Sprintf("+%d/-%d", add, del),
		})
	}
	slices.SortFunc(planned, func(a, b plannedChange) int {
		return strings.Compare(a.Path, b.Path)
	})
	return planned, nil
}

func diskSpec(metadataRoot string, disk map[specKey]string, key specKey) (string, bool, error) {
	if body, ok := disk[key]; ok {
		return body, true, nil
	}
	path := filepath.Join(metadataRoot, filepath.FromSlash(key.dir), key.name)
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read spec file %s: %w", relFromMetadata(metadataRoot, path), err)
	}
	return string(body), true, nil
}

func specRelPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func normalizeSpecBody(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimRight(s, "\r\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRightFunc(line, unicode.IsSpace)
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func lineChangeCounts(oldText, newText string) (add, del int) {
	for _, op := range lineDiffOps(diffLines(oldText), diffLines(newText)) {
		switch op.tag {
		case '+':
			add++
		case '-':
			del++
		}
	}
	return add, del
}

func writeChangeImportYAML(stdout io.Writer, id string, files []plannedChange) error {
	if _, err := fmt.Fprintf(stdout, "id: %s\nfile_changes:\n", id); err != nil {
		return err
	}
	for _, file := range files {
		if _, err := fmt.Fprintf(stdout, "  - path: %s\n    summary: %s\n", file.Path, file.Summary); err != nil {
			return err
		}
	}
	return nil
}

const changeIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

const (
	changeIDDigitCount         = 1000
	changeIDLetterDecimalCount = 26 * 100
	changeIDTailCount          = 26 * len(changeIDAlphabet)
	changeIDLetterDecimalStart = changeIDDigitCount
	changeIDTailStart          = changeIDDigitCount + changeIDLetterDecimalCount
	changeIDSpace              = changeIDTailStart + changeIDTailCount
)

func nextChangeID(ids []string) (string, error) {
	maxRank := -1
	for _, id := range ids {
		rank, err := changeIDRank(id)
		if err != nil {
			return "", err
		}
		if rank > maxRank {
			maxRank = rank
		}
	}
	return formatChangeID(maxRank + 1)
}

func changeIDRank(id string) (int, error) {
	suf, err := changeSuffix(id)
	if err != nil {
		return 0, err
	}
	if n, ok := decimalSuffix(suf); ok {
		return n, nil
	}
	if n, ok := letterDecimalSuffix(suf); ok {
		return changeIDLetterDecimalStart + n, nil
	}
	if n, ok := tailSuffix(suf); ok {
		return changeIDTailStart + n, nil
	}
	return 0, fmt.Errorf("invalid change id %q", id)
}

func formatChangeID(rank int) (string, error) {
	if rank < 0 || rank >= changeIDSpace {
		return "", fmt.Errorf("change id space exhausted")
	}
	var suf string
	switch {
	case rank < changeIDDigitCount:
		suf = fmt.Sprintf("%03d", rank)
	case rank < changeIDTailStart:
		n := rank - changeIDLetterDecimalStart
		suf = fmt.Sprintf("%c%02d", 'a'+rune(n/100), n%100)
	default:
		n := rank - changeIDTailStart
		suf = fmt.Sprintf("z%c%c", 'a'+rune(n/len(changeIDAlphabet)), changeIDAlphabet[n%len(changeIDAlphabet)])
	}
	return "ch-" + suf, nil
}

func changeSuffix(id string) (string, error) {
	if len(id) != 6 || id[:3] != "ch-" {
		return "", fmt.Errorf("invalid change id %q", id)
	}
	return id[3:], nil
}

func decimalSuffix(suf string) (int, bool) {
	n := 0
	for i := 0; i < len(suf); i++ {
		if suf[i] < '0' || suf[i] > '9' {
			return 0, false
		}
		n = n*10 + int(suf[i]-'0')
	}
	return n, true
}

func letterDecimalSuffix(suf string) (int, bool) {
	if len(suf) != 3 || suf[0] < 'a' || suf[0] > 'z' || !isDigit(suf[1]) || !isDigit(suf[2]) {
		return 0, false
	}
	return int(suf[0]-'a')*100 + int(suf[1]-'0')*10 + int(suf[2]-'0'), true
}

func tailSuffix(suf string) (int, bool) {
	if len(suf) != 3 || suf[0] != 'z' || suf[1] < 'a' || suf[1] > 'z' {
		return 0, false
	}
	third := strings.IndexByte(changeIDAlphabet, suf[2])
	if third < 0 {
		return 0, false
	}
	return int(suf[1]-'a')*36 + third, true
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func (s *doltStore) listWorkspaceBodies() ([]specDoc, error) {
	rows, err := s.queryCSV(
		"SELECT d.path, f.name, b.body FROM spec.bodies b" +
			" JOIN spec.files f ON f.id = b.file_id" +
			" JOIN spec.dirs d ON d.id = f.dir_id" +
			" WHERE b.state = 'workspace'" +
			" ORDER BY d.path, f.name",
	)
	if err != nil {
		return nil, err
	}
	docs := make([]specDoc, 0, len(rows))
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		docs = append(docs, specDoc{Dir: row[0], Name: row[1], Body: row[2]})
	}
	return docs, nil
}

func (s *doltStore) workspaceChangeID() (string, error) {
	rows, err := s.queryCSV("SELECT workspace_change_id FROM spec.current_changes")
	if err != nil {
		return "", err
	}
	if len(rows) == 0 || len(rows[0]) == 0 || rows[0][0] == "" {
		return "", fmt.Errorf("workspace change is not set")
	}
	return rows[0][0], nil
}

func (s *doltStore) changeIDs() ([]string, error) {
	rows, err := s.queryCSV("SELECT id FROM spec.changes")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 || row[0] == "" {
			continue
		}
		ids = append(ids, row[0])
	}
	return ids, nil
}

func (s *doltStore) applyChangeImport(id, parent string, files []plannedChange) error {
	if _, err := changeIDRank(id); err != nil {
		return err
	}
	if _, err := changeIDRank(parent); err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no file changes to import")
	}
	rows, err := s.queryCSV("SELECT COUNT(*) FROM spec.changes_closure WHERE child_id = " + sqlString(parent))
	if err != nil {
		return err
	}
	if len(rows) == 0 || len(rows[0]) == 0 || rows[0][0] == "0" {
		return fmt.Errorf("workspace change %s has no closure rows", parent)
	}

	type storedFile struct {
		fileID uint64
		file   plannedChange
	}
	stored := make([]storedFile, 0, len(files))
	for _, file := range files {
		dirID, err := s.ensureDir(file.Dir)
		if err != nil {
			return err
		}
		fileID, err := s.ensureFile(dirID, file.Name)
		if err != nil {
			return err
		}
		stored = append(stored, storedFile{fileID: fileID, file: file})
	}

	var script strings.Builder
	script.WriteString("START TRANSACTION;\n")
	fmt.Fprintf(&script, "INSERT INTO spec.changes (id, prompt) VALUES (%s, '');\n", sqlString(id))
	fmt.Fprintf(&script,
		"INSERT INTO spec.changes_closure (parent_id, child_id, depth) "+
			"SELECT parent_id, %s, depth + 1 FROM spec.changes_closure WHERE child_id = %s;\n",
		sqlString(id), sqlString(parent),
	)
	fmt.Fprintf(&script,
		"INSERT INTO spec.changes_closure (parent_id, child_id, depth) VALUES (%s, %s, 0);\n",
		sqlString(id), sqlString(id),
	)
	fmt.Fprintf(&script,
		"UPDATE spec.current_changes SET workspace_change_id = %s WHERE singleton = 1;\n",
		sqlString(id),
	)
	for _, item := range stored {
		fmt.Fprintf(&script,
			"INSERT INTO spec.file_changes (change_id, file_id, diff) VALUES (%s, %d, %s);\n",
			sqlString(id), item.fileID, sqlBodyLiteral(item.file.Diff),
		)
		fmt.Fprintf(&script,
			"DELETE FROM spec.bodies WHERE file_id = %d AND state = 'workspace';\n",
			item.fileID,
		)
		fmt.Fprintf(&script,
			"INSERT INTO spec.bodies (body, file_id, file_change_id, state, ieee_crc32, summary) "+
				"SELECT %s, %d, fc.id, 'workspace', %d, '' FROM spec.file_changes fc "+
				"WHERE fc.change_id = %s AND fc.file_id = %d;\n",
			sqlBodyLiteral(item.file.Body),
			item.fileID,
			crc32.ChecksumIEEE([]byte(item.file.Body)),
			sqlString(id),
			item.fileID,
		)
	}
	script.WriteString("COMMIT;\n")
	if err := s.run("sql", "-q", script.String()); err != nil {
		return err
	}
	countRows, err := s.queryCSV(fmt.Sprintf(
		"SELECT COUNT(*) FROM spec.bodies b JOIN spec.file_changes fc ON fc.id = b.file_change_id "+
			"WHERE fc.change_id = %s AND b.state = 'workspace'",
		sqlString(id),
	))
	if err != nil {
		return err
	}
	if len(countRows) == 0 || len(countRows[0]) == 0 || countRows[0][0] != fmt.Sprintf("%d", len(files)) {
		return fmt.Errorf("change %s did not record %d workspace bodies", id, len(files))
	}
	return nil
}
