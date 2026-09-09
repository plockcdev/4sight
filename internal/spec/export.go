package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Export(cwd string, loc []string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runExport(root, loc, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runExport(metadataRoot string, loc []string, st store, stdout io.Writer) error {
	dirRel, name, err := resolveSpecLocation(loc)
	if err != nil {
		return err
	}
	body, ok, err := st.latestBody(dirRel, name)
	if err != nil {
		return fmt.Errorf("lookup spec body: %w", err)
	}
	if !ok {
		return fmt.Errorf("spec body %s is not in the database", displaySpecPath(dirRel, name))
	}
	path, err := writeSpecFile(metadataRoot, dirRel, name, body)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote %s\n", relFromMetadata(metadataRoot, path))
	return err
}
