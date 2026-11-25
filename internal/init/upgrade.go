package init

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	intentpkg "foursight.dev/4sight/internal/intent"
	"foursight.dev/4sight/internal/migrate"
	"foursight.dev/4sight/internal/spec"
)

// Upgrade updates only the embedded internal databases of an existing project.
func Upgrade(cwd string) error {
	return applyInternal(cwd, false)
}

// MigrateSelf applies the prepared source chains after isolated verification.
func MigrateSelf(cwd string) error {
	return applyInternal(cwd, true)
}

func applyInternal(cwd string, agent bool) error {
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	gitRoot := exec.Command("git", "rev-parse", "--show-toplevel")
	gitRoot.Dir = root
	if out, err := gitRoot.Output(); err == nil {
		root = strings.TrimSpace(string(out))
	}
	database := filepath.Join(root, ".4sight", "db")
	if info, err := os.Stat(filepath.Join(database, ".dolt")); err != nil || !info.IsDir() {
		return fmt.Errorf("no complete internal installation at %s; run 4sight init only if the location is empty, otherwise restore or reconcile it", database)
	}
	self, err := isSelfProject(root)
	if err != nil {
		if !agent {
			return err
		}
		var answer string
		var inputErr error
		info, statErr := os.Stdin.Stat()
		if statErr != nil {
			return fmt.Errorf("project identity input is unavailable")
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			answer, inputErr = bufio.NewReader(os.Stdin).ReadString('\n')
		} else {
			var input []byte
			input, inputErr = io.ReadAll(io.LimitReader(os.Stdin, 128))
			answer = string(input)
		}
		if inputErr != nil {
			return fmt.Errorf("project identity must be clarified with exactly one line: 4sight or application")
		}
		self, err = parseIdentityAnswer(answer)
		if err != nil {
			return err
		}
	}
	lock, err := os.OpenFile(filepath.Join(database, ".4sight-preflight.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()
	chains, err := validateInternal(database)
	if err != nil {
		return err
	}
	if agent {
		if err := spec.CheckImplementationReady(root); err != nil {
			return err
		}
		if !self {
			for _, chain := range chains {
				if chain.applied != len(chain.migrations) {
					return fmt.Errorf("internal database %s requires an upgrade; run 4sight upgrade", chain.name)
				}
			}
			return migrate.AgentUp(root)
		}
		for _, chain := range chains {
			dir := filepath.Join(root, ".4sight", "migrations", "dolt_"+chain.name)
			entries, err := os.ReadDir(dir)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".delete") {
					return fmt.Errorf("internal database %s cannot have a retirement entry", chain.name)
				}
			}
			localChain, err := readEmbeddedMigrations(os.DirFS(dir), ".")
			if err != nil || len(localChain) != len(chain.migrations) {
				return fmt.Errorf("source chain for %s does not match the embedded chain", chain.name)
			}
			for i := range localChain {
				if localChain[i] != chain.migrations[i] {
					return fmt.Errorf("source chain for %s does not match the embedded chain", chain.name)
				}
			}
			target, err := os.Readlink(filepath.Join(dir, "submitted"))
			if err != nil || target != chain.migrations[len(chain.migrations)-1].name {
				return fmt.Errorf("submitted source chain for %s does not match the embedded chain", chain.name)
			}
			for _, migration := range chain.migrations {
				if err := verifySourceMigration(root, chain.name, migration); err != nil {
					return err
				}
			}
		}
	}
	intent := filepath.Join(database, ".4sight-internal-migration-intent")
	if body, err := os.ReadFile(intent); err == nil {
		fields := strings.Fields(string(body))
		resolved := false
		if len(fields) == 2 {
			for _, chain := range chains {
				if chain.name == fields[0] && chain.applied > 0 && chain.migrations[chain.applied-1].id == fields[1] {
					if self {
						if err := reconcileInternalLink(root, chain, chain.migrations[chain.applied-1]); err != nil {
							return err
						}
					}
					resolved = true
				}
			}
		}
		if !resolved {
			return fmt.Errorf("internal migration intent %s remains after an interrupted operation; reconcile the recorded and actual database state before retrying", intent)
		}
		if err := intentpkg.Remove(intent); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	store := doltBootstrap{dataDir: database}
	changed := false
	for _, chain := range chains {
		if self {
			if err := reconcileInternalLink(root, chain, chain.migrations[chain.applied-1]); err != nil {
				return err
			}
		}
		for _, migration := range chain.migrations[chain.applied:] {
			if self {
				if err := verifySourceMigration(root, chain.name, migration); err != nil {
					return err
				}
			}
			sql, err := embedded.ReadFile("migrations/dolt_" + chain.name + "/" + migration.name)
			if err != nil {
				return err
			}
			if err := intentpkg.Write(intent, []byte(chain.name+" "+migration.id+"\n")); err != nil {
				return err
			}
			if err := store.run("sql", "-q", string(sql)); err != nil {
				return fmt.Errorf("apply %s %s: %w; intent retained for manual reconciliation", chain.name, migration.id, err)
			}
			if err := store.recordApplied(chain.name, migration.id, time.Now().UTC().Truncate(time.Second)); err != nil {
				return fmt.Errorf("record %s %s: %w; intent retained for manual reconciliation", chain.name, migration.id, err)
			}
			if self {
				if err := reconcileInternalLink(root, chain, migration); err != nil {
					return fmt.Errorf("migration recorded but link reconciliation failed: %w", err)
				}
			}
			if err := intentpkg.Remove(intent); err != nil {
				return err
			}
			changed = true
		}
	}
	if agent {
		fmt.Printf("internal migrations changed: %t\n", changed)
		return nil
	}
	managedPaths := []string{filepath.Join(root, ".gitignore"), filepath.Join(root, "AGENTS.md"), filepath.Join(root, ".4sight", "AGENTS.md"), filepath.Join(root, ".4sight", "coding.md"), filepath.Join(root, ".4sight", "coding_suggestions.md")}
	before := make([][]byte, len(managedPaths))
	for i, path := range managedPaths {
		before[i], _ = os.ReadFile(path)
	}
	if err := setupManagedFiles(root, filepath.Join(root, ".4sight"), embedded); err != nil {
		return fmt.Errorf("internal migrations complete; managed files need repair: %w", err)
	}
	managedChanged := false
	for i, path := range managedPaths {
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before[i], after) {
			managedChanged = true
		}
	}
	fmt.Printf("internal migrations changed: %t; managed files changed: %t\n", changed, managedChanged)
	return nil
}

