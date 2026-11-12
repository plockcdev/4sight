package spec

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type doltStore struct {
	dataDir string
}

func newDoltStore(dataDir string) *doltStore {
	return &doltStore{dataDir: dataDir}
}

func (s *doltStore) latestBody(dirPath, fileName string) (string, bool, error) {
	query := "SELECT b.body FROM spec.bodies b" +
		" JOIN spec.files f ON f.id = b.file_id" +
		" JOIN spec.file_changes fc ON fc.id = b.file_change_id" +
		" JOIN spec.filepaths p ON p.id = fc.filepath_id" +
		" JOIN spec.dirs d ON d.id = p.dir_id" +
		" WHERE d.path = " + sqlString(dirPath) +
		" AND p.name = " + sqlString(fileName) +
		" AND b.state = 'workspace' AND b.deleted = FALSE" +
		" LIMIT 1"
	rows, err := s.queryCSV(query)
	if err != nil {
		return "", false, err
	}
	if len(rows) == 0 {
		return "", false, nil
	}
	return rows[0][0], true, nil
}

func (s *doltStore) listLatestBodies() ([]specDoc, error) {
	query := "SELECT d.path, p.name, b.body, b.deleted FROM spec.bodies b" +
		" JOIN spec.files f ON f.id = b.file_id" +
		" JOIN spec.file_changes fc ON fc.id = b.file_change_id" +
		" JOIN spec.filepaths p ON p.id = fc.filepath_id" +
		" JOIN spec.dirs d ON d.id = p.dir_id" +
		" WHERE b.state = 'workspace'" +
		" ORDER BY d.path, p.name, b.deleted DESC"
	rows, err := s.queryCSV(query)
	if err != nil {
		return nil, err
	}
	latest := make(map[string]specDoc)
	var order []string
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		doc := specDoc{Dir: row[0], Name: row[1], Body: row[2], Deleted: row[3] == "1" || strings.EqualFold(row[3], "true")}
		key := doc.Dir + "\x00" + doc.Name
		if _, ok := latest[key]; !ok {
			order = append(order, key)
		}
		latest[key] = doc
	}
	docs := make([]specDoc, 0, len(order))
	for _, key := range order {
		docs = append(docs, latest[key])
	}
	return docs, nil
}

func (s *doltStore) ensureDir(path string) (uint64, error) {
	query := "SELECT id FROM spec.dirs WHERE path = " + sqlString(path)
	id, ok, err := s.lookupID(query)
	if err != nil || ok {
		return id, err
	}
	if err := s.run("sql", "-q", "INSERT INTO spec.dirs (path) VALUES ("+sqlString(path)+")"); err != nil {
		id, ok, err2 := s.lookupID(query)
		if err2 == nil && ok {
			return id, nil
		}
		return 0, err
	}
	id, ok, err = s.lookupID(query)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("dir %s was not found after insert", path)
	}
	return id, nil
}

func (s *doltStore) ensureFile(dirID uint64, name string) (uint64, error) {
	filepathID, err := s.ensureFilepath(dirID, name)
	if err != nil {
		return 0, err
	}
	fileQuery := fmt.Sprintf("SELECT id FROM spec.files WHERE current_filepath_id = %d", filepathID)
	fileID, ok, err := s.lookupID(fileQuery)
	if err != nil || ok {
		return fileID, err
	}
	if err := s.run("sql", "-q", fmt.Sprintf("INSERT INTO spec.files (current_filepath_id) VALUES (%d)", filepathID)); err != nil {
		fileID, ok, err = s.lookupID(fileQuery)
		if err != nil || !ok {
			return 0, err
		}
		return fileID, nil
	}
	fileID, ok, err = s.lookupID(fileQuery)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("file %s was not found after insert", name)
	}
	return fileID, nil
}

func (s *doltStore) ensureFilepath(dirID uint64, name string) (uint64, error) {
	filepathQuery := fmt.Sprintf("SELECT id FROM spec.filepaths WHERE dir_id = %d AND name = %s", dirID, sqlString(name))
	filepathID, ok, err := s.lookupID(filepathQuery)
	if err != nil {
		return 0, err
	}
	if !ok {
		if err := s.run("sql", "-q", fmt.Sprintf("INSERT INTO spec.filepaths (dir_id, name) VALUES (%d, %s)", dirID, sqlString(name))); err != nil {
			filepathID, ok, err = s.lookupID(filepathQuery)
			if err != nil || !ok {
				return 0, err
			}
		} else {
			filepathID, ok, err = s.lookupID(filepathQuery)
			if err != nil {
				return 0, err
			}
		}
	}
	if !ok {
		return 0, fmt.Errorf("filepath %s was not found after insert", name)
	}
	return filepathID, nil
}

func (s *doltStore) lookupID(query string) (uint64, bool, error) {
	rows, err := s.queryCSV(query)
	if err != nil {
		return 0, false, err
	}
	if len(rows) == 0 {
		return 0, false, nil
	}
	id, err := strconv.ParseUint(rows[0][0], 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse id %q: %w", rows[0][0], err)
	}
	return id, true, nil
}

func (s *doltStore) queryCSV(query string) ([][]string, error) {
	out, err := s.output("sql", "-r", "csv", "-q", query)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil, nil
	}
	reader := csv.NewReader(strings.NewReader(trimmed))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse dolt csv: %w", err)
	}
	if len(records) <= 1 {
		return nil, nil
	}
	return records[1:], nil
}

func (s *doltStore) run(args ...string) error {
	_, err := s.output(args...)
	return err
}

func (s *doltStore) output(args ...string) (string, error) {
	cmd := exec.Command("dolt", args...)
	cmd.Dir = s.dataDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			return "", fmt.Errorf("dolt %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("dolt %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}
