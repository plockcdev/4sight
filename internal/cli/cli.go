package cli

import (
	"fmt"
	"os"
	"strings"

	initcmd "foursight.dev/4sight/internal/init"
	"foursight.dev/4sight/internal/migrate"
	"foursight.dev/4sight/internal/spec"
)

type deps struct {
	getwd                        func() (string, error)
	up                           func(cwd string) error
	implementPrompt              func(cwd, schemaRel string) error
	migrateImport                func(cwd, schemaRel string) error
	specImport                   func(cwd string, loc []string) error
	specExport                   func(cwd string, loc []string) error
	specSyncCheck                func(cwd string) error
	specSyncDiff                 func(cwd string) error
	specImplementPrompt          func(cwd string) error
	specChangeImport             func(cwd string) error
	specChangeList               func(cwd string) error
	specChangeRequest            func(cwd, prompt string) error
	specChangeStatus             func(cwd string) error
	specChangeSubmit             func(cwd string) error
	specChangeImplementationDone func(cwd string) error
	specChangeReviewPrompt       func(cwd string) error
	specChangeReviewResponse     func(cwd string) error
	specChangeReviewShow         func(cwd, changeID string) error
	init                         func(cwd string) error
}

func Run(args []string) int {
	return run(args, deps{
		getwd:                        os.Getwd,
		up:                           migrate.Up,
		implementPrompt:              migrate.ImplementPrompt,
		migrateImport:                spec.ImportSchema,
		specImport:                   spec.Import,
		specExport:                   spec.Export,
		specSyncCheck:                spec.SyncCheck,
		specSyncDiff:                 spec.SyncDiff,
		specImplementPrompt:          spec.ImplementPrompt,
		specChangeImport:             spec.ChangeImport,
		specChangeList:               spec.ChangeList,
		specChangeRequest:            spec.ChangeRequest,
		specChangeStatus:             spec.ChangeStatus,
		specChangeSubmit:             spec.ChangeSubmit,
		specChangeImplementationDone: spec.ChangeImplementationDone,
		specChangeReviewPrompt:       spec.ChangeReviewAgentPrompt,
		specChangeReviewResponse:     spec.ChangeReviewAgentResponse,
		specChangeReviewShow:         spec.ChangeReviewShow,
		init:                         initcmd.Run,
	})
}

func run(args []string, d deps) int {
	if code, handled := runUsage(args); handled {
		return code
	}
	if code, handled := runExplicitHelp(args); handled {
		return code
	}
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
	case "init":
		return parseInit(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printMainHelp()
		return 1
	}
	printMainHelp()
	fmt.Fprintf(os.Stderr, "4sight: unknown command %q\n", args[0])
	return 1
}

func parseInit(args []string, d deps) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "4sight init: does not take arguments")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight init:", err)
		return 1
	}
	if err := d.init(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight init:", err)
		return 1
	}
	return 0
}

func runHelpTopic(topic []string) int {
	switch strings.Join(topic, " ") {
	case "migrate":
		printMigrateHelp()
		return 0
	case "init":
		printInitHelp()
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
	case "spec change list":
		printSpecChangeListHelp()
		return 0
	case "spec change request":
		printSpecChangeRequestHelp()
		return 0
	case "spec change status":
		printSpecChangeStatusHelp()
		return 0
	case "spec change submit":
		printSpecChangeSubmitHelp()
		return 0
	case "spec change implement agent done":
		printSpecChangeImplementationDoneHelp()
		return 0
	case "spec change review":
		printSpecChangeReviewHelp()
		return 0
	case "spec change review agent":
		printSpecChangeReviewAgentHelp()
		return 0
	case "spec change review agent prompt":
		printSpecChangeReviewPromptHelp()
		return 0
	case "spec change review agent response":
		printSpecChangeReviewResponseHelp()
		return 0
	case "spec change review show":
		printSpecChangeReviewShowHelp()
		return 0
	case "migrate import":
		printMigrateImportHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "4sight: unknown help topic %q (try: migrate, migrate up, migrate import, migrate implement prompt, spec, spec import, spec export, spec sync-check, spec sync-diff, spec implement prompt, spec change, spec change import, spec change list, spec change request, spec change status)\n", strings.Join(topic, " "))
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
	printMigrateHelp()
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
	printMigrateImplementHelp()
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
		printMigrateImplementPromptHelp()
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
	printSpecHelp()
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
	printSpecImplementHelp()
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
		printSpecImplementPromptHelp()
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
	case "list":
		return parseSpecChangeList(args[1:], d)
	case "request":
		return parseSpecChangeRequest(args[1:], d)
	case "status":
		return parseSpecChangeStatus(args[1:], d)
	case "submit":
		return parseSpecChangeSubmit(args[1:], d)
	case "implement":
		return parseSpecChangeImplement(args[1:], d)
	case "review":
		return parseSpecChangeReview(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printSpecChangeHelp()
		return 1
	}
	printSpecChangeHelp()
	fmt.Fprintf(os.Stderr, "4sight spec change: unknown command %q\n", args[0])
	return 1
}

