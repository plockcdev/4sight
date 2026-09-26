package cli

import (
	"fmt"
	"os"
	"strings"

	"foursight.dev/4sight/internal/migrate"
	"foursight.dev/4sight/internal/spec"
)

type deps struct {
	getwd               func() (string, error)
	up                  func(cwd string) error
	implementPrompt     func(cwd, schemaRel string) error
	migrateImport       func(cwd, schemaRel string) error
	specImport          func(cwd string, loc []string) error
	specExport          func(cwd string, loc []string) error
	specSyncCheck       func(cwd string) error
	specSyncDiff        func(cwd string) error
	specImplementPrompt func(cwd string) error
	specChangeImport    func(cwd string) error
}

func Run(args []string) int {
	return run(args, deps{
		getwd:               os.Getwd,
		up:                  migrate.Up,
		implementPrompt:     migrate.ImplementPrompt,
		migrateImport:       spec.ImportSchema,
		specImport:          spec.Import,
		specExport:          spec.Export,
		specSyncCheck:       spec.SyncCheck,
		specSyncDiff:        spec.SyncDiff,
		specImplementPrompt: spec.ImplementPrompt,
		specChangeImport:    spec.ChangeImport,
	})
}

func run(args []string, d deps) int {
	if len(args) == 0 {
		printMainHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help", "help":
		if len(args) > 1 {
			return runHelpTopic(args[1:])
		}
		printMainHelp()
		return 0
	case "migrate":
		return parseMigrate(args[1:], d)
	case "spec":
		return parseSpec(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printMainHelp()
		return 1
	}
	fmt.Fprintf(os.Stderr, "4sight: unknown command %q\n", args[0])
	return 1
}

func runHelpTopic(topic []string) int {
	switch strings.Join(topic, " ") {
	case "migrate":
		printMigrateHelp()
		return 0
	case "migrate up":
		printMigrateUpHelp()
		return 0
	case "migrate implement":
		printMigrateImplementHelp()
		return 0
	case "migrate implement prompt":
		printMigrateImplementPromptHelp()
		return 0
	case "spec":
		printSpecHelp()
		return 0
	case "spec import":
		printSpecImportHelp()
		return 0
	case "spec implement":
		printSpecImplementHelp()
		return 0
	case "spec implement prompt":
		printSpecImplementPromptHelp()
		return 0
	case "spec export":
		printSpecExportHelp()
		return 0
	case "spec sync-check":
		printSpecSyncCheckHelp()
		return 0
	case "spec sync-diff":
		printSpecSyncDiffHelp()
		return 0
	case "spec change":
		printSpecChangeHelp()
		return 0
	case "spec change import":
		printSpecChangeImportHelp()
		return 0
	case "migrate import":
		printMigrateImportHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "4sight: unknown help topic %q (try: migrate, migrate up, migrate import, migrate implement prompt, spec, spec import, spec export, spec sync-check, spec sync-diff, spec implement prompt, spec change, spec change import)\n", strings.Join(topic, " "))
		return 1
	}
}

func parseMigrate(args []string, d deps) int {
	if len(args) == 0 {
		printMigrateHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printMigrateHelp()
		return 0
	case "up":
		return parseMigrateUp(args[1:], d)
	case "import":
		return parseMigrateImport(args[1:], d)
	case "implement":
		return parseMigrateImplement(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printMigrateHelp()
		return 1
	}
	fmt.Fprintf(os.Stderr, "4sight migrate: unknown command %q\n", args[0])
	return 1
}

func parseMigrateImplement(args []string, d deps) int {
	if len(args) == 0 {
		printMigrateImplementHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printMigrateImplementHelp()
		return 0
	case "prompt":
		return parseMigrateImplementPrompt(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printMigrateImplementHelp()
		return 1
	}
	fmt.Fprintf(os.Stderr, "4sight migrate implement: unknown command %q\n", args[0])
	return 1
}

func parseMigrateImplementPrompt(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printMigrateImplementPromptHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printMigrateImplementPromptHelp()
			return 1
		}
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "4sight migrate implement prompt: requires exactly one schema spec path")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight migrate implement prompt:", err)
		return 1
	}
	if err := d.implementPrompt(cwd, args[0]); err != nil {
		fmt.Fprintln(os.Stderr, "4sight migrate implement prompt:", err)
		return 1
	}
	return 0
}

