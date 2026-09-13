package mainscreen

import (
	"context"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.searching {
		return m.updateSearch(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, common.MainKeys.Quit):
		return func() tea.Msg { return common.QuitMsg{} }

	case key.Matches(msg, common.MainKeys.Enter):
		if repo, ok := m.SelectedRepo(); ok {
			return func() tea.Msg { return common.SwitchToManageMsg{Repo: repo} }
		}

	case msg.String() == " ":
		m.toggleSelectedGroup()

	case key.Matches(msg, common.MainKeys.Pull):
		return m.pullSelected()

	case key.Matches(msg, common.MainKeys.PullAll):
		return m.pullAll()

	case key.Matches(msg, common.MainKeys.Fetch):
		return m.fetchSelected()

	case key.Matches(msg, common.MainKeys.FetchAll):
		return m.fetchAll()

	case key.Matches(msg, common.MainKeys.SwitchDefault):
		return m.switchDefaultSelected()

	case key.Matches(msg, common.MainKeys.SwitchDefaultAll):
		return m.switchDefaultAll()

	case key.Matches(msg, common.MainKeys.Open):
		return m.openSelected()

	case key.Matches(msg, common.MainKeys.Search):
		m.searching = true
		m.searchText.Focus()
		return m.searchText.Cursor.BlinkCmd()

	case key.Matches(msg, common.MainKeys.Filter):
		m.cycleFilter()
		m.refilter()
		m.clampCursor()

	case key.Matches(msg, common.MainKeys.Sort1):
		m.toggleSort(SortName)
	case key.Matches(msg, common.MainKeys.Sort2):
		m.toggleSort(SortBranch)
	case key.Matches(msg, common.MainKeys.Sort3):
		m.toggleSort(SortAhead)
	case key.Matches(msg, common.MainKeys.Sort4):
		m.toggleSort(SortBehind)
	case key.Matches(msg, common.MainKeys.Sort5):
		m.toggleSort(SortChanges)
	case key.Matches(msg, common.MainKeys.Sort6):
		m.toggleSort(SortStatus)

	case msg.String() == "up" || msg.String() == "k":
		m.cursor--
		m.clampCursor()

	case msg.String() == "down" || msg.String() == "j":
		m.cursor++
		m.clampCursor()

	case msg.String() == "pgup":
		m.cursor -= m.TableHeight()
		m.clampCursor()

	case msg.String() == "pgdown":
		m.cursor += m.TableHeight()
		m.clampCursor()

	case msg.String() == "G":
		rows := m.visibleRows()
		if len(rows) > 0 {
			m.cursor = len(rows) - 1
		}
		m.clampCursor()

	case msg.String() == "g":
		m.cursor = 0
		m.clampCursor()

	}

	return nil
}

func (m *Model) toggleSelectedGroup() bool {
	rows := m.visibleRows()
	if m.cursor >= len(rows) || rows[m.cursor].groupKey == "" || (rows[m.cursor].repoIndex >= 0 && !rows[m.cursor].groupRoot) {
		return false
	}
	key := rows[m.cursor].groupKey
	m.collapsed[key] = !m.collapsed[key]
	m.clampCursor()
	return true
}

func (m *Model) updateSearch(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.searching = false
			m.searchText.Blur()
			m.searchText.SetValue("")
			m.refilter()
			return nil
		case "enter":
			m.searching = false
			m.searchText.Blur()
			return nil
		}
	}
	var cmd tea.Cmd
	m.searchText, cmd = m.searchText.Update(msg)
	m.refilter()
	return cmd
}

func (m *Model) toggleSort(col SortColumn) {
	if m.sortCol == col {
		m.sortAsc = !m.sortAsc
	} else {
		m.sortCol = col
		m.sortAsc = true
	}
	m.refilter()
}

func (m *Model) openSelected() tea.Cmd {
	if m.OpenCmd == "" {
		return nil
	}
	repo, ok := m.SelectedRepo()
	if !ok {
		return nil
	}
	parts := strings.Fields(m.OpenCmd)
	if len(parts) == 0 {
		return nil
	}
	args := append(parts[1:], repo.Path)
	cmd := exec.Command(parts[0], args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return common.OperationResultMsg{RepoName: repo.Name, Op: "open", Err: err}
	})
}