func parseSpecChangeSubmit(args []string, d deps) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printSpecChangeSubmitHelp()
		return 0
	}
	if len(args) != 0 {
		printSpecChangeSubmitHelp()
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change submit:", err)
		return 1
	}
	if err := d.specChangeSubmit(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change submit:", err)
		return 1
	}
	return 0
}

func parseSpecChangeList(args []string, d deps) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printSpecChangeListHelp()
		return 0
	}
	if len(args) != 0 {
		printSpecChangeListHelp()
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change list:", err)
		return 1
	}
	if err := d.specChangeList(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change list:", err)
		return 1
	}
	return 0
}

func parseSpecChangeStatus(args []string, d deps) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printSpecChangeStatusHelp()
		return 0
	}
	if len(args) != 0 {
		printSpecChangeStatusHelp()
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change status:", err)
		return 1
	}
	if err := d.specChangeStatus(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change status:", err)
		return 1
	}
	return 0
}

func parseSpecChangeImplement(args []string, d deps) int {
	if len(args) == 0 {
		printSpecChangeImplementHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printSpecChangeImplementHelp()
		return 0
	case "agent":
		if len(args) == 1 {
			printSpecChangeImplementHelp()
			return 0
		}
		if len(args) == 2 && args[1] == "done" {
			cwd, err := d.getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, "4sight spec change implement agent done:", err)
				return 1
			}
			if err := d.specChangeImplementationDone(cwd); err != nil {
				fmt.Fprintln(os.Stderr, "4sight spec change implement agent done:", err)
				return 1
			}
			return 0
		}
		if len(args) == 3 && (args[1] == "done" && (args[2] == "-h" || args[2] == "--help")) {
			printSpecChangeImplementationDoneHelp()
			return 0
		}
		printSpecChangeImplementHelp()
		return 1
	default:
		printSpecChangeImplementHelp()
		fmt.Fprintf(os.Stderr, "4sight spec change implement: unknown command %q\n", args[0])
		return 1
	}
}

func parseSpecChangeRequest(args []string, d deps) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printSpecChangeRequestHelp()
		return 0
	}
	if len(args) == 0 {
		printSpecChangeRequestHelp()
		fmt.Fprintln(os.Stderr, "4sight spec change request: requires a prompt")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change request:", err)
		return 1
	}
	if err := d.specChangeRequest(cwd, strings.Join(args, " ")); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change request:", err)
		return 1
	}
	return 0
}

