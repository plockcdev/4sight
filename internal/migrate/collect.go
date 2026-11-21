package migrate

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"foursight.dev/4sight/internal/intent"
)

type collectionRecord struct {
	original, trash, migration, trashed, state string
	id                                         physicalID
}

// GarbageCollect requires the caller to hold the shared project lock.
func GarbageCollect(cwd, selected string, stdin *os.File, stdout io.Writer) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	st := newDoltStore(filepath.Join(root, "db"))
	records, err := collectionRecords(st)
	if err != nil {
		return err
	}
	active, err := activeDatabaseNames(st)
	if err != nil {
		return err
	}
	var chosen []collectionRecord
	for _, record := range records {
		if active[strings.ToLower(record.trash)] || strings.EqualFold(record.original, "spec") || strings.EqualFold(record.original, "4sight_migration") {
			continue
		}
		if selected == "" || selected == record.trash {
			chosen = append(chosen, record)
		}
	}
	if selected != "" && len(chosen) == 0 {
		return fmt.Errorf("trash database %q is unknown, already collected, or incomplete", selected)
	}
	if len(chosen) == 0 {
		_, err = fmt.Fprintln(stdout, "no eligible databases to collect")
		return err
	}
	for _, record := range chosen {
		count := "unavailable"
		size := "unavailable"
		if ident(record.trash) == nil {
			rows, e := st.queryCSV("SELECT COUNT(*), SUM(data_length + index_length) FROM information_schema.tables WHERE table_schema='" + record.trash + "'")
			if e == nil && len(rows) == 1 && len(rows[0]) > 1 {
				count = rows[0][0]
				if rows[0][1] != "" && !strings.EqualFold(rows[0][1], "NULL") {
					size = rows[0][1] + " bytes"
				}
			}
		}
		if _, err := fmt.Fprintf(stdout, "%s -> %s; trashed %s; migration %s; size %s; tables %s\n", record.original, record.trash, record.trashed, record.migration, size, count); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(stdout, "%d selected; deletion is permanent. Type yes to continue: ", len(chosen)); err != nil {
		return err
	}
	info, err := stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return fmt.Errorf("collection canceled: interactive input is unavailable")
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "yes" {
		return fmt.Errorf("collection canceled")
	}
	var errors []string
	for _, record := range chosen {
		if err := collectOne(root, st, record); err != nil {
			if _, outputErr := fmt.Fprintf(stdout, "%s: failed: %v\n", record.trash, err); outputErr != nil {
				errors = append(errors, outputErr.Error())
			}
			errors = append(errors, record.trash+": "+err.Error())
		} else {
			if _, outputErr := fmt.Fprintf(stdout, "%s: collected\n", record.trash); outputErr != nil {
				errors = append(errors, outputErr.Error())
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("collection failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

func collectionRecords(st *doltStore) ([]collectionRecord, error) {
	rows, err := st.queryCSV("SELECT original_name,trash_name,migration_id,trashed_at,state,physical_device,physical_inode FROM 4sight_migration.database_deletions WHERE trashed_at IS NOT NULL AND garbage_collected_at IS NULL AND state IN ('trashed','collecting') ORDER BY trash_name")
	if err != nil {
		return nil, err
	}
	var out []collectionRecord
	for _, r := range rows {
		if len(r) != 7 {
			return nil, fmt.Errorf("malformed retirement record")
		}
		dev, e1 := strconv.ParseUint(r[5], 10, 64)
		ino, e2 := strconv.ParseUint(r[6], 10, 64)
		if e1 != nil || e2 != nil {
			return nil, fmt.Errorf("invalid physical identity for %s", r[1])
		}
		out = append(out, collectionRecord{r[0], r[1], r[2], r[3], r[4], physicalID{dev, ino}})
	}
	return out, nil
}

func activeDatabaseNames(st *doltStore) (map[string]bool, error) {
	rows, err := st.queryCSV("SELECT p.name FROM spec.bodies b JOIN spec.file_changes f ON f.id=b.file_change_id JOIN spec.filepaths p ON p.id=f.filepath_id JOIN spec.dirs d ON d.id=p.dir_id WHERE b.state IN ('submitted','workspace') AND b.deleted=0 AND d.path='schemas'")
	if err != nil {
		return nil, fmt.Errorf("inspect active database schemas: %w", err)
	}
	active := map[string]bool{}
	for _, row := range rows {
		if len(row) > 0 {
			stem, _ := strings.CutSuffix(row[0], ".md")
			_, name, ok := strings.Cut(stem, "_")
			if ok {
				active[strings.ToLower(name)] = true
			}
		}
	}
	return active, nil
}

func collectOne(root string, st *doltStore, want collectionRecord) error {
	if ident(want.original) != nil || ident(want.trash) != nil || !strings.HasPrefix(want.trash, "trashed_db_") || want.original == "spec" || want.original == "4sight_migration" {
		return fmt.Errorf("unsafe retirement identity")
	}
	active, err := activeDatabaseNames(st)
	if err != nil {
		return err
	}
	if active[strings.ToLower(want.trash)] {
		return fmt.Errorf("trash name belongs to an active database")
	}
	records, err := collectionRecords(st)
	if err != nil {
		return err
	}
	var current *collectionRecord
	for i := range records {
		if records[i].trash == want.trash {
			current = &records[i]
			break
		}
	}
	if current == nil || *current != want {
		return fmt.Errorf("retirement record changed after selection")
	}
	path := filepath.Join(root, "db", want.trash)
	identity, exists, err := identifyDirectory(path)
	if err != nil {
		return err
	}
	if !exists && want.state != "collecting" {
		return fmt.Errorf("trash database is unexpectedly absent; repair required")
	}
	if exists && identity != want.id {
		return fmt.Errorf("trash database identity differs; repair required")
	}
	if want.state == "trashed" {
		if err := st.run("sql", "-q", "UPDATE 4sight_migration.database_deletions SET state='collecting' WHERE trash_name='"+want.trash+"' AND state='trashed'"); err != nil {
			return err
		}
	}
	if exists {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		if err := intent.SyncDirectory(filepath.Dir(path)); err != nil {
			return err
		}
	}
	if _, exists, err := identifyDirectory(path); err != nil || exists {
		return fmt.Errorf("verify deletion failed: %v", err)
	}
	return st.run("sql", "-q", "UPDATE 4sight_migration.database_deletions SET state='collected', garbage_collected_at=NOW() WHERE trash_name='"+want.trash+"' AND state='collecting'")
}