func parseSpec(args []string, d deps) int {
	if len(args) == 0 {
		printSpecHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printSpecHelp()
		return 0
	case "import":
		return parseSpecImport(args[1:], d)
	case "export":
		return parseSpecExport(args[1:], d)
	case "sync-check":
		return parseSpecSyncCheck(args[1:], d)
	case "sync-diff":
		return parseSpecSyncDiff(args[1:], d)
	case "implement":
		return parseSpecImplement(args[1:], d)
	case "change":
		return parseSpecChange(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printSpecHelp()
		return 1
	}
	fmt.Fprintf(os.Stderr, "4sight spec: unknown command %q\n", args[0])
	return 1
}

func parseSpecImplement(args []string, d deps) int {
	if len(args) == 0 {
		printSpecImplementHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printSpecImplementHelp()
		return 0
	case "prompt":
		return parseSpecImplementPrompt(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printSpecImplementHelp()
		return 1
	}
	fmt.Fprintf(os.Stderr, "4sight spec implement: unknown command %q\n", args[0])
	return 1
}

func parseSpecImplementPrompt(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printSpecImplementPromptHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printSpecImplementPromptHelp()
			return 1
		}
	}
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "4sight spec implement prompt: does not take arguments")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec implement prompt:", err)
		return 1
	}
	if err := d.specImplementPrompt(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec implement prompt:", err)
		return 1
	}
	return 0
}

func parseSpecChange(args []string, d deps) int {
	if len(args) == 0 {
		printSpecChangeHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printSpecChangeHelp()
		return 0
	case "import":
		return parseSpecChangeImport(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printSpecChangeHelp()
		return 1
	}
	fmt.Fprintf(os.Stderr, "4sight spec change: unknown command %q\n", args[0])
	return 1
}

func parseSpecChangeImport(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printSpecChangeImportHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printSpecChangeImportHelp()
			return 1
		}
	}
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "4sight spec change import: does not take arguments")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change import:", err)
		return 1
	}
	if err := d.specChangeImport(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change import:", err)
		return 1
	}
	return 0
}

func parseSpecExport(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printSpecExportHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printSpecExportHelp()
			return 1
		}
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "4sight spec export: requires a spec file location (command path then file name)")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec export:", err)
		return 1
	}
	if err := d.specExport(cwd, args); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec export:", err)
		return 1
	}
	return 0
}

func parseSpecSyncDiff(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printSpecSyncDiffHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printSpecSyncDiffHelp()
			return 1
		}
	}
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "4sight spec sync-diff: does not take arguments")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec sync-diff:", err)
		return 1
	}
	if err := d.specSyncDiff(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec sync-diff:", err)
		return 1
	}
	return 0
}

func parseSpecSyncCheck(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printSpecSyncCheckHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printSpecSyncCheckHelp()
			return 1
		}
	}
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "4sight spec sync-check: does not take arguments")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec sync-check:", err)
		return 1
	}
	if err := d.specSyncCheck(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec sync-check:", err)
		return 1
	}
	return 0
}

func parseMigrateImport(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printMigrateImportHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printMigrateImportHelp()
			return 1
		}
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "4sight migrate import: requires exactly one schema spec path")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight migrate import:", err)
		return 1
	}
	if err := d.migrateImport(cwd, args[0]); err != nil {
		fmt.Fprintln(os.Stderr, "4sight migrate import:", err)
		return 1
	}
	return 0
}

func parseSpecImport(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printSpecImportHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printSpecImportHelp()
			return 1
		}
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "4sight spec import: requires a spec file location (command path then file name)")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec import:", err)
		return 1
	}
	if err := d.specImport(cwd, args); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec import:", err)
		return 1
	}
	return 0
}

func parseMigrateUp(args []string, d deps) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			printMigrateUpHelp()
			return 0
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printMigrateUpHelp()
			return 1
		}
	}
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "4sight migrate up: does not take arguments")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight migrate up:", err)
		return 1
	}
	if err := d.up(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight migrate up:", err)
		return 1
	}
	return 0
}

func printMainHelp() {
	fmt.Print(`4sight — project tooling

Commands:
  help              Show this help
  migrate           Apply database schema migrations
  spec              Import and manage specification bodies

Global options:
  -h, --help        Show this help

Run "4sight help <command>" or "<command> --help" for command help.
`)
}

