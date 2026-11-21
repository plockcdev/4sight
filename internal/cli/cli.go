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
	specExport                   func(cwd string, loc []string) error
	specSyncCheck                func(cwd string) error
	specSyncDiff                 func(cwd string) error
	specImplementPrompt          func(cwd string) error
	specChangeImport             func(cwd string) error
	specChangeList               func(cwd string) error
	specHistory                  func(cwd, file string) error
	specChangeRequest            func(cwd, prompt string) error
	specChangeStatus             func(cwd string) error
	specChangeSubmit             func(cwd string) error
	specChangeImplementationDone func(cwd string) error
	specChangeReviewPrompt       func(cwd string) error
	specChangeReviewResponse     func(cwd, changeID string) error
	specChangeReviewShow         func(cwd, changeID string) error
	specChangeReviewList         func(cwd, changeID string) error
	specChangeAuthorPrompt       func(cwd string) error
	specChangeBody               func(cwd, changeID string, locator []string) error
	init                         func(cwd string) error
	upgrade                      func(cwd string) error
	agentMigrate                 func(cwd string) error
	garbageCollect               func(cwd, name string) error
}

func Run(args []string) int {
	getwd := os.Getwd
	var release func()
	defer func() {
		if release != nil {
			release()
		}
	}()
	agentMigration := len(args) >= 6 && strings.Join(args[:6], " ") == "spec change implement agent migrate up"
	if len(args) > 0 && !agentMigration && (args[0] == "spec" || args[0] == "garbage_collect") {
		getwd = func() (string, error) {
			cwd, err := os.Getwd()
			if err == nil && release == nil {
				release, err = initcmd.AcquirePreflight(cwd, false)
			}
			return cwd, err
		}
	}
	return run(args, deps{
		getwd:                        getwd,
		specExport:                   spec.Export,
		specSyncCheck:                spec.SyncCheck,
		specSyncDiff:                 spec.SyncDiff,
		specImplementPrompt:          spec.ImplementPrompt,
		specChangeImport:             spec.ChangeImport,
		specChangeList:               spec.ChangeList,
		specHistory:                  spec.History,
		specChangeRequest:            spec.ChangeRequest,
		specChangeStatus:             spec.ChangeStatus,
		specChangeSubmit:             spec.ChangeSubmit,
		specChangeImplementationDone: spec.ChangeImplementationDone,
		specChangeReviewPrompt:       spec.ChangeReviewAgentPrompt,
		specChangeReviewResponse:     spec.ChangeReviewAgentResponse,
		specChangeReviewShow:         spec.ChangeReviewShow,
		specChangeReviewList:         spec.ChangeReviewList,
		specChangeAuthorPrompt:       spec.ChangeAuthorAgentPrompt,
		specChangeBody:               spec.ChangeBody,
		init:                         initcmd.Run,
		upgrade:                      initcmd.Upgrade,
		agentMigrate:                 initcmd.MigrateSelf,
		garbageCollect:               func(cwd, name string) error { return migrate.GarbageCollect(cwd, name, os.Stdin, os.Stdout) },
	})
}

