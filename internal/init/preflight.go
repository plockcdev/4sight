package init

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// These IDs identify the final migrations embedded in this binary.
const expectedMigrationID = "00016920"
const expectedSpecID = "00113758"

// Preflight validates and updates the internal databases of an initialized project.
func Preflight(cwd string) error {
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	for {
		metadata := filepath.Join(dir, ".4sight")
		if info, err := os.Stat(metadata); err == nil && info.IsDir() {
			return preflightDatabase(filepath.Join(metadata, "db"))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func preflightDatabase(database string) error {
	if info, err := os.Stat(filepath.Join(database, ".dolt")); err != nil || !info.IsDir() {
		return fmt.Errorf("internal database at %s is missing; run 4sight init for initialization or recovery", database)
	}
	lock, err := os.OpenFile(filepath.Join(database, ".4sight-preflight.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock internal databases: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()
	store := doltBootstrap{dataDir: database}
	type chain struct {
		name, expected string
		migrations     []embeddedMigration
		applied        int
	}
	chains := []chain{{name: "4sight_migration", expected: expectedMigrationID}, {name: "spec", expected: expectedSpecID}}
	for i := range chains {
		c := &chains[i]
		c.migrations, err = readEmbeddedMigrations(embedded, "migrations/dolt_"+c.name)
		if err != nil {
			return fmt.Errorf("%s embedded migrations: %w", c.name, err)
		}
		if c.migrations[len(c.migrations)-1].id != c.expected {
			return fmt.Errorf("%s embedded migration chain does not end at expected %s", c.name, c.expected)
		}
		rows, e := store.queryCSV("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = '" + c.name + "'")
		if e != nil || len(rows) == 0 || rows[0][0] == "0" {
			return fmt.Errorf("internal database %s at %s is missing or inaccessible; run 4sight init for initialization or recovery", c.name, database)
		}
		exists, e := store.trackingTableExists(c.name)
		if e != nil || !exists {
			return fmt.Errorf("tracking table for %s at %s is missing or inaccessible; run 4sight init for initialization or recovery", c.name, database)
		}
		rows, e = store.queryCSV("SELECT id FROM 4sight_migration." + c.name)
		if e != nil {
			return fmt.Errorf("read %s migration history: %w", c.name, e)
		}
		c.applied, err = validateHistory(c.name, c.migrations, rows)
		if err != nil {
			return err
		}

	}
	for _, c := range chains {
		for _, m := range c.migrations[c.applied:] {
			sql, e := embedded.ReadFile("migrations/dolt_" + c.name + "/" + m.name)
			if e != nil {
				return e
			}
			if e = store.run("sql", "-q", string(sql)); e != nil {
				return fmt.Errorf("apply %s migration %s: %w; the failed migration may require repair before retrying", c.name, m.id, e)
			}
			if e = store.recordApplied(c.name, m.id, time.Now().UTC().Truncate(time.Second)); e != nil {
				return fmt.Errorf("record %s migration %s: %w; the migration may require repair before retrying", c.name, m.id, e)
			}
		}
	}
	return nil
}

func validateHistory(name string, migrations []embeddedMigration, rows [][]string) (int, error) {
	present := make(map[string]bool)
	for _, row := range rows {
		if len(row) > 0 {
			present[row[0]] = true
		}
	}
	applied := 0
	for _, m := range migrations {
		if !present[m.id] {
			break
		}
		applied++
		delete(present, m.id)
	}
	for id := range present {
		return 0, fmt.Errorf("%s migration history is unknown, divergent, or incomplete at %s", name, id)
	}
	if applied == 0 {
		return 0, fmt.Errorf("%s migration history is incomplete at %s", name, migrations[0].id)
	}
	return applied, nil
}
