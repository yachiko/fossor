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
	GetRemoteDefaultBranch(ctx context.Context, path string) (string, error)
	GetBranch(ctx context.Context, path string) (string, error)
	GetRemote(ctx context.Context, path string) (string, error)
	GetAheadBehind(ctx context.Context, path, branch string) (int, int, error)
	GetChanges(ctx context.Context, path string) ([]ChangeInfo, error)
	GetFileDiff(ctx context.Context, path, filePath string, isSubmodule bool) (string, error)
	GetStagedDiff(ctx context.Context, path string) (string, error)
	GetStashes(ctx context.Context, path string) ([]StashInfo, error)
	GetStashDiff(ctx context.Context, path, ref string) (string, error)
	GetBranches(ctx context.Context, path, defaultBranch string) ([]BranchInfo, error)
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

// Command constructs a Git process for interactive terminal handoff. Callers
// use it with tea.ExecProcess when Git must own the terminal (for example an
// editor or interactive rebase); background work should use the Git interface.
func Command(path string, args ...string) *exec.Cmd {
	return exec.Command("git", append([]string{"-C", path}, args...)...)
}

func (g *ExecGit) run(ctx context.Context, path string, args ...string) (string, error) {
	out, err := g.runRaw(ctx, path, args...)
	if err == nil {
		return strings.TrimSpace(out), nil
	}
	return "", err
}

// runRaw is run's counterpart for Git output whose whitespace is significant.
func (g *ExecGit) runRaw(ctx context.Context, path string, args ...string) (string, error) {
	out, err := runGitRawOnce(ctx, path, args...)
	if err == nil || !looksLikeLockError(err.Error()) {
		return out, err
	}
	cleared, _ := tryClearStaleLocks(ctx, path)
	if len(cleared) == 0 || ctx.Err() != nil {
		return out, err
	}
	return runGitRawOnce(ctx, path, args...)
}

// IsWorktree validates both regular and linked worktrees and rejects
// submodules, which are intentionally not independent Fossor repositories.
func (g *ExecGit) IsWorktree(ctx context.Context, path string) bool {
	inside, err := g.run(ctx, path, "rev-parse", "--is-inside-work-tree")
	if err != nil || inside != "true" {
		return false
	}
	superproject, err := g.run(ctx, path, "rev-parse", "--show-superproject-working-tree")
	return err == nil && superproject == ""
}

func runGitOnce(ctx context.Context, path string, args ...string) (string, error) {
	out, err := runGitRawOnce(ctx, path, args...)
	return strings.TrimSpace(out), err
}

