package mainscreen

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"

	"github.com/ahoma/fossor/internal/git"
)

// SortColumn identifies which column to sort by.
type SortColumn int

const (
	SortName SortColumn = iota
	SortBranch
	SortAhead
	SortBehind
	SortChanges
	SortStatus
)

// FilterMode controls which repos are visible.
type FilterMode int

const (
	FilterAll        FilterMode = iota
	FilterError                 // Only repos with StatusError
	FilterNonDefault            // Only repos with StatusNonDefault
	FilterDiverged              // Only repos with StatusDiverged
	FilterBehind                // Only repos with StatusBehind
	FilterAhead                 // Only repos with StatusAhead
	FilterDirty                 // Only repos with StatusDirty
	FilterUpToDate              // Only repos with StatusUpToDate
	filterModeCount             // sentinel for cycling
)

var filterModeNames = [filterModeCount]string{"All", "Error", "Non-default", "Diverged", "Behind", "Ahead", "Dirty", "Up to date"}

func (f FilterMode) String() string {
	if int(f) < len(filterModeNames) {
		return filterModeNames[f]
	}
	return "All"
}

// Model is the main screen model.
type Model struct {
	Repos        []git.RepoInfo
	Git          git.Git
	RootDir      string
	cursor       int
	scrollOffset int
	sortCol      SortColumn
	sortAsc      bool
	searching    bool
	searchText   textinput.Model
	filtered     []int // indices into Repos
	filterMode   FilterMode
	width        int
	height       int
	statusMsg    string
}

// New creates a new main screen model.
func New(g git.Git, rootDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "Search repos..."
	ti.CharLimit = 100

	return Model{
		Git:        g,
		RootDir:    rootDir,
		sortCol:    SortName,
		sortAsc:    true,
		searchText: ti,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetStatus(msg string) {
	m.statusMsg = msg
}

// TableHeight returns the number of visible table rows.
func (m *Model) TableHeight() int {
	// title(2) + header(1) + top_sep(1) + bottom_sep(1) + statusbar(2) + padding(1) = 8
	h := m.height - 8
	if m.searching || m.searchText.Value() != "" {
		h -= 2
	}
	if m.filterMode != FilterAll {
		h -= 1
	}
	if h < 1 {
		h = 1
	}
	return h
}

// clampScroll adjusts scrollOffset so the cursor is always visible.
func (m *Model) clampScroll() {
	th := m.TableHeight()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+th {
		m.scrollOffset = m.cursor - th + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// UpdateRepo updates or inserts a repo in the list.
func (m *Model) UpdateRepo(repo git.RepoInfo) {
	for i, r := range m.Repos {
		if r.Path == repo.Path {
			m.Repos[i] = repo
			m.refilter()
			return
		}
	}
	m.Repos = append(m.Repos, repo)
	m.refilter()
}

// SelectedRepo returns the currently selected repo, if any.
func (m *Model) SelectedRepo() (git.RepoInfo, bool) {
	indices := m.visibleIndices()
	if len(indices) == 0 || m.cursor >= len(indices) {
		return git.RepoInfo{}, false
	}
	return m.Repos[indices[m.cursor]], true
}

func (m *Model) visibleIndices() []int {
	if m.filtered != nil {
		return m.filtered
	}
	idx := make([]int, len(m.Repos))
	for i := range idx {
		idx[i] = i
	}
	return idx
}

func (m *Model) refilter() {
	m.sortRepos()

	query := strings.ToLower(m.searchText.Value())

	if query == "" && m.filterMode == FilterAll {
		m.filtered = nil
		return
	}

	m.filtered = []int{}
	for i, r := range m.Repos {
		if !m.matchesFilter(r) {
			continue
		}
		if query != "" {
			if !strings.Contains(strings.ToLower(r.Name), query) &&
				!strings.Contains(strings.ToLower(r.Branch), query) &&
				!strings.Contains(strings.ToLower(r.Status.String()), query) {
				continue
			}
		}
		m.filtered = append(m.filtered, i)
	}
	if m.cursor >= len(m.visibleIndices()) {
		m.cursor = max(0, len(m.visibleIndices())-1)
	}
}

func (m *Model) matchesFilter(r git.RepoInfo) bool {
	switch m.filterMode {
	case FilterError:
		return r.Status == git.StatusError
	case FilterNonDefault:
		return r.Status == git.StatusNonDefault
	case FilterDiverged:
		return r.Status == git.StatusDiverged
	case FilterBehind:
		return r.Status == git.StatusBehind
	case FilterAhead:
		return r.Status == git.StatusAhead
	case FilterDirty:
		return r.Status == git.StatusDirty
	case FilterUpToDate:
		return r.Status == git.StatusUpToDate
	default:
		return true
	}
}

// cycleFilter advances to the next filter mode that has repos, or back to All.
func (m *Model) cycleFilter() {
	counts := make(map[git.RepoStatus]int)
	for _, r := range m.Repos {
		counts[r.Status]++
	}

	// Filter modes in cycle order, mapped to their required status.
	type entry struct {
		mode   FilterMode
		status git.RepoStatus
	}
	order := []entry{
		{FilterError, git.StatusError},
		{FilterNonDefault, git.StatusNonDefault},
		{FilterDiverged, git.StatusDiverged},
		{FilterBehind, git.StatusBehind},
		{FilterAhead, git.StatusAhead},
		{FilterDirty, git.StatusDirty},
		{FilterUpToDate, git.StatusUpToDate},
	}

	// Find current position in order, then scan forward for next non-empty.
	start := 0
	for i, e := range order {
		if e.mode == m.filterMode {
			start = i + 1
			break
		}
	}
	for i := 0; i < len(order); i++ {
		e := order[(start+i)%len(order)]
		if counts[e.status] > 0 {
			m.filterMode = e.mode
			return
		}
	}
	m.filterMode = FilterAll
}

func (m *Model) sortRepos() {
	sort.SliceStable(m.Repos, func(i, j int) bool {
		a, b := m.Repos[i], m.Repos[j]
		var less bool
		switch m.sortCol {
		case SortName:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case SortBranch:
			less = strings.ToLower(a.Branch) < strings.ToLower(b.Branch)
		case SortAhead:
			less = a.Ahead < b.Ahead
		case SortBehind:
			less = a.Behind < b.Behind
		case SortChanges:
			less = a.Changes < b.Changes
		case SortStatus:
			less = a.Status < b.Status
		}
		if !m.sortAsc {
			return !less
		}
		return less
	})
}

func (m *Model) clampCursor() {
	vis := m.visibleIndices()
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(vis) > 0 && m.cursor >= len(vis) {
		m.cursor = len(vis) - 1
	}
	m.clampScroll()
}
