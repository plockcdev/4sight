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
	Marker  string
}

func ChangeList(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeList(newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runChangeList(st *doltStore, out io.Writer) error {
	return listChanges(st, out, true)
}

func listChanges(st *doltStore, out io.Writer, active bool) error {
	pointers, err := st.queryCSV("SELECT submitted_change_id, implemented_change_id, workspace_change_id FROM spec.current_changes WHERE singleton = 1")
	if err != nil {
		return fmt.Errorf("read current change pointers: %w", err)
	}
	if len(pointers) != 1 || len(pointers[0]) < 3 || pointers[0][0] == "" || pointers[0][1] == "" || pointers[0][2] == "" {
		return fmt.Errorf("current change pointers are not initialized")
	}
	submitted, implemented := pointers[0][0], pointers[0][1]
	query := "SELECT c.id, COALESCE(cr.summary, 'no summary yet') FROM spec.changes c LEFT JOIN spec.change_review cr ON cr.change_id = c.id AND cr.attempt = (SELECT MAX(r2.attempt) FROM spec.change_review r2 WHERE r2.change_id=c.id)"
	if active {
		query += " JOIN spec.changes_closure a ON a.child_id=c.id AND a.parent_id=" + sqlString(implemented) + " AND a.depth>0 JOIN spec.changes_closure w ON w.parent_id=c.id AND w.child_id=" + sqlString(pointers[0][2])
	}
	rows, err := st.queryCSV(query + " ORDER BY c.created_at ASC, c.id ASC")
	if err != nil {
		return fmt.Errorf("list changes: %w", err)
	}
	if len(rows) == 0 {
		message := "no changes found"
		if active {
			message = "no active changes"
		}
		_, err := fmt.Fprintln(out, message)
		return err
	}
	entries := make([]changeListEntry, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			return fmt.Errorf("incomplete change list row")
		}
		implementedMarker, submittedMarker := " ", " "
		if row[0] == implemented {
			implementedMarker = "I"
		}
		if row[0] == submitted {
			submittedMarker = "S"
		}
		entries = append(entries, changeListEntry{ID: row[0], Summary: row[1], Marker: implementedMarker + " " + submittedMarker})
	}
	for _, entry := range entries {
		if _, err := fmt.Fprintf(out, "%s  %s: %s\n", entry.Marker, entry.ID, entry.Summary); err != nil {
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
