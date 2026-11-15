package spec

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// History lists all changes, or the ancestry history of a file path.
func History(cwd, file string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	st := newDoltStore(filepath.Join(root, "db"))
	if file == "" {
		return listChanges(st, os.Stdout, false)
	}
	file, err = historyPath(root, cwd, file)
	if err != nil {
		return err
	}
	return fileHistory(st, os.Stdout, file)
}

func historyPath(metadataRoot, cwd, file string) (string, error) {
	if !filepath.IsAbs(file) {
		file = filepath.Join(cwd, file)
	}
	absolute, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("resolve history path: %w", err)
	}
	rel, err := filepath.Rel(metadataRoot, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("history path %s is outside .4sight", file)
	}
	return filepath.ToSlash(rel), nil
}

func fileHistory(st *doltStore, out io.Writer, file string) error {
	dir, name := path.Split(file)
	dir = strings.TrimSuffix(dir, "/")
	ids, err := st.queryCSV("SELECT DISTINCT fc.file_id FROM spec.file_changes fc JOIN spec.filepaths p ON p.id=fc.filepath_id JOIN spec.dirs d ON d.id=p.dir_id JOIN spec.current_changes cc ON cc.singleton=1 JOIN spec.changes_closure a ON a.child_id=cc.workspace_change_id AND a.parent_id=fc.change_id WHERE d.path=" + sqlString(dir) + " AND p.name=" + sqlString(name))
	if err != nil {
		return fmt.Errorf("resolve file history: %w", err)
	}
	if len(ids) == 0 {
		return fmt.Errorf("%s has no tracked history", file)
	}
	if len(ids) > 1 {
		return fmt.Errorf("%s is ambiguous: multiple file identities match", file)
	}
	rows, err := st.queryCSV("SELECT fc.change_id, cr.id, cr.description FROM spec.file_changes fc JOIN spec.current_changes cc ON cc.singleton=1 JOIN spec.changes_closure a ON a.child_id=cc.workspace_change_id AND a.parent_id=fc.change_id LEFT JOIN spec.change_review cr ON cr.change_id=fc.change_id AND cr.attempt=(SELECT MAX(r.attempt) FROM spec.change_review r WHERE r.change_id=fc.change_id) WHERE fc.file_id=" + ids[0][0] + " ORDER BY a.depth DESC")
	if err != nil {
		return fmt.Errorf("read file history: %w", err)
	}
	for _, row := range rows {
		if len(row) < 3 {
			return fmt.Errorf("incomplete file history row")
		}
		desc := row[2]
		if row[1] == "" || row[1] == "NULL" {
			desc = "no review yet"
		} else if desc == "" {
			desc = "no description"
		}
		if _, err := fmt.Fprintln(out, row[0]); err != nil {
			return err
		}
		for _, line := range wrapHistory(desc, 78) {
			if _, err := fmt.Fprintln(out, "  "+line); err != nil {
				return err
			}
		}
	}
	return nil
}

func wrapHistory(text string, width int) []string {
	var result []string
	for paragraph := range strings.SplitSeq(text, "\n") {
		words := strings.Fields(paragraph)
		line := ""
		for _, word := range words {
			r := []rune(word)
			for len(r) > 0 {
				available := width - len([]rune(line))
				if line != "" {
					available--
				}
				if available == 0 {
					result = append(result, line)
					line = ""
					continue
				}
				if len(r) > available && line != "" {
					result = append(result, line)
					line = ""
					continue
				}
				n := min(len(r), available)
				if line != "" {
					line += " "
				}
				line += string(r[:n])
				r = r[n:]
				if len(r) > 0 {
					result = append(result, line)
					line = ""
				}
			}
		}
		result = append(result, line)
	}
	return result
}
