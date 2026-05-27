package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
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
	if err := os.Mkdir(filepath.Join(root, "not-a-repo"), 0755); err != nil {
		t.Fatalf("mkdir not-a-repo: %v", err)
	}

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

func TestSanitize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain ascii", "plain ascii"},
		{"with\ttab", "with\ttab"},
		{"esc \x1b[2J clear", "esc ?[2J clear"},
		{"bel\x07 newline\n cr\r", "bel? newline? cr?"},
		{"del\x7f", "del?"},
		{"C1 \x9bevil", "C1 ?evil"},
		{"", ""},
	}
	for _, c := range cases {
		if got := Sanitize(c.in); got != c.want {
			t.Errorf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGetLogSanitizesCommitFields(t *testing.T) {
	dir := setupTestRepo(t)
	// Plant a commit with ANSI escape in subject and author.
	cmds := [][]string{
		{"git", "config", "user.email", "evil@example.com"},
		{"git", "config", "user.name", "Mal\x1b[31mlory"},
		{"git", "commit", "--allow-empty", "-m", "subject \x1b[2J\x1b[H pwned"},
	}
	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s: %v", args, out, err)
		}
	}
	g := NewExecGit()
	commits, err := g.GetLog(context.Background(), dir, 1)
	if err != nil || len(commits) != 1 {
		t.Fatalf("GetLog: err=%v len=%d", err, len(commits))
	}
	if containsControl(commits[0].Subject) {
		t.Errorf("Subject still contains control chars: %q", commits[0].Subject)
	}
	if containsControl(commits[0].Author) {
		t.Errorf("Author still contains control chars: %q", commits[0].Author)
	}
}

func TestDetectDefaultBranchSanitizesPoisonedHEAD(t *testing.T) {
	dir := setupTestRepo(t)
	// Plant a poisoned refs/remotes/origin/HEAD whose ref name is an arg-injection payload.
	headPath := filepath.Join(dir, ".git", "refs", "remotes", "origin")
	if err := os.MkdirAll(headPath, 0755); err != nil {
		t.Fatal(err)
	}
	const poisoned = "ref: refs/remotes/origin/--exec=\x1b[Aevil\n"
	if err := os.WriteFile(filepath.Join(headPath, "HEAD"), []byte(poisoned), 0644); err != nil {
		t.Fatal(err)
	}
	g := NewExecGit()
	got := g.DetectDefaultBranch(context.Background(), dir)
	if containsControl(got) {
		t.Errorf("DetectDefaultBranch returned control chars: %q", got)
	}
	// Note: the leading "--" is intentionally still allowed through here; PR-2
	// addresses that with the -- separator on the consumer side.
}

// containsControl reports whether s contains any C0/C1 control byte or DEL.
func containsControl(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\t' {
			continue
		}
		if c < 0x20 || c == 0x7f || (c >= 0x80 && c < 0xa0) {
			return true
		}
	}
	return false
}

func TestLooksLikeLockError(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{"fatal: Unable to create '/foo/.git/index.lock': File exists.", true},
		{"Another git process seems to be running in this repository", true},
		{"fatal: could not lock config file .git/config", true},
		{"error: cannot lock ref 'refs/heads/main'", true},
		{"fatal: not a git repository", false},
		{"Updating ab12cd3..ef45gh6", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := looksLikeLockError(tt.stderr); got != tt.want {
			t.Errorf("looksLikeLockError(%q) = %v, want %v", tt.stderr, got, tt.want)
		}
	}
}

func TestTryClearStaleLocks(t *testing.T) {
	// Shorten the threshold so the test runs fast without sleeping seconds.
	orig := staleLockThreshold
	staleLockThreshold = 50 * time.Millisecond
	t.Cleanup(func() { staleLockThreshold = orig })

	makeRepoDirs := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		for _, sub := range []string{".git", ".git/refs/remotes/origin", ".git/refs/heads"} {
			if err := os.MkdirAll(filepath.Join(root, sub), 0755); err != nil {
				t.Fatal(err)
			}
		}
		return root
	}

	writeLock := func(t *testing.T, path string, age time.Duration) {
		t.Helper()
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		past := time.Now().Add(-age)
		if err := os.Chtimes(path, past, past); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("removes stale index.lock", func(t *testing.T) {
		root := makeRepoDirs(t)
		lock := filepath.Join(root, ".git", "index.lock")
		writeLock(t, lock, 500*time.Millisecond)

		cleared, err := tryClearStaleLocks(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(cleared) != 1 || cleared[0] != lock {
			t.Errorf("expected to clear %s, got %v", lock, cleared)
		}
		if _, err := os.Stat(lock); !os.IsNotExist(err) {
			t.Errorf("expected lock to be removed, stat err: %v", err)
		}
	})

	t.Run("keeps fresh lock", func(t *testing.T) {
		root := makeRepoDirs(t)
		lock := filepath.Join(root, ".git", "index.lock")
		writeLock(t, lock, 0) // brand new

		cleared, err := tryClearStaleLocks(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(cleared) != 0 {
			t.Errorf("expected no clears for fresh lock, got %v", cleared)
		}
		if _, err := os.Stat(lock); err != nil {
			t.Errorf("fresh lock should still exist: %v", err)
		}
	})

	t.Run("removes stale ref locks under refs/", func(t *testing.T) {
		root := makeRepoDirs(t)
		l1 := filepath.Join(root, ".git", "refs", "remotes", "origin", "main.lock")
		l2 := filepath.Join(root, ".git", "refs", "heads", "feature.lock")
		writeLock(t, l1, 500*time.Millisecond)
		writeLock(t, l2, 500*time.Millisecond)

		cleared, err := tryClearStaleLocks(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(cleared) != 2 {
			t.Errorf("expected 2 clears, got %v", cleared)
		}
	})

	t.Run("no-op when no locks exist", func(t *testing.T) {
		root := makeRepoDirs(t)
		cleared, err := tryClearStaleLocks(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(cleared) != 0 {
			t.Errorf("expected no clears on clean repo, got %v", cleared)
		}
	})
}

func TestRunRetriesAfterStaleLock(t *testing.T) {
	// End-to-end check: a stale .git/index.lock blocks a write op (e.g.
	// `git commit --allow-empty`). With our retry in place, the op should
	// succeed after the stale lock is cleared.
	orig := staleLockThreshold
	staleLockThreshold = 50 * time.Millisecond
	t.Cleanup(func() { staleLockThreshold = orig })

	dir := setupTestRepo(t)
	lock := filepath.Join(dir, ".git", "index.lock")
	if err := os.WriteFile(lock, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-500 * time.Millisecond)
	if err := os.Chtimes(lock, past, past); err != nil {
		t.Fatal(err)
	}

	g := NewExecGit()
	_, err := g.run(context.Background(), dir, "commit", "--allow-empty", "-m", "retry test")
	if err != nil {
		t.Fatalf("expected commit to succeed after stale-lock recovery, got: %v", err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Errorf("expected lock to be gone after recovery, stat err: %v", err)
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
