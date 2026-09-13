package manageview

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
)

const (
	TabStatus   = 0
	TabHistory  = 1
	TabStash    = 2
	TabBranches = 3
	NumTabs     = 4
)

var tabNames = [NumTabs]string{"Status", "History", "Stash", "Branches"}

type mode int

const (
	modeNormal mode = iota
	modeConfirm
	modeInput
	modeCommit // inline commit message editor
)

// Model is the manage screen model.
type Model struct {
	Repo                    git.RepoInfo
	Git                     git.Git
	Coordinator             *git.OperationCoordinator
	remote                  string
	verified                bool
	verifyAfterRefresh      bool
	remoteErrorAfterRefresh bool
	ctx                     context.Context
	sessionID               uint64

	changesRequest, diffRequest, commitsRequest, remoteRequest   uint64
	repoRequest, stashRequest, stashDiffRequest, branchesRequest uint64
	stagedDiffRequest                                            uint64

	activeTab int

	// Action system
	actions    []Action
	keyMap     map[string]int
	mode       mode
	pendingIdx int
	textInput  textinput.Model

	// Status tab: file panels
	changes    []git.ChangeInfo
	fileCursor int
	fileScroll int
	diffView   viewport.Model
	diffLoaded bool
	changesErr error
	diffErr    error

	// Status tab: last action
	lastAction string
	lastOutput string
	lastErr    error
	stashErr   error

	// Commit mode
	commitInput    textarea.Model
	commitDiffView viewport.Model

	// History tab
	commits       []git.CommitInfo
	commitsLoaded bool
	commitsView   viewport.Model
	commitsErr    error

	// Stash tab
	stashEntries    []git.StashInfo
	stashCursor     int
	stashScroll     int
	stashDiffView   viewport.Model
	stashDiffLoaded bool
	stashDiffErr    error

	// Branches tab
	branches          []git.BranchInfo
	branchesLoaded    bool
	branchCursor      int
	branchScroll      int
	branchInputMode   bool   // true when entering new branch name or rename
	branchInputAction string // "create" or "rename"
	branchInput       textinput.Model
	branchesErr       error

	width     int
	height    int
	statusMsg string
}

// Internal messages

type execFinishedMsg struct {
	action  string
	err     error
	session uint64
}

type remoteOperationReadyMsg struct {
	action  string
	cmd     *exec.Cmd
	release func()
	session uint64
}

type stashInfoMsg struct {
	stashes []git.StashInfo
	err     error
	session uint64
	path    string
	request uint64
}

type repoRefreshedMsg struct {
	repo        git.RepoInfo
	verified    bool
	remoteError bool
	err         error
	session     uint64
	path        string
	request     uint64
}

type changesLoadedMsg struct {
	changes []git.ChangeInfo
	err     error
	session uint64
	path    string
	request uint64
}

type diffLoadedMsg struct {
	path    string
	diff    string
	err     error
	session uint64
	request uint64
}

type commitsLoadedMsg struct {
	commits []git.CommitInfo
	err     error
	session uint64
	path    string
	request uint64
}

type remoteLoadedMsg struct {
	remote  string
	err     error
	session uint64
	path    string
	request uint64
}

type stashDiffLoadedMsg struct {
	diff    string
	err     error
	entry   string
	session uint64
	path    string
	request uint64
}

type stagedDiffLoadedMsg struct {
	diff    string
	err     error
	session uint64
	path    string
	request uint64
}

type branchesLoadedMsg struct {
	branches []git.BranchInfo
	err      error
	session  uint64
	path     string
	request  uint64
}

// New creates a new manage screen model.
func New(g git.Git, repo git.RepoInfo, verified ...bool) Model {
	isVerified := true
	if len(verified) > 0 {
		isVerified = verified[0]
	}
	return newModel(g, repo, isVerified, nil)
}

// NewWithCoordinator creates a manage view whose remote actions share the
// main screen's worktree-family coordinator.
func NewWithCoordinator(g git.Git, repo git.RepoInfo, verified bool, coordinator *git.OperationCoordinator, sessionID uint64) Model {
	m := newModel(g, repo, verified, coordinator)
	m.sessionID = sessionID
	return m
}

func newModel(g git.Git, repo git.RepoInfo, isVerified bool, coordinator *git.OperationCoordinator) Model {
	actions := AllActions()
	km := make(map[string]int, len(actions))
	for i, a := range actions {
		km[a.Key] = i
	}

	ti := textinput.New()
	ti.CharLimit = 120

	ci := textarea.New()
	ci.Placeholder = "Commit message..."
	ci.CharLimit = 0
	ci.SetHeight(5)

	bi := textinput.New()
	bi.Placeholder = "Branch name..."
	bi.CharLimit = 100

	return Model{
		Repo:           repo,
		Git:            g,
		Coordinator:    coordinator,
		verified:       isVerified,
		ctx:            context.Background(),
		actions:        actions,
		keyMap:         km,
		textInput:      ti,
		commitInput:    ci,
		branchInput:    bi,
		diffView:       viewport.New(80, 10),
		commitsView:    viewport.New(80, 20),
		stashDiffView:  viewport.New(80, 10),
		commitDiffView: viewport.New(80, 10),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.refreshStash(), m.loadChanges(), m.loadRemote())
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetStatus(msg string) {
	m.statusMsg = msg
}

func (m *Model) UpdateRepo(repo git.RepoInfo) {
	m.Repo = repo
}

// SetVerified updates remote verification supplied by the application shell.
func (m *Model) SetVerified(verified bool) { m.verified = verified }

// SetContext binds background loaders to the active manage-view session.
func (m *Model) SetContext(ctx context.Context, sessionID uint64) {
	m.ctx = ctx
	m.sessionID = sessionID
}

// Data loaders

func (m *Model) loadChanges() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	m.changesRequest++
	request, session := m.changesRequest, m.sessionID
	return func() tea.Msg {
		changes, err := g.GetChanges(m.ctx, path)
		return changesLoadedMsg{changes: changes, err: err, session: session, path: path, request: request}
	}
}

