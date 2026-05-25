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
	Key          string                                          // keybinding
	Name         string                                          // human label
	Category     Category                                        // grouping
	Dangerous    bool                                            // requires y/n confirmation
	NeedsInput   bool                                            // prompts for text input first
	InputPrompt  string                                          // prompt text when NeedsInput
	UsesSelected bool                                            // passes selected file path as input
	Enabled      func(git.RepoInfo) bool                         // enable condition
	BuildCmd     func(repo git.RepoInfo, input string) *exec.Cmd // builds the command
}

// gitCmd builds an exec.Cmd for a git command in the given repo path.
func gitCmd(path string, args ...string) *exec.Cmd {
	return exec.Command("git", append([]string{"-C", path}, args...)...)
}
