package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ChangeBody(cwd, changeID string, locator []string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	dir, name, err := resolveSpecLocation(locator)
	if err != nil {
		return err
	}
	st := newDoltStore(filepath.Join(root, "db"))
	return st.writeChangeBody(changeID, dir, name, os.Stdout)
}

func (s *doltStore) writeChangeBody(changeID, dir, name string, out io.Writer) error {
	exists, err := s.queryCSV("SELECT id FROM spec.changes WHERE id=" + sqlString(changeID))
	if err != nil {
		return err
	}
	if len(exists) == 0 {
		return fmt.Errorf("change %s does not exist", changeID)
	}
	pathQuery := "SELECT fc.file_id FROM spec.file_changes fc JOIN spec.changes_closure ancestry ON ancestry.child_id=" + sqlString(changeID) + " AND ancestry.parent_id=fc.change_id" +
		" JOIN spec.filepaths p ON p.id=fc.filepath_id JOIN spec.dirs d ON d.id=p.dir_id" +
		" WHERE d.path=" + sqlString(dir) + " AND p.name=" + sqlString(name) +
		" AND NOT EXISTS (SELECT 1 FROM spec.file_changes newer JOIN spec.changes_closure later ON later.child_id=" + sqlString(changeID) + " AND later.parent_id=newer.change_id" +
		" WHERE newer.file_id=fc.file_id AND later.depth<ancestry.depth) ORDER BY ancestry.depth ASC LIMIT 1"
	rows, err := s.queryCSV(pathQuery)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("spec file %s is not present in change %s", displaySpecPath(dir, name), changeID)
	}
	fileID := rows[0][0]
	changes, err := s.queryCSV("SELECT fc.diff,fc.deleted FROM spec.file_changes fc JOIN spec.changes_closure ancestry ON ancestry.child_id=" + sqlString(changeID) + " AND ancestry.parent_id=fc.change_id WHERE fc.file_id=" + fileID + " ORDER BY ancestry.depth DESC")
	if err != nil {
		return err
	}
	body := ""
	for _, change := range changes {
		if len(change) < 2 {
			continue
		}
		body, err = applyUnifiedDiff(body, change[0])
		if err != nil {
			return fmt.Errorf("reconstruct %s at change %s: %w", displaySpecPath(dir, name), changeID, err)
		}
		if change[1] == "1" || strings.EqualFold(change[1], "true") {
			body = ""
		}
	}
	_, err = io.WriteString(out, body)
	return err
}

func applyUnifiedDiff(base, diff string) (string, error) {
	lines := diffLines(base)
	patch := strings.Split(strings.TrimSuffix(diff, "\n"), "\n")
	if len(patch) < 3 {
		return base, nil
	}
	targetLineCount := len(lines)
	for _, line := range patch[2:] {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			targetLineCount++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			targetLineCount--
		}
	}
	var output []string
	oldIndex := 0
	hasFinalNewline := strings.HasSuffix(base, "\n")
	lastTag := byte(0)
	lastWasFinal := false
	for i := 2; i < len(patch); {
		if !strings.HasPrefix(patch[i], "@@ ") {
			i++
			continue
		}
		var oldStart, oldCount, newStart, newCount int
		if _, err := fmt.Sscanf(patch[i], "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount); err != nil {
			return "", fmt.Errorf("invalid hunk header %q", patch[i])
		}
		_ = newStart
		_ = newCount
		i++
		newIndex := max(newStart-1, 0)
		if newCount == 0 {
			newIndex = newStart
		}
		hunkStart := max(oldStart-1, 0)
		if oldCount == 0 {
			if oldStart == 0 {
				hunkStart = oldIndex + max(newStart-1-len(output), 0)
			} else {
				hunkStart = oldStart
			}
		}
		if hunkStart < oldIndex || hunkStart > len(lines) {
			return "", fmt.Errorf("hunk starts outside the current body")
		}
		output = append(output, lines[oldIndex:hunkStart]...)
		oldIndex = hunkStart
		for i < len(patch) && !strings.HasPrefix(patch[i], "@@ ") {
			line := patch[i]
			i++
			if line == "\\ No newline at end of file" {
				if (lastTag == '+' || lastTag == ' ') && lastWasFinal {
					hasFinalNewline = false
				}
				continue
			}
			if line == "" {
				return "", fmt.Errorf("invalid empty patch line")
			}
			switch line[0] {
			case ' ':
				if oldIndex >= len(lines) || lines[oldIndex] != line[1:] {
					return "", fmt.Errorf("context does not match current body")
				}
				output = append(output, line[1:])
				oldIndex++
				lastTag = ' '
				lastWasFinal = newIndex == targetLineCount-1
				if lastWasFinal {
					hasFinalNewline = true
				}
				newIndex++
			case '-':
				if oldIndex >= len(lines) || lines[oldIndex] != line[1:] {
					return "", fmt.Errorf("removed line does not match current body")
				}
				oldIndex++
				lastTag = '-'
				lastWasFinal = false
			case '+':
				output = append(output, line[1:])
				lastTag = '+'
				lastWasFinal = newIndex == targetLineCount-1
				if lastWasFinal {
					hasFinalNewline = true
				}
				newIndex++
			default:
				return "", fmt.Errorf("invalid patch line %q", line)
			}
		}
	}
	output = append(output, lines[oldIndex:]...)
	if len(output) == 0 {
		return "", nil
	}
	result := strings.Join(output, "\n")
	if hasFinalNewline {
		result += "\n"
	}
	return result, nil
}
