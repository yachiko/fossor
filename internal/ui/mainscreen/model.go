package mainscreen

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yachiko/fossor/internal/git"
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

// VerificationState is UI metadata, deliberately separate from git.RepoStatus.
type VerificationState int

const (
	Unverified VerificationState = iota
	Verified
	RemoteError
)

// Model is the main screen model.
type Model struct {
	Repos        []git.RepoInfo
	Git          git.Git
	Coordinator  *git.OperationCoordinator
	RootDir      string
	OpenCmd      string
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
	verification map[string]VerificationState
	collapsed    map[string]bool
}

type tableRow struct {
	repoIndex int
	groupKey  string
	groupName string
	groupSize int
	groupRoot bool
}

// New creates a new main screen model.
func New(g git.Git, rootDir, openCmd string, coordinators ...*git.OperationCoordinator) Model {
	ti := textinput.New()
	ti.Placeholder = "Search repos..."
	ti.CharLimit = 100

	m := Model{
		Git:          g,
		RootDir:      rootDir,
		OpenCmd:      openCmd,
		sortCol:      SortName,
		sortAsc:      true,
		searchText:   ti,
		verification: make(map[string]VerificationState),
		collapsed:    make(map[string]bool),
	}
	if len(coordinators) > 0 {
		m.Coordinator = coordinators[0]
	}
	return m
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
	// title(1) + blank(1) + search/filter(1) + header(1) + top_sep(1) + bottom_sep(1) + statusbar(2) = 8
	h := m.height - 8
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

// AddCachedRepo hydrates a persisted row without treating it as live state.
func (m *Model) AddCachedRepo(repo git.RepoInfo) {
	m.UpdateRepo(repo)
	m.verification[repo.Path] = Unverified
}

func (m *Model) SetVerification(path string, state VerificationState) {
	m.verification[path] = state
}

// Prune removes rows and verification metadata absent from a completed scan.
func (m *Model) Prune(paths map[string]bool) {
	kept := m.Repos[:0]
	for _, repo := range m.Repos {
		if paths[repo.Path] {
			kept = append(kept, repo)
		} else {
			delete(m.verification, repo.Path)
		}
	}
	m.Repos = kept
	m.refilter()
	m.clampCursor()
}

func (m *Model) Verification(path string) VerificationState {
	if state, ok := m.verification[path]; ok {
		return state
	}
	return Verified
}

func (m *Model) IsActionable(repo git.RepoInfo) bool {
	return m.Verification(repo.Path) == Verified
}

// SelectedRepo returns the currently selected repo, if any.
func (m *Model) SelectedRepo() (git.RepoInfo, bool) {
	rows := m.visibleRows()
	if len(rows) == 0 || m.cursor >= len(rows) || rows[m.cursor].repoIndex < 0 {
		return git.RepoInfo{}, false
	}
	return m.Repos[rows[m.cursor].repoIndex], true
}

// visibleIndices returns actionable checkout rows, never group headers.
func (m *Model) visibleIndices() []int {
	rows := m.visibleRows()
	indices := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.repoIndex >= 0 {
			indices = append(indices, row.repoIndex)
		}
	}
	return indices
}

func (m *Model) visibleRows() []tableRow {
	var indices []int
	if m.filtered != nil {
		indices = m.filtered
	} else {
		indices = make([]int, len(m.Repos))
		for i := range indices {
			indices[i] = i
		}
	}
	groups := make(map[string][]int)
	var order []string
	for _, i := range indices {
		key := m.Repos[i].CoordinatorKey()
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], i)
	}
	rows := make([]tableRow, 0, len(indices))
	for _, key := range order {
		members := groups[key]
		if len(members) < 2 {
			rows = append(rows, tableRow{repoIndex: members[0]})
			continue
		}
		primary := -1
		groupName := m.Repos[members[0]].Name
		for _, i := range members {
			if !m.Repos[i].LinkedWorktree {
				groupName = m.Repos[i].Name
				primary = i
				break
			}
		}
		if primary >= 0 {
			rows = append(rows, tableRow{repoIndex: primary, groupKey: key, groupSize: len(members), groupRoot: true})
		} else {
			rows = append(rows, tableRow{repoIndex: -1, groupKey: key, groupName: groupName, groupSize: len(members)})
		}
		if !m.collapsed[key] {
			for _, i := range members {
				if i == primary {
					continue
				}
				rows = append(rows, tableRow{repoIndex: i, groupKey: key})
			}
		}
	}
	return rows
}

func (m *Model) refilter() {
	m.sortRepos()

	query := strings.ToLower(m.searchText.Value())

	if query == "" && m.filterMode == FilterAll {
		m.filtered = nil
		m.clampCursor()
		return
	}

	m.filtered = []int{}
	for i, r := range m.Repos {
		if !m.matchesFilter(r) {
			continue
		}
		if query != "" {
			if !strings.Contains(strings.ToLower(r.Name), query) &&
				!strings.Contains(strings.ToLower(r.Path), query) &&
				!strings.Contains(strings.ToLower(r.CommonGitDir), query) &&
				!strings.Contains(strings.ToLower(r.Branch), query) &&
				!strings.Contains(strings.ToLower(r.DefaultBranch), query) &&
				!strings.Contains(strings.ToLower(r.Remote), query) &&
				!strings.Contains(strings.ToLower(r.Status.String()), query) {
				continue
			}
		}
		m.filtered = append(m.filtered, i)
	}
	m.clampCursor()
}

func (m *Model) matchesFilter(r git.RepoInfo) bool {
	verification := m.Verification(r.Path)
	if verification == Unverified {
		return false
	}
	if verification == RemoteError {
		return m.filterMode == FilterAll || m.filterMode == FilterError
	}
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
		verification := m.Verification(r.Path)
		if verification == Unverified {
			continue
		}
		if verification == RemoteError {
			counts[git.StatusError]++
			continue
		}
		counts[r.Status]++
	}

	// Cycle order: status filters that have repos, then All.
	type entry struct {
		mode   FilterMode
		status git.RepoStatus
	}
	statusOrder := []entry{
		{FilterError, git.StatusError},
		{FilterNonDefault, git.StatusNonDefault},
		{FilterDiverged, git.StatusDiverged},
		{FilterBehind, git.StatusBehind},
		{FilterAhead, git.StatusAhead},
		{FilterDirty, git.StatusDirty},
		{FilterUpToDate, git.StatusUpToDate},
	}

	// Build the active cycle: non-empty statuses + All at the end
	var cycle []FilterMode
	for _, e := range statusOrder {
		if counts[e.status] > 0 {
			cycle = append(cycle, e.mode)
		}
	}
	cycle = append(cycle, FilterAll)

	// Find current position and advance to next
	for i, fm := range cycle {
		if fm == m.filterMode {
			m.filterMode = cycle[(i+1)%len(cycle)]
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
	vis := m.visibleRows()
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(vis) > 0 && m.cursor >= len(vis) {
		m.cursor = len(vis) - 1
	}
	m.clampScroll()
}
