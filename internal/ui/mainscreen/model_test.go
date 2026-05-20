package mainscreen

import (
	"testing"

	"github.com/ahoma/fossor/internal/git"
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
