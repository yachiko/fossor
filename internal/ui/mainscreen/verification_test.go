package mainscreen

import (
	"strings"
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

func TestRemoteErrorsMatchOnlyTheErrorFilter(t *testing.T) {
	m := New(nil, "", "")
	repo := git.RepoInfo{Path: "/remote-error", Status: git.StatusBehind}
	m.UpdateRepo(repo)
	m.SetVerification(repo.Path, RemoteError)
	m.filterMode = FilterError
	if !m.matchesFilter(repo) {
		t.Error("remote error did not match the error filter")
	}
	m.filterMode = FilterBehind
	if m.matchesFilter(repo) {
		t.Error("remote error matched its retained local status filter")
	}
	m.filterMode = FilterAll
	if !m.matchesFilter(repo) {
		t.Error("remote error did not match the all filter")
	}
	m.cycleFilter()
	if m.filterMode != FilterError {
		t.Errorf("filter = %v, want Error", m.filterMode)
	}
	if !strings.Contains(m.statusCountsView(), "1 error") {
		t.Errorf("status counts = %q, want one error", m.statusCountsView())
	}
}

func TestCheckingRowsAreCountedFilterableAndActionable(t *testing.T) {
	m := New(nil, "", "")
	repo := git.RepoInfo{Path: "/checking", Status: git.StatusBehind, Behind: 2}
	m.UpdateRepo(repo)
	m.SetVerification(repo.Path, Checking)
	if !m.IsActionable(repo) {
		t.Error("checking repo is not actionable")
	}
	if !strings.Contains(m.statusCountsView(), "1 behind") {
		t.Errorf("status counts = %q, want one behind", m.statusCountsView())
	}
	m.cycleFilter()
	if m.filterMode != FilterBehind || !m.matchesFilter(repo) {
		t.Errorf("filter = %v, want Behind matching the checking row", m.filterMode)
	}
	if cmd := m.pullAll(); cmd == nil {
		t.Error("pull-all skipped the checking row")
	}
}

func TestSetVerificationRefiltersActiveFilter(t *testing.T) {
	m := New(nil, "", "")
	repo := git.RepoInfo{Path: "/repo", Status: git.StatusBehind}
	m.AddCachedRepo(repo)
	m.filterMode = FilterBehind
	m.refilter()
	if len(m.visibleIndices()) != 0 {
		t.Fatal("cached row matched the Behind filter")
	}
	m.SetVerification(repo.Path, Checking)
	if len(m.visibleIndices()) != 1 {
		t.Error("row did not appear under the active filter once checking")
	}
}

func TestCheckingRowShowsLiveStatusNotStale(t *testing.T) {
	m := New(nil, "", "")
	m.SetSize(120, 20)
	repo := git.RepoInfo{Path: "/repo", Name: "repo", Branch: "main", DefaultBranch: "main", Status: git.StatusBehind, Behind: 2}
	m.UpdateRepo(repo)
	m.SetVerification(repo.Path, Checking)
	view := m.View()
	if strings.Contains(view, "stale") || !strings.Contains(view, "Behind") {
		t.Errorf("checking row should show its live status, got:\n%s", view)
	}
}