func run(args []string, d deps) int {
	if code, handled := runUsage(args); handled {
		return code
	}
	if code, handled := runExplicitHelp(args); handled {
		return code
	}
	switch strings.Join(args, " ") {
	case "spec", "spec implement", "spec change", "spec change implement", "spec change implement agent", "spec change implement agent migrate", "spec change review", "spec change review agent", "spec change author", "spec change author agent":
		return runHelpTopic(args)
	}
	if len(args) > 1 && args[len(args)-1] == "help" {
		topic := strings.Join(args[:len(args)-1], " ")
		if helpTopics[topic] && plainUsageCommands[topic] {
			return runHelpTopic(args[:len(args)-1])
		}
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
	case "spec":
		return parseSpec(args[1:], d)
	case "init":
		return parseInit(args[1:], d)
	case "upgrade":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "4sight upgrade: takes no arguments")
			return 1
		}
		cwd, err := d.getwd()
		if err == nil {
			err = d.upgrade(cwd)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "4sight upgrade:", err)
			return 1
		}
		return 0
	case "garbage_collect":
		if len(args) > 2 {
			fmt.Fprintln(os.Stderr, "4sight garbage_collect: expected at most one trash name")
			return 1
		}
		name := ""
		if len(args) == 2 {
			name = args[1]
		}
		cwd, err := d.getwd()
		if err == nil {
			err = d.garbageCollect(cwd, name)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "4sight garbage_collect:", err)
			return 1
		}
		return 0
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
	name := strings.Join(topic, " ")
	if helpTopics[name] && name != "upgrade" && name != "garbage_collect" && name != "spec change implement agent migrate" && name != "spec change implement agent migrate up" {
		defer func() {
			if name != "spec implement prompt" {
				fmt.Print("\nOptions:\n  --usage           Show usage guidance\n")
			}
		}()
	}
	switch name {
	case "upgrade":
		fmt.Print("4sight upgrade\n\nUpgrade internal databases and refresh managed setup.\n\nOptions:\n  -h, --help        Show this help\n  --usage           Show usage guidance\n")
		return 0
	case "garbage_collect":
		fmt.Print("4sight garbage_collect [trash_name]\n\nPermanently delete completed trashed databases after interactive confirmation.\n\nOptions:\n  -h, --help        Show this help\n  --usage           Show usage guidance\n")
		return 0
	case "spec change implement agent migrate":
		fmt.Print("4sight spec change implement agent migrate\n\nCommands:\n  up                Apply prepared migrations\n\nOptions:\n  -h, --help        Show this help\n  --usage           Show usage guidance\n")
		return 0
	case "spec change implement agent migrate up":
		fmt.Print("4sight spec change implement agent migrate up\n\nApply prepared migrations after implementation tests pass.\n\nOptions:\n  -h, --help        Show this help\n  --usage           Show usage guidance\n")
		return 0
	case "init":
		printInitHelp()
		return 0
	case "spec":
		printSpecHelp()
		return 0
	case "spec implement":
		printSpecImplementHelp()
		return 0
	case "spec implement prompt":
		printSpecImplementPromptHelp()
		return 0
	case "spec history":
		printSpecHistoryHelp()
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
	case "spec status":
		printSpecChangeStatusHelp()
		return 0
	case "spec change submit":
		printSpecChangeSubmitHelp()
		return 0
	case "spec change implement", "spec change implement agent":
		printSpecChangeImplementHelp()
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
	case "spec change review list":
		fmt.Print("4sight spec change review list [change-id]\n")
		return 0
	case "spec change review show":
		printSpecChangeReviewShowHelp()
		return 0
	case "spec change author":
		printSpecChangeAuthorHelp()
		return 0
	case "spec change author agent":
		printSpecChangeAuthorAgentHelp()
		return 0
	case "spec change author agent prompt":
		printSpecChangeAuthorAgentPromptHelp()
		return 0
	case "spec body":
		printSpecChangeBodyHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "4sight: unknown help topic %q (try: init, upgrade, garbage_collect, spec)\n", strings.Join(topic, " "))
		return 1
	}
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
	case "history":
		return parseSpecHistory(args[1:], d)
	case "status":
		return parseSpecChangeStatus(args[1:], d)
	case "body":
		return parseSpecChangeBody(args[1:], d)
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

func parseSpecHistory(args []string, d deps) int {
	if len(args) > 1 || (len(args) == 1 && strings.HasPrefix(args[0], "-")) {
		printSpecHistoryHelp()
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec history:", err)
		return 1
	}
	file := ""
	if len(args) == 1 {
		file = args[0]
	}
	if err := d.specHistory(cwd, file); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec history:", err)
		return 1
	}
	return 0
}
func printSpecHistoryHelp() {
	fmt.Print("4sight spec history [file-path]\n\nList all changes or the history of a tracked file.\n\nOptions:\n  -h, --help        Show this help\n")
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
	case "submit":
		return parseSpecChangeSubmit(args[1:], d)
	case "implement":
		return parseSpecChangeImplement(args[1:], d)
	case "review":
		return parseSpecChangeReview(args[1:], d)
	case "author":
		return parseSpecChangeAuthor(args[1:], d)
	}
	if strings.HasPrefix(args[0], "-") {
		printSpecChangeHelp()
		return 1
	}
	printSpecChangeHelp()
	fmt.Fprintf(os.Stderr, "4sight spec change: unknown command %q\n", args[0])
	return 1
}

func parseSpecChangeBody(args []string, d deps) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "-h" || args[0] == "--help")) {
		printSpecChangeBodyHelp()
		return 0
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			printSpecChangeBodyHelp()
			return 1
		}
	}
	if len(args) < 2 {
		printSpecChangeBodyHelp()
		fmt.Fprintln(os.Stderr, "4sight spec body: requires a change ID and spec file locator")
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec body:", err)
		return 1
	}
	if err := d.specChangeBody(cwd, args[0], args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec body:", err)
		return 1
	}
	return 0
}

