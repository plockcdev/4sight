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
	Dir            string
	Name           string
	Path           string
	PriorPath      string
	RelatedPath    string
	Body           string
	OldBody        string
	Diff           string
	Summary        string
	Deleted        bool
	Similarity     float64
	CopyCandidates []string
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
	for _, file := range planned {
		if len(file.CopyCandidates) > 0 {
			if _, err := fmt.Fprintf(stderr, "ambiguous copy candidates for %s: %s\n", file.Path, strings.Join(file.CopyCandidates, ", ")); err != nil {
				return fmt.Errorf("report ambiguous copy candidates: %w", err)
			}
		}
	}
	return writeChangeImportYAML(stdout, id, planned)
}

func planChangeImport(metadataRoot string, disk map[specKey]string, bodies []specDoc) ([]plannedChange, error) {
	dbBodies := make(map[specKey]specDoc, len(bodies))
	for _, doc := range bodies {
		dbBodies[specKey{dir: doc.Dir, name: doc.Name}] = doc
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
		oldDeleted := true
		if body, ok := dbBodies[key]; ok {
			oldDeleted = body.Deleted
		}
		if body, ok := dbBodies[key]; ok && !body.Deleted {
			oldText = normalizeTrackedBody(key.dir, key.name, body.Body)
		}
		newText := ""
		if onDisk {
			newText = normalizeTrackedBody(key.dir, key.name, rawDisk)
		}
		newDeleted := !onDisk
		if oldText == newText && oldDeleted == newDeleted {
			continue
		}
		path := specRelPath(key.dir, key.name)
		oldDiffPath, newDiffPath := path, path
		if oldDeleted {
			oldDiffPath = "/dev/null"
		}
		if newDeleted {
			newDiffPath = "/dev/null"
		}
		diff := unifiedDiffPaths(oldDiffPath, newDiffPath, oldText, newText)
		add, del := lineChangeCounts(oldText, newText)
		planned = append(planned, plannedChange{
			Dir:     key.dir,
			Name:    key.name,
			Path:    path,
			Body:    newText,
			OldBody: oldText,
			Diff:    diff,
			Summary: fmt.Sprintf("+%d/-%d", add, del),
			Deleted: newDeleted,
		})
	}
	planned = associateRenames(planned)
	planned = associateCopies(planned, dbBodies, disk)
	slices.SortFunc(planned, func(a, b plannedChange) int {
		return strings.Compare(a.Path, b.Path)
	})
	return planned, nil
}

