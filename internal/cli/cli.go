package cli

import (
	"fmt"
	"os"
	"strings"

	"foursight.dev/4sight/internal/migrate"
)

type deps struct {
	getwd           func() (string, error)
	up              func(cwd string) error
	implementPrompt func(cwd, schemaRel string) error
}

func Run(args []string) int {
	return run(args, deps{
		getwd:           os.Getwd,
		up:              migrate.Up,
		implementPrompt: migrate.ImplementPrompt,
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
	default:
		fmt.Fprintf(os.Stderr, "4sight: unknown help topic %q (try: migrate, migrate up, migrate implement prompt)\n", strings.Join(topic, " "))
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

func printMigrateUpHelp() {
	fmt.Print(`4sight migrate up

Apply pending migrations, starting with 4sight_migration, then every
database named by .4sight/schemas.

This command takes no arguments.

Options:
  -h, --help        Show this help
`)
}
