package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

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

func TestRefreshResultsDebounceCacheSave(t *testing.T) {
	path := useTempCache(t)
	a := newDiscoveringApp()
	a.localDone = true
	ch := closedDiscovery()

	for i := range 50 {
		repo := git.RepoInfo{Path: fmt.Sprintf("/repos/r%d", i)}
		_, cmd := a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Ch: ch})
		want := 1
		if i == 0 {
			want = 2 // discovery wait plus the debounced save
		}
		if got := batchLen(cmd); got != want {
			t.Fatalf("result %d returned %d commands, want %d", i, got, want)
		}
	}
	assertNoCacheFile(t, path)

	_, cmd := a.Update(cacheSaveMsg{})
	if cmd == nil {
		t.Fatal("debounce fired without a save command")
	}
	assertNoCacheFile(t, path)
	cmd()
	if got := git.LoadDiscoveryCache("/repos", false); len(got) != 50 {
		t.Fatalf("saved %d repos, want 50", len(got))
	}

	repo := git.RepoInfo{Path: "/repos/late"}
	if _, cmd := a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Ch: ch}); batchLen(cmd) != 2 {
		t.Fatal("result after a fired debounce did not schedule another save")
	}
}

func TestLocalDoneAndCompleteSaveCache(t *testing.T) {
	path := useTempCache(t)
	a := newDiscoveringApp()
	ch := closedDiscovery()
	repo := git.RepoInfo{Path: "/repos/one", Status: git.StatusBehind}

	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Local: true, Ch: ch})
	_, cmd := a.Update(common.DiscoveryCompleteMsg{LocalDone: true, Ch: ch})
	assertNoCacheFile(t, path)
	runCmds(cmd)
	if got := git.LoadDiscoveryCache("/repos", false); len(got) != 1 || got[0].Status != git.StatusBehind {
		t.Fatalf("LocalDone cache = %#v, want local snapshot", got)
	}

	repo.Status = git.StatusUpToDate
	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Ch: ch})
	_, cmd = a.Update(common.DiscoveryCompleteMsg{Complete: true})
	runCmds(cmd)
	if got := git.LoadDiscoveryCache("/repos", false); len(got) != 1 || got[0].Status != git.StatusUpToDate {
		t.Fatalf("Complete cache = %#v, want refreshed snapshot", got)
	}
}

func TestQuitFlushesCacheOnlyAfterLocalDone(t *testing.T) {
	useTempCache(t)
	repo := git.RepoInfo{Path: "/repos/one"}

	a := newDiscoveringApp()
	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path, Local: true})
	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path})
	a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if got := git.LoadDiscoveryCache("/repos", false); got != nil {
		t.Fatalf("cancelled scan saved %#v", got)
	}

	a = newDiscoveringApp()
	a.localDone = true
	a.Update(common.RepoDiscoveredMsg{Repo: repo, Path: repo.Path})
	a.Update(common.QuitMsg{})
	if got := git.LoadDiscoveryCache("/repos", false); len(got) != 1 {
		t.Fatalf("quit cache = %#v, want pending refresh flushed", got)
	}
}

func useTempCache(t *testing.T) string {
	t.Helper()
	old := git.CacheDir
	git.CacheDir = t.TempDir()
	t.Cleanup(func() { git.CacheDir = old })
	return filepath.Join(git.CacheDir, "repositories.json")
}

func newDiscoveringApp() *App {
	a := NewApp(nil, "/repos", false, false, true, "")
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	return a
}

func closedDiscovery() <-chan git.DiscoveryResult {
	ch := make(chan git.DiscoveryResult)
	close(ch)
	return ch
}

func assertNoCacheFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cache written synchronously (stat err %v)", err)
	}
}

// batchLen counts the commands in cmd; batched children are not run.
func batchLen(cmd tea.Cmd) int {
	if cmd == nil {
		return 0
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		return len(batch)
	}
	return 1
}

// runCmds runs cmd and its batched children, abandoning timers still pending.
func runCmds(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		return
	}
	var wg sync.WaitGroup
	for _, c := range batch {
		wg.Go(func() { c() })
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}
