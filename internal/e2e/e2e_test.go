package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ahoma/fossor/internal/git"
)

// fixedDate is used for deterministic commits.
const fixedDate = "2025-01-15T12:00:00+00:00"

// git runs a git command in the given directory and returns stdout.
// It sets environment variables for deterministic commits and to allow
// the file:// protocol (needed for submodule/clone operations).
func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+fixedDate,
		"GIT_COMMITTER_DATE="+fixedDate,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=protocol.file.allow",
		"GIT_CONFIG_VALUE_0=always",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v (in %s): %s: %v", args, dir, out, err)
	}
	return strings.TrimSpace(string(out))
}

// setupRepo creates a temporary git repo with an initial commit on the "main" branch.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-b", "main")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "config", "user.name", "Test")
	gitCmd(t, dir, "commit", "--allow-empty", "-m", "initial commit")
	return dir
}

// setupRepoWithRemote creates a local repo and a bare remote, pushes main to the remote,
// and configures the local repo to track it. Returns (repoPath, barePath).
func setupRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()

	// Create bare remote
	bare := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(bare, 0755); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, bare, "init", "--bare", "-b", "main")

	// Create local repo
	repo := setupRepo(t)
	gitCmd(t, repo, "remote", "add", "origin", bare)
	gitCmd(t, repo, "push", "-u", "origin", "main")

	return repo, bare
}

// commitToBare clones the bare repo into a temporary directory, makes a commit, and pushes it.
// This simulates another user pushing to the remote.
func commitToBare(t *testing.T, barePath, msg string) {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "clone")

	cmd := exec.Command("git", "clone", barePath, tmp)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=protocol.file.allow",
		"GIT_CONFIG_VALUE_0=always",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clone bare: %s: %v", out, err)
	}

	gitCmd(t, tmp, "config", "user.email", "other@test.com")
	gitCmd(t, tmp, "config", "user.name", "Other")
	gitCmd(t, tmp, "commit", "--allow-empty", "-m", msg)
	gitCmd(t, tmp, "push")
}

// ---------------------------------------------------------------------------
// Status computation tests
// ---------------------------------------------------------------------------

