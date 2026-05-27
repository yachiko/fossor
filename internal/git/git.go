package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Git defines the interface for git operations.
type Git interface {
	GetRepoInfo(ctx context.Context, path string) (RepoInfo, error)
	DetectDefaultBranch(ctx context.Context, path string) string
	GetBranch(ctx context.Context, path string) (string, error)
	GetRemote(ctx context.Context, path string) (string, error)
	GetAheadBehind(ctx context.Context, path, branch string) (int, int, error)
	GetChanges(ctx context.Context, path string) ([]ChangeInfo, error)
	GetLog(ctx context.Context, path string, n int) ([]CommitInfo, error)
	Fetch(ctx context.Context, path string) error
	Pull(ctx context.Context, path string) (string, error)
	Push(ctx context.Context, path string) (string, error)
	RunCommand(ctx context.Context, path string, args ...string) (string, error)
	SwitchBranch(ctx context.Context, path, branch string) (string, error)
	RunShellCommand(ctx context.Context, dir string, name string, args ...string) (string, error)
}

// ExecGit implements Git using os/exec.
type ExecGit struct{}

func NewExecGit() *ExecGit {
	return &ExecGit{}
}

func (g *ExecGit) run(ctx context.Context, path string, args ...string) (string, error) {
	out, err := runGitOnce(ctx, path, args...)
	if err == nil {
		return out, nil
	}
	if !looksLikeLockError(err.Error()) {
		return "", err
	}
	cleared, _ := tryClearStaleLocks(path)
	if len(cleared) == 0 {
		return "", err
	}
	return runGitOnce(ctx, path, args...)
}

func runGitOnce(ctx context.Context, path string, args ...string) (string, error) {
	allArgs := append([]string{"-C", path}, args...)
	cmd := exec.CommandContext(ctx, "git", allArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Sanitize strips C0 control characters and DEL (except tab) from a string
// before it crosses into the UI. Repo-supplied data — commit subjects, author
// names, branch refs, file paths — can contain ANSI escape sequences that
// would otherwise reach the terminal directly and enable UI spoofing or
// cursor hijack. Replaces stripped bytes with '?'.
//
// Operates on bytes, not runes, so it also catches stray C1 controls that
// appear as invalid UTF-8 byte sequences (a single 0x9b byte, for instance).
func Sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\t':
			b.WriteByte(c)
		case c < 0x20, c == 0x7f, c >= 0x80 && c < 0xa0:
			b.WriteByte('?')
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// staleLockThreshold is how long a *.lock file must be untouched before we
// consider it abandoned. Package-level so tests can override it.
var staleLockThreshold = 5 * time.Second

// lockErrorMarkers are substrings git uses when refusing to run because of an
// existing lock file. Matching any of them in stderr triggers the stale-lock
// recovery path.
var lockErrorMarkers = []string{
	"Another git process seems to be running",
	"Unable to create '",
	"could not lock",
	"cannot lock ref",
}

func looksLikeLockError(s string) bool {
	for _, m := range lockErrorMarkers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

// tryClearStaleLocks scans the well-known lock files under repoPath/.git and
// removes any that look genuinely abandoned (mtime older than
// staleLockThreshold and not held by any process per lsof, if available). It
// returns the list of removed paths. Never removes a lock that could still be
// held by a live process.
func tryClearStaleLocks(repoPath string) ([]string, error) {
	gitDir := filepath.Join(repoPath, ".git")
	candidates := []string{
		filepath.Join(gitDir, "index.lock"),
		filepath.Join(gitDir, "HEAD.lock"),
		filepath.Join(gitDir, "packed-refs.lock"),
	}
	if matches, err := filepath.Glob(filepath.Join(gitDir, "refs", "remotes", "origin", "*.lock")); err == nil {
		candidates = append(candidates, matches...)
	}
	if matches, err := filepath.Glob(filepath.Join(gitDir, "refs", "heads", "*.lock")); err == nil {
		candidates = append(candidates, matches...)
	}

	var cleared []string
	for _, lock := range candidates {
		info, err := os.Stat(lock)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < staleLockThreshold {
			continue
		}
		if lockHasHolder(lock) {
			continue
		}
		age := time.Since(info.ModTime())
		if err := os.Remove(lock); err == nil {
			cleared = append(cleared, lock)
			debugLog("stale-lock cleared repo=%s lock=%s age=%s", repoPath, lock, age.Truncate(time.Millisecond))
		}
	}
	return cleared, nil
}

// debugLog appends a line to ~/.cache/fossor/debug.log when FOSSOR_DEBUG=1.
// Stays silent (and never errors out the caller) when the env var is unset or
// the cache dir is unwritable — diagnostics are best-effort.
func debugLog(format string, args ...any) {
	if os.Getenv("FOSSOR_DEBUG") != "1" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".cache", "fossor")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format(time.RFC3339)}, args...)...)
}

