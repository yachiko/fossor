package mainscreen

import (
	"testing"

	"github.com/yachiko/fossor/internal/git"
)

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter FilterMode
		repo   git.RepoInfo
		want   bool
	}{
		// FilterAll matches everything
		{"all matches up-to-date", FilterAll, git.RepoInfo{Status: git.StatusUpToDate}, true},
		{"all matches error", FilterAll, git.RepoInfo{Status: git.StatusError}, true},
		{"all matches zero value", FilterAll, git.RepoInfo{}, true},

		// FilterBehind
		{"behind matches behind", FilterBehind, git.RepoInfo{Status: git.StatusBehind}, true},
		{"behind rejects ahead", FilterBehind, git.RepoInfo{Status: git.StatusAhead}, false},
		{"behind rejects up-to-date", FilterBehind, git.RepoInfo{Status: git.StatusUpToDate}, false},

		// FilterAhead
		{"ahead matches ahead", FilterAhead, git.RepoInfo{Status: git.StatusAhead}, true},
		{"ahead rejects behind", FilterAhead, git.RepoInfo{Status: git.StatusBehind}, false},

		// FilterDirty
		{"dirty matches dirty", FilterDirty, git.RepoInfo{Status: git.StatusDirty}, true},
		{"dirty rejects clean", FilterDirty, git.RepoInfo{Status: git.StatusUpToDate}, false},

		// FilterDiverged
		{"diverged matches diverged", FilterDiverged, git.RepoInfo{Status: git.StatusDiverged}, true},
		{"diverged rejects ahead", FilterDiverged, git.RepoInfo{Status: git.StatusAhead}, false},

		// FilterNonDefault
		{"non-default matches non-default", FilterNonDefault, git.RepoInfo{Status: git.StatusNonDefault}, true},
		{"non-default rejects up-to-date", FilterNonDefault, git.RepoInfo{Status: git.StatusUpToDate}, false},

		// FilterError
		{"error matches error", FilterError, git.RepoInfo{Status: git.StatusError}, true},
		{"error rejects up-to-date", FilterError, git.RepoInfo{Status: git.StatusUpToDate}, false},

		// FilterUpToDate
		{"up-to-date matches up-to-date", FilterUpToDate, git.RepoInfo{Status: git.StatusUpToDate}, true},
		{"up-to-date rejects dirty", FilterUpToDate, git.RepoInfo{Status: git.StatusDirty}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(nil, "", "")
			m.filterMode = tt.filter
			got := m.matchesFilter(tt.repo)
			if got != tt.want {
				t.Errorf("matchesFilter(%v, %+v) = %v, want %v", tt.filter, tt.repo.Status, got, tt.want)
			}
		})
	}
}

func TestCycleFilter(t *testing.T) {
	t.Run("starts at All with one Behind repo, goes to Behind, then back to All", func(t *testing.T) {
		m := New(nil, "", "")
		m.Repos = []git.RepoInfo{
			{Name: "repo1", Status: git.StatusBehind},
		}
		if m.filterMode != FilterAll {
			t.Fatalf("expected initial filter to be All, got %v", m.filterMode)
		}

		m.cycleFilter()
		if m.filterMode != FilterBehind {
			t.Errorf("after first cycle expected Behind, got %v", m.filterMode)
		}

		m.cycleFilter()
		if m.filterMode != FilterAll {
			t.Errorf("after second cycle expected All, got %v", m.filterMode)
		}
	})

	t.Run("all statuses empty stays at All", func(t *testing.T) {
		m := New(nil, "", "")
		m.Repos = nil
		m.filterMode = FilterAll

		m.cycleFilter()
		if m.filterMode != FilterAll {
			t.Errorf("expected to stay at All with no repos, got %v", m.filterMode)
		}
	})

	t.Run("skips statuses with 0 repos", func(t *testing.T) {
		m := New(nil, "", "")
		m.Repos = []git.RepoInfo{
			{Name: "repo1", Status: git.StatusAhead},
			{Name: "repo2", Status: git.StatusDirty},
		}
		m.filterMode = FilterAll

		// Cycle order with these repos: Ahead, Dirty, All
		// (Behind, Diverged, NonDefault, Error, UpToDate all have 0 and are skipped)
		m.cycleFilter()
		if m.filterMode != FilterAhead {
			t.Errorf("expected Ahead, got %v", m.filterMode)
		}

		m.cycleFilter()
		if m.filterMode != FilterDirty {
			t.Errorf("expected Dirty, got %v", m.filterMode)
		}

		m.cycleFilter()
		if m.filterMode != FilterAll {
			t.Errorf("expected All, got %v", m.filterMode)
		}
	})
}

