package cli

import (
	"embed"
	"fmt"
	"os"
	"strings"
)

//go:embed usage_docs/*.txt
var usageDocs embed.FS

var helpTopics = map[string]bool{
	"upgrade": true, "garbage_collect": true, "spec change implement agent migrate": true, "spec change implement agent migrate up": true,
	"": true, "init": true,

	"spec": true, "spec history": true, "spec export": true, "spec sync-check": true,
	"spec sync-diff": true, "spec implement": true, "spec implement prompt": true,
	"spec change": true, "spec change import": true, "spec change list": true,
	"spec change request": true, "spec status": true, "spec change submit": true,
	"spec change implement": true, "spec change implement agent": true,
	"spec change implement agent done": true, "spec change review": true,
	"spec change review agent": true, "spec change review agent prompt": true,
	"spec change review agent response": true, "spec change review show": true, "spec change review list": true,
	"spec change author": true, "spec change author agent": true,
	"spec change author agent prompt": true,
	"spec body":                       true,
}

var usageCommands = map[string]bool{
	"upgrade": true, "garbage_collect": true, "spec change implement agent migrate": true, "spec change implement agent migrate up": true,
	"": true, "init": true,

	"spec": true, "spec history": true, "spec export": true, "spec sync-check": true,
	"spec sync-diff": true, "spec implement": true, "spec implement prompt": true,
	"spec change": true, "spec change import": true, "spec change list": true,
	"spec change request": true, "spec status": true, "spec change submit": true,
	"spec change implement": true, "spec change implement agent": true,
	"spec change implement agent done": true, "spec change review": true,
	"spec change review agent": true, "spec change review agent prompt": true,
	"spec change review agent response": true, "spec change review show": true, "spec change review list": true,
	"spec change author": true, "spec change author agent": true,
	"spec change author agent prompt": true,
	"spec body":                       true,
}

var plainUsageCommands = map[string]bool{
	"upgrade": true, "garbage_collect": true, "spec change implement agent migrate": true, "spec change implement agent migrate up": true,
	"": true, "init": true,
	"spec": true, "spec history": true, "spec status": true, "spec sync-check": true, "spec sync-diff": true,
	"spec implement": true, "spec implement prompt": true, "spec change": true,
	"spec change import": true, "spec change list": true,
	"spec change submit": true, "spec change implement": true,
	"spec change implement agent": true, "spec change implement agent done": true,
	"spec change review": true, "spec change review agent": true,
	"spec change review agent prompt": true, "spec change review agent response": true,
	"spec change review show": true, "spec change review list": true,
	"spec change author": true, "spec change author agent": true,
	"spec change author agent prompt": true, "spec body": true,
}

func runExplicitHelp(args []string) (int, bool) {
	for i, arg := range args {
		if arg != "-h" && arg != "--help" {
			continue
		}
		prefix := args[:i]
		printNearestHelp(prefix)
		if i != len(args)-1 {
			return 1, true
		}
		for n := len(prefix); n >= 0; n-- {
			if helpTopics[strings.Join(prefix[:n], " ")] {
				if n == len(prefix) {
					return 0, true
				}
				return 1, true
			}
		}
		return 1, true
	}
	return 0, false
}

func runUsage(args []string) (int, bool) {
	for i, arg := range args {
		if arg != "--usage" && arg != "usage" {
			continue
		}
		prefix := strings.Join(args[:i], " ")
		if arg == "usage" && i != 0 {
			continue
		}
		if !usageCommands[prefix] {
			continue
		}
		if i != len(args)-1 {
			printNearestHelp(args[:i])
			return 1, true
		}
		name := "USAGE"
		if prefix != "" {
			name += "_" + strings.ReplaceAll(prefix, " ", "_")
		}
		path := "usage_docs/" + name + ".txt"
		body, err := usageDocs.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "4sight: embedded usage file %s: %v\n", path, err)
			return 1, true
		}
		if _, err := fmt.Print(string(body)); err != nil {
			fmt.Fprintln(os.Stderr, "4sight: write usage:", err)
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func printNearestHelp(path []string) {
	for len(path) > 0 {
		if helpTopics[strings.Join(path, " ")] {
			runHelpTopic(path)
			return
		}
		path = path[:len(path)-1]
	}
	printMainHelp()
}