func (m *Model) loadDiff(change git.ChangeInfo) tea.Cmd {
	repoPath := m.Repo.Path
	filePath := change.DestinationPath
	isSubmodule := change.IsSubmodule
	m.diffRequest++
	request, session := m.diffRequest, m.sessionID
	return func() tea.Msg {
		var diff string
		if isSubmodule {
			// Show commit log range for submodule changes
			cmd := gitPathCmd(repoPath, []string{"diff", "--submodule=log", "HEAD"}, change.Pathspecs()...)
			out, _ := cmd.Output()
			diff = string(out)
		} else {
			cmd := gitPathCmd(repoPath, []string{"diff", "HEAD"}, change.Pathspecs()...)
			out, _ := cmd.Output()
			diff = string(out)
			if diff == "" {
				cmd = gitPathCmd(repoPath, []string{"diff", "--no-index", "/dev/null"}, filePath)
				out, _ = cmd.Output()
				diff = string(out)
			}
		}
		return diffLoadedMsg{path: filePath, diff: diff, session: session, request: request}
	}
}

func (m *Model) loadRemote() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	m.remoteRequest++
	request, session := m.remoteRequest, m.sessionID
	return func() tea.Msg {
		remote, err := g.GetRemote(m.ctx, path)
		return remoteLoadedMsg{remote: remote, err: err, session: session, path: path, request: request}
	}
}

func (m *Model) loadCommits() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	m.commitsRequest++
	request, session := m.commitsRequest, m.sessionID
	return func() tea.Msg {
		commits, err := g.GetLog(m.ctx, path, 50)
		return commitsLoadedMsg{commits: commits, err: err, session: session, path: path, request: request}
	}
}

func (m *Model) loadStashDiff(index int) tea.Cmd {
	repoPath := m.Repo.Path
	if index < 0 || index >= len(m.stashEntries) {
		return nil
	}
	entry := m.stashEntries[index]
	m.stashDiffRequest++
	request, session := m.stashDiffRequest, m.sessionID
	return func() tea.Msg {
		diff, err := m.Git.GetStashDiff(m.ctx, repoPath, entry.Ref)
		return stashDiffLoadedMsg{diff: diff, err: err, entry: entry.Ref, session: session, path: repoPath, request: request}
	}
}

func (m *Model) loadBranches() tea.Cmd {
	repoPath, defaultBranch := m.Repo.Path, m.Repo.DefaultBranch
	m.branchesRequest++
	request, session := m.branchesRequest, m.sessionID
	return func() tea.Msg {
		branches, err := m.Git.GetBranches(m.ctx, repoPath, defaultBranch)
		return branchesLoadedMsg{branches: branches, err: err, session: session, path: repoPath, request: request}
	}
}

func (m *Model) loadStagedDiff() tea.Cmd {
	repoPath := m.Repo.Path
	m.stagedDiffRequest++
	request, session := m.stagedDiffRequest, m.sessionID
	return func() tea.Msg {
		diff, err := m.Git.GetStagedDiff(m.ctx, repoPath)
		return stagedDiffLoadedMsg{diff: diff, err: err, session: session, path: repoPath, request: request}
	}
}

func (m *Model) refreshStash() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	m.stashRequest++
	request, session := m.stashRequest, m.sessionID
	return func() tea.Msg {
		stashes, err := g.GetStashes(m.ctx, path)
		return stashInfoMsg{stashes: stashes, err: err, session: session, path: path, request: request}
	}
}

func (m *Model) refreshRepo() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	verified := m.verifyAfterRefresh
	remoteError := m.remoteErrorAfterRefresh
	m.repoRequest++
	request, session := m.repoRequest, m.sessionID
	m.verifyAfterRefresh = false
	m.remoteErrorAfterRefresh = false
	return func() tea.Msg {
		updated, err := g.GetRepoInfo(m.ctx, path)
		return repoRefreshedMsg{repo: updated, err: err, verified: verified, remoteError: remoteError, session: session, path: path, request: request}
	}
}

func (m *Model) selectedFilePath() string {
	if len(m.changes) == 0 || m.fileCursor >= len(m.changes) {
		return ""
	}
	return m.changes[m.fileCursor].DestinationPath
}

func (m *Model) selectedChange() (git.ChangeInfo, bool) {
	if len(m.changes) == 0 || m.fileCursor >= len(m.changes) {
		return git.ChangeInfo{}, false
	}
	return m.changes[m.fileCursor], true
}

// renderCommits formats commit log for the History tab viewport.
func renderCommits(commits []git.CommitInfo) string {
	if len(commits) == 0 {
		return "  No commits found."
	}

	hashStyle := lipgloss.NewStyle().Foreground(common.ColorAccent)
	authorStyle := lipgloss.NewStyle().Foreground(common.ColorMuted)

	var b strings.Builder
	for _, c := range commits {
		fmt.Fprintf(&b, "  %s %s\n", hashStyle.Render(git.Sanitize(c.Short)), git.Sanitize(c.Subject))
		fmt.Fprintf(&b, "  %s  %s\n\n", authorStyle.Render(git.Sanitize(c.Author)), authorStyle.Render(c.Date.Format("2006-01-02 15:04")))
	}
	return b.String()
}