func parseSpecChangeReview(args []string, d deps) int {
	if len(args) == 0 {
		printSpecChangeReviewHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printSpecChangeReviewHelp()
		return 0
	case "agent":
		return parseSpecChangeReviewAgent(args[1:], d)
	case "show":
		if len(args) > 2 {
			printSpecChangeReviewShowHelp()
			return 1
		}
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			printSpecChangeReviewShowHelp()
			return 0
		}
		cwd, err := d.getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "4sight spec change review show:", err)
			return 1
		}
		changeID := ""
		if len(args) == 2 {
			changeID = args[1]
		}
		if err := d.specChangeReviewShow(cwd, changeID); err != nil {
			fmt.Fprintln(os.Stderr, "4sight spec change review show:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(os.Stderr, "4sight spec change review: unknown command %q\n", args[0])
	printSpecChangeReviewHelp()
	return 1
}

func parseSpecChangeReviewAgent(args []string, d deps) int {
	if len(args) == 0 {
		printSpecChangeReviewAgentHelp()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printSpecChangeReviewAgentHelp()
		return 0
	case "prompt":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent prompt: takes no arguments")
			return 1
		}
		cwd, err := d.getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent prompt:", err)
			return 1
		}
		if err := d.specChangeReviewPrompt(cwd); err != nil {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent prompt:", err)
			return 1
		}
		return 0
	case "response":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent response: takes no arguments")
			return 1
		}
		cwd, err := d.getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent response:", err)
			return 1
		}
		if err := d.specChangeReviewResponse(cwd); err != nil {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent response:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(os.Stderr, "4sight spec change review agent: unknown command %q\n", args[0])
	printSpecChangeReviewAgentHelp()
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
		printSpecChangeImportHelp()
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
		printSpecExportHelp()
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
		printSpecSyncDiffHelp()
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
		printSpecSyncCheckHelp()
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
		printMigrateImportHelp()
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
		printSpecImportHelp()
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
		printMigrateUpHelp()
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
  init              Initialize a repository for 4sight
  help              Show this help
  migrate           Apply database schema migrations
  spec              Import and manage specification bodies

Global options:
  -h, --help        Show this help

Run "4sight help <command>" or "<command> --help" for command help.
`)
}

func printInitHelp() {
	fmt.Print(`4sight init

Initialize the current git repository for 4sight.

Options:
  -h, --help        Show this help
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
  change            Record filesystem changes and prompts as a change
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
  list              List changes with their summaries
  request           Save a prompt as a new workspace change
  status            Show the current change pointers
  submit            Submit the current workspace change for implementation
  implement         Mark a submitted change as implemented
  review            Review changes to the specification

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeListHelp() {
	fmt.Print(`4sight spec change list

List specification changes with their IDs and summaries.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeStatusHelp() {
	fmt.Print(`4sight spec change status

Display the current workspace, submitted, and implemented change IDs.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeRequestHelp() {
	fmt.Print(`4sight spec change request <prompt>

Save a prompt in a new change, make it a child of the current workspace
change, and update the workspace pointer.

Prints YAML with the new change id.

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeSubmitHelp() {
	fmt.Print(`4sight spec change submit

Verify on-disk specs match workspace bodies, then advance the submitted
change pointer and submitted body cache to the workspace change.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeImplementHelp() {
	fmt.Print(`4sight spec change implement

Mark the current submitted change as implemented.

Commands:
  agent done        Advance the implemented change pointer

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeImplementationDoneHelp() {
	fmt.Print(`4sight spec change implement agent done

Mark the current submitted change as implemented by advancing the
implemented change pointer and implemented body cache.

This command takes no arguments.

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

func printSpecChangeReviewHelp() {
	fmt.Print("4sight spec change review\n\nCommands:\n  agent             Generate or submit an agent review\n  show [change-id]  Display a saved review\n\n")
}
func printSpecChangeReviewAgentHelp() {
	fmt.Print("4sight spec change review agent\n\nCommands:\n  prompt            Output review instructions\n  response          Read and persist a JSON review from stdin\n\n")
}
func printSpecChangeReviewPromptHelp() {
	fmt.Print("4sight spec change review agent prompt\n\nOutput instructions for reviewing the workspace spec change.\n\n")
}
func printSpecChangeReviewResponseHelp() {
	fmt.Print("4sight spec change review agent response\n\nRead one JSON review from stdin and persist it for the workspace change.\n\n")
}
func printSpecChangeReviewShowHelp() {
	fmt.Print("4sight spec change review show [change-id]\n\nDisplay a saved review for the specified change, or the current workspace change when omitted.\n\n")
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

Compare workspace specification bodies with on-disk spec files in both
directions. Trailing whitespace and empty lines at the end of a file
are ignored.

Prints two sections. The import section shows "4sight spec change import"
when any spec differs, which overwrites the database from disk. The
export section lists "4sight spec export <command...> <file>" for each
workspace body that differs from disk.

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
