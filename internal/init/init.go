package init

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed assets
var embedded embed.FS

// Run initializes a repository from the current working directory.
func Run(cwd string) error { return run(cwd, embedded, os.Stdout) }

func run(cwd string, source fs.FS, stdout interface{ Write([]byte) (int, error) }) error {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}
	git := exec.Command("git", "rev-parse", "--show-toplevel")
	git.Dir = cwd
	out, err := git.Output()
	if err != nil {
		return fmt.Errorf("%s is not inside a git repository; initialize one with `git init %s` and run 4sight from its root", abs, abs)
	}
	root, err := filepath.Abs(strings.TrimSpace(string(out)))
	if err != nil {
		return fmt.Errorf("resolve git root: %w", err)
	}
	if root != abs {
		return fmt.Errorf("run 4sight init from the git repository root %s (current directory is %s)", root, abs)
	}

	metadata := filepath.Join(root, ".4sight")
	if err := os.MkdirAll(metadata, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", metadata, err)
	}
	database := filepath.Join(metadata, "db")
	if err := os.MkdirAll(database, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", database, err)
	}
	if _, err := os.Stat(filepath.Join(database, ".dolt")); os.IsNotExist(err) {
		cmd := exec.Command("dolt", "init")
		cmd.Dir = database
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("initialize Dolt in %s: %s; install Dolt and rerun 4sight init", database, strings.TrimSpace(string(output)))
		}
	} else if err != nil {
		return fmt.Errorf("check Dolt repository %s: %w", database, err)
	}

	rootAsset, err := fs.ReadFile(source, "assets/root_AGENTS.md")
	if err != nil {
		return fmt.Errorf("read embedded root instructions: %w", err)
	}
	oldVersions, err := fs.Glob(source, "assets/history/root_AGENTS_*.md")
	if err != nil {
		return fmt.Errorf("list historical root instructions: %w", err)
	}
	var historical [][]byte
	for _, path := range oldVersions {
		body, err := fs.ReadFile(source, path)
		if err != nil {
			return fmt.Errorf("read historical root instructions %s: %w", path, err)
		}
		historical = append(historical, body)
	}
	if err := rewriteRootInstructions(filepath.Join(root, "AGENTS.md"), string(rootAsset), historical); err != nil {
		return err
	}
	for _, item := range []struct {
		name, target string
		preserve     bool
	}{
		{"4sight_AGENTS.md", filepath.Join(metadata, "AGENTS.md"), false},
		{"coding.md", filepath.Join(metadata, "coding.md"), true},
	} {
		if item.preserve {
			if _, err := os.Stat(item.target); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("check %s: %w", item.target, err)
			}
		}
		body, err := fs.ReadFile(source, "assets/"+item.name)
		if err != nil {
			return fmt.Errorf("read embedded asset %s: %w", item.name, err)
		}
		if err := os.WriteFile(item.target, body, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", item.target, err)
		}
	}
	_, err = fmt.Fprintf(stdout, "initialized 4sight in %s\n", root)
	return err
}

func rewriteRootInstructions(path, current string, historical [][]byte) error {
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(old), "\r\n", "\n"), "\n")
	versions := append(historical, []byte(current))
	var blocks [][]string
	for _, version := range versions {
		blocks = append(blocks, normalizedLines(string(version)))
	}
	var kept []string
	var localAdditions []string
	for i := 0; i < len(lines); {
		matched := false
		matchLen := 0
		for _, block := range blocks {
			if len(block) == 0 || i+len(block) > len(lines) {
				continue
			}
			ok := true
			for j, want := range block {
				if !sameManagedLine(lines[i+j], want) {
					ok = false
					break
				}
			}
			if ok {
				matched, matchLen = true, len(block)
				break
			}
		}
		if matched {
			for _, block := range blocks {
				if len(block) != matchLen {
					continue
				}
				for j, want := range block {
					actual := strings.TrimSpace(lines[i+j])
					if actual == want {
						continue
					}
					extra := strings.TrimSpace(strings.TrimPrefix(actual, strings.TrimRight(want, ".")))
					extra = strings.TrimLeft(extra, ",;: ")
					if extra != "" {
						localAdditions = append(localAdditions, extra)
					}
				}
				break
			}
			i += matchLen
			continue
		}
		kept = append(kept, lines[i])
		i++
	}
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}
	if len(localAdditions) > 0 {
		kept = append(kept, localAdditions...)
	}
	if len(kept) > 0 {
		kept = append(kept, "")
	}
	kept = append(kept, normalizedLines(current)...)
	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func sameManagedLine(actual, expected string) bool {
	actual = strings.TrimSpace(actual)
	if actual == expected {
		return true
	}
	return strings.HasPrefix(actual, strings.TrimRight(expected, ".")+",")
}

func normalizedLines(s string) []string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return lines
}