func TestWorktreeGroupsProjectHeadersAndKeepCheckoutsActionable(t *testing.T) {
	m := New(nil, "", "")
	m.Repos = []git.RepoInfo{
		{Name: "primary", Path: "/repos/primary", CommonGitDir: "/repos/primary/.git", Status: git.StatusUpToDate},
		{Name: "feature", Path: "/repos/feature", CommonGitDir: "/repos/primary/.git", LinkedWorktree: true, Status: git.StatusBehind},
		{Name: "other", Path: "/repos/other", CommonGitDir: "/repos/other/.git", Status: git.StatusUpToDate},
	}
	m.refilter()
	if rows := m.visibleRows(); len(rows) != 3 || m.Repos[rows[0].repoIndex].Path != "/repos/primary" || !rows[0].groupRoot || rows[0].groupSize != 2 {
		t.Fatalf("rows = %#v, want primary root plus two checkouts", rows)
	}
	if got := m.visibleIndices(); len(got) != 3 {
		t.Fatalf("actionable rows = %v, want 3 checkout rows", got)
	}
	if !m.toggleSelectedGroup() {
		t.Fatal("group header did not toggle")
	}
	if rows := m.visibleRows(); len(rows) != 2 || m.Repos[rows[0].repoIndex].Path != "/repos/primary" || !rows[0].groupRoot {
		t.Fatalf("collapsed rows = %#v", rows)
	}
}

func TestSearchMatchesWorktreePath(t *testing.T) {
	m := New(nil, "", "")
	m.Repos = []git.RepoInfo{{Name: "checkout", Path: "/repos/feature-123", Status: git.StatusUpToDate}}
	m.searchText.SetValue("feature-123")
	m.refilter()
	if got := m.visibleIndices(); len(got) != 1 {
		t.Fatalf("path search rows = %v, want checkout", got)
	}
}

func selectedPath(t *testing.T, m *Model) string {
	t.Helper()
	repo, ok := m.SelectedRepo()
	if !ok {
		t.Fatalf("no repo selected at cursor %d", m.cursor)
	}
	return repo.Path
}

func TestSelectionFollowsRepoInsertedAbove(t *testing.T) {
	m := New(nil, "", "")
	m.SetSize(80, 11) // three table rows
	for _, name := range []string{"b", "c", "d"} {
		m.UpdateRepo(git.RepoInfo{Name: name, Path: "/repos/" + name})
	}
	m.cursor = 2
	m.UpdateRepo(git.RepoInfo{Name: "a", Path: "/repos/a"})
	if got := selectedPath(t, &m); got != "/repos/d" {
		t.Fatalf("selected %q, want /repos/d", got)
	}
	if m.cursor != 3 || m.scrollOffset != 1 {
		t.Fatalf("cursor/scroll = %d/%d, want 3/1 so d keeps its screen line", m.cursor, m.scrollOffset)
	}
}

func TestSelectionFollowsRepoResortedByStatus(t *testing.T) {
	m := New(nil, "", "")
	m.sortCol = SortStatus
	m.UpdateRepo(git.RepoInfo{Name: "a", Path: "/repos/a", Status: git.StatusUpToDate})
	m.UpdateRepo(git.RepoInfo{Name: "b", Path: "/repos/b", Status: git.StatusAhead})
	m.UpdateRepo(git.RepoInfo{Name: "c", Path: "/repos/c", Status: git.StatusBehind})
	m.cursor = 0
	m.UpdateRepo(git.RepoInfo{Name: "a", Path: "/repos/a", Status: git.StatusDirty})
	if got := selectedPath(t, &m); got != "/repos/a" {
		t.Fatalf("selected %q, want /repos/a", got)
	}
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}
}

func TestSelectionFallsBackWhenFilteredOut(t *testing.T) {
	m := New(nil, "", "")
	for _, name := range []string{"keep1", "keep2", "other"} {
		m.UpdateRepo(git.RepoInfo{Name: name, Path: "/repos/" + name})
	}
	m.cursor = 2
	m.searchText.SetValue("keep")
	m.refilter()
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want clamped to 1", m.cursor)
	}
	if got := selectedPath(t, &m); got != "/repos/keep2" {
		t.Fatalf("selected %q, want /repos/keep2", got)
	}
}

func TestSelectionFollowsWorktreeHeader(t *testing.T) {
	m := New(nil, "", "")
	m.UpdateRepo(git.RepoInfo{Name: "feat-a", Path: "/repos/feat-a", CommonGitDir: "/repos/x/.git", LinkedWorktree: true})
	m.UpdateRepo(git.RepoInfo{Name: "feat-b", Path: "/repos/feat-b", CommonGitDir: "/repos/x/.git", LinkedWorktree: true})
	m.UpdateRepo(git.RepoInfo{Name: "zed", Path: "/repos/zed"})
	m.cursor = 0
	if rows := m.visibleRows(); rows[m.cursor].repoIndex >= 0 {
		t.Fatalf("cursor row = %#v, want worktree header", rows[m.cursor])
	}
	m.UpdateRepo(git.RepoInfo{Name: "alpha", Path: "/repos/alpha"})
	rows := m.visibleRows()
	if m.cursor != 1 || rows[m.cursor].repoIndex >= 0 || rows[m.cursor].groupKey != "/repos/x/.git" {
		t.Fatalf("cursor = %d row = %#v, want header at 1", m.cursor, rows[m.cursor])
	}
}