func parseIdentityAnswer(answer string) (bool, error) {
	switch answer {
	case "4sight\n":
		return true, nil
	case "application\n":
		return false, nil
	default:
		return false, fmt.Errorf("project identity must be clarified with exactly one line: 4sight or application")
	}
}

func isSelfProject(root string) (bool, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = root
	out, err := cmd.Output()
	remote := strings.TrimRight(strings.TrimSpace(string(out)), "/")
	if i := strings.LastIndexAny(remote, "/:"); i >= 0 {
		remote = remote[i+1:]
	}
	remote = strings.TrimSuffix(remote, ".git")
	if err != nil || remote == "" {
		return false, fmt.Errorf("project identity is ambiguous; specify whether this is 4sight itself or an application before upgrading")
	}
	localSelf := strings.EqualFold(filepath.Base(root), "4sight")
	remoteSelf := strings.EqualFold(remote, "4sight")
	if localSelf != remoteSelf {
		return false, fmt.Errorf("project identity is ambiguous; specify whether this is 4sight itself or an application before upgrading")
	}
	return localSelf, nil
}

func verifySourceMigration(root, name string, migration embeddedMigration) error {
	path := filepath.Join(root, ".4sight", "migrations", "dolt_"+name, migration.name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("source migration %s is missing or unsafe", path)
	}
	local, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	expected, err := embedded.ReadFile("migrations/dolt_" + name + "/" + migration.name)
	if err != nil {
		return err
	}
	if !bytes.Equal(local, expected) {
		return fmt.Errorf("source migration %s differs from embedded migration", path)
	}
	return nil
}

func reconcileInternalLink(root string, chain internalChain, migration embeddedMigration) error {
	dir := filepath.Join(root, ".4sight", "migrations", "dolt_"+chain.name)
	link := filepath.Join(dir, "current")
	info, err := os.Lstat(link)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("unsafe current link %s", link)
	}
	target, err := os.Readlink(link)
	if err != nil {
		return err
	}
	valid := false
	currentIndex, appliedIndex := -1, -1
	for i, item := range chain.migrations {
		if item.name == migration.name {
			appliedIndex = i
		}
		if item.name == target {
			if err := verifySourceMigration(root, chain.name, item); err != nil {
				return err
			}
			valid = true
			currentIndex = i
		}
	}
	if !valid || appliedIndex < 0 || currentIndex > appliedIndex {
		return fmt.Errorf("unsafe current link %s -> %s", link, target)
	}
	if err := verifySourceMigration(root, chain.name, migration); err != nil {
		return err
	}
	if target == migration.name {
		return nil
	}
	tmp := link + ".4sight-new"
	if err := os.Symlink(migration.name, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
