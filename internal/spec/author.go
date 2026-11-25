package spec

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

//go:embed assets/application-database-lifecycle.md
var applicationDatabaseLifecycle string

func ChangeAuthorAgentPrompt(cwd string) error {
	root, err := findMetadataRoot(cwd)
	if err != nil {
		return err
	}
	return runChangeAuthorPrompt(root, newDoltStore(filepath.Join(root, "db")), os.Stdout)
}

func runChangeAuthorPrompt(root string, st *doltStore, out io.Writer) error {
	state, err := st.reviewState()
	if err != nil {
		return fmt.Errorf("read current changes: %w", err)
	}
	if state.Prompt == "" {
		_, err := fmt.Fprintf(out, "Workspace change: %s\nSubmitted change: %s\nThere is no pending spec authoring request.\n", state.Workspace, state.Submitted)
		return err
	}
	problems, err := st.workspaceSyncProblems(root)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return fmt.Errorf("workspace specs are out of sync:\n- %s\nRun `4sight spec change import` or export the affected workspace bodies, then retry", strings.Join(problems, "\n- "))
	}
	if err := os.MkdirAll(filepath.Join(root, "changes", state.Workspace), 0o755); err != nil {
		return err
	}
	diff, err := st.compositeDiff(state.Submitted, state.Workspace)
	if err != nil {
		return err
	}
	diffPath := filepath.Join(root, "changes", state.Workspace, "unified.diff")
	if err := os.WriteFile(diffPath, []byte(diff), 0o644); err != nil {
		return err
	}
	docs, err := st.listWorkspaceBodies()
	if err != nil {
		return err
	}
	var codeSpecs, schemaSpecs []string
	for _, doc := range docs {
		if doc.Deleted || isTextAssetDir(doc.Dir) {
			continue
		}
		path := specRelPath(doc.Dir, doc.Name)
		if doc.Dir == "schemas" {
			schemaSpecs = append(schemaSpecs, path)
		} else {
			codeSpecs = append(codeSpecs, path)
		}
	}
	slices.Sort(codeSpecs)
	slices.Sort(schemaSpecs)
	var b strings.Builder
	fmt.Fprintf(&b, "Workspace change: %s\nSubmitted change: %s\nComposite diff: .4sight/changes/%s/unified.diff\n", state.Workspace, state.Submitted, state.Workspace)
	b.WriteString("\nLive workspace code specs:\n")
	for _, path := range codeSpecs {
		fmt.Fprintf(&b, "- .4sight/%s\n", path)
	}
	b.WriteString("\nLive workspace schema specs:\n")
	for _, path := range schemaSpecs {
		fmt.Fprintf(&b, "- .4sight/%s\n", path)
	}
	b.WriteString("\nWhen persistence is needed, default to file-backed Dolt without a daemon unless a different database is already established. Other database backends have not been tested. Preserve project-specific requirements and established choices. Users should commit Dolt database files alongside application changes; Git-based database rollback is deferred. Establish the initial schema spec and specify application access as needed. 4sight implementation prepares and applies migrations to bootstrap the working database; do not invent automatic runtime migration requirements. When existing local data needs importing, suggest a one-off tool after database setup and before using the application, not an application spec subcommand by default. Follow the standard application migration tracking requirements.\n")
	b.WriteString("\nIf the application uses a database, specify its initialization and upgrade lifecycle, including deployed instances and existing data. Use the following reference as the default design unless the application has a different migration design or its spec explicitly excludes migration support. Capture application behavior in design or command specs and persisted tracking requirements in schema specs. Preserve established choices and ask the user when consequential decisions cannot be inferred. Distinguish the application lifecycle from 4sight's working-database migrations.\n\n")
	b.WriteString(applicationDatabaseLifecycle)
	b.WriteString("\n")
	b.WriteString("\nRead the composite diff, the full relevant workspace specs, all relevant schema specs, and related design and coding standards. Follow applicable coding standards, including referenced standards and command-path overrides, unless directly instructed otherwise by the user. Consider both code and schema specifications for every request and edit all relevant specs, including both kinds when affected, preserving their respective responsibilities. Program-wide specs live at .4sight/design.md and .4sight/coding.md; main command specs live under .4sight/main/; nested commands use .4sight/subcommands/<name>/subcommands/<name>/cmd.md, with inherited design.md and coding.md along the path; schema specs live under .4sight/schemas/<dbtype>_<dbname>.md. Use appropriate paths and create missing directories and files. Do not generate SQL migrations or modify application code. Schema references must be described semantically by conceptual name and target table, with relevant constraints and without implementation-specific `_id` suffixes. Entity tables imply an `id` primary key; pure join tables use their referenced fields as a composite key unless otherwise specified. When working on a schema without a competing design pattern, suggest that each entity has a core identity table, with optional data stored in one-to-one extension tables keyed by the entity ID. Present this as a flexible default, not a requirement that overrides an established schema design.\n\nSaved user request:\n")
	b.WriteString(state.Prompt)
	b.WriteString("\n\nIf any specs changed, run `4sight spec change import`, then run `4sight spec change review agent prompt` and follow the refreshed review instructions. If no specs changed, tell the user no review is needed because the request caused no spec changes, then stop.\n")
	_, err = io.WriteString(out, b.String())
	return err
}
