package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type changeListEntry struct {
	ID      string
	Summary string
}

func ChangeList(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeList(newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runChangeList(st *doltStore, out io.Writer) error {
	rows, err := st.queryCSV("SELECT c.id, COALESCE(cr.summary, 'no summary yet') FROM spec.changes c LEFT JOIN spec.change_review cr ON cr.change_id = c.id ORDER BY c.created_at ASC, c.id ASC")
	if err != nil {
		return fmt.Errorf("list changes: %w", err)
	}
	if len(rows) == 0 {
		_, err := fmt.Fprintln(out, "no changes found")
		return err
	}
	entries := make([]changeListEntry, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			return fmt.Errorf("incomplete change list row")
		}
		entries = append(entries, changeListEntry{ID: row[0], Summary: row[1]})
	}
	for _, entry := range entries {
		if _, err := fmt.Fprintf(out, "%s: %s\n", entry.ID, entry.Summary); err != nil {
			return err
		}
	}
	return nil
}

func ChangeStatus(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeStatus(newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runChangeStatus(st *doltStore, out io.Writer) error {
	rows, err := st.queryCSV("SELECT workspace_change_id, submitted_change_id, implemented_change_id FROM spec.current_changes WHERE singleton = 1")
	if err != nil {
		return fmt.Errorf("read current change pointers: %w", err)
	}
	if len(rows) != 1 || len(rows[0]) < 3 || rows[0][0] == "" || rows[0][1] == "" || rows[0][2] == "" {
		return fmt.Errorf("current change pointers are not initialized")
	}
	if _, err := fmt.Fprintf(out, "Workspace: %s\nSubmitted: %s\nImplemented: %s\n", rows[0][0], rows[0][1], rows[0][2]); err != nil {
		return err
	}
	return nil
}
