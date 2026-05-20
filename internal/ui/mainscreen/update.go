package mainscreen

import (
	"context"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
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
		return tea.Quit

	case key.Matches(msg, common.MainKeys.Enter):
		if repo, ok := m.SelectedRepo(); ok {
			return func() tea.Msg { return common.SwitchToManageMsg{Repo: repo} }
		}

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
		vis := m.visibleIndices()
		if len(vis) > 0 {
			m.cursor = len(vis) - 1
		}

	case msg.String() == "g":
		m.cursor = 0
	}

	return nil
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

func (m *Model) pullSelected() tea.Cmd {
	repo, ok := m.SelectedRepo()
	if !ok {
		return nil
	}
	g := m.Git
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Pulling " + repo.Name + "..."}
		},
		func() tea.Msg {
			ctx := context.Background()
			output, err := g.Pull(ctx, repo.Path)
			return common.OperationResultMsg{RepoName: repo.Name, Op: "pull", Output: output, Err: err}
		},
		func() tea.Msg {
			ctx := context.Background()
			updated, _ := g.GetRepoInfo(ctx, repo.Path)
			return common.RepoUpdatedMsg{Repo: updated}
		},
	)
}

func (m *Model) fetchSelected() tea.Cmd {
	repo, ok := m.SelectedRepo()
	if !ok {
		return nil
	}
	g := m.Git
	return tea.Sequence(
		func() tea.Msg {
			return common.StatusMsg{Text: "Fetching " + repo.Name + "..."}
		},
		func() tea.Msg {
			ctx := context.Background()
			err := g.Fetch(ctx, repo.Path)
			if err != nil {
				return common.OperationResultMsg{RepoName: repo.Name, Op: "fetch", Err: err}
			}
			return common.OperationResultMsg{RepoName: repo.Name, Op: "fetch"}
		},
		func() tea.Msg {
			ctx := context.Background()
			updated, _ := g.GetRepoInfo(ctx, repo.Path)
			return common.RepoUpdatedMsg{Repo: updated}
		},
	)
}

func (m *Model) switchDefaultSelected() tea.Cmd {
	repo, ok := m.SelectedRepo()
	if !ok {
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
			ctx := context.Background()
			_, err := g.SwitchBranch(ctx, repo.Path, repo.DefaultBranch)
			return common.OperationResultMsg{RepoName: repo.Name, Op: "switch", Err: err}
		},
		func() tea.Msg {
			ctx := context.Background()
			updated, _ := g.GetRepoInfo(ctx, repo.Path)
			return common.RepoUpdatedMsg{Repo: updated}
		},
	)
}

func (m *Model) switchDefaultAll() tea.Cmd {
	g := m.Git
	var repos []git.RepoInfo
	for _, idx := range m.visibleIndices() {
		r := m.Repos[idx]
		if r.Branch != r.DefaultBranch {
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
	var cmds []tea.Cmd
	for _, r := range repos {
		r := r
		cmds = append(cmds, func() tea.Msg {
			ctx := context.Background()
			_, err := g.SwitchBranch(ctx, r.Path, r.DefaultBranch)
			done := atomic.AddInt64(&counter, 1) == total
			return common.BulkOperationTickMsg{RepoName: r.Name, Op: "switch", Err: err, Done: done}
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
	indices := m.visibleIndices()
	if len(indices) == 0 {
		return nil
	}
	total := int64(len(indices))
	var counter int64
	g := m.Git
	var cmds []tea.Cmd
	for _, idx := range indices {
		r := m.Repos[idx]
		cmds = append(cmds, func() tea.Msg {
			ctx := context.Background()
			_, err := g.Pull(ctx, r.Path)
			done := atomic.AddInt64(&counter, 1) == total
			return common.BulkOperationTickMsg{RepoName: r.Name, Op: "pull", Err: err, Done: done}
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
	indices := m.visibleIndices()
	if len(indices) == 0 {
		return nil
	}
	total := int64(len(indices))
	var counter int64
	g := m.Git
	var cmds []tea.Cmd
	for _, idx := range indices {
		r := m.Repos[idx]
		cmds = append(cmds, func() tea.Msg {
			ctx := context.Background()
			err := g.Fetch(ctx, r.Path)
			done := atomic.AddInt64(&counter, 1) == total
			return common.BulkOperationTickMsg{RepoName: r.Name, Op: "fetch", Err: err, Done: done}
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
func RefreshRepoCmd(g git.Git, path string, fetch bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
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
	return RefreshRepoCmd(g, repo.Path, true)
}