// lockHasHolder uses lsof (if available) to check whether any process holds
// the lock file open. Returns true on positive identification of a holder,
// false otherwise (including when lsof is missing or errors out — we only want
// to *block* removal on confirmed live holders, not on tool absence).
func lockHasHolder(lock string) bool {
	if _, err := exec.LookPath("lsof"); err != nil {
		return false
	}
	out, err := exec.Command("lsof", "-t", lock).Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func (g *ExecGit) DetectDefaultBranch(ctx context.Context, path string) string {
	// Fast path: read symref file directly (avoids spawning git)
	if data, err := os.ReadFile(filepath.Join(path, ".git", "refs", "remotes", "origin", "HEAD")); err == nil {
		ref := strings.TrimSpace(string(data))
		const prefix = "ref: refs/remotes/origin/"
		if strings.HasPrefix(ref, prefix) {
			return Sanitize(ref[len(prefix):])
		}
	}

	// Fallback: git symbolic-ref (handles packed refs)
	out, err := g.run(ctx, path, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		parts := strings.Split(out, "/")
		if len(parts) > 0 {
			return Sanitize(parts[len(parts)-1])
		}
	}

	// Check common branch names via filesystem first
	for _, name := range []string{"main", "master"} {
		if _, err := os.Stat(filepath.Join(path, ".git", "refs", "heads", name)); err == nil {
			return name
		}
	}

	return "main"
}

func (g *ExecGit) GetBranch(ctx context.Context, path string) (string, error) {
	out, err := g.run(ctx, path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return Sanitize(out), nil
}

func (g *ExecGit) GetRemote(ctx context.Context, path string) (string, error) {
	out, err := g.run(ctx, path, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return Sanitize(out), nil
}

func (g *ExecGit) GetAheadBehind(ctx context.Context, path, branch string) (int, int, error) {
	upstream := "origin/" + branch
	// `--` separator: branch may be repo-controlled (poisoned HEAD); without
	// the separator a leading `-` turns the refspec into a git flag.
	out, err := g.run(ctx, path, "rev-list", "--left-right", "--count", "--", branch+"..."+upstream)
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", out)
	}

	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return ahead, behind, nil
}

func (g *ExecGit) GetChanges(ctx context.Context, path string) ([]ChangeInfo, error) {
	// Run directly instead of via g.run() — porcelain output has significant
	// leading spaces (e.g. " M file") that TrimSpace would destroy.
	cmd := exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain", "-uall")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}

	out := strings.TrimRight(stdout.String(), "\n\r ")
	if out == "" {
		return nil, nil
	}

	// Detect submodule paths from .gitmodules
	submodulePaths := getSubmodulePaths(path)

	var changes []ChangeInfo
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		p := strings.TrimSpace(line[3:])
		changes = append(changes, ChangeInfo{
			Staged:      line[0],
			Unstaged:    line[1],
			Path:        Sanitize(p),
			IsSubmodule: submodulePaths[p],
		})
	}
	return changes, nil
}

// getSubmodulePaths reads .gitmodules and returns a set of submodule paths.
func getSubmodulePaths(repoPath string) map[string]bool {
	cmd := exec.Command("git", "-C", repoPath, "config", "--file", ".gitmodules", "--get-regexp", "^submodule\\..*\\.path$")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	paths := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		// Format: "submodule.<name>.path <value>"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			paths[parts[1]] = true
		}
	}
	return paths
}

