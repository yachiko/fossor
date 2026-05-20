package common

import "github.com/ahoma/fossor/internal/git"

// RepoDiscoveredMsg is sent when a repo is discovered during scanning.
// It carries the channel so the listener can re-subscribe.
type RepoDiscoveredMsg struct {
	Repo git.RepoInfo
	Ch   <-chan git.DiscoveryResult
}

// DiscoveryCompleteMsg is sent when repo discovery finishes.
type DiscoveryCompleteMsg struct{}

// RepoUpdatedMsg is sent after a repo operation completes (pull, fetch, etc).
type RepoUpdatedMsg struct {
	Repo git.RepoInfo
}

// OperationResultMsg carries the result of a git operation for display.
type OperationResultMsg struct {
	RepoName string
	Op       string
	Output   string
	Err      error
}

// BulkOperationTickMsg is sent as each bulk operation completes.
type BulkOperationTickMsg struct {
	RepoName string
	Op       string
	Err      error
	Done     bool // true when this is the last operation in the batch
}

// StatusClearMsg clears the status bar message after a delay.
type StatusClearMsg struct{}

// SwitchToManageMsg requests navigation to the manage screen.
type SwitchToManageMsg struct {
	Repo git.RepoInfo
}

// SwitchToMainMsg requests navigation back to the main screen.
type SwitchToMainMsg struct{}

// StatusMsg sets a transient status message in the status bar.
type StatusMsg struct {
	Text      string
	AutoClear bool // if true, message auto-clears after a few seconds
}

// RefreshTickMsg triggers a periodic background refresh of visible repos.
type RefreshTickMsg struct{}