func parseSpecChangeAuthor(args []string, d deps) int {
	if len(args) == 0 {
		printSpecChangeAuthorHelp()
		return 0
	}
	if args[0] == "-h" || args[0] == "--help" {
		printSpecChangeAuthorHelp()
		return 0
	}
	if args[0] != "agent" {
		printSpecChangeAuthorHelp()
		fmt.Fprintf(os.Stderr, "4sight spec change author: unknown command %q\n", args[0])
		return 1
	}
	if len(args) == 1 {
		printSpecChangeAuthorAgentHelp()
		return 0
	}
	if args[1] == "-h" || args[1] == "--help" {
		printSpecChangeAuthorAgentHelp()
		if len(args) == 2 {
			return 0
		}
		return 1
	}
	if args[1] != "prompt" {
		printSpecChangeAuthorAgentHelp()
		fmt.Fprintf(os.Stderr, "4sight spec change author agent: unknown command %q\n", args[1])
		return 1
	}
	if len(args) != 2 {
		printSpecChangeAuthorAgentPromptHelp()
		if len(args) == 3 && (args[2] == "-h" || args[2] == "--help") {
			return 0
		}
		return 1
	}
	cwd, err := d.getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change author agent prompt:", err)
		return 1
	}
	if err := d.specChangeAuthorPrompt(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec change author agent prompt:", err)
		return 1
	}
	return 0
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
		fmt.Fprintln(os.Stderr, "4sight spec status:", err)
		return 1
	}
	if err := d.specChangeStatus(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "4sight spec status:", err)
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
		if len(args) == 2 && args[1] == "migrate" {
			return runHelpTopic([]string{"spec", "change", "implement", "agent", "migrate"})
		}
		if len(args) == 3 && args[1] == "migrate" && args[2] == "up" {
			cwd, err := d.getwd()
			if err == nil {
				err = d.agentMigrate(cwd)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "4sight spec change implement agent migrate up:", err)
				return 1
			}
			return 0
		}
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
	case "list":
		if len(args) > 2 {
			fmt.Fprintln(os.Stderr, "too many arguments")
			return 1
		}
		cwd, err := d.getwd()
		if err != nil {
			return 1
		}
		id := ""
		if len(args) == 2 {
			id = args[1]
		}
		if err := d.specChangeReviewList(cwd, id); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "show":
		if len(args) > 3 {
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
		if len(args) >= 2 {
			changeID = strings.Join(args[1:], " ")
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
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent response: requires one change-id argument")
			return 1
		}
		cwd, err := d.getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "4sight spec change review agent response:", err)
			return 1
		}
		if err := d.specChangeReviewResponse(cwd, args[1]); err != nil {
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

func printMainHelp() {
	fmt.Print(`4sight — project tooling

Commands:
  init              Initialize a repository for 4sight
  upgrade           Upgrade internal databases
  garbage_collect   Permanently delete completed trashed databases
  help              Show this help
  usage             Show practical usage guidance
  spec              Import and manage specification bodies

Global options:
  -h, --help        Show this help
  --usage           Show usage guidance for the current command

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

func printSpecHelp() {
	fmt.Print(`4sight spec

Import, export, and check specification bodies in the spec database.

Commands:
  body              Render a spec body at a change
  history           List changes or a file history
  status            Show the current change pointers
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

Output instructions for implementing submitted code and schema specifications,
preparing and testing migrations, and applying them after verification.

This command takes no arguments.

Options:
  -h, --help        Show this help
  --usage           Show usage guidance
`)
}

func printSpecChangeHelp() {
	fmt.Print(`4sight spec change

Record specification filesystem changes in the spec database.

Commands:
  import            Persist filesystem spec deltas as a workspace change
  list              List changes with their summaries
  request           Save a prompt as a new workspace change
  submit            Submit the current workspace change for implementation
  implement         Mark a submitted change as implemented
  review            Review changes to the specification
  author            Prepare prompts for authoring spec changes

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeAuthorHelp() {
	fmt.Print("4sight spec change author\n\nGenerate agent instructions for authoring spec changes.\n\nCommands:\n  agent             Prepare spec-authoring agent prompts\n\nOptions:\n  -h, --help        Show this help\n")
}

func printSpecChangeAuthorAgentHelp() {
	fmt.Print("4sight spec change author agent\n\nPrepare agent prompts for spec authoring.\n\nCommands:\n  prompt            Output the authoring instructions\n\nOptions:\n  -h, --help        Show this help\n")
}

func printSpecChangeAuthorAgentPromptHelp() {
	fmt.Print("4sight spec change author agent prompt\n\nOutput instructions for authoring spec changes from the saved request.\n\nThis command takes no arguments.\n\nOptions:\n  -h, --help        Show this help\n")
}

func printSpecChangeBodyHelp() {
	fmt.Print("4sight spec body <change-id> <spec file locator>\n\nRender a specification body at the given change. The locator is the command path followed by the filename, such as `mysubcommand design.md` or `main cmd.md`.\n\nOptions:\n  -h, --help        Show this help\n")
}

func printSpecChangeListHelp() {
	fmt.Print(`4sight spec change list

List active specification changes with their IDs and summaries.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}

func printSpecChangeStatusHelp() {
	fmt.Print(`4sight spec status

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
  agent migrate up  Apply prepared database migrations

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
	fmt.Print("4sight spec change review\n\nCommands:\n  agent             Generate or submit an agent review\n  list [change-id]  List reviews\n  show [change-id] [attempt]  Display a saved review\n\n")
}
func printSpecChangeReviewAgentHelp() {
	fmt.Print("4sight spec change review agent\n\nCommands:\n  prompt            Output review instructions\n  response <change-id>  Read and persist a JSON review from stdin for the latest workspace change\n\n")
}
func printSpecChangeReviewPromptHelp() {
	fmt.Print("4sight spec change review agent prompt\n\nOutput instructions for reviewing the workspace spec change.\n\n")
}
func printSpecChangeReviewResponseHelp() {
	fmt.Print("4sight spec change review agent response <change-id>\n\nRead one JSON review from stdin and persist it for the latest workspace change.\n\n")
}
func printSpecChangeReviewShowHelp() {
	fmt.Print("4sight spec change review show [change-id] [attempt]\n\nDisplay a saved review for the specified change, or the current workspace change when omitted.\n\n")
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
