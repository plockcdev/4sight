package spec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func ImplementPrompt(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runImplementPrompt(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runImplementPrompt(metadataRoot string, st store, stdout io.Writer) error {
	docs, err := st.listLatestBodies()
	if err != nil {
		return fmt.Errorf("list spec bodies: %w", err)
	}
	if state, ok := st.(interface {
		changePointers() (string, string, error)
		implementedChangeID() (string, error)
	}); ok {
		if err := verifyImplementSpecsInSync(metadataRoot, docs); err != nil {
			return err
		}
		workspace, submitted, err := state.changePointers()
		if err != nil {
			return fmt.Errorf("read workspace and submitted pointers: %w", err)
		}
		if workspace != submitted {
			message := fmt.Sprintf("workspace change %s differs from submitted change %s", workspace, submitted)
			if summaries, ok := st.(interface {
				changesBetween(string, string) ([]string, error)
			}); ok {
				items, err := summaries.changesBetween(submitted, workspace)
				if err != nil {
					return fmt.Errorf("%s; list changes since submission: %w", message, err)
				}
				message += "; changes since submission:\n- " + strings.Join(items, "\n- ")
			}
			return fmt.Errorf("%s; submit the workspace change before implementing", message)
		}
		implemented, err := state.implementedChangeID()
		if err != nil {
			return fmt.Errorf("read implemented pointer: %w", err)
		}
		if implemented == submitted {
			_, err := fmt.Fprintf(stdout, "nothing to implement: submitted change %s is already implemented\n", submitted)
			return err
		}
		if err := verifySchemaMigrations(metadataRoot); err != nil {
			return err
		}
	}
	var codeDocs []specDoc
	for _, doc := range docs {
		if isCodeSpecName(doc.Name) {
			codeDocs = append(codeDocs, doc)
		}
	}
	slices.SortFunc(codeDocs, func(a, b specDoc) int {
		if c := strings.Compare(a.Dir, b.Dir); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})

	schemas, err := listSchemaSpecs(filepath.Join(metadataRoot, "schemas"))
	if err != nil {
		return err
	}

	projectRoot := filepath.Dir(metadataRoot)
	var b strings.Builder
	b.WriteString("Read these design and command specification documents from the spec database to understand the application behavior. These code specification files inform application behavior; read all listed files and applicable inherited design and coding standards:\n")
	for _, doc := range codeDocs {
		fmt.Fprintf(&b, "\n----- SPEC %s -----\n", displaySpecPath(doc.Dir, doc.Name))
		b.WriteString(doc.Body)
		if !strings.HasSuffix(doc.Body, "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("----- END SPEC -----\n")
	}
	b.WriteByte('\n')
	b.WriteString("Read these database schema specifications:\nGroup the schemas by database type:\n")
	lastType := ""
	for _, schema := range schemas {
		base := filepath.Base(schema)
		dbType, _, _ := strings.Cut(base, "_")
		if dbType != lastType {
			fmt.Fprintf(&b, "\n%s:\n", dbType)
			lastType = dbType
		}
		fmt.Fprintf(&b, "- %s\n", relFromProject(projectRoot, schema))
	}
	b.WriteByte('\n')
	b.WriteString("Review the application code specification and update the code to match the spec.\n")
	b.WriteString("Do not update the schema.\n")
	b.WriteString("If the schema needs a spec change or a migration to be compatible with the code spec, stop and tell the user the gap (schema spec or migration, and what kinds of changes are needed).\n")
	b.WriteString("\nCheck the schema specification and implementation for logic, integrity, and performance problems, mismatches with the code specification, and behavior that cannot be inferred. If a problem prevents safe implementation, stop and report the file, line number, and a search string, then recommend how to resolve it using the 4sight spec or migration commands. Before code that depends on a schema change, verify its current and submitted migration links match; if not, stop and ask the user to run `4sight migrate up`.\n")
	b.WriteString("\nUpdate command help when command structure, arguments, options, or flags change. Show help for syntax errors and for --help at any command depth. If help is combined with other arguments, show help and return an error. Help lists commands, options, and arguments, not usage. Semantic errors should explain the problem without help.\n")
	b.WriteString("\nMaintain docs/USAGE.txt and docs/USAGE_<subcommand>.txt for every command recursively. Usage is brief practical guidance with examples. `usage` or `--usage` prints the corresponding usage file; extra arguments are an error and show help.\n")
	b.WriteString("\nAfter successfully implementing the submitted change, run `4sight spec change implement agent done` to mark implementation complete.\n")
	_, err = io.WriteString(stdout, b.String())
	return err
}

func verifyImplementSpecsInSync(root string, bodies []specDoc) error {
	disk, err := listOnDiskCodeSpecs(root)
	if err != nil {
		return err
	}
	db := make(map[specKey]string, len(bodies))
	for _, doc := range bodies {
		db[specKey{dir: doc.Dir, name: doc.Name}] = doc.Body
	}
	keys := make(map[specKey]bool, len(disk)+len(db))
	for key := range disk {
		keys[key] = true
	}
	for key := range db {
		keys[key] = true
	}
	var mismatches []string
	for key := range keys {
		local, onDisk := disk[key]
		cached, inDB := db[key]
		if normalizeSpecBody(local) == normalizeSpecBody(cached) {
			continue
		}
		path := specRelPath(key.dir, key.name)
		if !inDB || key.dir == "schemas" {
			mismatches = append(mismatches, path+": run `4sight spec change import` to update workspace bodies")
		} else if !onDisk {
			mismatches = append(mismatches, path+": restore or edit the file, then run `4sight spec change import`")
		} else {
			mismatches = append(mismatches, path+": export the database body or run `4sight spec change import` to accept disk contents")
		}
	}
	if len(mismatches) == 0 {
		return nil
	}
	slices.Sort(mismatches)
	return fmt.Errorf("workspace spec files do not match their database bodies:\n- %s", strings.Join(mismatches, "\n- "))
}

func verifySchemaMigrations(root string) error {
	schemas, err := listSchemaSpecs(filepath.Join(root, "schemas"))
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, schema := range schemas {
		base := strings.TrimSuffix(filepath.Base(schema), ".md")
		dbType, dbName, ok := strings.Cut(base, "_")
		if !ok {
			continue
		}
		db := dbType + "_" + dbName
		if seen[db] {
			continue
		}
		seen[db] = true
		dir := filepath.Join(root, "migrations", db)
		current, err := filepath.EvalSymlinks(filepath.Join(dir, "current"))
		if err != nil {
			return fmt.Errorf("schema %s has no valid current migration link in %s; submit and apply its migration before implementing dependent code", filepath.Base(schema), dir)
		}
		submitted, err := filepath.EvalSymlinks(filepath.Join(dir, "submitted"))
		if err != nil {
			return fmt.Errorf("schema %s has no valid submitted migration link in %s; submit and apply its migration before implementing dependent code", filepath.Base(schema), dir)
		}
		if current != submitted {
			return fmt.Errorf("schema %s has different current and submitted migrations in %s; ask the user to run `4sight migrate up` before implementing dependent code", filepath.Base(schema), dir)
		}
	}
	return nil
}

func listSchemaSpecs(schemasDir string) ([]string, error) {
	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read schemas dir %s: %w", schemasDir, err)
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		paths = append(paths, filepath.Join(schemasDir, entry.Name()))
	}
	slices.Sort(paths)
	return paths, nil
}

func relFromProject(projectRoot, path string) string {
	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
