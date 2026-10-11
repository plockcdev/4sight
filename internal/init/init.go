package init

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed assets migrations
var embedded embed.FS

// Run initializes a repository from the current working directory.
func Run(cwd string) error { return run(cwd, embedded, os.Stdout) }

func run(cwd string, source fs.FS, stdout interface{ Write([]byte) (int, error) }) error {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}
	git := exec.Command("git", "rev-parse", "--show-toplevel")
	git.Dir = cwd
	out, err := git.Output()
	if err != nil {
		return fmt.Errorf("%s is not inside a git repository; initialize one with `git init %s` and run 4sight from its root", abs, abs)
	}
	root, err := filepath.Abs(strings.TrimSpace(string(out)))
	if err != nil {
		return fmt.Errorf("resolve git root: %w", err)
	}
	if root != abs {
		return fmt.Errorf("run 4sight init from the git repository root %s (current directory is %s)", root, abs)
	}

	metadata := filepath.Join(root, ".4sight")
	if err := os.MkdirAll(metadata, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", metadata, err)
	}
	database := filepath.Join(metadata, "db")
	if err := os.MkdirAll(database, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", database, err)
	}
	if _, err := os.Stat(filepath.Join(database, ".dolt")); os.IsNotExist(err) {
		cmd := exec.Command("dolt", "init")
		cmd.Dir = database
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("initialize Dolt in %s: %s; install Dolt and rerun 4sight init", database, strings.TrimSpace(string(output)))
		}
	} else if err != nil {
		return fmt.Errorf("check Dolt repository %s: %w", database, err)
	}
	if err := bootstrapDatabases(database, embedded, time.Now); err != nil {
		return err
	}

	rootAsset, err := fs.ReadFile(source, "assets/root_AGENTS.md")
	if err != nil {
		return fmt.Errorf("read embedded root instructions: %w", err)
	}
	oldVersions, err := fs.Glob(source, "assets/history/root_AGENTS_*.md")
	if err != nil {
		return fmt.Errorf("list historical root instructions: %w", err)
	}
	var historical [][]byte
	for _, path := range oldVersions {
		body, err := fs.ReadFile(source, path)
		if err != nil {
			return fmt.Errorf("read historical root instructions %s: %w", path, err)
		}
		historical = append(historical, body)
	}
	if err := rewriteRootInstructions(filepath.Join(root, "AGENTS.md"), string(rootAsset), historical); err != nil {
		return err
	}
	for _, item := range []struct {
		name, target string
		preserve     bool
	}{
		{"4sight_AGENTS.md", filepath.Join(metadata, "AGENTS.md"), false},
		{"coding.md", filepath.Join(metadata, "coding.md"), true},
	} {
		if item.preserve {
			if _, err := os.Stat(item.target); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("check %s: %w", item.target, err)
			}
		}
		body, err := fs.ReadFile(source, "assets/"+item.name)
		if err != nil {
			return fmt.Errorf("read embedded asset %s: %w", item.name, err)
		}
		if err := os.WriteFile(item.target, body, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", item.target, err)
		}
	}
	_, err = fmt.Fprintf(stdout, "initialized 4sight in %s\n", root)
	return err
}

type embeddedMigration struct {
	name, id, parent string
}

func bootstrapDatabases(database string, source fs.FS, now func() time.Time) error {
	store := doltBootstrap{dataDir: database}
	for _, item := range []struct {
		name  string
		table string
	}{
		{name: "dolt_4sight_migration", table: "4sight_migration"},
		{name: "dolt_spec", table: "spec"},
	} {
		migrations, err := readEmbeddedMigrations(source, "migrations/"+item.name)
		if err != nil {
			return fmt.Errorf("read embedded %s migrations: %w", item.name, err)
		}
		for _, migration := range migrations {
			exists, err := store.trackingTableExists(item.table)
			if err != nil {
				return fmt.Errorf("check tracking table 4sight_migration.%s: %w", item.table, err)
			}
			if !exists && migration.id != "00000000" {
				return fmt.Errorf("tracking table 4sight_migration.%s is missing before migration %s", item.table, migration.name)
			}
			if exists {
				applied, err := store.migrationApplied(item.table, migration.id)
				if err != nil {
					return fmt.Errorf("check applied migration %s for %s: %w", migration.id, item.name, err)
				}
				if applied {
					continue
				}
			}
			sql, err := fs.ReadFile(source, "migrations/"+item.name+"/"+migration.name)
			if err != nil {
				return fmt.Errorf("read embedded migration %s: %w", migration.name, err)
			}
			if err := store.run("sql", "-q", string(sql)); err != nil {
				return fmt.Errorf("apply embedded migration %s: %w", migration.name, err)
			}
			if err := store.recordApplied(item.table, migration.id, now().UTC().Truncate(time.Second)); err != nil {
				return fmt.Errorf("record embedded migration %s in 4sight_migration.%s: %w", migration.id, item.table, err)
			}
		}
	}
	return nil
}