func printMigrateHelp() {
	fmt.Print(`4sight migrate

Apply SQL migrations so databases match the submitted schema.

Commands:
  up                Apply pending migrations
  import            Import a schema spec into bodies
  implement         Generate migration implementation prompts

Options:
  -h, --help        Show this help
`)
}

func printMigrateImplementHelp() {
	fmt.Print(`4sight migrate implement

Generate prompts for creating migration SQL from schema specs.

Commands:
  prompt            Output a prompt for an agent to generate a migration

Options:
  -h, --help        Show this help
`)
}

func printMigrateImplementPromptHelp() {
	fmt.Print(`4sight migrate implement prompt <schema>

Output a prompt instructing an agent to read the schema specification and
related design documents, then generate a migration SQL file.

The schema argument is a path relative to .4sight/schemas, such as
dolt_spec.md or dolt_spec.

Options:
  -h, --help        Show this help
`)
}

func printSpecHelp() {
	fmt.Print(`4sight spec

Import, export, and check specification bodies in the spec database.

Commands:
  import            Import a spec file into bodies
  change            Record filesystem spec changes as a change
  export            Write a spec body from the database to disk
  sync-check        Compare database bodies with on-disk spec files
  sync-diff         Show unified diffs for spec differences
  implement         Generate implementation prompts from specs

Options:
  -h, --help        Show this help
`)
}

func printSpecImplementHelp() {
	fmt.Print(`4sight spec implement

Generate prompts for updating the application from specification documents.

Commands:
  prompt            Output a prompt for an agent to align the codebase with the spec

Options:
  -h, --help        Show this help
`)
}

func printSpecImplementPromptHelp() {
	fmt.Print(`4sight spec implement prompt

Output a prompt instructing an agent to read design.md, cmd.md, and coding.md
bodies from the spec database, plus schema specifications on disk, then
update application code to match without changing the schema. If a schema
spec or migration change is required, the agent should stop and report the
gap.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeHelp() {
	fmt.Print(`4sight spec change

Record specification filesystem changes in the spec database.

Commands:
  import            Persist filesystem spec deltas as a workspace change

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeImportHelp() {
	fmt.Print(`4sight spec change import

Persist filesystem changes to spec files as a new change.

When spec files differ from workspace bodies, creates a child of the
current workspace change with an empty prompt, records a file change
for each delta, and updates the workspace pointer. Trailing whitespace
and empty lines at the end of a file are ignored.

If nothing differs, prints "no changes found" to stderr and exits
successfully.

Prints YAML with the new change id and each file change path and
+/- line summary.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printSpecExportHelp() {
	fmt.Print(`4sight spec export <command...> <file>

Write a specification body from the spec database onto disk, overwriting
the on-disk file.

The location is the command/subcommand path as separate arguments,
then the file name. For example:

  4sight spec export spec implement prompt cmd.md

If the body is not in the database, the command fails.

Options:
  -h, --help        Show this help
`)
}

func printSpecSyncDiffHelp() {
	fmt.Print(`4sight spec sync-diff

Render unified diffs for specification bodies that differ between the
spec database and on-disk design.md, cmd.md, coding.md, and schema
spec files.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printSpecSyncCheckHelp() {
	fmt.Print(`4sight spec sync-check

Compare specification bodies in the database with on-disk design.md,
cmd.md, and coding.md files.

Prints three sections: import commands for new spec files, import
commands for updated spec files, and export commands for bodies that
do not match. Ends with a summary count of matching, not in db, and
mismatched files.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printMigrateImportHelp() {
	fmt.Print(`4sight migrate import <schema>

Import a schema specification file into the spec.bodies table.

The schema argument is a path relative to .4sight/schemas, such as
dolt_spec.md or dolt_spec.

If the file does not exist, the command fails.

Options:
  -h, --help        Show this help
`)
}

func printSpecImportHelp() {
	fmt.Print(`4sight spec import <command...> <file>

Import a specification file into the spec.bodies table.

The location is the command/subcommand path as separate arguments,
then the file name. For example, the cmd.md spec for "myprog foo bar":

  4sight spec import myprog foo bar cmd.md

If the file does not exist, the command fails.

Options:
  -h, --help        Show this help
`)
}

func printMigrateUpHelp() {
	fmt.Print(`4sight migrate up

Apply pending migrations, starting with 4sight_migration, then every
database named by .4sight/schemas.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}
