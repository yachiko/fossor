package manageview

import (
	"os/exec"

	"github.com/yachiko/fossor/internal/git"
)

// Category groups actions in the grid view.
type Category int

const (
	CatRemote Category = iota
	CatBranch
	CatChanges
	CatHistory
)

var categoryNames = [...]string{"Remote", "Branch", "Changes", "History"}

func (c Category) String() string { return categoryNames[c] }

// AllCategories returns categories in display order.
func AllCategories() []Category {
	return []Category{CatRemote, CatBranch, CatChanges, CatHistory}
}

// Action defines a single keybinding-driven git operation.
type Action struct {
	Key              string                                          // keybinding
	Name             string                                          // human label
	Category         Category                                        // grouping
	Dangerous        bool                                            // requires y/n confirmation
	NeedsInput       bool                                            // prompts for text input first
	InputPrompt      string                                          // prompt text when NeedsInput
	UsesSelected     bool                                            // passes the selected change to BuildSelectedCmd
	Enabled          func(git.RepoInfo) bool                         // enable condition
	BuildCmd         func(repo git.RepoInfo, input string) *exec.Cmd // builds the command
	BuildSelectedCmd func(repo git.RepoInfo, change git.ChangeInfo) *exec.Cmd
}

// gitCmd builds an exec.Cmd for a git command in the given repo path.
func gitCmd(path string, args ...string) *exec.Cmd {
	return git.Command(path, args...)
}

// gitRefCmd builds an exec.Cmd for a git command that takes a single
// repo-controlled or user-controlled refspec / path. The refspec is appended
// after a `--` separator so a leading `-` cannot turn the value into a git
// flag (e.g. `--exec=…` in git rebase, which would otherwise enable RCE).
func gitRefCmd(path string, subArgs []string, ref string) *exec.Cmd {
	all := append([]string{}, subArgs...)
	all = append(all, "--", ref)
	return git.Command(path, all...)
}

// gitPathCmd appends literal pathspecs. `--` only ends option parsing; without
// `:(literal)`, Git still interprets glob and magic pathspec characters.
func gitPathCmd(path string, subArgs []string, paths ...string) *exec.Cmd {
	args := append([]string{}, subArgs...)
	args = append(args, "--")
	for _, path := range paths {
		args = append(args, ":(literal)"+path)
	}
	return gitCmd(path, args...)
}

// restoreChangeCmds restores a rename's source from the index, then removes its
// destination. Passing both endpoints to checkout fails because the destination
// does not exist in the index for an unstaged rename.
func restoreChangeCmds(repoPath string, change git.ChangeInfo) []*exec.Cmd {
	if change.SourcePath == "" {
		return []*exec.Cmd{gitPathCmd(repoPath, []string{"checkout"}, change.DestinationPath)}
	}
	return []*exec.Cmd{
		gitPathCmd(repoPath, []string{"checkout"}, change.SourcePath),
		gitPathCmd(repoPath, []string{"clean", "-f"}, change.DestinationPath),
	}
}
