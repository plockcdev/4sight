package spec

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strings"
)

type doltStore struct {
	dataDir string
}

func newDoltStore(dataDir string) *doltStore {
	return &doltStore{dataDir: dataDir}
}

func (s *doltStore) insertBody(body string) error {
	query := "INSERT INTO spec.bodies (body) VALUES (" + sqlBodyLiteral(body) + ")"
	return s.run("sql", "-q", query)
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
