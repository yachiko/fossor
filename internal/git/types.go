package git

import "time"

// RepoStatus represents the high-level status of a repository.
type RepoStatus int

const (
	StatusUnknown    RepoStatus = iota
	StatusUpToDate              // on default branch, no ahead/behind, no changes
	StatusAhead                 // ahead of remote
	StatusBehind                // behind remote
	StatusDiverged              // both ahead and behind
	StatusNonDefault            // on a non-default branch
	StatusDirty                 // has uncommitted changes on default branch
	StatusError                 // something went wrong
)

func (s RepoStatus) String() string {
	switch s {
	case StatusUpToDate:
		return "Up to date"
	case StatusAhead:
		return "Ahead"
	case StatusBehind:
		return "Behind"
	case StatusDiverged:
		return "Diverged"
	case StatusNonDefault:
		return "Non-default"
	case StatusDirty:
		return "Dirty"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

// RepoInfo holds all the information about a single repository.
type RepoInfo struct {
	Name          string
	Path          string
	Branch        string
	DefaultBranch string
	Remote        string
	Ahead         int
	Behind        int
	Changes       int
	Status        RepoStatus
	Error         error
	// CommonGitDir identifies all checkouts that share refs and remotes.
	CommonGitDir   string
	LinkedWorktree bool
}

// CoordinatorKey identifies the shared git state that remote operations must
// serialize. Repositories without resolved metadata retain path-level safety.
func (r RepoInfo) CoordinatorKey() string {
	if r.CommonGitDir != "" {
		return r.CommonGitDir
	}
	return r.Path
}

// CommitInfo represents a single commit.
type CommitInfo struct {
	Hash    string
	Short   string
	Author  string
	Date    time.Time
	Subject string
}

// ChangeInfo represents a file change from git status.
type ChangeInfo struct {
	Staged      byte // first char of porcelain status
	Unstaged    byte // second char of porcelain status
	Path        string
	IsSubmodule bool
}

// StashInfo describes one entry in the stash reflog.
type StashInfo struct {
	Ref     string
	Message string
}

// BranchInfo describes a local branch and its relationship to the default
// branch. A comparison error makes only that comparison unavailable; the rest
// of the branch list remains usable.
type BranchInfo struct {
	Name            string
	IsCurrent       bool
	Merged          bool
	MergedError     error
	LastDate        string
	LastMsg         string
	Ahead           int
	Behind          int
	ComparisonError error
}
