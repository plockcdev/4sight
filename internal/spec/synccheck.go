package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func SyncCheck(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runSyncCheck(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

type specKey struct {
	dir  string
	name string
}

type workspaceBodyStore interface {
	listWorkspaceBodies() ([]specDoc, error)
}

func runSyncCheck(metadataRoot string, st workspaceBodyStore, stdout io.Writer) error {
	docs, err := st.listWorkspaceBodies()
	if err != nil {
		return fmt.Errorf("list workspace bodies: %w", err)
	}
	disk, err := listOnDiskCodeSpecs(metadataRoot)
	if err != nil {
		return err
	}

	dbBodies := make(map[specKey]string, len(docs))
	for _, doc := range docs {
		dbBodies[specKey{dir: doc.Dir, name: doc.Name}] = doc.Body
	}
	keys := make(map[specKey]bool, len(disk)+len(dbBodies))
	for key := range disk {
		keys[key] = true
	}
	for key := range dbBodies {
		keys[key] = true
	}

	needsImport := false
	var exportCmds []string
	for key := range keys {
		rawDisk, onDisk, err := diskSpec(metadataRoot, disk, key)
		if err != nil {
			return err
		}
		dbText := ""
		_, inDB := dbBodies[key]
		if inDB {
			dbText = normalizeSpecBody(dbBodies[key])
		}
		diskText := ""
		if onDisk {
			diskText = normalizeSpecBody(rawDisk)
		}
		if dbText == diskText {
			continue
		}
		needsImport = true
		if !inDB || key.dir == "schemas" {
			continue
		}
		loc := strings.Join(locatorFromDirName(key.dir, key.name), " ")
		exportCmds = append(exportCmds, "4sight spec export "+loc)
	}
	slices.Sort(exportCmds)

	var importCmds []string
	if needsImport {
		importCmds = []string{"4sight spec change import"}
	}
	if _, err := io.WriteString(stdout, "Import (overwrite database from disk):\n"); err != nil {
		return err
	}
	if err := writeCommands(stdout, importCmds); err != nil {
		return err
	}
	if _, err := io.WriteString(stdout, "\nExport (overwrite disk from database):\n"); err != nil {
		return err
	}
	return writeCommands(stdout, exportCmds)
}

func writeCommands(stdout io.Writer, cmds []string) error {
	if len(cmds) == 0 {
		_, err := io.WriteString(stdout, "(none)\n")
		return err
	}
	for _, cmd := range cmds {
		if _, err := fmt.Fprintln(stdout, cmd); err != nil {
			return err
		}
	}
	return nil
}

func listOnDiskCodeSpecs(metadataRoot string) (map[specKey]string, error) {
	out := make(map[specKey]string)
	err := filepath.WalkDir(metadataRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != metadataRoot && (d.Name() == "db" || d.Name() == "migrations" || d.Name() == "schemas") {
				return filepath.SkipDir
			}
			return nil
		}
		if !isCodeSpecName(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(metadataRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		dir := ""
		name := d.Name()
		if rel != name {
			dir = strings.TrimSuffix(rel, "/"+name)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read spec file %s: %w", relFromMetadata(metadataRoot, path), err)
		}
		out[specKey{dir: dir, name: name}] = string(body)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list on-disk spec files under %s: %w", metadataRoot, err)
	}
	if err := addOnDiskSchemaSpecs(metadataRoot, out); err != nil {
		return nil, err
	}
	return out, nil
}

func addOnDiskSchemaSpecs(metadataRoot string, out map[specKey]string) error {
	schemasDir := filepath.Join(metadataRoot, "schemas")
	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read schemas dir %s: %w", schemasDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(schemasDir, entry.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read spec file %s: %w", relFromMetadata(metadataRoot, path), err)
		}
		out[specKey{dir: "schemas", name: entry.Name()}] = string(body)
	}
	return nil
}
