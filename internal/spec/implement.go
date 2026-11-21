package spec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

const implementPromptTemplate = `Read these design and command specification documents from the spec database to understand the application behavior. These code specification files inform application behavior; read all listed files and applicable inherited design and coding standards:
{{range .CodeSpecs}}
----- SPEC {{.Path}} -----
{{.Body}}{{if not (endsInNewline .Body)}}
{{end}}----- END SPEC -----
{{end}}{{if .RemovedCodeSpecs}}
These code specs were removed (their workspace bodies are empty):
{{range .RemovedCodeSpecs}}- {{.}}
{{end}}{{end}}
Read these database schema specifications:
Group the schemas by database type:
{{range .SchemaGroups}}
{{.DBType}}:
{{range .Paths}}- {{.}}
{{end}}{{end}}
{{if .CompositeDiffPath}}Read the composite diff at {{.CompositeDiffPath}} to inspect path changes, deletions, text asset changes, and rename/copy relationship metadata. Text assets are omitted from the spec list; read relevant text asset files from disk.
{{end}}
Compare implemented and submitted spec bodies, including schema additions, changes, and deletions, and inspect migration state. Do not edit or import specs. Account for startup preflight: do not run newly built binaries against the working project if they would apply migrations outside the requested scope or before the permitted step. Use isolated test projects as needed; stop if this cannot be done safely.

Check code and schema specifications and implementation for logic, integrity, performance, mismatches, and ambiguous behavior. If a problem prevents safe implementation, stop and report the file, line number, and a search string, and recommend how to resolve it with 4sight spec or migration commands.

{{.ProjectIdentity}}

Maintain a durable task list when available, marking steps complete only after success.

For a full implementation, compare implemented and submitted schema identities and prepare all necessary migrations before changing code. Follow the migration design for metadata, parent chains, and retirements; reuse matching migrations, and stop on ambiguous baselines or unsafe dependencies. Do not prepare migrations for unchanged schemas. Do not apply migrations to the working project during preparation or testing.

Read the applicable schema and migration design specifications. Infer indexes and constraints. Entity tables have an implied id primary key unless specified otherwise; pure join tables use their referenced columns as a composite key. Resolve references against the referenced primary key, using _id for single-column references and one <reference>_<component>_id column per composite component. Stop if the component mapping is ambiguous.

Inspect each live schema's submitted migration. Reuse a migration only when its recorded schema identity and normalized CRC32 match; a match does not prove SQL correctness or application. If metadata is absent and the baseline cannot otherwise be established, stop for reconciliation. SHA-256 is metadata, not a matching key. Do not create empty SQL solely for metadata. For a new database prepare an initial database-creation migration followed by a schema migration when SQL is required. Validate and reuse existing chain members on retry. For a deleted schema use its implemented identity and body; if its database was created, prepare a final retirement entry. Reject internal database deletion and conflicting lifecycles.

Make the code changes, then test migration chains on isolated fresh databases and representative prior databases with histories and data. Verify ordering, preservation, behavior, and failure and retry handling. Once checks pass, apply migrations with {{.MigrateUpCommand}}; for 4sight itself invoke the new source with go run ./cmd/4sight. Verify histories and current links, then run {{.CompleteCommand}} only after all work succeeds. Stop and report repair or a staged rollout if the transition is unsafe.

Verify application tracking-table creation, internal upgrade gating, and migration ordering. Do not run 4sight upgrade for the self-hosted application migration. If project identity requires a user answer, provide that answer to the migration command on standard input as its specification requires. Do not upgrade the working project during preparation or testing.

For an explicitly code-only request, do not generate or apply migrations. Stop if code depends on a pending migration. After verification, mark complete only if all other required work is already complete; otherwise report the remaining work.

For an explicitly schema-only request, prepare the required migrations and report what remains pending; do not apply them or mark the change complete. For internal databases the new binary must embed and validate the complete chains.

Remove command implementations whose cmd.md specs no longer exist. Update command routing, help, and usage accordingly.

Update command help when command structure, arguments, options, or flags change. Show help for syntax errors and for --help at any command depth. If help is combined with other arguments, show help and return an error. Help lists commands, options, and arguments, not usage. Semantic errors should explain the problem without help.

Maintain docs/USAGE.txt and docs/USAGE_<subcommand>.txt for every command recursively. Usage is brief practical guidance with examples. Explain design principles and conventions where users need them, in the relevant usage document. Provide {{.UsageCommand}} only as a top-level subcommand, and {{.UsageFlag}} on the main command and every subcommand. Each prints the corresponding usage file; extra arguments or flags are an error and show help. List the usage subcommand only in top-level help and the flag in help at every command level.

Ensure each current internal database migration ID is embedded in the binary and its embedded chain includes all migrations through that ID.
`

type promptCodeSpec struct {
	Path string
	Body string
}

type promptSchemaGroup struct {
	DBType string
	Paths  []string
}

type implementPromptData struct {
	CodeSpecs         []promptCodeSpec
	RemovedCodeSpecs  []string
	SchemaGroups      []promptSchemaGroup
	CompositeDiffPath string
	MigrateUpCommand  string
	UsageCommand      string
	UsageFlag         string
	CompleteCommand   string
	ProjectIdentity   string
}

