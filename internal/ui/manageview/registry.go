package manageview

import (
	"os/exec"

	"github.com/yachiko/fossor/internal/git"
)

func always(_ git.RepoInfo) bool { return true }

// AllActions returns the complete action registry.
// To add a new action, add an Action{} literal here.
func AllActions() []Action {
	return []Action{
		// ── Remote ──────────────────────────────────────────
		{
			Key:      "p",
			Name:     "pull",
			Category: CatRemote,
			Enabled:  always,
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "pull")
			},
		},
		{
			Key:      "R",
			Name:     "pull --rebase",
			Category: CatRemote,
			Enabled:  always,
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "pull", "--rebase")
			},
		},
		{
			Key:      "u",
			Name:     "push",
			Category: CatRemote,
			Enabled:  func(r git.RepoInfo) bool { return r.Ahead > 0 },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "push")
			},
		},
		{
			Key:      "f",
			Name:     "fetch",
			Category: CatRemote,
			Enabled:  always,
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "fetch", "--prune")
			},
		},

		// ── Branch ──────────────────────────────────────────
		{
			Key:      "d",
			Name:     "switch default",
			Category: CatBranch,
			Enabled:  func(r git.RepoInfo) bool { return r.Branch != r.DefaultBranch },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitRefCmd(r.Path, []string{"switch"}, r.DefaultBranch)
			},
		},
		{
			Key:         "m",
			Name:        "merge",
			Category:    CatBranch,
			NeedsInput:  true,
			InputPrompt: "Branch to merge (empty = tracking):",
			Enabled:     func(r git.RepoInfo) bool { return r.Branch != r.DefaultBranch },
			BuildCmd: func(r git.RepoInfo, input string) *exec.Cmd {
				if input == "" {
					return gitCmd(r.Path, "merge")
				}
				return gitRefCmd(r.Path, []string{"merge"}, input)
			},
		},
		{
			Key:      "b",
			Name:     "rebase",
			Category: CatBranch,
			Enabled:  func(r git.RepoInfo) bool { return r.Branch != r.DefaultBranch },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				// `--` separator: prevents r.DefaultBranch from being
				// interpreted as a git rebase flag. Without it, a poisoned
				// .git/refs/remotes/origin/HEAD containing `--exec=<cmd>`
				// yields RCE on key `b`.
				return gitRefCmd(r.Path, []string{"rebase"}, r.DefaultBranch)
			},
		},
		{
			Key:      "B",
			Name:     "rebase -i",
			Category: CatBranch,
			Enabled:  func(r git.RepoInfo) bool { return r.Branch != r.DefaultBranch },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitRefCmd(r.Path, []string{"rebase", "-i"}, r.DefaultBranch)
			},
		},

		{
			Key:      "U",
			Name:     "submodule update",
			Category: CatBranch,
			Enabled:  always,
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "submodule", "update", "--init", "--recursive")
			},
		},

		// ── Changes ─────────────────────────────────────────
		{
			Key:      "s",
			Name:     "stash",
			Category: CatChanges,
			Enabled:  func(r git.RepoInfo) bool { return r.Changes > 0 },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "stash")
			},
		},
		{
			Key:      "S",
			Name:     "stash pop",
			Category: CatChanges,
			Enabled:  always,
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "stash", "pop")
			},
		},
		{
			Key:      "a",
			Name:     "stage all",
			Category: CatChanges,
			Enabled:  func(r git.RepoInfo) bool { return r.Changes > 0 },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "add", "-A")
			},
		},
		{
			Key:          "i",
			Name:         "stage selected",
			Category:     CatChanges,
			UsesSelected: true,
			Enabled:      func(r git.RepoInfo) bool { return r.Changes > 0 },
			BuildCmd: func(r git.RepoInfo, input string) *exec.Cmd {
				return gitCmd(r.Path, "add", "--", input)
			},
		},
		{
			Key:          "I",
			Name:         "unstage selected",
			Category:     CatChanges,
			UsesSelected: true,
			Enabled:      func(r git.RepoInfo) bool { return r.Changes > 0 },
			BuildCmd: func(r git.RepoInfo, input string) *exec.Cmd {
				return gitCmd(r.Path, "reset", "HEAD", "--", input)
			},
		},
		{
			Key:      "c",
			Name:     "commit",
			Category: CatChanges,
			Enabled:  func(r git.RepoInfo) bool { return r.Changes > 0 },
			BuildCmd: nil, // handled specially — opens inline commit mode
		},
		{
			Key:      "C",
			Name:     "commit (editor)",
			Category: CatChanges,
			Enabled:  func(r git.RepoInfo) bool { return r.Changes > 0 },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "commit")
			},
		},

		// ── History ─────────────────────────────────────────
		{
			Key:      "z",
			Name:     "reset --soft HEAD~1",
			Category: CatHistory,
			Enabled:  func(r git.RepoInfo) bool { return r.Ahead > 0 },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "reset", "--soft", "HEAD~1")
			},
		},
		{
			Key:       "Z",
			Name:      "reset --hard HEAD~1",
			Category:  CatHistory,
			Dangerous: true,
			Enabled:   func(r git.RepoInfo) bool { return r.Ahead > 0 },
			BuildCmd: func(r git.RepoInfo, _ string) *exec.Cmd {
				return gitCmd(r.Path, "reset", "--hard", "HEAD~1")
			},
		},
		{
			Key:         "k",
			Name:        "cherry-pick",
			Category:    CatHistory,
			NeedsInput:  true,
			InputPrompt: "Commit hash:",
			Enabled:     always,
			BuildCmd: func(r git.RepoInfo, input string) *exec.Cmd {
				return gitRefCmd(r.Path, []string{"cherry-pick"}, input)
			},
		},
	}
}
