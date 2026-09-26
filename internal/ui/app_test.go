package ui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
	"github.com/yachiko/fossor/internal/ui/mainscreen"
	"github.com/yachiko/fossor/internal/ui/manageview"
)

func TestDiscoveryLocalRowIsCheckingUntilRemoteTerminal(t *testing.T) {
	a := NewApp(nil, "", false, false, true, "")
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	repo := git.RepoInfo{Path: "/repo", Status: git.StatusBehind}

	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Local: true})
	if got := a.mainScreen.Verification(repo.Path); got != mainscreen.Checking {
		t.Fatalf("local verification = %v, want checking", got)
	}
	if !a.mainScreen.IsActionable(repo) {
		t.Fatal("locally read repo is not actionable while its fetch is pending")
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

func TestQuitCancelsApplicationAndManageContexts(t *testing.T) {
	a := NewApp(nil, "", false, true, true, "")
	ctx, cancel := context.WithCancel(a.appCtx)
	a.cancelManage = cancel
	a.manageModel = &manageview.Model{}
	a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if a.appCtx.Err() == nil {
		t.Fatal("quit did not cancel application context")
	}
	if ctx.Err() == nil {
		t.Fatal("quit did not cancel manage context")
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

func TestManageViewIsUsableWhileFetchIsPending(t *testing.T) {
	for _, tc := range []struct {
		state    mainscreen.VerificationState
		unlocked bool
	}{
		{mainscreen.Unverified, false},
		{mainscreen.Checking, true},
		{mainscreen.RemoteError, false},
	} {
		a := NewApp(nil, t.TempDir(), false, false, true, "")
		repo := git.RepoInfo{Path: "/repo", Name: "repo"}
		a.mainScreen.UpdateRepo(repo)
		a.mainScreen.SetVerification(repo.Path, tc.state)
		a.Update(common.SwitchToManageMsg{Repo: repo})
		// "U" (submodule update) is a local action gated on verification.
		if got := a.manageModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("U")}) != nil; got != tc.unlocked {
			t.Errorf("state %v: manage view unlocked = %v, want %v", tc.state, got, tc.unlocked)
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