func ImplementPrompt(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runImplementPrompt(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

// CheckImplementationReady verifies the workspace before applying migrations.
func CheckImplementationReady(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	st := newDoltStore(filepath.Join(root, "db"))
	docs, err := st.listLatestBodies()
	if err != nil {
		return err
	}
	if err := verifyImplementSpecsInSync(root, docs); err != nil {
		return err
	}
	workspace, submitted, err := st.changePointers()
	if err != nil {
		return err
	}
	implemented, err := st.implementedChangeID()
	if err != nil {
		return err
	}
	if workspace != submitted {
		return fmt.Errorf("workspace %s differs from submitted change %s", workspace, submitted)
	}
	if submitted == implemented {
		return fmt.Errorf("no submitted change awaits implementation")
	}
	return nil
}

func runImplementPrompt(metadataRoot string, st store, stdout io.Writer) error {
	docs, err := st.listLatestBodies()
	if err != nil {
		return fmt.Errorf("list spec bodies: %w", err)
	}
	data := implementPromptData{
		MigrateUpCommand: "`4sight spec change implement agent migrate up`",
		UsageCommand:     "`usage`",
		UsageFlag:        "`--usage`",
		CompleteCommand:  "`4sight spec change implement agent done`",
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
		if diffStore, ok := st.(interface {
			compositeDiff(from, to string) (string, error)
		}); ok {
			diff, err := diffStore.compositeDiff(implemented, submitted)
			if err != nil {
				return fmt.Errorf("render submitted composite diff: %w", err)
			}
			rel := filepath.Join("changes", submitted, "unified.diff")
			path := filepath.Join(metadataRoot, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("create composite diff directory: %w", err)
			}
			if err := os.WriteFile(path, []byte(diff), 0o644); err != nil {
				return fmt.Errorf("write submitted composite diff: %w", err)
			}
			dataPath := filepath.ToSlash(filepath.Join(".4sight", rel))
			data.CompositeDiffPath = dataPath
		}
	}
	var removedCodeDocs []specDoc
	for _, doc := range docs {
		if isCodeSpecName(doc.Name) {
			if doc.Deleted {
				removedCodeDocs = append(removedCodeDocs, doc)
				continue
			}
			data.CodeSpecs = append(data.CodeSpecs, promptCodeSpec{
				Path: displaySpecPath(doc.Dir, doc.Name),
				Body: doc.Body,
			})
		}
	}
	slices.SortFunc(data.CodeSpecs, func(a, b promptCodeSpec) int {
		return strings.Compare(a.Path, b.Path)
	})
	for _, doc := range removedCodeDocs {
		data.RemovedCodeSpecs = append(data.RemovedCodeSpecs, displaySpecPath(doc.Dir, doc.Name))
	}
	slices.Sort(data.RemovedCodeSpecs)

	schemas, err := listSchemaSpecs(filepath.Join(metadataRoot, "schemas"))
	if err != nil {
		return err
	}

	projectRoot := filepath.Dir(metadataRoot)
	data.ProjectIdentity = implementationProjectIdentity(projectRoot)
	lastType := ""
	for _, schema := range schemas {
		base := filepath.Base(schema)
		dbType, _, _ := strings.Cut(base, "_")
		if dbType != lastType {
			data.SchemaGroups = append(data.SchemaGroups, promptSchemaGroup{DBType: dbType})
			lastType = dbType
		}
		group := &data.SchemaGroups[len(data.SchemaGroups)-1]
		group.Paths = append(group.Paths, relFromProject(projectRoot, schema))
	}
	tmpl, err := template.New("implement-prompt").Funcs(template.FuncMap{
		"endsInNewline": func(value string) bool { return strings.HasSuffix(value, "\n") },
	}).Parse(implementPromptTemplate)
	if err != nil {
		return fmt.Errorf("parse implementation prompt template: %w", err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return fmt.Errorf("render implementation prompt: %w", err)
	}
	_, err = io.WriteString(stdout, output.String())
	return err
}

func implementationProjectIdentity(root string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = root
	output, err := cmd.Output()
	remote := strings.TrimRight(strings.TrimSpace(string(output)), "/")
	if pos := strings.LastIndexAny(remote, "/:"); pos >= 0 {
		remote = remote[pos+1:]
	}
	remote = strings.TrimSuffix(remote, ".git")
	local := filepath.Base(root)
	if err != nil || remote == "" || local == "." || local == string(filepath.Separator) {
		return "Project identity is ambiguous. Stop and ask the user whether this is 4sight itself or an ordinary application; retain the answer for the migration step."
	}
	selfLocal := strings.EqualFold(local, "4sight")
	selfRemote := strings.EqualFold(remote, "4sight")
	if selfLocal != selfRemote {
		return "Project identity is ambiguous. Stop and ask the user whether this is 4sight itself or an ordinary application; retain the answer for the migration step."
	}
	if selfLocal {
		return "This project is 4sight developing itself."
	}
	return "This project is an ordinary application."
}

func verifyImplementSpecsInSync(root string, bodies []specDoc) error {
	disk, err := listOnDiskCodeSpecs(root)
	if err != nil {
		return err
	}
	db := make(map[specKey]specDoc, len(bodies))
	for _, doc := range bodies {
		db[specKey{dir: doc.Dir, name: doc.Name}] = doc
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
		if inDB {
			if cached.Deleted && !onDisk {
				continue
			}
			if !cached.Deleted && onDisk && normalizeTrackedBody(key.dir, key.name, local) == normalizeTrackedBody(key.dir, key.name, cached.Body) {
				continue
			}
		} else if !onDisk {
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
