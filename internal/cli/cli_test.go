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
	if !strings.Contains(stdout, "upgrade") {
		t.Fatalf("help missing upgrade: %s", stdout)
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

func TestHelpTopicSpecExport(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "spec", "export"}, deps{})
	})
	if !strings.Contains(stdout, "4sight spec export") {
		t.Fatalf("help %q", stdout)
	}
}

func TestHelpTopicSpecSyncDiff(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "spec", "sync-diff"}, deps{})
	})
	if !strings.Contains(stdout, "4sight spec sync-diff") {
		t.Fatalf("help %q", stdout)
	}
}

func TestHelpTopicSpecSyncCheck(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "spec", "sync-check"}, deps{})
	})
	if !strings.Contains(stdout, "4sight spec sync-check") {
		t.Fatalf("help %q", stdout)
	}
}

func TestSpecExportRequiresArg(t *testing.T) {
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "export"}, deps{})
	})
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr, "requires a spec file location") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestSpecExportDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "export", "spec", "implement", "prompt", "cmd.md"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			specExport: func(cwd string, loc []string) error {
				called = cwd + " " + strings.Join(loc, " ")
				return nil
			},
		})
	})
	if code != 0 {
		t.Fatalf("code %d stderr %q", code, stderr)
	}
	if called != "/tmp/repo spec implement prompt cmd.md" {
		t.Fatalf("called %q", called)
	}
}

func TestSpecSyncDiffDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "sync-diff"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			specSyncDiff: func(cwd string) error {
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

func TestSpecSyncCheckDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "sync-check"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			specSyncCheck: func(cwd string) error {
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

func TestHelpTopicSpecChangeImport(t *testing.T) {
	stdout, _ := capture(t, func() int {
		return run([]string{"help", "spec", "change", "import"}, deps{})
	})
	if !strings.Contains(stdout, "4sight spec change import") {
		t.Fatalf("help %q", stdout)
	}
	if !strings.Contains(stdout, "no changes found") {
		t.Fatalf("help missing empty case: %q", stdout)
	}
}

func TestSpecChangeImportRejectsArgs(t *testing.T) {
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "change", "import", "extra"}, deps{})
	})
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr, "does not take arguments") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestSpecChangeImportDispatches(t *testing.T) {
	called := ""
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "change", "import"}, deps{
			getwd: func() (string, error) { return "/tmp/repo", nil },
			specChangeImport: func(cwd string) error {
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

func TestSpecChangeUnknownCommand(t *testing.T) {
	code, _, stderr := captureCode(t, func() int {
		return run([]string{"spec", "change", "nope"}, deps{})
	})
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr, `unknown command "nope"`) {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestNewCommandHelpListsUsage(t *testing.T) {
	for _, args := range [][]string{{"upgrade", "--help"}, {"garbage_collect", "--help"}, {"spec", "change", "implement", "agent", "migrate", "up", "--help"}} {
		t.Run(strings.Join(args, "/"), func(t *testing.T) {
			code, stdout, _ := captureCode(t, func() int { return run(args, deps{}) })
			if code != 0 || strings.Contains(stdout, "  usage ") || !strings.Contains(stdout, "--usage") {
				t.Fatalf("help code %d: %s", code, stdout)
			}
		})
	}
}

func TestAllCommandHelpTopicsExposeUsage(t *testing.T) {
	for topic := range helpTopics {
		if topic == "" {
			continue
		}
		t.Run(strings.ReplaceAll(topic, " ", "/"), func(t *testing.T) {
			args := append(strings.Fields(topic), "--help")
			code, stdout, stderr := captureCode(t, func() int { return run(args, deps{}) })
			if code != 0 || !strings.Contains(stdout, "--usage") {
				t.Fatalf("help code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if strings.Contains(stdout, "  usage ") {
				t.Fatalf("nested help lists usage command: %q", stdout)
			}
		})
	}
}

func TestUsageRouting(t *testing.T) {
	for topic := range helpTopics {
		t.Run(strings.ReplaceAll("flag/"+topic, " ", "/"), func(t *testing.T) {
			args := append(strings.Fields(topic), "--usage")
			code, stdout, stderr := captureCode(t, func() int { return run(args, deps{}) })
			if code != 0 || stdout == "" {
				t.Fatalf("usage code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
		})
	}
	code, command, _ := captureCode(t, func() int { return run([]string{"usage"}, deps{}) })
	flagCode, flag, _ := captureCode(t, func() int { return run([]string{"--usage"}, deps{}) })
	if code != 0 || flagCode != 0 || command != flag {
		t.Fatalf("top-level usage differs: command=%d flag=%d", code, flagCode)
	}
	if _, handled := runUsage([]string{"spec", "usage"}); handled {
		t.Fatal("nested usage was treated as a usage command")
	}
	code, _, _ = captureCode(t, func() int { return run([]string{"spec", "--usage", "extra"}, deps{}) })
	if code == 0 {
		t.Fatal("usage with an extra argument succeeded")
	}
}

func TestParentHelpListsUsage(t *testing.T) {
	for _, topic := range []string{"spec", "spec implement", "spec change", "spec change implement", "spec change implement agent", "spec change implement agent migrate", "spec change review", "spec change review agent", "spec change author", "spec change author agent"} {
		t.Run(strings.ReplaceAll(topic, " ", "/"), func(t *testing.T) {
			code, stdout, _ := captureCode(t, func() int { return run(strings.Fields(topic), deps{}) })
			if code != 0 || strings.Contains(stdout, "  usage ") || !strings.Contains(stdout, "--usage") {
				t.Fatalf("help code %d: %s", code, stdout)
			}
		})
	}
}
