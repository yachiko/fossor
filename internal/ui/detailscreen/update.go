package detailscreen

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
	"github.com/ahoma/fossor/internal/ui/detailscreen/tabs"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// CLI tab gets first crack at input when focused
	if m.activeTab == TabCLI && m.cliTab.IsFocused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.cliTab.Blur()
				return nil
			}
			if msg.String() == "shift+tab" {
				m.switchTab((m.activeTab - 1 + NumTabs) % NumTabs)
				return m.onTabSwitch()
			}
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				// don't quit from CLI input
				if msg.String() == "ctrl+c" {
					return tea.Quit
				}
				// let 'q' go to the input
			}
		case tabs.CLICmdMsg:
			return m.runCLICommand(msg)
		case common.CLIOutputMsg:
			// handled in app.go
			return nil
		}
		cmd := m.cliTab.Update(msg)
		return cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tabs.CLICmdMsg:
		return m.runCLICommand(msg)
	}
	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, common.DetailKeys.Quit):
		return tea.Quit

	case key.Matches(msg, common.DetailKeys.Back):
		return func() tea.Msg { return common.SwitchToMainMsg{} }

	case key.Matches(msg, common.DetailKeys.Tab):
		m.switchTab((m.activeTab + 1) % NumTabs)
		return m.onTabSwitch()

	case key.Matches(msg, common.DetailKeys.ShiftTab):
		m.switchTab((m.activeTab - 1 + NumTabs) % NumTabs)
		return m.onTabSwitch()

	case key.Matches(msg, common.DetailKeys.Tab1):
		m.switchTab(TabInfo)
		return m.onTabSwitch()
	case key.Matches(msg, common.DetailKeys.Tab2):
		m.switchTab(TabCommits)
		return m.onTabSwitch()
	case key.Matches(msg, common.DetailKeys.Tab3):
		m.switchTab(TabChanges)
		return m.onTabSwitch()
	case key.Matches(msg, common.DetailKeys.Tab4):
		m.switchTab(TabCLI)
		return m.onTabSwitch()

	case key.Matches(msg, common.DetailKeys.Pull):
		return m.pull()

	case key.Matches(msg, common.DetailKeys.Push):
		return m.push()

	default:
		// Forward viewport keys for scrollable tabs
		switch m.activeTab {
		case TabCommits:
			var cmd tea.Cmd
			m.commitsTab.Viewport, cmd = m.commitsTab.Viewport.Update(msg)
			return cmd
		case TabChanges:
			var cmd tea.Cmd
			m.changesTab.Viewport, cmd = m.changesTab.Viewport.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *Model) switchTab(tab int) {
	if m.activeTab == TabCLI {
		m.cliTab.Blur()
	}
	m.activeTab = tab
}

func (m *Model) onTabSwitch() tea.Cmd {
	if m.activeTab == TabCLI {
		return m.cliTab.Focus()
	}
	return m.loadTabData()
}

func (m *Model) loadTabData() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	switch m.activeTab {
	case TabInfo:
		if m.Repo.Remote == "" {
			return func() tea.Msg {
				ctx := context.Background()
				remote, _ := g.GetRemote(ctx, path)
				return remoteLoadedMsg{remote: remote}
			}
		}
	case TabCommits:
		return func() tea.Msg {
			ctx := context.Background()
			commits, err := g.GetLog(ctx, path, 50)
			if err != nil {
				return commitsLoadedMsg{err: err}
			}
			return commitsLoadedMsg{commits: commits}
		}
	case TabChanges:
		return func() tea.Msg {
			ctx := context.Background()
			changes, err := g.GetChanges(ctx, path)
			if err != nil {
				return changesLoadedMsg{err: err}
			}
			diffStat, _ := g.RunCommand(ctx, path, "diff", "--stat")
			return changesLoadedMsg{changes: changes, diffStat: diffStat}
		}
	}
	return nil
}

func (m *Model) pull() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	name := m.Repo.Name
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Pulling " + name + "..."}
		},
		func() tea.Msg {
			ctx := context.Background()
			output, err := g.Pull(ctx, path)
			return common.OperationResultMsg{RepoName: name, Op: "pull", Output: output, Err: err}
		},
		func() tea.Msg {
			ctx := context.Background()
			updated, _ := g.GetRepoInfo(ctx, path)
			return common.RepoUpdatedMsg{Repo: updated}
		},
	)
}

func (m *Model) push() tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	name := m.Repo.Name
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Pushing " + name + "..."}
		},
		func() tea.Msg {
			ctx := context.Background()
			output, err := g.Push(ctx, path)
			return common.OperationResultMsg{RepoName: name, Op: "push", Output: output, Err: err}
		},
		func() tea.Msg {
			ctx := context.Background()
			updated, _ := g.GetRepoInfo(ctx, path)
			return common.RepoUpdatedMsg{Repo: updated}
		},
	)
}

func (m *Model) runCLICommand(msg tabs.CLICmdMsg) tea.Cmd {
	g := m.Git
	path := m.Repo.Path
	raw := msg.Raw
	args := msg.Args
	isGit := msg.IsGit
	return func() tea.Msg {
		ctx := context.Background()
		var output string
		var err error
		if isGit {
			output, err = g.RunCommand(ctx, path, args...)
		} else {
			if len(args) == 0 {
				return cliResultMsg{raw: raw, err: fmt.Errorf("empty command")}
			}
			output, err = g.RunShellCommand(ctx, path, args[0], args[1:]...)
		}
		return cliResultMsg{raw: raw, output: output, err: err}
	}
}

// Internal messages for tab data loading
type commitsLoadedMsg struct {
	commits []git.CommitInfo
	err     error
}

type changesLoadedMsg struct {
	changes  []git.ChangeInfo
	diffStat string
	err      error
}

type remoteLoadedMsg struct {
	remote string
}

type cliResultMsg struct {
	raw    string
	output string
	err    error
}

// HandleInternalMsg processes internal messages. Returns true if handled.
func (m *Model) HandleInternalMsg(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case remoteLoadedMsg:
		m.Repo.Remote = msg.remote
		return true
	case commitsLoadedMsg:
		if msg.err == nil {
			m.commitsTab.SetCommits(msg.commits)
		}
		return true
	case changesLoadedMsg:
		if msg.err == nil {
			m.changesTab.SetChanges(msg.changes, msg.diffStat)
		}
		return true
	case cliResultMsg:
		m.cliTab.AppendOutput(msg.raw, msg.output, msg.err)
		return true
	}
	return false
}
