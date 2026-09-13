package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
	"github.com/yachiko/fossor/internal/ui/mainscreen"
	"github.com/yachiko/fossor/internal/ui/manageview"
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

	discovering         bool
	discovered          int
	liveRepos           map[string]git.RepoInfo
	localRepos          map[string]git.RepoInfo
	localDone           bool
	refreshTotal        int
	refreshDone         int
	spinner             spinner.Model
	appCtx              context.Context
	cancelApp           context.CancelFunc
	cancelCtx           context.CancelFunc
	cancelManage        context.CancelFunc
	cancelled           bool
	coordinator         *git.OperationCoordinator
	manageSession       uint64
	discoveryGeneration uint64
	userUpdateDiscovery map[string]uint64

	width  int
	height int
}

// NewApp creates the root application model.
const autoRefreshInterval = 30 * time.Second

func NewApp(g git.Git, rootDir string, recursive, noFetch, noAutoRefresh bool, openCmd string) *App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(common.ColorAccent)

	coordinator := git.NewOperationCoordinator()
	appCtx, cancelApp := context.WithCancel(context.Background())
	a := &App{
		git:                 g,
		rootDir:             rootDir,
		recursive:           recursive,
		noFetch:             noFetch,
		noAutoRefresh:       noAutoRefresh,
		openCmd:             openCmd,
		coordinator:         coordinator,
		mainScreen:          mainscreen.New(g, rootDir, openCmd, coordinator),
		spinner:             s,
		appCtx:              appCtx,
		cancelApp:           cancelApp,
		userUpdateDiscovery: make(map[string]uint64),
	}
	a.mainScreen.SetContext(appCtx)
	return a
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
			a.cancelled = true
			if a.cancelApp != nil {
				a.cancelApp()
			}
			if a.cancelCtx != nil {
				a.cancelCtx()
			}
			return a, tea.Quit
		}

	case common.QuitMsg:
		a.cancelled = true
		if a.cancelApp != nil {
			a.cancelApp()
		}
		if a.cancelCtx != nil {
			a.cancelCtx()
		}
		return a, tea.Quit

	case spinner.TickMsg:
		if a.discovering {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return a, cmd
		}

	case common.RepoDiscoveredMsg:
		if msg.Local {
			superseded := a.discoverySuperseded(msg.Repo.Path, msg.DiscoveryGeneration)
			a.discovered++
			a.localRepos[msg.Repo.Path] = msg.Repo
			a.liveRepos[msg.Repo.Path] = msg.Repo
			if !superseded {
				a.mainScreen.UpdateRepo(msg.Repo)
				if a.noFetch {
					a.mainScreen.SetVerification(msg.Repo.Path, mainscreen.Verified)
				} else {
					a.refreshTotal++
					a.mainScreen.SetVerification(msg.Repo.Path, mainscreen.Unverified)
				}
				a.updateManageVerification(msg.Repo.Path)
			} else if !a.noFetch {
				a.refreshTotal++
			}
			a.mainScreen.SetStatus(a.scanStatus())
			return a, waitForDiscovery(msg.Ch, msg.DiscoveryGeneration)
		}
		a.refreshDone++
		// Discovery refreshes for sibling worktrees share one coordinator key.
		// A later sibling fetch advances that key's revision but must not discard
		// this checkout's completed status result.
		if !msg.Skipped && !a.discoverySuperseded(msg.Repo.Path, msg.DiscoveryGeneration) {
			a.liveRepos[msg.Repo.Path] = msg.Repo
			a.localRepos[msg.Repo.Path] = msg.Repo
			a.saveLocalRepos()
			a.mainScreen.UpdateRepo(msg.Repo)
			if msg.FetchErr != nil {
				a.mainScreen.SetVerification(msg.Repo.Path, mainscreen.RemoteError)
			} else {
				a.mainScreen.SetVerification(msg.Repo.Path, mainscreen.Verified)
			}
			a.updateManageVerification(msg.Repo.Path)
		}
		if a.localDone {
			a.mainScreen.SetStatus(a.refreshStatus())
		} else {
			a.mainScreen.SetStatus(a.scanStatus())
		}
		return a, waitForDiscovery(msg.Ch, msg.DiscoveryGeneration)

	case common.DiscoveryCompleteMsg:
		if msg.LocalDone && !a.cancelled {
			a.finishLocalScan()
			if !a.noFetch {
				a.mainScreen.SetStatus(a.refreshStatus())
			}
			return a, waitForDiscovery(msg.Ch, msg.DiscoveryGeneration)
		}
		if msg.Complete {
			if !a.localDone && !a.cancelled {
				a.finishLocalScan()
			}
			a.discovering = false
			a.mainScreen.SetStatus(fmt.Sprintf("Scan complete: %d repos", a.discovered))
			return a, a.scheduleClearStatus()
		}

	case common.SwitchToManageMsg:
		if a.cancelManage != nil {
			a.cancelManage()
		}
		a.manageSession++
		manageCtx, cancelManage := context.WithCancel(a.appCtx)
		a.cancelManage = cancelManage
		fm := manageview.NewWithCoordinator(a.git, msg.Repo, a.mainScreen.Verification(msg.Repo.Path) == mainscreen.Verified, a.coordinator, a.manageSession)
		fm.SetContext(manageCtx, a.manageSession)
		fm.SetSize(a.width, a.height)
		a.manageModel = &fm
		a.screen = screenManage
		return a, fm.Init()

	case common.SwitchToMainMsg:
		// Local-only refresh: manage view actions already touched local state,
		// and no remote interaction is needed for the round-trip back.
		var refreshCmd tea.Cmd
		if a.manageModel != nil {
			refreshCmd = mainscreen.RefreshRepoCmd(a.git, a.manageModel.Repo.Path, false, a.appCtx)
		}
		if a.cancelManage != nil {
			a.cancelManage()
			a.cancelManage = nil
		}
		a.screen = screenMain
		a.manageModel = nil
		return a, refreshCmd

	case common.RepoUpdatedMsg:
		if msg.Source == common.RepoUpdateUserAction {
			a.userUpdateDiscovery[msg.Repo.Path] = a.discoveryGeneration
		}
		a.mainScreen.UpdateRepo(msg.Repo)
		if msg.RemoteError {
			a.mainScreen.SetVerification(msg.Repo.Path, mainscreen.RemoteError)
		} else if msg.Verified {
			a.mainScreen.SetVerification(msg.Repo.Path, mainscreen.Verified)
		}
		if a.manageModel != nil && a.manageModel.Repo.Path == msg.Repo.Path {
			a.manageModel.UpdateRepo(msg.Repo)
			a.updateManageVerification(msg.Repo.Path)
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
		path := msg.Path
		done := msg.Done
		refreshCmd := func() tea.Msg {
			for _, r := range a.mainScreen.Repos {
				if r.Path == path {
					ctx := a.appCtx
					updated, _ := g.GetRepoInfo(ctx, r.Path)
					return common.RepoUpdatedMsg{Repo: updated, Source: common.RepoUpdateUserAction}
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
		if a.refreshDone < a.refreshTotal {
			return a, nil
		}
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
	a.discoveryGeneration++
	a.discovering = true
	a.discovered = 0
	a.cancelled = false
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	a.localDone = false
	a.refreshTotal = 0
	a.refreshDone = 0

	ctx, cancel := context.WithCancel(a.appCtx)
	a.cancelCtx = cancel
	cachedRepos := make(map[string]git.RepoInfo)
	for _, repo := range git.LoadDiscoveryCache(a.rootDir, a.recursive) {
		cachedRepos[repo.Path] = repo
		a.mainScreen.AddCachedRepo(repo)
	}

	opts := git.DiscoveryOptions{
		RootDir:     a.rootDir,
		Recursive:   a.recursive,
		Git:         a.git,
		Fetch:       !a.noFetch,
		Coordinator: a.coordinator,
		CachedRepos: cachedRepos,
	}

	ch := git.Discover(ctx, opts)
	return waitForDiscovery(ch, a.discoveryGeneration)
}

func (a *App) updateManageVerification(path string) {
	if a.manageModel != nil && a.manageModel.Repo.Path == path {
		a.manageModel.SetVerified(a.mainScreen.Verification(path) == mainscreen.Verified)
	}
}

func (a *App) discoverySuperseded(path string, generation uint64) bool {
	updatedGeneration, ok := a.userUpdateDiscovery[path]
	return ok && updatedGeneration == generation
}

func (a *App) refreshStatus() string {
	return fmt.Sprintf("%s Refreshing remotes: %d/%d", a.spinner.View(), a.refreshDone, a.refreshTotal)
}

func (a *App) scanStatus() string {
	return fmt.Sprintf("%s Scanning... (%d repos found; refreshing %d/%d)", a.spinner.View(), a.discovered, a.refreshDone, a.refreshTotal)
}

func (a *App) finishLocalScan() {
	a.localDone = true
	paths := make(map[string]bool, len(a.localRepos))
	for path := range a.localRepos {
		paths[path] = true
	}
	a.mainScreen.Prune(paths)
	a.saveLocalRepos()
}

func (a *App) saveLocalRepos() {
	repos := make([]git.RepoInfo, 0, len(a.localRepos))
	for _, repo := range a.localRepos {
		repos = append(repos, repo)
	}
	git.SaveDiscoveryCache(a.rootDir, a.recursive, repos)
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
func waitForDiscovery(ch <-chan git.DiscoveryResult, generations ...uint64) tea.Cmd {
	generation := uint64(0)
	if len(generations) > 0 {
		generation = generations[0]
	}
	return func() tea.Msg {
		result, ok := <-ch
		if !ok {
			return common.DiscoveryCompleteMsg{Complete: true, DiscoveryGeneration: generation}
		}
		if result.LocalDone {
			return common.DiscoveryCompleteMsg{LocalDone: true, DiscoveryGeneration: generation, Ch: ch}
		}
		return common.RepoDiscoveredMsg{Repo: result.Repo, FetchErr: result.FetchErr, Path: result.Path, Skipped: result.Skipped, Revision: result.Revision, DiscoveryGeneration: generation, Local: result.Local, Ch: ch}
	}
}