func runGitRawOnce(ctx context.Context, path string, args ...string) (string, error) {
	allArgs := append([]string{"-C", path}, args...)
	cmd := exec.CommandContext(ctx, "git", allArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return stdout.String(), nil
}

// Sanitize makes untrusted text safe for terminal display. It preserves valid
// UTF-8, replaces malformed byte sequences with U+FFFD, and neutralizes C0,
// C1, and DEL controls (tabs remain allowed for existing layout behavior).
// It is deliberately a rendering-boundary function: Git acquisition retains
// raw values for subsequent Git commands.
func Sanitize(s string) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		switch {
		case c == '\t':
			b.WriteRune(c)
		case c < 0x20, c == 0x7f, c >= 0x80 && c < 0xa0:
			b.WriteByte('?')
		default:
			b.WriteRune(c)
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

// tryClearStaleLocks scans the well-known lock files under the resolved git dir and
// removes any that look genuinely abandoned (mtime older than
// staleLockThreshold and not held by any process per lsof, if available). It
// returns the list of removed paths. Never removes a lock that could still be
// held by a live process.
func tryClearStaleLocks(ctx context.Context, repoPath string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	gitDir, err := runGitOnce(ctx, repoPath, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return nil, err
	}
	gitDirs := []string{gitDir}
	if commonDir, err := runGitOnce(ctx, repoPath, "rev-parse", "--path-format=absolute", "--git-common-dir"); err == nil && filepath.Clean(commonDir) != filepath.Clean(gitDir) {
		gitDirs = append(gitDirs, commonDir)
	}
	var candidates []string
	for _, dir := range gitDirs {
		candidates = append(candidates,
			filepath.Join(dir, "index.lock"),
			filepath.Join(dir, "HEAD.lock"),
			filepath.Join(dir, "packed-refs.lock"),
		)
		if matches, err := filepath.Glob(filepath.Join(dir, "refs", "remotes", "origin", "*.lock")); err == nil {
			candidates = append(candidates, matches...)
		}
		if matches, err := filepath.Glob(filepath.Join(dir, "refs", "heads", "*.lock")); err == nil {
			candidates = append(candidates, matches...)
		}
	}

	var cleared []string
	for _, lock := range candidates {
		if err := ctx.Err(); err != nil {
			return cleared, err
		}
		info, err := os.Stat(lock)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < staleLockThreshold {
			continue
		}
		if lockHasHolder(ctx, lock) {
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
func lockHasHolder(ctx context.Context, lock string) bool {
	if _, err := exec.LookPath("lsof"); err != nil {
		return false
	}
	out, err := exec.CommandContext(ctx, "lsof", "-t", lock).Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func (g *ExecGit) DetectDefaultBranch(ctx context.Context, path string) string {
	// Fast path: read symref file directly from Git's resolved common dir.
	if commonDir, err := g.commonGitDir(ctx, path); err == nil {
		if data, err := os.ReadFile(filepath.Join(commonDir, "refs", "remotes", "origin", "HEAD")); err == nil {
			ref := strings.TrimSpace(string(data))
			const prefix = "ref: refs/remotes/origin/"
			if strings.HasPrefix(ref, prefix) {
				return ref[len(prefix):]
			}
		}
	}

	// Fallback: git symbolic-ref (handles packed refs)
	out, err := g.run(ctx, path, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		const prefix = "refs/remotes/origin/"
		if strings.HasPrefix(out, prefix) {
			return out[len(prefix):]
		}
	}

	// Check common branch names through Git so packed refs and linked worktrees work.
	for _, name := range []string{"main", "master"} {
		if _, err := g.run(ctx, path, "show-ref", "--verify", "--quiet", "refs/heads/"+name); err == nil {
			return name
		}
	}

	return "main"
}

// GetRemoteDefaultBranch returns origin's advertised HEAD branch without
// changing any local refs.
func (g *ExecGit) GetRemoteDefaultBranch(ctx context.Context, path string) (string, error) {
	out, err := g.run(ctx, path, "ls-remote", "--symref", "origin", "HEAD")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "ref:" && fields[2] == "HEAD" {
			const prefix = "refs/heads/"
			if strings.HasPrefix(fields[1], prefix) {
				return fields[1][len(prefix):], nil
			}
		}
	}
	return "", fmt.Errorf("remote HEAD symref not found")
}

func (g *ExecGit) GetBranch(ctx context.Context, path string) (string, error) {
	out, err := g.run(ctx, path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (g *ExecGit) GetRemote(ctx context.Context, path string) (string, error) {
	out, err := g.run(ctx, path, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (g *ExecGit) GetAheadBehind(ctx context.Context, path, branch string) (int, int, error) {
	upstream := "origin/" + branch
	// --end-of-options keeps a poisoned ref name from being parsed as a flag.
	out, err := g.run(ctx, path, "rev-list", "--left-right", "--count", "--end-of-options", branch+"..."+upstream)
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
	// NUL-delimited porcelain preserves arbitrary paths and emits rename entries
	// as separate destination and source paths rather than a display string.
	out, err := g.runRaw(ctx, path, "status", "--porcelain=v1", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	submodulePaths := g.getSubmodulePaths(ctx, path)
	data := []byte(out)
	var changes []ChangeInfo
	for len(data) > 0 {
		i := bytes.IndexByte(data, 0)
		if i < 0 {
			return nil, fmt.Errorf("malformed porcelain status: missing NUL terminator")
		}
		record := data[:i]
		data = data[i+1:]
		if len(record) < 3 {
			continue
		}

		destination := string(record[3:])
		var source string
		if record[0] == 'R' || record[0] == 'C' || record[1] == 'R' || record[1] == 'C' {
			i := bytes.IndexByte(data, 0)
			if i < 0 {
				return nil, fmt.Errorf("malformed porcelain status: missing rename source")
			}
			source = string(data[:i])
			data = data[i+1:]
		}
		changes = append(changes, ChangeInfo{
			Staged:          record[0],
			Unstaged:        record[1],
			SourcePath:      source,
			DestinationPath: destination,
			IsSubmodule:     submodulePaths[destination],
		})
	}
	return changes, nil
}

// GetFileDiff returns the working-tree diff for a changed path. Untracked
// files are compared against /dev/null because ordinary git diff omits them.
func (g *ExecGit) GetFileDiff(ctx context.Context, path, filePath string, isSubmodule bool) (string, error) {
	if isSubmodule {
		return g.runRaw(ctx, path, "diff", "--submodule=log", "HEAD", "--", filePath)
	}
	diff, err := g.runRaw(ctx, path, "diff", "HEAD", "--", filePath)
	if err != nil {
		return "", err
	}
	if diff != "" {
		return diff, nil
	}
	// git diff --no-index exits with status 1 when it finds a difference.
	diff, err = g.runRaw(ctx, path, "diff", "--no-index", "--", "/dev/null", filePath)
	if err != nil && diff == "" {
		return "", err
	}
	return diff, nil
}

func (g *ExecGit) GetStagedDiff(ctx context.Context, path string) (string, error) {
	return g.runRaw(ctx, path, "diff", "--cached")
}

func (g *ExecGit) GetStashes(ctx context.Context, path string) ([]StashInfo, error) {
	out, err := g.run(ctx, path, "stash", "list", "--format=%gd%x09%gs")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	stashes := make([]StashInfo, 0, strings.Count(out, "\n")+1)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		stashes = append(stashes, StashInfo{Ref: Sanitize(parts[0]), Message: Sanitize(parts[1])})
	}
	return stashes, nil
}

func (g *ExecGit) GetStashDiff(ctx context.Context, path, ref string) (string, error) {
	return g.runRaw(ctx, path, "stash", "show", "-p", "--", ref)
}

func (g *ExecGit) GetBranches(ctx context.Context, path, defaultBranch string) ([]BranchInfo, error) {
	out, err := g.run(ctx, path, "for-each-ref", "--sort=-committerdate", "--format=%(refname:short)\t%(HEAD)\t%(committerdate:short)\t%(subject)", "refs/heads/")
	if err != nil {
		return nil, err
	}

	// The equals form keeps a poisoned ref name from being parsed as an option.
	mergedOut, mergedErr := g.run(ctx, path, "branch", "--merged="+defaultBranch)
	merged := make(map[string]bool)
	if mergedErr == nil {
		for _, line := range strings.Split(mergedOut, "\n") {
			if name := strings.TrimSpace(strings.TrimPrefix(line, "*")); name != "" {
				merged[name] = true
			}
		}
	}

	var branches []BranchInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		branch := BranchInfo{
			Name:        Sanitize(parts[0]),
			IsCurrent:   parts[1] == "*",
			Merged:      merged[parts[0]],
			MergedError: mergedErr,
			LastDate:    Sanitize(parts[2]),
			LastMsg:     Sanitize(parts[3]),
		}
		if parts[0] != defaultBranch {
			comparison, comparisonErr := g.run(ctx, path, "rev-list", "--left-right", "--count", "--end-of-options", defaultBranch+"..."+parts[0])
			if comparisonErr != nil {
				branch.ComparisonError = comparisonErr
			} else if values := strings.Fields(comparison); len(values) == 2 {
				branch.Behind, comparisonErr = strconv.Atoi(values[0])
				if comparisonErr == nil {
					branch.Ahead, comparisonErr = strconv.Atoi(values[1])
				}
				branch.ComparisonError = comparisonErr
			} else {
				branch.ComparisonError = fmt.Errorf("unexpected rev-list output: %q", comparison)
			}
		}
		branches = append(branches, branch)
	}
	return branches, nil
}

// getSubmodulePaths reads .gitmodules and returns a set of submodule paths.
func (g *ExecGit) getSubmodulePaths(ctx context.Context, repoPath string) map[string]bool {
	out, err := g.run(ctx, repoPath, "config", "--file", ".gitmodules", "--get-regexp", "^submodule\\..*\\.path$")
	if err != nil {
		return nil
	}
	paths := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
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
	if commonDir, err := g.commonGitDir(ctx, path); err == nil {
		info.CommonGitDir = commonDir
		if gitDir, err := g.gitDir(ctx, path); err == nil {
			info.LinkedWorktree = filepath.Clean(gitDir) != filepath.Clean(commonDir)
		}
	}

	info.Status = computeStatus(info)
	return info, nil
}

func (g *ExecGit) commonGitDir(ctx context.Context, path string) (string, error) {
	return g.run(ctx, path, "rev-parse", "--path-format=absolute", "--git-common-dir")
}

func (g *ExecGit) gitDir(ctx context.Context, path string) (string, error) {
	return g.run(ctx, path, "rev-parse", "--path-format=absolute", "--git-dir")
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
			r.branch = line[len("# branch.head "):]
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
			Author:  lines[i+2],
			Date:    date,
			Subject: lines[i+4],
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