func (g *ExecGit) GetRepoInfo(ctx context.Context, path string) (RepoInfo, error) {
	name := pathBaseName(path)
	info := RepoInfo{
		Name: name,
		Path: path,
	}

	// Single command replaces GetBranch + GetAheadBehind + GetChanges
	si, err := g.getStatusInfo(ctx, path)
	if err != nil {
		info.Status = StatusError
		info.Error = fmt.Errorf("get status: %w", err)
		return info, nil
	}
	info.Branch = si.branch
	info.Ahead = si.ahead
	info.Behind = si.behind
	info.Changes = si.changes

	info.DefaultBranch = g.DetectDefaultBranch(ctx, path)

	info.Status = computeStatus(info)
	return info, nil
}

// getStatusInfo runs a single git command to get branch, ahead/behind, and change count.
func (g *ExecGit) getStatusInfo(ctx context.Context, path string) (struct {
	branch  string
	ahead   int
	behind  int
	changes int
}, error) {
	type result struct {
		branch  string
		ahead   int
		behind  int
		changes int
	}

	out, err := g.run(ctx, path, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return result{}, err
	}

	var r result
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			r.branch = Sanitize(line[len("# branch.head "):])
		case strings.HasPrefix(line, "# branch.ab "):
			// Format: # branch.ab +N -M
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				r.ahead, _ = strconv.Atoi(parts[2][1:])  // skip '+'
				r.behind, _ = strconv.Atoi(parts[3][1:]) // skip '-'
			}
		case line[0] != '#':
			r.changes++
		}
	}

	return r, nil
}

func (g *ExecGit) GetLog(ctx context.Context, path string, n int) ([]CommitInfo, error) {
	format := "%H%n%h%n%an%n%aI%n%s"
	out, err := g.run(ctx, path, "log", fmt.Sprintf("-%d", n), fmt.Sprintf("--format=%s", format))
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	var commits []CommitInfo
	for i := 0; i+4 < len(lines); i += 5 {
		date, _ := time.Parse(time.RFC3339, lines[i+3])
		commits = append(commits, CommitInfo{
			Hash:    lines[i],
			Short:   lines[i+1],
			Author:  Sanitize(lines[i+2]),
			Date:    date,
			Subject: Sanitize(lines[i+4]),
		})
	}
	return commits, nil
}

func (g *ExecGit) Fetch(ctx context.Context, path string) error {
	_, err := g.run(ctx, path, "fetch", "--prune")
	return err
}

func (g *ExecGit) Pull(ctx context.Context, path string) (string, error) {
	return g.run(ctx, path, "pull")
}

func (g *ExecGit) Push(ctx context.Context, path string) (string, error) {
	return g.run(ctx, path, "push")
}

func (g *ExecGit) RunCommand(ctx context.Context, path string, args ...string) (string, error) {
	return g.run(ctx, path, args...)
}

func (g *ExecGit) SwitchBranch(ctx context.Context, path, branch string) (string, error) {
	// `--` separator: branch is repo-controlled when called from
	// switchDefault* with r.DefaultBranch derived from refs/remotes/origin/HEAD.
	return g.run(ctx, path, "switch", "--", branch)
}

func (g *ExecGit) RunShellCommand(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func computeStatus(info RepoInfo) RepoStatus {
	if info.Branch != info.DefaultBranch {
		return StatusNonDefault
	}
	if info.Ahead > 0 && info.Behind > 0 {
		return StatusDiverged
	}
	if info.Ahead > 0 {
		return StatusAhead
	}
	if info.Behind > 0 {
		return StatusBehind
	}
	if info.Changes > 0 {
		return StatusDirty
	}
	return StatusUpToDate
}

func pathBaseName(path string) string {
	// Trim trailing slashes then find last component
	path = strings.TrimRight(path, "/\\")
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		return path[i+1:]
	}
	return path
}
