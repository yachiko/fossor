package mainscreen

import (
	"testing"

	"github.com/yachiko/fossor/internal/git"
)

func TestUnverifiedRowsExcludedFromCountsAndBulkEligibility(t *testing.T) {
	m := New(nil, "", "")
	cached := git.RepoInfo{Path: "/cached", Status: git.StatusBehind}
	live := git.RepoInfo{Path: "/live", Status: git.StatusAhead}
	m.AddCachedRepo(cached)
	m.UpdateRepo(live)
	if m.IsActionable(cached) {
		t.Error("cached repo is actionable")
	}
	if !m.IsActionable(live) {
		t.Error("live repo is not actionable")
	}
	m.cycleFilter()
	if m.filterMode != FilterAhead {
		t.Errorf("filter = %v, want Ahead; cached Behind must be excluded", m.filterMode)
	}
	m.filterMode = FilterBehind
	if m.matchesFilter(cached) {
		t.Error("unverified row matched status filter")
	}
}

func TestPruneRemovesStaleCachedRows(t *testing.T) {
	m := New(nil, "", "")
	m.AddCachedRepo(git.RepoInfo{Path: "/gone"})
	m.AddCachedRepo(git.RepoInfo{Path: "/live"})
	m.Prune(map[string]bool{"/live": true})
	if len(m.Repos) != 1 || m.Repos[0].Path != "/live" {
		t.Fatalf("repos = %#v", m.Repos)
	}
	if _, ok := m.verification["/gone"]; ok {
		t.Error("stale verification metadata retained")
	}
}

func TestBulkOperationsIgnoreUnverifiedRows(t *testing.T) {
	m := New(nil, "", "")
	m.AddCachedRepo(git.RepoInfo{Path: "/cached"})
	if cmd := m.pullAll(); cmd != nil {
		t.Error("pull-all returned a command with only an unverified row")
	}
	if cmd := m.fetchAll(); cmd != nil {
		t.Error("fetch-all returned a command with only an unverified row")
	}
}

func TestRefreshingRowsRemainVisibleButNotActionable(t *testing.T) {
	m := New(nil, "", "")
	repo := git.RepoInfo{Path: "/refreshing", Status: git.StatusBehind}
	m.UpdateRepo(repo)
	m.SetVerification(repo.Path, Refreshing)
	if m.IsActionable(repo) {
		t.Error("refreshing repo is actionable")
	}
	m.filterMode = FilterBehind
	if !m.matchesFilter(repo) {
		t.Error("refreshing repo did not match its local status filter")
	}
}
