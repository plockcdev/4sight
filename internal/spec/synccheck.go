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

func runSyncCheck(metadataRoot string, st store, stdout io.Writer) error {
	docs, err := st.listLatestBodies()
	if err != nil {
		return fmt.Errorf("list spec bodies: %w", err)
	}
	disk, err := listOnDiskCodeSpecs(metadataRoot)
	if err != nil {
		return err
	}

	dbBodies := make(map[specKey]string)
	for _, doc := range docs {
		dbBodies[specKey{dir: doc.Dir, name: doc.Name}] = doc.Body
	}

	var importNew []string
	var importUpdated []string
	var exportCmds []string
	matching := 0
	seen := make(map[specKey]bool)
	for key, diskBody := range disk {
		seen[key] = true
		dbBody, ok := dbBodies[key]
		if !ok {
			imp, _ := syncCommands(key)
			importNew = append(importNew, imp)
			continue
		}
		if dbBody == diskBody {
			matching++
			continue
		}
		imp, exp := syncCommands(key)
		importUpdated = append(importUpdated, imp)
		if exp != "" {
			exportCmds = append(exportCmds, exp)
		}
	}
	for key := range dbBodies {
		if seen[key] {
			continue
		}
		if _, exp := syncCommands(key); exp != "" {
			exportCmds = append(exportCmds, exp)
		}
	}
	slices.Sort(importNew)
	slices.Sort(importUpdated)
	slices.Sort(exportCmds)

	if _, err := io.WriteString(stdout, "Import new spec files (overwrite database from disk):\n"); err != nil {
		return err
	}
	if err := writeCommands(stdout, importNew); err != nil {
		return err
	}
	if _, err := io.WriteString(stdout, "\nImport updated spec files (overwrite database from disk):\n"); err != nil {
		return err
	}
	if err := writeCommands(stdout, importUpdated); err != nil {
		return err
	}
	if _, err := io.WriteString(stdout, "\nExport (overwrite disk from database):\n"); err != nil {
		return err
	}
	if err := writeCommands(stdout, exportCmds); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "\nSummary: matching %d, not in db %d, mismatched %d\n", matching, len(importNew), len(importUpdated))
	return err
}

func syncCommands(key specKey) (importCmd, exportCmd string) {
	if key.dir == "schemas" {
		return "4sight migrate import " + key.name, ""
	}
	loc := strings.Join(locatorFromDirName(key.dir, key.name), " ")
	return "4sight spec import " + loc, "4sight spec export " + loc
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