func readEmbeddedMigrations(source fs.FS, dir string) ([]embeddedMigration, error) {
	entries, err := fs.ReadDir(source, dir)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]embeddedMigration)
	children := make(map[string][]embeddedMigration)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		stem := strings.TrimSuffix(entry.Name(), ".sql")
		id, parent, ok := strings.Cut(stem, "-")
		if !ok || len(id) != 8 || len(parent) != 8 || !digits(id) || !digits(parent) {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		migration := embeddedMigration{name: entry.Name(), id: id, parent: parent}
		if _, exists := byID[id]; exists {
			return nil, fmt.Errorf("duplicate migration id %s", id)
		}
		byID[id] = migration
		if id != "00000000" {
			children[parent] = append(children[parent], migration)
		}
	}
	first, ok := byID["00000000"]
	if !ok || first.parent != "00000000" {
		return nil, fmt.Errorf("missing initial migration 00000000-00000000.sql")
	}
	ordered := []embeddedMigration{first}
	seen := map[string]bool{first.id: true}
	for parent := first.id; ; {
		next := children[parent]
		if len(next) == 0 {
			break
		}
		if len(next) != 1 {
			return nil, fmt.Errorf("multiple migrations have parent id %s", parent)
		}
		if seen[next[0].id] {
			return nil, fmt.Errorf("cycle in migration chain at %s", next[0].id)
		}
		ordered = append(ordered, next[0])
		seen[next[0].id] = true
		parent = next[0].id
	}
	if len(seen) != len(byID) {
		return nil, fmt.Errorf("embedded migrations in %s do not form one chain", dir)
	}
	return ordered, nil
}

func digits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type doltBootstrap struct{ dataDir string }

func (s doltBootstrap) trackingTableExists(table string) (bool, error) {
	rows, err := s.queryCSV(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '4sight_migration' AND table_name = '%s'", table))
	if err != nil {
		return false, err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return false, fmt.Errorf("empty table existence result")
	}
	return rows[0][0] != "0", nil
}

func (s doltBootstrap) migrationApplied(table, id string) (bool, error) {
	rows, err := s.queryCSV(fmt.Sprintf("SELECT id FROM 4sight_migration.%s WHERE id = '%s'", table, id))
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (s doltBootstrap) recordApplied(table, id string, at time.Time) error {
	return s.run("sql", "-q", fmt.Sprintf("INSERT INTO 4sight_migration.%s (id, time) VALUES ('%s', '%s')", table, id, at.Format("2006-01-02 15:04:05")))
}

func (s doltBootstrap) queryCSV(query string) ([][]string, error) {
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
	if len(records) < 2 {
		return nil, nil
	}
	return records[1:], nil
}

func (s doltBootstrap) run(args ...string) error {
	_, err := s.output(args...)
	return err
}

func (s doltBootstrap) output(args ...string) (string, error) {
	cmd := exec.Command("dolt", args...)
	cmd.Dir = s.dataDir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			return "", fmt.Errorf("dolt %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("dolt %s: %s", strings.Join(args, " "), message)
	}
	return stdout.String(), nil
}

func rewriteRootInstructions(path, current string, historical [][]byte) error {
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(old), "\r\n", "\n"), "\n")
	versions := append(historical, []byte(current))
	var blocks [][]string
	for _, version := range versions {
		blocks = append(blocks, normalizedLines(string(version)))
	}
	var kept []string
	var localAdditions []string
	for i := 0; i < len(lines); {
		matched := false
		matchLen := 0
		for _, block := range blocks {
			if len(block) == 0 || i+len(block) > len(lines) {
				continue
			}
			ok := true
			for j, want := range block {
				if !sameManagedLine(lines[i+j], want) {
					ok = false
					break
				}
			}
			if ok {
				matched, matchLen = true, len(block)
				break
			}
		}
		if matched {
			for _, block := range blocks {
				if len(block) != matchLen {
					continue
				}
				for j, want := range block {
					actual := strings.TrimSpace(lines[i+j])
					if actual == want {
						continue
					}
					extra := strings.TrimSpace(strings.TrimPrefix(actual, strings.TrimRight(want, ".")))
					extra = strings.TrimLeft(extra, ",;: ")
					if extra != "" {
						localAdditions = append(localAdditions, extra)
					}
				}
				break
			}
			i += matchLen
			continue
		}
		kept = append(kept, lines[i])
		i++
	}
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}
	if len(localAdditions) > 0 {
		kept = append(kept, localAdditions...)
	}
	if len(kept) > 0 {
		kept = append(kept, "")
	}
	kept = append(kept, normalizedLines(current)...)
	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func sameManagedLine(actual, expected string) bool {
	actual = strings.TrimSpace(actual)
	if actual == expected {
		return true
	}
	return strings.HasPrefix(actual, strings.TrimRight(expected, ".")+",")
}

func normalizedLines(s string) []string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return lines
}
