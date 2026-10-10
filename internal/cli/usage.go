package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var helpTopics = map[string]bool{
	"": true, "init": true, "migrate": true, "migrate up": true,
	"migrate import": true, "migrate implement": true, "migrate implement prompt": true,
	"spec": true, "spec import": true, "spec export": true, "spec sync-check": true,
	"spec sync-diff": true, "spec implement": true, "spec implement prompt": true,
	"spec change": true, "spec change import": true, "spec change list": true,
	"spec change request": true, "spec change status": true, "spec change submit": true,
	"spec change implement": true, "spec change implement agent": true,
	"spec change implement agent done": true, "spec change review": true,
	"spec change review agent": true, "spec change review agent prompt": true,
	"spec change review agent response": true, "spec change review show": true,
}

var usageCommands = map[string]bool{
	"": true, "init": true, "migrate": true, "migrate up": true,
	"migrate import": true, "migrate implement": true, "migrate implement prompt": true,
	"spec": true, "spec import": true, "spec export": true, "spec sync-check": true,
	"spec sync-diff": true, "spec implement": true, "spec implement prompt": true,
	"spec change": true, "spec change import": true, "spec change list": true,
	"spec change request": true, "spec change status": true, "spec change submit": true,
	"spec change implement": true, "spec change implement agent": true,
	"spec change implement agent done": true, "spec change review": true,
	"spec change review agent": true, "spec change review agent prompt": true,
	"spec change review agent response": true, "spec change review show": true,
}

var plainUsageCommands = map[string]bool{
	"": true, "init": true, "migrate": true, "migrate up": true, "migrate implement": true,
	"spec": true, "spec sync-check": true, "spec sync-diff": true,
	"spec implement": true, "spec implement prompt": true, "spec change": true,
	"spec change import": true, "spec change list": true, "spec change status": true,
	"spec change submit": true, "spec change implement": true,
	"spec change implement agent": true, "spec change implement agent done": true,
	"spec change review": true, "spec change review agent": true,
	"spec change review agent prompt": true, "spec change review agent response": true,
	"spec change review show": true,
}

func runExplicitHelp(args []string) (int, bool) {
	for i, arg := range args {
		if arg != "-h" && arg != "--help" {
			continue
		}
		printNearestHelp(args[:i])
		if i == len(args)-1 {
			return 0, true
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
		if arg == "usage" && !plainUsageCommands[prefix] {
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
		path := filepath.Join("docs", name+".txt")
		body, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "4sight: usage file %s: %v\n", path, err)
			return 1, true
		}
		if _, err := os.Stdout.Write(body); err != nil {
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
