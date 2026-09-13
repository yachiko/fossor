package manageview

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yachiko/fossor/internal/git"
)

func TestAllActions(t *testing.T) {
	actions := AllActions()

	t.Run("returns 19 actions", func(t *testing.T) {
		if len(actions) != 19 {
			t.Errorf("expected 19 actions, got %d", len(actions))
		}
	})

	t.Run("no duplicate keys", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, a := range actions {
			if seen[a.Key] {
				t.Errorf("duplicate key: %q", a.Key)
			}
			seen[a.Key] = true
		}
	})

	t.Run("all have non-empty Name, Category label, and Key", func(t *testing.T) {
		for _, a := range actions {
			if a.Name == "" {
				t.Errorf("action with key %q has empty Name", a.Key)
			}
			if a.Key == "" {
				t.Errorf("action %q has empty Key", a.Name)
			}
			if a.Category.String() == "" {
				t.Errorf("action %q has empty Category string", a.Key)
			}
		}
	})

	t.Run("BuildCmd non-nil for all except commit inline (key c)", func(t *testing.T) {
		for _, a := range actions {
			if a.Key == "c" {
				if a.BuildCmd != nil {
					t.Errorf("action 'c' (commit) should have nil BuildCmd")
				}
			} else {
				if a.BuildCmd == nil {
					t.Errorf("action %q (%s) has nil BuildCmd", a.Key, a.Name)
				}
			}
		}
	})

	t.Run("Enabled functions do not panic with zero-value RepoInfo", func(t *testing.T) {
		zero := git.RepoInfo{}
		for _, a := range actions {
			// Should not panic
			a.Enabled(zero)
		}
	})

	t.Run("each of 4 categories has at least one action", func(t *testing.T) {
		cats := make(map[Category]int)
		for _, a := range actions {
			cats[a.Category]++
		}
		for _, cat := range AllCategories() {
			if cats[cat] == 0 {
				t.Errorf("category %q has no actions", cat.String())
			}
		}
	})

	t.Run("pull (key p) is always enabled", func(t *testing.T) {
		var pullAction Action
		for _, a := range actions {
			if a.Key == "p" {
				pullAction = a
				break
			}
		}
		if !pullAction.Enabled(git.RepoInfo{}) {
			t.Error("pull should be enabled with zero-value RepoInfo")
		}
		if !pullAction.Enabled(git.RepoInfo{Ahead: 5, Behind: 3, Changes: 10}) {
			t.Error("pull should always be enabled")
		}
	})

	t.Run("push (key u) disabled when Ahead==0, enabled when Ahead>0", func(t *testing.T) {
		var pushAction Action
		for _, a := range actions {
			if a.Key == "u" {
				pushAction = a
				break
			}
		}
		if pushAction.Enabled(git.RepoInfo{Ahead: 0}) {
			t.Error("push should be disabled when Ahead==0")
		}
		if !pushAction.Enabled(git.RepoInfo{Ahead: 1}) {
			t.Error("push should be enabled when Ahead>0")
		}
	})

	t.Run("commit (key c) disabled when Changes==0, enabled when Changes>0", func(t *testing.T) {
		var commitAction Action
		for _, a := range actions {
			if a.Key == "c" {
				commitAction = a
				break
			}
		}
		if commitAction.Enabled(git.RepoInfo{Changes: 0}) {
			t.Error("commit should be disabled when Changes==0")
		}
		if !commitAction.Enabled(git.RepoInfo{Changes: 1}) {
			t.Error("commit should be enabled when Changes>0")
		}
	})
}

// TestRebaseArgInjection asserts that a poisoned DefaultBranch value cannot
// be interpreted by git as a flag. Without the `--` separator inserted by
// gitRefCmd, `git rebase --exec=<cmd>` runs the command for every replayed
// commit on a tracking branch — silent RCE on key `b`.
func TestRebaseArgInjection(t *testing.T) {
	cases := []string{
		"--exec=echo PWNED",
		"--exec=touch /tmp/pwn",
		"-i",
		"--onto",
		"--root",
	}
	for _, name := range cases {
		t.Run("DefaultBranch="+name, func(t *testing.T) {
			actions := AllActions()
			var rebase Action
			for _, a := range actions {
				if a.Key == "b" {
					rebase = a
					break
				}
			}
			if rebase.BuildCmd == nil {
				t.Fatal("rebase action not found")
			}
			cmd := rebase.BuildCmd(git.RepoInfo{Path: "/tmp/x", Branch: "feat", DefaultBranch: name}, "")
			if cmd == nil {
				t.Fatal("BuildCmd returned nil")
			}

			// Locate the poisoned value's index and the `--` separator.
			args := cmd.Args
			poisonIdx, sepIdx := -1, -1
			for i, a := range args {
				if a == "--" && sepIdx == -1 {
					sepIdx = i
				}
				if a == name {
					poisonIdx = i
				}
			}
			if sepIdx == -1 {
				t.Fatalf("rebase cmd lacks `--` separator: %v", args)
			}
			if poisonIdx == -1 {
				t.Fatalf("rebase cmd missing DefaultBranch arg: %v", args)
			}
			if poisonIdx < sepIdx {
				t.Errorf("DefaultBranch passed before `--`, would be parsed as flag: %v", args)
			}
		})
	}
}

