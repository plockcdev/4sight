package init

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoltInitDiagnostics(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, script, want string }{
		{"missing", "", "install Dolt"},
		{"identity", "echo 'Author identity unknown: fatal: empty ident name not allowed' >&2; exit 1", "configure Dolt user.name and user.email"},
		{"other", "echo 'permission denied' >&2; exit 1", "permission denied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			cmd := exec.Command(git, "init", root)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git init: %s: %v", out, err)
			}
			bin := t.TempDir()
			if err := os.Symlink(git, filepath.Join(bin, "git")); err != nil {
				t.Fatal(err)
			}
			if tc.script != "" {
				if err := os.WriteFile(filepath.Join(bin, "dolt"), []byte("#!/bin/sh\n"+tc.script+"\n"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin)
			err := run(root, embedded, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v; want %q", err, tc.want)
			}
			if tc.name != "missing" && strings.Contains(err.Error(), "install Dolt") {
				t.Fatalf("misdiagnosed: %v", err)
			}
		})
	}
}
