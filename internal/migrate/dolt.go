package migrate

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

type doltStore struct {
	dataDir string
}

func newDoltStore(dataDir string) *doltStore {
	return &doltStore{dataDir: dataDir}
}

func (s *doltStore) databaseExists(name string) (bool, error) {
	if err := ident(name); err != nil {
		return false, err
	}
	rows, err := s.queryCSV("SHOW DATABASES")
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if len(row) > 0 && row[0] == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *doltStore) applyFile(path string) error {
	return s.run("sql", "-f", path)
}

func (s *doltStore) trackingTableExists(table string) (bool, error) {
	if err := ident(table); err != nil {
		return false, err
	}
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s'",
		migrationsDB, table,
	)
	rows, err := s.queryCSV(query)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return false, fmt.Errorf("empty table existence result")
	}
	return rows[0][0] != "0", nil
}

func (s *doltStore) insertApplied(table, id string, at time.Time) error {
	if err := ident(table); err != nil {
		return err
	}
	if !migrationIDRe.MatchString(id) {
		return fmt.Errorf("invalid migration id %q", id)
	}
	query := fmt.Sprintf(
		"INSERT INTO %s.%s (id, time) VALUES ('%s', '%s')",
		migrationsDB, table, id, at.Format("2006-01-02 15:04:05"),
	)
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

func ident(name string) error {
	if name == "" {
		return fmt.Errorf("empty SQL identifier")
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("invalid SQL identifier %q", name)
		}
	}
	return nil
}
