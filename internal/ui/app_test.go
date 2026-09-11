package ui

import (
	"errors"
	"testing"

	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
	"github.com/yachiko/fossor/internal/ui/mainscreen"
)

func TestDiscoveryLocalRowStaysStaleUntilRemoteTerminal(t *testing.T) {
	a := NewApp(nil, "", false, false, true, "")
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	repo := git.RepoInfo{Path: "/repo", Status: git.StatusBehind}

	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Local: true})
	if got := a.mainScreen.Verification(repo.Path); got != mainscreen.Unverified {
		t.Fatalf("local verification = %v, want unverified", got)
	}
	if a.refreshTotal != 1 || a.refreshDone != 0 {
		t.Fatalf("progress = %d/%d, want 0/1", a.refreshDone, a.refreshTotal)
	}

	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, FetchErr: errors.New("offline")})
	if got := a.mainScreen.Verification(repo.Path); got != mainscreen.RemoteError {
		t.Fatalf("terminal verification = %v, want remote error", got)
	}
	if a.refreshDone != 1 {
		t.Fatalf("refresh done = %d, want 1", a.refreshDone)
	}
}

func TestNoFetchLocalRowIsVerified(t *testing.T) {
	a := NewApp(nil, "", false, true, true, "")
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	repo := git.RepoInfo{Path: "/repo"}

	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Local: true})
	if got := a.mainScreen.Verification(repo.Path); got != mainscreen.Verified {
		t.Fatalf("no-fetch verification = %v, want verified", got)
	}
}
