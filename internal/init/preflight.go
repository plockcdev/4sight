package init

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// These IDs identify the final migrations embedded in this binary.
const expectedMigrationID = "00022367"
const expectedSpecID = "00113758"

// Preflight validates the internal databases without changing them.
func Preflight(cwd string) error {
	release, err := AcquirePreflight(cwd, false)
	if err == nil {
		release()
	}
	return err
}

// AcquirePreflight holds the project lock until release is called.
func AcquirePreflight(cwd string, allowOlder bool) (func(), error) {
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	for {
		metadata := filepath.Join(dir, ".4sight")
		if info, err := os.Stat(metadata); err == nil && info.IsDir() {
			return acquireDatabase(filepath.Join(metadata, "db"), allowOlder)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return func() {}, nil
		}
		dir = parent
	}
}

type internalChain struct {
	name, expected string
	migrations     []embeddedMigration
	applied        int
}

func preflightDatabase(database string) error {
	release, err := acquireDatabase(database, false)
	if err == nil {
		release()
	}
	return err
}

func acquireDatabase(database string, allowOlder bool) (func(), error) {
	if info, err := os.Stat(filepath.Join(database, ".dolt")); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("internal database at %s is missing; restore or reconcile the installation", database)
	}
	lock, err := os.OpenFile(filepath.Join(database, ".4sight-preflight.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("lock internal databases: %w", err)
	}
	release := func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); _ = lock.Close() }
	chains, err := validateInternal(database)
	if err != nil {
		release()
		return nil, err
	}
	for _, c := range chains {
		if !allowOlder && c.applied != len(c.migrations) {
			release()
			return nil, fmt.Errorf("internal database %s requires an upgrade; run 4sight upgrade", c.name)
		}
	}
	return release, nil
}

func validateInternal(database string) ([]internalChain, error) {
	store := doltBootstrap{dataDir: database}
	var err error
	chains := []internalChain{{name: "4sight_migration", expected: expectedMigrationID}, {name: "spec", expected: expectedSpecID}}
	for i := range chains {
		c := &chains[i]
		c.migrations, err = readEmbeddedMigrations(embedded, "migrations/dolt_"+c.name)
		if err != nil {
			return nil, fmt.Errorf("%s embedded migrations: %w", c.name, err)
		}
		if c.migrations[len(c.migrations)-1].id != c.expected {
			return nil, fmt.Errorf("%s embedded migration chain does not end at expected %s", c.name, c.expected)
		}
		rows, e := store.queryCSV("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = '" + c.name + "'")
		if e != nil || len(rows) == 0 || rows[0][0] == "0" {
			return nil, fmt.Errorf("internal database %s at %s is missing or inaccessible; restore or reconcile the installation before proceeding", c.name, database)
		}
		exists, e := store.trackingTableExists(c.name)
		if e != nil || !exists {
			return nil, fmt.Errorf("tracking table for %s at %s is missing or inaccessible; restore or reconcile the installation before proceeding", c.name, database)
		}
		rows, e = store.queryCSV("SELECT id, time FROM 4sight_migration." + c.name)
		if e != nil {
			return nil, fmt.Errorf("read %s migration history: %w", c.name, e)
		}
		for _, row := range rows {
			if len(row) != 2 {
				return nil, fmt.Errorf("%s migration history has malformed rows; restore or reconcile it", c.name)
			}
			if _, err := time.Parse("2006-01-02 15:04:05", row[1]); err != nil {
				return nil, fmt.Errorf("%s migration history has invalid timestamp for %s; restore or reconcile it", c.name, row[0])
			}
		}
		c.applied, err = validateHistory(c.name, c.migrations, rows)
		if err != nil {
			return nil, err
		}

	}
	return chains, nil
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
		return 0, fmt.Errorf("%s migration history is unknown, divergent, or incomplete at %s; restore or reconcile the recorded history before proceeding", name, id)
	}
	if applied == 0 {
		return 0, fmt.Errorf("%s migration history is incomplete at %s; restore or reconcile the recorded history before proceeding", name, migrations[0].id)
	}
	return applied, nil
}
