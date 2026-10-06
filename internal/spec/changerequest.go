package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ChangeRequest(cwd, prompt string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeRequest(newDoltStore(filepath.Join(root, "db")), prompt, os.Stdout)
}

type changeRequestStore interface {
	workspaceChangeID() (string, error)
	changeIDs() ([]string, error)
	applyChangeRequest(id, parent, prompt string) error
}

func runChangeRequest(st changeRequestStore, prompt string, stdout io.Writer) error {
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt must not be empty")
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
	if err := st.applyChangeRequest(id, parent, prompt); err != nil {
		return fmt.Errorf("record change request: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "id: %s\n", id)
	return err
}

func (s *doltStore) applyChangeRequest(id, parent, prompt string) error {
	if _, err := changeIDRank(id); err != nil {
		return err
	}
	if _, err := changeIDRank(parent); err != nil {
		return err
	}
	rows, err := s.queryCSV("SELECT COUNT(*) FROM spec.changes_closure WHERE child_id = " + sqlString(parent))
	if err != nil {
		return err
	}
	if len(rows) == 0 || len(rows[0]) == 0 || rows[0][0] == "0" {
		return fmt.Errorf("workspace change %s has no closure rows", parent)
	}
	var script strings.Builder
	script.WriteString("START TRANSACTION;\n")
	fmt.Fprintf(&script, "INSERT INTO spec.changes (id, prompt) VALUES (%s, %s);\n", sqlString(id), sqlBodyLiteral(prompt))
	fmt.Fprintf(&script,
		"INSERT INTO spec.changes_closure (parent_id, child_id, depth) "+
			"SELECT parent_id, %s, depth + 1 FROM spec.changes_closure WHERE child_id = %s;\n",
		sqlString(id), sqlString(parent),
	)
	fmt.Fprintf(&script, "INSERT INTO spec.changes_closure (parent_id, child_id, depth) VALUES (%s, %s, 0);\n", sqlString(id), sqlString(id))
	fmt.Fprintf(&script, "UPDATE spec.current_changes SET workspace_change_id = %s WHERE singleton = 1;\n", sqlString(id))
	script.WriteString("COMMIT;\n")
	if err := s.run("sql", "-q", script.String()); err != nil {
		return err
	}
	rows, err = s.queryCSV("SELECT workspace_change_id FROM spec.current_changes WHERE singleton = 1")
	if err != nil {
		return err
	}
	if len(rows) != 1 || len(rows[0]) == 0 || rows[0][0] != id {
		return fmt.Errorf("workspace pointer was not updated to %s", id)
	}
	return nil
}
