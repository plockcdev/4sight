package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	stdout, stderr := capture(t, func() int {
		return run([]string{"--help"}, deps{})
	})
	if !strings.Contains(stdout, "migrate") {
		t.Fatalf("help missing migrate: %s", stdout)
	}
	if !strings.Contains(stdout, "spec") {
		t.Fatalf("help missing spec: %s", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	_, stderr := capture(t, func() int {
		return run([]string{"nope"}, deps{})
	})
	if !strings.Contains(stderr, `unknown command "nope"`) {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestMigrateUpRejectsArgs(t *testing.T) {
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"migrate", "up", "spec"}, deps{})
	})
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr, "does not take arguments") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestMigrateUpDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"migrate", "up"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			up: func(cwd string) error {
				called = cwd
				return nil
			},
		})
	})
	if code != 0 {
		t.Fatalf("code %d stderr %q", code, stderr)
	}
	if called != "/tmp/repo" {
		t.Fatalf("cwd %q", called)
	}
}

func capture(t *testing.T, fn func() int) (string, string) {
	t.Helper()
	code, stdout, stderr := captureCode(t, fn)
	if code != 0 && stdout == "" && stderr == "" {
		t.Fatalf("exit %d with no output", code)
	}
	return stdout, stderr
}

func captureCode(t *testing.T, fn func() int) (int, string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW
	code := fn()
	_ = outW.Close()
	_ = errW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	stdout, err := io.ReadAll(outR)
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := io.ReadAll(errR)
	if err != nil {
		t.Fatal(err)
	}
	return code, string(stdout), string(stderr)
}

func TestHelpTopicMigrateUp(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "migrate", "up"}, deps{})
	})
	if !strings.Contains(stdout, "4sight migrate up") {
		t.Fatalf("help %q", stdout)
	}
}

func TestHelpTopicMigrateImplementPrompt(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "migrate", "implement", "prompt"}, deps{})
	})
	if !strings.Contains(stdout, "4sight migrate implement prompt") {
		t.Fatalf("help %q", stdout)
	}
}

func TestMigrateImplementPromptRequiresArg(t *testing.T) {
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"migrate", "implement", "prompt"}, deps{})
	})
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr, "requires exactly one schema spec path") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestHelpTopicSpecImport(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "spec", "import"}, deps{})
	})
	if !strings.Contains(stdout, "4sight spec import") {
		t.Fatalf("help %q", stdout)
	}
	if !strings.Contains(stdout, "myprog foo bar cmd.md") {
		t.Fatalf("help missing example: %q", stdout)
	}
}

func TestSpecImportRequiresArg(t *testing.T) {
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "import"}, deps{})
	})
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr, "requires a spec file location") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestHelpTopicSpecImplementPrompt(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "spec", "implement", "prompt"}, deps{})
	})
	if !strings.Contains(stdout, "4sight spec implement prompt") {
		t.Fatalf("help %q", stdout)
	}
}

func TestSpecImplementPromptRejectsArgs(t *testing.T) {
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "implement", "prompt", "extra"}, deps{})
	})
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr, "does not take arguments") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestSpecImplementPromptDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "implement", "prompt"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			specImplementPrompt: func(cwd string) error {
				called = cwd
				return nil
			},
		})
	})
	if code != 0 {
		t.Fatalf("code %d stderr %q", code, stderr)
	}
	if called != "/tmp/repo" {
		t.Fatalf("called %q", called)
	}
}

func TestSpecImportDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "import", "myprog", "foo", "bar", "cmd.md"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			specImport: func(cwd string, loc []string) error {
				called = cwd + " " + strings.Join(loc, " ")
				return nil
			},
		})
	})
	if code != 0 {
		t.Fatalf("code %d stderr %q", code, stderr)
	}
	if called != "/tmp/repo myprog foo bar cmd.md" {
		t.Fatalf("called %q", called)
	}
}

func TestMigrateImplementPromptDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"migrate", "implement", "prompt", "dolt_spec.md"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			implementPrompt: func(cwd, schemaRel string) error {
				called = cwd + " " + schemaRel
				return nil
			},
		})
	})
	if code != 0 {
		t.Fatalf("code %d stderr %q", code, stderr)
	}
	if called != "/tmp/repo dolt_spec.md" {
		t.Fatalf("called %q", called)
	}
}
