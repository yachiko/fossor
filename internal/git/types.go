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
	Staged   byte // first char of porcelain status
	Unstaged byte // second char of porcelain status
	Path     string
}
