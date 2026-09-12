package ui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
	"github.com/yachiko/fossor/internal/ui/mainscreen"
)

func TestDiscoveryLocalRowStaysStaleUntilRemoteTerminal(t *testing.T) {
	a := NewApp(nil, t.TempDir(), false, false, true, "")
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

func TestOpenManageViewReceivesDiscoveryVerification(t *testing.T) {
	a := NewApp(nil, t.TempDir(), false, false, true, "")
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	a.discoveryGeneration = 1
	repo := git.RepoInfo{Path: "/repo", Name: "repo"}
	a.mainScreen.UpdateRepo(repo)
	a.mainScreen.SetVerification(repo.Path, mainscreen.Unverified)

	a.Update(common.SwitchToManageMsg{Repo: repo})
	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, DiscoveryGeneration: 1})

	if a.manageModel == nil {
		t.Fatal("manage view was not opened")
	}
	if cmd := a.manageModel.Update(tea.KeyMsg{Type: tea.KeyTab}); cmd == nil {
		t.Fatal("verified discovery result did not unlock the open manage view")
	}
}

func TestUserUpdateSupersedesOnlySameCheckoutDiscovery(t *testing.T) {
	a := NewApp(nil, t.TempDir(), false, false, true, "")
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	a.discoveryGeneration = 1
	first := git.RepoInfo{Path: "/first", Name: "first", Status: git.StatusAhead}
	second := git.RepoInfo{Path: "/second", Name: "second", Status: git.StatusBehind}

	a.Update(common.RepoUpdatedMsg{Repo: first, Source: common.RepoUpdateUserAction})
	a.Update(common.RepoDiscoveredMsg{Repo: git.RepoInfo{Path: first.Path, Name: first.Name, Status: git.StatusBehind}, Path: first.Path, DiscoveryGeneration: 1})
	a.Update(common.RepoDiscoveredMsg{Repo: second, Path: second.Path, DiscoveryGeneration: 1})

	for _, repo := range a.mainScreen.Repos {
		switch repo.Path {
		case first.Path:
			if repo.Status != git.StatusAhead {
				t.Fatalf("discovery overwrote user result: %v", repo.Status)
			}
		case second.Path:
			if repo.Status != git.StatusBehind {
				t.Fatalf("sibling checkout discovery was discarded: %v", repo.Status)
			}
		}
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