func (m *Model) pullSelected() tea.Cmd {
	repo, ok := m.SelectedRepo()
	if !ok {
		return nil
	}
	g := m.Git
	var operationErr error
	var revision uint64
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Pulling " + repo.Name + "..."}
		},
		func() tea.Msg {
			ctx := m.operationContext()
			var output string
			var err error
			if m.Coordinator != nil {
				revision, _, err = m.Coordinator.Run(ctx, repo.CoordinatorKey(), true, func(ctx context.Context) error {
					output, err = g.Pull(ctx, repo.Path)
					return err
				})
			} else {
				output, err = g.Pull(ctx, repo.Path)
			}
			operationErr = err
			return common.OperationResultMsg{RepoName: repo.Name, Op: "pull", Output: output, Err: err}
		},
		func() tea.Msg {
			ctx := m.operationContext()
			updated, _ := g.GetRepoInfo(ctx, repo.Path)
			return common.RepoUpdatedMsg{Repo: updated, Verified: operationErr == nil, RemoteError: operationErr != nil, Revision: revision, Source: common.RepoUpdateUserAction}
		},
	)
}

func (m *Model) fetchSelected() tea.Cmd {
	repo, ok := m.SelectedRepo()
	if !ok {
		return nil
	}
	g := m.Git
	var operationErr error
	var revision uint64
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Fetching " + repo.Name + "..."}
		},
		func() tea.Msg {
			ctx := m.operationContext()
			var err error
			if m.Coordinator != nil {
				revision, _, err = m.Coordinator.Run(ctx, repo.CoordinatorKey(), true, func(ctx context.Context) error { return g.Fetch(ctx, repo.Path) })
			} else {
				err = g.Fetch(ctx, repo.Path)
			}
			operationErr = err
			if err != nil {
				return common.OperationResultMsg{RepoName: repo.Name, Op: "fetch", Err: err}
			}
			return common.OperationResultMsg{RepoName: repo.Name, Op: "fetch"}
		},
		func() tea.Msg {
			ctx := m.operationContext()
			updated, _ := g.GetRepoInfo(ctx, repo.Path)
			return common.RepoUpdatedMsg{Repo: updated, Verified: operationErr == nil, RemoteError: operationErr != nil, Revision: revision, Source: common.RepoUpdateUserAction}
		},
	)
}

func (m *Model) switchDefaultSelected() tea.Cmd {
	repo, ok := m.SelectedRepo()
	if !ok || !m.IsActionable(repo) {
		return nil
	}
	if repo.Branch == repo.DefaultBranch {
		return func() tea.Msg {
			return common.StatusMsg{Text: repo.Name + ": already on " + repo.DefaultBranch, AutoClear: true}
		}
	}
	g := m.Git
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Switching " + repo.Name + " to " + repo.DefaultBranch + "..."}
		},
		func() tea.Msg {
			ctx := m.operationContext()
			_, err := g.SwitchBranch(ctx, repo.Path, repo.DefaultBranch)
			return common.OperationResultMsg{RepoName: repo.Name, Op: "switch", Err: err}
		},
		func() tea.Msg {
			ctx := m.operationContext()
			updated, _ := g.GetRepoInfo(ctx, repo.Path)
			return common.RepoUpdatedMsg{Repo: updated, Source: common.RepoUpdateUserAction}
		},
	)
}

// bulkOpConcurrency caps how many parallel git subprocesses we spawn for
// pull-all / fetch-all / switch-all. Bounds fd / process pressure on large
// scans and shrinks the race window where stale .git/*.lock files can
// otherwise be left behind.
const bulkOpConcurrency = 8

func (m *Model) switchDefaultAll() tea.Cmd {
	g := m.Git
	var repos []git.RepoInfo
	for _, idx := range m.visibleIndices() {
		r := m.Repos[idx]
		if m.IsActionable(r) && r.Branch != r.DefaultBranch {
			repos = append(repos, r)
		}
	}
	if len(repos) == 0 {
		return func() tea.Msg {
			return common.StatusMsg{Text: "All visible repos already on default branch", AutoClear: true}
		}
	}
	total := int64(len(repos))
	var counter int64
	sem := make(chan struct{}, bulkOpConcurrency)
	var cmds []tea.Cmd
	for _, r := range repos {
		r := r
		cmds = append(cmds, func() tea.Msg {
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx := m.operationContext()
			_, err := g.SwitchBranch(ctx, r.Path, r.DefaultBranch)
			done := atomic.AddInt64(&counter, 1) == total
			return common.BulkOperationTickMsg{RepoName: r.Name, Path: r.Path, Op: "switch", Err: err, Done: done}
		})
	}
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Switching visible repos to default branch..."}
		},
		tea.Batch(cmds...),
	)
}

