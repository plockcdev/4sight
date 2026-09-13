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

	dbBodies := make(map[specKey]string)
	for _, doc := range docs {
		dbBodies[specKey{dir: doc.Dir, name: doc.Name}] = doc.Body
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
		dbBody := dbBodies[key]
		diskBody := disk[key]
		if dbBody == diskBody {
			continue
		}
		path := specDisplayPath(key)
		diff := unifiedDiffPaths(path, path, dbBody, diskBody)
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
