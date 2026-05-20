package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
	"github.com/ahoma/fossor/internal/ui/mainscreen"
	"github.com/ahoma/fossor/internal/ui/manageview"
)

type screen int

const (
	screenMain screen = iota
	screenManage
)

// App is the root bubbletea model.
type App struct {
	git           git.Git
	rootDir       string
	recursive     bool
	noFetch       bool
	noAutoRefresh bool
	openCmd       string

	screen      screen
	mainScreen  mainscreen.Model
	manageModel *manageview.Model

	discovering bool
	discovered  int
	spinner     spinner.Model
	cancelCtx   context.CancelFunc

	width  int
	height int
}

// NewApp creates the root application model.
const autoRefreshInterval = 30 * time.Second

func NewApp(g git.Git, rootDir string, recursive, noFetch, noAutoRefresh bool, openCmd string) *App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(common.ColorAccent)

	return &App{
		git:           g,
		rootDir:       rootDir,
		recursive:     recursive,
		noFetch:       noFetch,
		noAutoRefresh: noAutoRefresh,
		openCmd:       openCmd,
		mainScreen:    mainscreen.New(g, rootDir, openCmd),
		spinner:       s,
	}
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{
		a.startDiscovery(),
		a.spinner.Tick,
	}
	if !a.noAutoRefresh {
		cmds = append(cmds, a.scheduleAutoRefresh())
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.mainScreen.SetSize(msg.Width, msg.Height)
		if a.manageModel != nil {
			a.manageModel.SetSize(msg.Width, msg.Height)
		}
		return a, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if a.cancelCtx != nil {
				a.cancelCtx()
			}
			return a, tea.Quit
		}

	case spinner.TickMsg:
		if a.discovering {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return a, cmd
		}

	case common.RepoDiscoveredMsg:
		a.discovered++
		a.mainScreen.UpdateRepo(msg.Repo)
		a.mainScreen.SetStatus(fmt.Sprintf("%s Scanning... (%d repos found)", a.spinner.View(), a.discovered))
		return a, waitForDiscovery(msg.Ch)

	case common.DiscoveryCompleteMsg:
		a.discovering = false
		a.mainScreen.SetStatus(fmt.Sprintf("Scan complete: %d repos", a.discovered))
		return a, a.scheduleClearStatus()

	case common.SwitchToManageMsg:
		fm := manageview.New(a.git, msg.Repo)
		fm.SetSize(a.width, a.height)
		a.manageModel = &fm
		a.screen = screenManage
		return a, fm.Init()

	case common.SwitchToMainMsg:
		// Local-only refresh: manage view actions already touched local state,
		// and no remote interaction is needed for the round-trip back.
		var refreshCmd tea.Cmd
		if a.manageModel != nil {
			refreshCmd = mainscreen.RefreshRepoCmd(a.git, a.manageModel.Repo.Path, false)
		}
		a.screen = screenMain
		a.manageModel = nil
		return a, refreshCmd

	case common.RepoUpdatedMsg:
		a.mainScreen.UpdateRepo(msg.Repo)
		if a.manageModel != nil && a.manageModel.Repo.Path == msg.Repo.Path {
			a.manageModel.UpdateRepo(msg.Repo)
		}
		return a, nil

	case common.OperationResultMsg:
		status := msg.Op + " " + msg.RepoName
		if msg.Err != nil {
			status += ": " + msg.Err.Error()
		} else {
			status += ": done"
		}
		a.mainScreen.SetStatus(status)
		if a.manageModel != nil {
			a.manageModel.SetStatus(status)
		}
		return a, a.scheduleClearStatus()

	case common.BulkOperationTickMsg:
		status := msg.Op + " " + msg.RepoName
		if msg.Err != nil {
			status += " (error)"
		} else {
			status += " done"
		}
		if msg.Done {
			status = fmt.Sprintf("Bulk %s complete", msg.Op)
		}
		a.mainScreen.SetStatus(status)
		g := a.git
		name := msg.RepoName
		done := msg.Done
		refreshCmd := func() tea.Msg {
			for _, r := range a.mainScreen.Repos {
				if r.Name == name {
					ctx := context.Background()
					updated, _ := g.GetRepoInfo(ctx, r.Path)
					return common.RepoUpdatedMsg{Repo: updated}
				}
			}
			return nil
		}
		if done {
			return a, tea.Batch(refreshCmd, a.scheduleClearStatus())
		}
		return a, refreshCmd

	case common.StatusMsg:
		a.mainScreen.SetStatus(msg.Text)
		if a.manageModel != nil {
			a.manageModel.SetStatus(msg.Text)
		}
		if msg.AutoClear {
			return a, a.scheduleClearStatus()
		}
		return a, nil

	case common.StatusClearMsg:
		a.mainScreen.SetStatus("")
		if a.manageModel != nil {
			a.manageModel.SetStatus("")
		}
		return a, nil

	case common.RefreshTickMsg:
		// Periodic background refresh — only when on main screen and not discovering
		if a.screen == screenMain && !a.discovering {
			cmd := a.mainScreen.RefreshSelected(a.git)
			return a, tea.Batch(cmd, a.scheduleAutoRefresh())
		}
		return a, a.scheduleAutoRefresh()
	}

	// Forward to active screen
	switch a.screen {
	case screenMain:
		cmd := a.mainScreen.Update(msg)
		return a, cmd
	case screenManage:
		if a.manageModel != nil {
			if handled, cmd := a.manageModel.HandleInternalMsg(msg); handled {
				return a, cmd
			}
			cmd := a.manageModel.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

func (a *App) View() string {
	switch a.screen {
	case screenManage:
		if a.manageModel != nil {
			return a.manageModel.View()
		}
	}
	return a.mainScreen.View()
}

func (a *App) startDiscovery() tea.Cmd {
	a.discovering = true
	a.discovered = 0

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelCtx = cancel

	opts := git.DiscoveryOptions{
		RootDir:   a.rootDir,
		Recursive: a.recursive,
		NoFetch:   a.noFetch,
		Git:       a.git,
	}

	ch := git.Discover(ctx, opts)
	return waitForDiscovery(ch)
}

func (a *App) scheduleAutoRefresh() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(time.Time) tea.Msg {
		return common.RefreshTickMsg{}
	})
}

func (a *App) scheduleClearStatus() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return common.StatusClearMsg{}
	})
}

// waitForDiscovery reads one result from the channel and returns it as a message.
func waitForDiscovery(ch <-chan git.DiscoveryResult) tea.Cmd {
	return func() tea.Msg {
		result, ok := <-ch
		if !ok {
			return common.DiscoveryCompleteMsg{}
		}
		return common.RepoDiscoveredMsg{Repo: result.Repo, Ch: ch}
	}
}