// TestRefArgSeparator asserts every action whose BuildCmd takes user/repo
// input passes that input AFTER a `--` separator.
func TestRefArgSeparator(t *testing.T) {
	type refCase struct {
		key   string
		repo  git.RepoInfo
		input string
	}
	cases := []refCase{
		{"d", git.RepoInfo{Path: "/p", Branch: "feat", DefaultBranch: "--evil"}, ""},
		{"m", git.RepoInfo{Path: "/p", Branch: "feat", DefaultBranch: "main"}, "--evil"},
		{"b", git.RepoInfo{Path: "/p", Branch: "feat", DefaultBranch: "--evil"}, ""},
		{"B", git.RepoInfo{Path: "/p", Branch: "feat", DefaultBranch: "--evil"}, ""},
		{"k", git.RepoInfo{Path: "/p"}, "--evil"},
	}
	actions := AllActions()
	byKey := make(map[string]Action, len(actions))
	for _, a := range actions {
		byKey[a.Key] = a
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			a, ok := byKey[c.key]
			if !ok || a.BuildCmd == nil {
				t.Skip("action not present or no BuildCmd")
			}
			cmd := a.BuildCmd(c.repo, c.input)
			joined := strings.Join(cmd.Args, " ")
			if !strings.Contains(joined, " -- ") {
				t.Errorf("expected `--` separator in %q", joined)
			}
		})
	}
}

func TestSelectedPathActionsUseLiteralRenameEndpoints(t *testing.T) {
	change := git.ChangeInfo{
		SourcePath:      " old [*].txt",
		DestinationPath: "new \t🚀?.txt ",
	}
	for _, action := range AllActions() {
		if !action.UsesSelected {
			continue
		}
		if action.BuildSelectedCmd == nil {
			t.Fatalf("%s has no selected change builder", action.Name)
		}
		args := action.BuildSelectedCmd(git.RepoInfo{Path: "/repo"}, change).Args
		for _, path := range change.Pathspecs() {
			want := ":(literal)" + path
			found := false
			for _, arg := range args {
				if arg == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s args %q missing literal pathspec %q", action.Name, args, want)
			}
		}
	}
}

func TestStageSelectedRenameWithLiteralPathspecs(t *testing.T) {
	repo := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %s: %v", strings.Join(args, " "), out, err)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	oldPath := " old [*].txt"
	newPath := "new \t🚀?.txt "
	if err := os.WriteFile(filepath.Join(repo, oldPath), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", oldPath)
	runGit("commit", "-m", "initial")
	runGit("mv", oldPath, newPath)
	runGit("reset") // Make the filesystem rename unstaged for the selected-stage action.

	var stage Action
	for _, action := range AllActions() {
		if action.Key == "i" {
			stage = action
			break
		}
	}
	if stage.BuildSelectedCmd == nil {
		t.Fatal("stage selected action missing")
	}
	change := git.ChangeInfo{SourcePath: oldPath, DestinationPath: newPath}
	if out, err := stage.BuildSelectedCmd(git.RepoInfo{Path: repo}, change).CombinedOutput(); err != nil {
		t.Fatalf("stage selected rename: %s: %v", out, err)
	}
	cmd := exec.Command("git", "-C", repo, "diff", "--cached", "--name-status", "-z")
	out, err := cmd.Output()
	if err != nil || !bytes.Contains(out, []byte(oldPath)) || !bytes.Contains(out, []byte(newPath)) {
		t.Fatalf("staged rename = %q, err=%v", out, err)
	}
}

func TestRestoreUnstagedRenameUsesLiteralEndpoints(t *testing.T) {
	repo := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %s: %v", strings.Join(args, " "), out, err)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	oldPath := " old [*].txt"
	newPath := "new \t🚀?.txt "
	if err := os.WriteFile(filepath.Join(repo, oldPath), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", oldPath)
	runGit("commit", "-m", "initial")
	runGit("mv", oldPath, newPath)
	runGit("reset")

	change := git.ChangeInfo{SourcePath: oldPath, DestinationPath: newPath}
	for _, cmd := range restoreChangeCmds(repo, change) {
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("restore command %q: %s: %v", cmd.Args, out, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, oldPath)); err != nil {
		t.Fatalf("restored source missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, newPath)); !os.IsNotExist(err) {
		t.Fatalf("rename destination still exists: %v", err)
	}
	if out, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output(); err != nil || len(out) != 0 {
		t.Fatalf("repo not restored: %q, err=%v", out, err)
	}
}