func TestE2E_CleanRepo(t *testing.T) {
	repo, _ := setupRepoWithRemote(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Fetch so the remote tracking ref is current
	if err := g.Fetch(ctx, repo); err != nil {
		t.Fatal(err)
	}

	info, err := g.GetRepoInfo(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != git.StatusUpToDate {
		t.Errorf("expected StatusUpToDate, got %s", info.Status)
	}
	if info.Ahead != 0 {
		t.Errorf("expected Ahead==0, got %d", info.Ahead)
	}
	if info.Behind != 0 {
		t.Errorf("expected Behind==0, got %d", info.Behind)
	}
}

func TestE2E_DirtyUnstaged(t *testing.T) {
	repo, _ := setupRepoWithRemote(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Create a tracked file, commit it, then modify it
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", "file.txt")
	gitCmd(t, repo, "commit", "-m", "add file")
	gitCmd(t, repo, "push")

	// Now modify the tracked file without staging
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := g.GetRepoInfo(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != git.StatusDirty {
		t.Errorf("expected StatusDirty, got %s", info.Status)
	}
	if info.Changes < 1 {
		t.Errorf("expected Changes > 0, got %d", info.Changes)
	}
}

func TestE2E_Ahead(t *testing.T) {
	repo, _ := setupRepoWithRemote(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Make a local commit without pushing
	gitCmd(t, repo, "commit", "--allow-empty", "-m", "local only")

	info, err := g.GetRepoInfo(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != git.StatusAhead {
		t.Errorf("expected StatusAhead, got %s", info.Status)
	}
	if info.Ahead != 1 {
		t.Errorf("expected Ahead==1, got %d", info.Ahead)
	}
}

func TestE2E_Behind(t *testing.T) {
	repo, bare := setupRepoWithRemote(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Simulate another user pushing a commit
	commitToBare(t, bare, "remote commit")

	// Fetch to learn about the new remote commit
	if err := g.Fetch(ctx, repo); err != nil {
		t.Fatal(err)
	}

	info, err := g.GetRepoInfo(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != git.StatusBehind {
		t.Errorf("expected StatusBehind, got %s", info.Status)
	}
	if info.Behind != 1 {
		t.Errorf("expected Behind==1, got %d", info.Behind)
	}
}

func TestE2E_Diverged(t *testing.T) {
	repo, bare := setupRepoWithRemote(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Remote gets a commit
	commitToBare(t, bare, "remote diverge")

	// Fetch so we know about the remote commit
	if err := g.Fetch(ctx, repo); err != nil {
		t.Fatal(err)
	}

	// Local also gets a commit (not pushed)
	gitCmd(t, repo, "commit", "--allow-empty", "-m", "local diverge")

	info, err := g.GetRepoInfo(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != git.StatusDiverged {
		t.Errorf("expected StatusDiverged, got %s", info.Status)
	}
	if info.Ahead < 1 {
		t.Errorf("expected Ahead >= 1, got %d", info.Ahead)
	}
	if info.Behind < 1 {
		t.Errorf("expected Behind >= 1, got %d", info.Behind)
	}
}

func TestE2E_NonDefault(t *testing.T) {
	repo, _ := setupRepoWithRemote(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Create and switch to a feature branch
	gitCmd(t, repo, "checkout", "-b", "feature")

	info, err := g.GetRepoInfo(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != git.StatusNonDefault {
		t.Errorf("expected StatusNonDefault, got %s", info.Status)
	}
	if info.Branch != "feature" {
		t.Errorf("expected branch 'feature', got %q", info.Branch)
	}
}

func TestE2E_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-b", "main")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "config", "user.name", "Test")
	// No commits at all

	g := git.NewExecGit()
	ctx := context.Background()

	// The key requirement: this must not panic
	info, err := g.GetRepoInfo(ctx, dir)
	if err != nil {
		// An error is acceptable for an empty repo; a panic is not.
		t.Logf("GetRepoInfo returned error for empty repo (acceptable): %v", err)
	}
	t.Logf("EmptyRepo info: %+v", info)
}

// ---------------------------------------------------------------------------
// Changes detection tests
// ---------------------------------------------------------------------------

func TestE2E_UntrackedDir(t *testing.T) {
	repo := setupRepo(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Create a subdirectory with two files
	sub := filepath.Join(repo, "subdir")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := g.GetChanges(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}

	// With -uall, individual files should be listed, not just "subdir/"
	paths := make(map[string]bool)
	for _, c := range changes {
		paths[c.Path] = true
	}
	if !paths["subdir/a.txt"] {
		t.Errorf("expected subdir/a.txt in changes, got %v", paths)
	}
	if !paths["subdir/b.txt"] {
		t.Errorf("expected subdir/b.txt in changes, got %v", paths)
	}
}

func TestE2E_PorcelainLeadingSpace(t *testing.T) {
	repo := setupRepo(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Create a file, add and commit it
	filePath := filepath.Join(repo, "tracked.txt")
	if err := os.WriteFile(filePath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", "tracked.txt")
	gitCmd(t, repo, "commit", "-m", "add tracked")

	// Now modify it without staging -- porcelain output should be " M tracked.txt"
	// The leading space for the staged column is significant.
	if err := os.WriteFile(filePath, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := g.GetChanges(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}

	c := changes[0]
	// Regression test: Staged byte must be ' ' (space), not empty or 'M'
	if c.Staged != ' ' {
		t.Errorf("expected Staged==' ' (0x%02x), got 0x%02x (%c)", ' ', c.Staged, c.Staged)
	}
	if c.Unstaged != 'M' {
		t.Errorf("expected Unstaged=='M', got %c", c.Unstaged)
	}
	if c.Path != "tracked.txt" {
		t.Errorf("expected Path=='tracked.txt', got %q", c.Path)
	}
}

func TestE2E_StagedAndUnstaged(t *testing.T) {
	repo := setupRepo(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Create two tracked files
	for _, name := range []string{"staged.txt", "unstaged.txt"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte("init"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	gitCmd(t, repo, "add", "staged.txt", "unstaged.txt")
	gitCmd(t, repo, "commit", "-m", "add files")

	// Modify both, but only stage one
	if err := os.WriteFile(filepath.Join(repo, "staged.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", "staged.txt")

	if err := os.WriteFile(filepath.Join(repo, "unstaged.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := g.GetChanges(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}

	byPath := make(map[string]git.ChangeInfo)
	for _, c := range changes {
		byPath[c.Path] = c
	}

	// staged.txt should have Staged=='M' and Unstaged==' '
	if s, ok := byPath["staged.txt"]; !ok {
		t.Error("staged.txt not found in changes")
	} else {
		if s.Staged != 'M' {
			t.Errorf("staged.txt: expected Staged=='M', got %c (0x%02x)", s.Staged, s.Staged)
		}
		if s.Unstaged != ' ' {
			t.Errorf("staged.txt: expected Unstaged==' ', got %c (0x%02x)", s.Unstaged, s.Unstaged)
		}
	}

	// unstaged.txt should have Staged==' ' and Unstaged=='M'
	if u, ok := byPath["unstaged.txt"]; !ok {
		t.Error("unstaged.txt not found in changes")
	} else {
		if u.Staged != ' ' {
			t.Errorf("unstaged.txt: expected Staged==' ', got %c (0x%02x)", u.Staged, u.Staged)
		}
		if u.Unstaged != 'M' {
			t.Errorf("unstaged.txt: expected Unstaged=='M', got %c (0x%02x)", u.Unstaged, u.Unstaged)
		}
	}
}

// ---------------------------------------------------------------------------
// Stash test
// ---------------------------------------------------------------------------

func TestE2E_StashList(t *testing.T) {
	repo := setupRepo(t)
	g := git.NewExecGit()
	ctx := context.Background()

	// Create a file and stage it (stash needs at least staged/tracked content)
	if err := os.WriteFile(filepath.Join(repo, "stashme.txt"), []byte("stash this"), 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", "stashme.txt")
	gitCmd(t, repo, "stash", "push", "-m", "test stash")

	out, err := g.RunCommand(ctx, repo, "stash", "list")
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Error("expected non-empty stash list output")
	}
	if !strings.Contains(out, "test stash") {
		t.Errorf("expected stash list to contain 'test stash', got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Discovery test
// ---------------------------------------------------------------------------

func TestE2E_Discovery(t *testing.T) {
	root := t.TempDir()
	g := git.NewExecGit()
	ctx := context.Background()

	// Create three repos with different states
	names := []string{"clean-repo", "dirty-repo", "ahead-repo"}
	for _, name := range names {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		gitCmd(t, dir, "init", "-b", "main")
		gitCmd(t, dir, "config", "user.email", "test@test.com")
		gitCmd(t, dir, "config", "user.name", "Test")
		gitCmd(t, dir, "commit", "--allow-empty", "-m", "init "+name)
	}

	// Make dirty-repo dirty
	if err := os.WriteFile(filepath.Join(root, "dirty-repo", "dirt.txt"), []byte("dirt"), 0644); err != nil {
		t.Fatal(err)
	}

	// Also create a non-repo directory that should be ignored
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo"), 0755); err != nil {
		t.Fatal(err)
	}

	ch := git.Discover(ctx, git.DiscoveryOptions{
		RootDir:   root,
		Recursive: false,
		NoFetch:   true,
		Git:       g,
	})

	found := make(map[string]bool)
	for result := range ch {
		found[result.Repo.Name] = true
	}

	for _, name := range names {
		if !found[name] {
			t.Errorf("Discover did not find repo %q; found: %v", name, found)
		}
	}
	if len(found) != 3 {
		t.Errorf("expected exactly 3 repos, found %d: %v", len(found), found)
	}
}
