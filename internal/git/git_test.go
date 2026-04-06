package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s: %v", args, out, err)
		}
	}
	return dir
}

func TestGetBranch(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewExecGit()
	branch, err := g.GetBranch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should be main or master depending on git config
	if branch != "main" && branch != "master" {
		t.Errorf("unexpected branch: %s", branch)
	}
}

func TestDetectDefaultBranch(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewExecGit()
	branch := g.DetectDefaultBranch(context.Background(), dir)
	if branch != "main" && branch != "master" {
		t.Errorf("unexpected default branch: %s", branch)
	}
}

func TestGetChanges(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewExecGit()

	// No changes initially
	changes, err := g.GetChanges(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}

	// Create an untracked file
	if err := os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err = g.GetChanges(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
}

func TestGetRepoInfo(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewExecGit()

	info, err := g.GetRepoInfo(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name == "" {
		t.Error("expected non-empty name")
	}
	if info.Branch == "" {
		t.Error("expected non-empty branch")
	}
	// No remote, so ahead/behind should be 0
	if info.Ahead != 0 || info.Behind != 0 {
		t.Errorf("expected 0 ahead/behind, got %d/%d", info.Ahead, info.Behind)
	}
}

func TestGetLog(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewExecGit()

	commits, err := g.GetLog(context.Background(), dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Subject != "initial commit" {
		t.Errorf("unexpected commit subject: %s", commits[0].Subject)
	}
}

func TestDiscovery(t *testing.T) {
	root := t.TempDir()

	// Create 3 repos
	for _, name := range []string{"repo-a", "repo-b", "repo-c"} {
		dir := filepath.Join(root, name)
		os.Mkdir(dir, 0755)
		cmds := [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@test.com"},
			{"git", "config", "user.name", "Test"},
			{"git", "commit", "--allow-empty", "-m", "init"},
		}
		for _, args := range cmds {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("setup %s %v: %s: %v", name, args, out, err)
			}
		}
	}

	// Also create a non-repo directory
	os.Mkdir(filepath.Join(root, "not-a-repo"), 0755)

	g := NewExecGit()
	ch := Discover(context.Background(), DiscoveryOptions{
		RootDir:   root,
		Recursive: false,
		NoFetch:   true,
		Git:       g,
	})

	var found []string
	for result := range ch {
		found = append(found, result.Repo.Name)
	}

	if len(found) != 3 {
		t.Errorf("expected 3 repos, found %d: %v", len(found), found)
	}
}

func TestSwitchBranch(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewExecGit()
	ctx := context.Background()

	// Create a feature branch and switch to it
	_, err := g.RunCommand(ctx, dir, "checkout", "-b", "feature")
	if err != nil {
		t.Fatal(err)
	}
	branch, _ := g.GetBranch(ctx, dir)
	if branch != "feature" {
		t.Fatalf("expected feature, got %s", branch)
	}

	// Switch back to default
	defaultBranch := g.DetectDefaultBranch(ctx, dir)
	_, err = g.SwitchBranch(ctx, dir, defaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	branch, _ = g.GetBranch(ctx, dir)
	if branch != defaultBranch {
		t.Errorf("expected %s, got %s", defaultBranch, branch)
	}
}

func TestRunShellCommand(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewExecGit()
	ctx := context.Background()

	out, err := g.RunShellCommand(ctx, dir, "ls", "-la")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, ".git") {
		t.Errorf("expected .git in ls output, got: %s", out)
	}
}

func TestComputeStatus(t *testing.T) {
	tests := []struct {
		name   string
		info   RepoInfo
		expect RepoStatus
	}{
		{"up to date", RepoInfo{Branch: "main", DefaultBranch: "main"}, StatusUpToDate},
		{"ahead", RepoInfo{Branch: "main", DefaultBranch: "main", Ahead: 2}, StatusAhead},
		{"behind", RepoInfo{Branch: "main", DefaultBranch: "main", Behind: 3}, StatusBehind},
		{"diverged", RepoInfo{Branch: "main", DefaultBranch: "main", Ahead: 1, Behind: 1}, StatusDiverged},
		{"non-default branch", RepoInfo{Branch: "feat", DefaultBranch: "main"}, StatusNonDefault},
		{"dirty", RepoInfo{Branch: "main", DefaultBranch: "main", Changes: 5}, StatusDirty},
		{
			"dirty + non-default returns non-default (checked first)",
			RepoInfo{Branch: "feat", DefaultBranch: "main", Changes: 10},
			StatusNonDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeStatus(tt.info)
			if got != tt.expect {
				t.Errorf("computeStatus(%+v) = %s, want %s", tt.info, got, tt.expect)
			}
		})
	}
}

func TestPathBaseName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple path", "/foo/bar", "bar"},
		{"trailing slash", "/foo/bar/", "bar"},
		{"single component", "bar", "bar"},
		{"root path child", "/baz", "baz"},
		{"trailing backslash", `C:\foo\bar\`, "bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathBaseName(tt.path)
			if got != tt.want {
				t.Errorf("pathBaseName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
