package fixscreen

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
)

const (
	TabStatus  = 0
	TabHistory = 1
	TabStash   = 2
	NumTabs    = 3
)

var tabNames = [NumTabs]string{"Status", "History", "Stash"}

type mode int

const (
	modeNormal  mode = iota
	modeConfirm
	modeInput
)

// Model is the manage screen model.
type Model struct {
	Repo   git.RepoInfo
	Git    git.Git
	remote string

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

	// Status tab: last action
	lastAction string
	lastOutput string
	lastErr    error
	stashInfo  string

	// History tab
	commits       []git.CommitInfo
	commitsLoaded bool
	commitsView   viewport.Model

	// Stash tab
	stashEntries    []string
	stashCursor     int
	stashScroll     int
	stashDiffView   viewport.Model
	stashDiffLoaded bool

	width     int
	height    int
	statusMsg string
}

// Internal messages

type execFinishedMsg struct {
	action string
	err    error
}

type stashInfoMsg struct {
	info string
}

type repoRefreshedMsg struct {
	repo git.RepoInfo
}

type changesLoadedMsg struct {
	changes []git.ChangeInfo
}

type diffLoadedMsg struct {
	path string
	diff string
}

type commitsLoadedMsg struct {
	commits []git.CommitInfo
}

type remoteLoadedMsg struct {
	remote string
}

type stashDiffLoadedMsg struct {
	diff string
}

// New creates a new manage screen model.
func New(g git.Git, repo git.RepoInfo) Model {
	actions := AllActions()
	km := make(map[string]int, len(actions))
	for i, a := range actions {
		km[a.Key] = i
	}

	ti := textinput.New()
	ti.CharLimit = 120

	return Model{
		Repo:          repo,
		Git:           g,
		actions:       actions,
		keyMap:        km,
		textInput:     ti,
		diffView:      viewport.New(80, 10),
		commitsView:   viewport.New(80, 20),
		stashDiffView: viewport.New(80, 10),
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

// Data loaders

func (m *Model) loadChanges() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	return func() tea.Msg {
		ctx := context.Background()
		changes, _ := g.GetChanges(ctx, path)
		return changesLoadedMsg{changes: changes}
	}
}

func (m *Model) loadDiff(change git.ChangeInfo) tea.Cmd {
	repoPath := m.Repo.Path
	filePath := change.Path
	isSubmodule := change.IsSubmodule
	return func() tea.Msg {
		var diff string
		if isSubmodule {
			// Show commit log range for submodule changes
			cmd := exec.Command("git", "-C", repoPath, "diff", "--submodule=log", "HEAD", "--", filePath)
			out, _ := cmd.Output()
			diff = string(out)
		} else {
			cmd := exec.Command("git", "-C", repoPath, "diff", "HEAD", "--", filePath)
			out, _ := cmd.Output()
			diff = string(out)
			if diff == "" {
				cmd = exec.Command("git", "-C", repoPath, "diff", "--no-index", "--", "/dev/null", filePath)
				out, _ = cmd.Output()
				diff = string(out)
			}
		}
		return diffLoadedMsg{path: filePath, diff: diff}
	}
}

func (m *Model) loadRemote() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	return func() tea.Msg {
		ctx := context.Background()
		remote, _ := g.GetRemote(ctx, path)
		return remoteLoadedMsg{remote: remote}
	}
}

func (m *Model) loadCommits() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	return func() tea.Msg {
		ctx := context.Background()
		commits, _ := g.GetLog(ctx, path, 50)
		return commitsLoadedMsg{commits: commits}
	}
}

func (m *Model) loadStashDiff(index int) tea.Cmd {
	repoPath := m.Repo.Path
	return func() tea.Msg {
		ref := fmt.Sprintf("stash@{%d}", index)
		cmd := exec.Command("git", "-C", repoPath, "stash", "show", "-p", ref)
		out, _ := cmd.Output()
		return stashDiffLoadedMsg{diff: string(out)}
	}
}

func (m *Model) refreshStash() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	return func() tea.Msg {
		ctx := context.Background()
		out, _ := g.RunCommand(ctx, path, "stash", "list")
		return stashInfoMsg{info: out}
	}
}

func (m *Model) refreshRepo() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	return func() tea.Msg {
		ctx := context.Background()
		updated, _ := g.GetRepoInfo(ctx, path)
		return repoRefreshedMsg{repo: updated}
	}
}

func (m *Model) selectedFilePath() string {
	if len(m.changes) == 0 || m.fileCursor >= len(m.changes) {
		return ""
	}
	return m.changes[m.fileCursor].Path
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
		b.WriteString(fmt.Sprintf("  %s %s\n", hashStyle.Render(c.Short), c.Subject))
		b.WriteString(fmt.Sprintf("  %s  %s\n\n", authorStyle.Render(c.Author), authorStyle.Render(c.Date.Format("2006-01-02 15:04"))))
	}
	return b.String()
}
