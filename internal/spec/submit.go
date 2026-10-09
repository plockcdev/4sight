package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func ChangeSubmit(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeSubmit(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

type changeSubmitStore interface {
	listWorkspaceBodies() ([]specDoc, error)
	changePointers() (workspace, submitted string, err error)
	applyChangeSubmit(workspace string) error
}

func runChangeSubmit(root string, st changeSubmitStore, stdout io.Writer) error {
	bodies, err := st.listWorkspaceBodies()
	if err != nil {
		return fmt.Errorf("list workspace bodies: %w", err)
	}
	disk, err := listOnDiskCodeSpecs(root)
	if err != nil {
		return err
	}
	db := make(map[specKey]string, len(bodies))
	for _, body := range bodies {
		db[specKey{dir: body.Dir, name: body.Name}] = body.Body
	}
	keys := make(map[specKey]struct{}, len(db)+len(disk))
	for key := range db {
		keys[key] = struct{}{}
	}
	for key := range disk {
		keys[key] = struct{}{}
	}
	var mismatches []string
	for key := range keys {
		local, onDisk := disk[key]
		cached, inDB := db[key]
		if onDisk == inDB && normalizeSpecBody(local) == normalizeSpecBody(cached) {
			continue
		}
		path := specRelPath(key.dir, key.name)
		if !inDB || key.dir == "schemas" {
			mismatches = append(mismatches, path+": run `4sight spec change import` to update the workspace bodies from disk")
		} else if !onDisk {
			mismatches = append(mismatches, path+": restore or edit the file, then run `4sight spec change import`")
		} else {
			loc := strings.Join(locatorFromDirName(key.dir, key.name), " ")
			mismatches = append(mismatches, path+": run `4sight spec export "+loc+"` to restore the workspace body to disk, or run `4sight spec change import` to accept disk contents")
		}
	}
	if len(mismatches) > 0 {
		slices.Sort(mismatches)
		return fmt.Errorf("on-disk specs do not match workspace bodies:\n- %s", strings.Join(mismatches, "\n- "))
	}
	workspace, submitted, err := st.changePointers()
	if err != nil {
		return fmt.Errorf("read change pointers: %w", err)
	}
	if workspace == submitted {
		_, err := fmt.Fprintln(stdout, "no change to submit")
		return err
	}
	if err := st.applyChangeSubmit(workspace); err != nil {
		return fmt.Errorf("submit change %s: %w", workspace, err)
	}
	_, err = fmt.Fprintf(stdout, "submitted change %s\n", workspace)
	return err
}

func (s *doltStore) changePointers() (string, string, error) {
	rows, err := s.queryCSV("SELECT workspace_change_id, submitted_change_id FROM spec.current_changes WHERE singleton = 1")
	if err != nil {
		return "", "", err
	}
	if len(rows) != 1 || len(rows[0]) < 2 || rows[0][0] == "" || rows[0][1] == "" {
		return "", "", fmt.Errorf("workspace or submitted change pointer is not set")
	}
	return rows[0][0], rows[0][1], nil
}

func (s *doltStore) applyChangeSubmit(workspace string) error {
	var script strings.Builder
	script.WriteString("START TRANSACTION;\n")
	script.WriteString("DELETE FROM spec.bodies WHERE state = 'submitted';\n")
	script.WriteString("INSERT INTO spec.bodies (body, file_id, file_change_id, state, ieee_crc32, summary) " +
		"SELECT body, file_id, file_change_id, 'submitted', ieee_crc32, summary FROM spec.bodies WHERE state = 'workspace';\n")
	fmt.Fprintf(&script, "UPDATE spec.current_changes SET submitted_change_id = %s WHERE singleton = 1;\n", sqlString(workspace))
	script.WriteString("COMMIT;\n")
	if err := s.run("sql", "-q", script.String()); err != nil {
		return err
	}
	rows, err := s.queryCSV("SELECT submitted_change_id FROM spec.current_changes WHERE singleton = 1")
	if err != nil {
		return err
	}
	if len(rows) != 1 || len(rows[0]) == 0 || rows[0][0] != workspace {
		return fmt.Errorf("submitted pointer was not updated to %s", workspace)
	}
	return nil
}

func ChangeImplementationDone(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeImplementationDone(newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

type implementationDoneStore interface {
	changePointers() (workspace, submitted string, err error)
	implementedChangeID() (string, error)
	applyImplementationDone(submitted string) error
}

func runChangeImplementationDone(st implementationDoneStore, stdout io.Writer) error {
	_, submitted, err := st.changePointers()
	if err != nil {
		return fmt.Errorf("read change pointers: %w", err)
	}
	implemented, err := st.implementedChangeID()
	if err != nil {
		return fmt.Errorf("read implemented change: %w", err)
	}
	if submitted == implemented {
		_, err := fmt.Fprintln(stdout, "no submitted change awaiting implementation")
		return err
	}
	if err := st.applyImplementationDone(submitted); err != nil {
		return fmt.Errorf("mark change %s implemented: %w", submitted, err)
	}
	_, err = fmt.Fprintf(stdout, "implemented change %s\n", submitted)
	return err
}

func (s *doltStore) implementedChangeID() (string, error) {
	rows, err := s.queryCSV("SELECT implemented_change_id FROM spec.current_changes WHERE singleton = 1")
	if err != nil {
		return "", err
	}
	if len(rows) != 1 || len(rows[0]) == 0 || rows[0][0] == "" {
		return "", fmt.Errorf("implemented change pointer is not set")
	}
	return rows[0][0], nil
}

func (s *doltStore) applyImplementationDone(submitted string) error {
	var script strings.Builder
	script.WriteString("START TRANSACTION;\n")
	script.WriteString("DELETE FROM spec.bodies WHERE state = 'implemented';\n")
	script.WriteString("INSERT INTO spec.bodies (body, file_id, file_change_id, state, ieee_crc32, summary) " +
		"SELECT body, file_id, file_change_id, 'implemented', ieee_crc32, summary FROM spec.bodies WHERE state = 'submitted';\n")
	fmt.Fprintf(&script, "UPDATE spec.current_changes SET implemented_change_id = %s WHERE singleton = 1;\n", sqlString(submitted))
	script.WriteString("COMMIT;\n")
	if err := s.run("sql", "-q", script.String()); err != nil {
		return err
	}
	rows, err := s.queryCSV("SELECT implemented_change_id FROM spec.current_changes WHERE singleton = 1")
	if err != nil {
		return err
	}
	if len(rows) != 1 || len(rows[0]) == 0 || rows[0][0] != submitted {
		return fmt.Errorf("implemented pointer was not updated to %s", submitted)
	}
	return nil
}
