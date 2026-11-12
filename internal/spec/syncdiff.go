package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func SyncDiff(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runSyncDiff(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runSyncDiff(metadataRoot string, st store, stdout io.Writer) error {
	docs, err := st.listLatestBodies()
	if err != nil {
		return fmt.Errorf("list spec bodies: %w", err)
	}
	disk, err := listOnDiskCodeSpecs(metadataRoot)
	if err != nil {
		return err
	}

	dbBodies := make(map[specKey]specDoc)
	for _, doc := range docs {
		dbBodies[specKey{dir: doc.Dir, name: doc.Name}] = doc
	}

	keys := make(map[specKey]bool)
	for key := range disk {
		keys[key] = true
	}
	for key := range dbBodies {
		keys[key] = true
	}

	var ordered []specKey
	for key := range keys {
		ordered = append(ordered, key)
	}
	slices.SortFunc(ordered, func(a, b specKey) int {
		if c := strings.Compare(a.dir, b.dir); c != 0 {
			return c
		}
		return strings.Compare(a.name, b.name)
	})

	var diffs []string
	for _, key := range ordered {
		dbDoc, inDB := dbBodies[key]
		diskBody, onDisk := disk[key]
		if inDB && dbDoc.Deleted && !onDisk {
			continue
		}
		dbBody := ""
		if inDB && !dbDoc.Deleted {
			dbBody = dbDoc.Body
		}
		if inDB == onDisk && (!inDB || normalizeTrackedBody(key.dir, key.name, dbBody) == normalizeTrackedBody(key.dir, key.name, diskBody)) {
			continue
		}
		path := specDisplayPath(key)
		oldPath, newPath := path, path
		if !inDB {
			oldPath = "/dev/null"
		}
		if !onDisk || (inDB && dbDoc.Deleted) {
			newPath = "/dev/null"
		}
		oldText, newText := dbBody, diskBody
		if inDB && dbDoc.Deleted {
			oldText, newText = diskBody, ""
		} else if !inDB {
			oldText = ""
		} else if !onDisk {
			newText = ""
		}
		diff := unifiedDiffPaths(oldPath, newPath, oldText, newText)
		if diff != "" {
			diffs = append(diffs, diff)
		}
	}

	for i, diff := range diffs {
		if i > 0 {
			if _, err := io.WriteString(stdout, "\n"); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(stdout, diff); err != nil {
			return err
		}
	}
	return nil
}

func specDisplayPath(key specKey) string {
	if key.dir == "schemas" {
		return ".4sight/schemas/" + key.name
	}
	return displaySpecPath(key.dir, key.name)
}