func associateCopies(planned []plannedChange, bodies map[specKey]specDoc, disk map[specKey]string) []plannedChange {
	type source struct {
		path, lookupPath, body string
	}
	sources := make(map[string]source)
	for key, doc := range bodies {
		if doc.Deleted {
			continue
		}
		if content, ok := disk[key]; ok {
			path := specRelPath(key.dir, key.name)
			sources[path] = source{path: path, lookupPath: path, body: normalizeTrackedBody(key.dir, key.name, content)}
		}
	}
	for _, change := range planned {
		if change.PriorPath != "" && !change.Deleted {
			sources[change.Path] = source{path: change.Path, lookupPath: change.PriorPath, body: change.Body}
		}
	}
	for i := range planned {
		copy := &planned[i]
		if copy.Deleted || copy.PriorPath != "" || copy.OldBody != "" || copy.Body == "" {
			continue
		}
		var candidates []source
		var scores []float64
		for _, src := range sources {
			if src.path == copy.Path || src.body == "" {
				continue
			}
			score := fileSimilarity(src.body, copy.Body)
			if score >= 0.60 {
				candidates = append(candidates, src)
				scores = append(scores, score)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		best, second := 0, 0.0
		for i := range candidates {
			if scores[i] > scores[best] {
				second = scores[best]
				best = i
			} else if scores[i] > second {
				second = scores[i]
			}
		}
		if scores[best]-second >= 0.10 {
			copy.RelatedPath = candidates[best].lookupPath
			copy.Similarity = scores[best]
			continue
		}
		for i, candidate := range candidates {
			copy.CopyCandidates = append(copy.CopyCandidates, fmt.Sprintf("%s (%.5f)", candidate.path, scores[i]))
		}
		slices.Sort(copy.CopyCandidates)
	}
	return planned
}

func associateRenames(planned []plannedChange) []plannedChange {
	var removed, added []int
	for i, change := range planned {
		if change.Deleted && change.OldBody != "" {
			removed = append(removed, i)
		} else if !change.Deleted && change.OldBody == "" && change.Body != "" {
			added = append(added, i)
		}
	}
	if len(removed) == 0 || len(added) == 0 {
		return planned
	}
	type candidate struct {
		removed, added int
		score          float64
	}
	var candidates []candidate
	for _, oldIndex := range removed {
		for _, newIndex := range added {
			oldFile, newFile := planned[oldIndex], planned[newIndex]
			score := fileSimilarity(oldFile.OldBody, newFile.Body)
			if score >= 0.60 {
				candidates = append(candidates, candidate{removed: oldIndex, added: newIndex, score: score})
			}
		}
	}
	bestForOld := make(map[int]candidate)
	secondForOld := make(map[int]float64)
	bestForNew := make(map[int]candidate)
	secondForNew := make(map[int]float64)
	for _, item := range candidates {
		if best, ok := bestForOld[item.removed]; !ok || item.score > best.score {
			if ok {
				secondForOld[item.removed] = best.score
			}
			bestForOld[item.removed] = item
		} else if item.score > secondForOld[item.removed] {
			secondForOld[item.removed] = item.score
		}
		if best, ok := bestForNew[item.added]; !ok || item.score > best.score {
			if ok {
				secondForNew[item.added] = best.score
			}
			bestForNew[item.added] = item
		} else if item.score > secondForNew[item.added] {
			secondForNew[item.added] = item.score
		}
	}
	paired := make(map[int]int)
	for oldIndex, best := range bestForOld {
		if best.score-secondForOld[oldIndex] < 0.10 {
			continue
		}
		newBest, ok := bestForNew[best.added]
		if !ok || newBest.removed != oldIndex || newBest.score-secondForNew[best.added] < 0.10 {
			continue
		}
		paired[oldIndex] = best.added
	}
	if len(paired) == 0 {
		return planned
	}
	removedIndexes := make(map[int]bool)
	for oldIndex, newIndex := range paired {
		oldFile, newFile := planned[oldIndex], planned[newIndex]
		newFile.PriorPath = oldFile.Path
		newFile.OldBody = oldFile.OldBody
		newFile.Diff = unifiedDiffPaths(oldFile.Path, newFile.Path, oldFile.OldBody, newFile.Body)
		add, del := lineChangeCounts(oldFile.OldBody, newFile.Body)
		newFile.Summary = fmt.Sprintf("+%d/-%d", add, del)
		newFile.Similarity = bestForOld[oldIndex].score
		planned[newIndex] = newFile
		removedIndexes[oldIndex] = true
	}
	result := make([]plannedChange, 0, len(planned)-len(removedIndexes))
	for i, change := range planned {
		if !removedIndexes[i] {
			result = append(result, change)
		}
	}
	return result
}

func fileSimilarity(oldText, newText string) float64 {
	oldLines, newLines := diffLines(oldText), diffLines(newText)
	if len(oldLines) == 0 || len(newLines) == 0 {
		return 0
	}
	var added, deleted int
	for _, op := range lineDiffOps(oldLines, newLines) {
		switch op.tag {
		case '+':
			added++
		case '-':
			deleted++
		}
	}
	denominator := max(len(oldLines), len(newLines))
	return maxFloat(0, 1-float64(added+deleted)/float64(denominator))
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
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

func splitSpecRelPath(path string) (dir, name string) {
	index := strings.LastIndexByte(path, '/')
	if index < 0 {
		return "", path
	}
	return path[:index], path[index+1:]
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
		"SELECT d.path, p.name, b.body, b.deleted FROM spec.bodies b" +
			" JOIN spec.files f ON f.id = b.file_id" +
			" JOIN spec.file_changes fc ON fc.id = b.file_change_id" +
			" JOIN spec.filepaths p ON p.id = fc.filepath_id" +
			" JOIN spec.dirs d ON d.id = p.dir_id" +
			" WHERE b.state = 'workspace'" +
			" ORDER BY d.path, p.name, b.deleted DESC",
	)
	if err != nil {
		return nil, err
	}
	docs := make([]specDoc, 0, len(rows))
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		docs = append(docs, specDoc{Dir: row[0], Name: row[1], Body: row[2], Deleted: row[3] == "1" || strings.EqualFold(row[3], "true")})
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
		fileID        uint64
		pathID        uint64
		relatedFileID uint64
		file          plannedChange
	}
	stored := make([]storedFile, 0, len(files))
	for _, file := range files {
		dirID, err := s.ensureDir(file.Dir)
		if err != nil {
			return err
		}
		var fileID, pathID uint64
		if file.PriorPath != "" {
			oldDir, oldName := splitSpecRelPath(file.PriorPath)
			oldDirID, err := s.ensureDir(oldDir)
			if err != nil {
				return err
			}
			oldPathID, ok, err := s.lookupID(fmt.Sprintf("SELECT id FROM spec.filepaths WHERE dir_id = %d AND name = %s", oldDirID, sqlString(oldName)))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("previous filepath %s was not found", file.PriorPath)
			}
			fileID, ok, err = s.lookupID(fmt.Sprintf("SELECT id FROM spec.files WHERE current_filepath_id = %d", oldPathID))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("live file at previous filepath %s was not found", file.PriorPath)
			}
			pathID, err = s.ensureFilepath(dirID, file.Name)
			if err != nil {
				return err
			}
		} else {
			fileID, err = s.ensureFile(dirID, file.Name)
			if err != nil {
				return err
			}
			pathID, err = s.ensureFilepath(dirID, file.Name)
			if err != nil {
				return err
			}
		}
		_, ok, err := s.lookupID(fmt.Sprintf("SELECT id FROM spec.filepaths WHERE id = %d", pathID))
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("filepath %s was not found after insert", file.Path)
		}
		var relatedFileID uint64
		if file.RelatedPath != "" {
			relatedDir, relatedName := splitSpecRelPath(file.RelatedPath)
			relatedDirID, err := s.ensureDir(relatedDir)
			if err != nil {
				return err
			}
			relatedPathID, found, err := s.lookupID(fmt.Sprintf("SELECT id FROM spec.filepaths WHERE dir_id = %d AND name = %s", relatedDirID, sqlString(relatedName)))
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("copy source filepath %s was not found", file.RelatedPath)
			}
			relatedFileID, found, err = s.lookupID(fmt.Sprintf("SELECT id FROM spec.files WHERE current_filepath_id = %d", relatedPathID))
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("live copy source %s was not found", file.RelatedPath)
			}
		}
		stored = append(stored, storedFile{fileID: fileID, pathID: pathID, relatedFileID: relatedFileID, file: file})
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
		deleted := 0
		if item.file.Deleted {
			deleted = 1
		}
		similarity := "NULL"
		if item.file.PriorPath != "" || item.file.RelatedPath != "" {
			similarity = fmt.Sprintf("%.5f", item.file.Similarity)
		}
		relatedFileID := "NULL"
		if item.relatedFileID != 0 {
			relatedFileID = fmt.Sprintf("%d", item.relatedFileID)
		}
		fmt.Fprintf(&script,
			"INSERT INTO spec.file_changes (change_id, file_id, filepath_id, diff, deleted, related_file_id, similarity) VALUES (%s, %d, %d, %s, %d, %s, %s);\n",
			sqlString(id), item.fileID, item.pathID, sqlBodyLiteral(item.file.Diff), deleted, relatedFileID, similarity,
		)
		fmt.Fprintf(&script,
			"DELETE FROM spec.bodies WHERE file_id = %d AND state = 'workspace';\n",
			item.fileID,
		)
		fmt.Fprintf(&script,
			"INSERT INTO spec.bodies (body, file_id, file_change_id, state, ieee_crc32, summary, deleted) "+
				"SELECT %s, %d, fc.id, 'workspace', %d, '', %d FROM spec.file_changes fc "+
				"WHERE fc.change_id = %s AND fc.file_id = %d;\n",
			sqlBodyLiteral(item.file.Body),
			item.fileID,
			crc32.ChecksumIEEE([]byte(item.file.Body)),
			deleted,
			sqlString(id),
			item.fileID,
		)
		if item.file.Deleted {
			fmt.Fprintf(&script, "UPDATE spec.files SET current_filepath_id = NULL WHERE id = %d;\n", item.fileID)
		} else if item.file.PriorPath != "" {
			fmt.Fprintf(&script, "UPDATE spec.files SET current_filepath_id = %d WHERE id = %d;\n", item.pathID, item.fileID)
		}
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