func (m *Model) pullAll() tea.Cmd {
	var repos []git.RepoInfo
	for _, idx := range m.visibleIndices() {
		r := m.Repos[idx]
		if m.IsActionable(r) {
			repos = append(repos, r)
		}
	}
	if len(repos) == 0 {
		return nil
	}
	total := int64(len(repos))
	var counter int64
	g := m.Git
	sem := make(chan struct{}, bulkOpConcurrency)
	var cmds []tea.Cmd
	for _, r := range repos {
		r := r
		cmds = append(cmds, func() tea.Msg {
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx := m.operationContext()
			var err error
			if m.Coordinator != nil {
				_, _, err = m.Coordinator.Run(ctx, r.CoordinatorKey(), true, func(ctx context.Context) error {
					_, err := g.Pull(ctx, r.Path)
					return err
				})
			} else {
				_, err = g.Pull(ctx, r.Path)
			}
			done := atomic.AddInt64(&counter, 1) == total
			return common.BulkOperationTickMsg{RepoName: r.Name, Path: r.Path, Op: "pull", Err: err, Done: done}
		})
	}
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Pulling visible repos..."}
		},
		tea.Batch(cmds...),
	)
}

func (m *Model) fetchAll() tea.Cmd {
	var repos []git.RepoInfo
	for _, idx := range m.visibleIndices() {
		r := m.Repos[idx]
		if m.IsActionable(r) {
			repos = append(repos, r)
		}
	}
	if len(repos) == 0 {
		return nil
	}
	total := int64(len(repos))
	var counter int64
	g := m.Git
	sem := make(chan struct{}, bulkOpConcurrency)
	var cmds []tea.Cmd
	for _, r := range repos {
		r := r
		cmds = append(cmds, func() tea.Msg {
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx := m.operationContext()
			var err error
			if m.Coordinator != nil {
				_, _, err = m.Coordinator.Run(ctx, r.CoordinatorKey(), true, func(ctx context.Context) error {
					return g.Fetch(ctx, r.Path)
				})
			} else {
				err = g.Fetch(ctx, r.Path)
			}
			done := atomic.AddInt64(&counter, 1) == total
			return common.BulkOperationTickMsg{RepoName: r.Name, Path: r.Path, Op: "fetch", Err: err, Done: done}
		})
	}
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Fetching visible repos..."}
		},
		tea.Batch(cmds...),
	)
}

// RefreshRepoCmd refreshes a single repo's status in the background. When
// fetch is true, runs git fetch before reading status; otherwise only re-reads
// local state. Use fetch=false after local operations where the remote is
// unchanged.
func RefreshRepoCmd(g git.Git, path string, fetch bool, contexts ...context.Context) tea.Cmd {
	ctx := context.Background()
	if len(contexts) > 0 && contexts[0] != nil {
		ctx = contexts[0]
	}
	return func() tea.Msg {
		if fetch {
			_ = g.Fetch(ctx, path)
		}
		info, err := g.GetRepoInfo(ctx, path)
		if err == nil {
			return common.RepoUpdatedMsg{Repo: info}
		}
		return nil
	}
}

// RefreshSelected fetches and refreshes the currently selected repo in the background.
func (m *Model) RefreshSelected(g git.Git) tea.Cmd {
	repo, ok := m.SelectedRepo()
	if !ok {
		return nil
	}
	return func() tea.Msg {
		ctx := m.operationContext()
		if m.Coordinator != nil {
			_, _, _ = m.Coordinator.Run(ctx, repo.CoordinatorKey(), false, func(ctx context.Context) error {
				return g.Fetch(ctx, repo.Path)
			})
		} else {
			_ = g.Fetch(ctx, repo.Path)
		}
		info, err := g.GetRepoInfo(ctx, repo.Path)
		if err == nil {
			return common.RepoUpdatedMsg{Repo: info}
		}
		return nil
	}
}
