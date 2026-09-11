package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type discoveryTestGit struct {
	mu                 sync.Mutex
	fetched            []string
	fetchErr           map[string]error
	infoErr            map[string]error
	infoHook           func(string)
	fetchHook          func(string)
	remoteDefault      string
	remoteDefaultErr   error
	remoteDefaultCalls int
}

func (g *discoveryTestGit) GetRepoInfo(_ context.Context, path string) (RepoInfo, error) {
	g.mu.Lock()
	hook := g.infoHook
	err := g.infoErr[path]
	g.mu.Unlock()
	if hook != nil {
		hook(path)
	}
	return RepoInfo{Name: filepath.Base(path), Path: path, Branch: "main", DefaultBranch: "main", Status: StatusUpToDate}, err
}
func (g *discoveryTestGit) DetectDefaultBranch(context.Context, string) string { return "main" }
func (g *discoveryTestGit) GetRemoteDefaultBranch(context.Context, string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.remoteDefaultCalls++
	return g.remoteDefault, g.remoteDefaultErr
}
func (g *discoveryTestGit) GetBranch(context.Context, string) (string, error) { return "main", nil }
func (g *discoveryTestGit) GetRemote(context.Context, string) (string, error) { return "origin", nil }
func (g *discoveryTestGit) GetAheadBehind(context.Context, string, string) (int, int, error) {
	return 0, 0, nil
}
func (g *discoveryTestGit) GetChanges(context.Context, string) ([]ChangeInfo, error) { return nil, nil }
func (g *discoveryTestGit) GetLog(context.Context, string, int) ([]CommitInfo, error) {
	return nil, nil
}
func (g *discoveryTestGit) Fetch(_ context.Context, path string) error {
	g.mu.Lock()
	g.fetched = append(g.fetched, path)
	hook := g.fetchHook
	err := g.fetchErr[path]
	g.mu.Unlock()
	if hook != nil {
		hook(path)
	}
	return err
}

func TestDiscoverOverlapsLocalScanAndFetches(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "one")
	second := filepath.Join(root, "two")
	for _, path := range []string{first, second} {
		if err := os.MkdirAll(filepath.Join(path, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	releaseSecond := make(chan struct{})
	fetchStarted := make(chan string, 2)
	var mu sync.Mutex
	infoCalls := make(map[string]int)
	fetchErr := errors.New("offline")
	g := &discoveryTestGit{fetchErr: map[string]error{first: fetchErr}, infoErr: make(map[string]error)}
	g.infoHook = func(path string) {
		mu.Lock()
		infoCalls[path]++
		call := infoCalls[path]
		mu.Unlock()
		if path == second && call == 1 {
			<-releaseSecond
		}
	}
	g.fetchHook = func(path string) { fetchStarted <- path }

	ch := Discover(context.Background(), DiscoveryOptions{RootDir: root, Git: g, Fetch: true, Coordinator: NewOperationCoordinator()})
	result := <-ch
	if !result.Local || result.Path != first {
		t.Fatalf("first result = %#v, want local result for %s", result, first)
	}
	select {
	case path := <-fetchStarted:
		if path != first {
			t.Fatalf("fetch started for %s, want %s", path, first)
		}
	case <-time.After(time.Second):
		t.Fatal("fetch did not start while the second local status was blocked")
	}
	close(releaseSecond)

	localDone := false
	terminals := 0
	terminalResults := make(map[string]DiscoveryResult)
	for result := range ch {
		if result.LocalDone {
			localDone = true
		} else if !result.Local {
			terminals++
			terminalResults[result.Path] = result
		}
	}
	if !localDone || terminals != 2 {
		t.Fatalf("localDone=%v terminals=%d, want true and 2", localDone, terminals)
	}
	if !errors.Is(terminalResults[first].FetchErr, fetchErr) {
		t.Errorf("fetch error = %v, want %v", terminalResults[first].FetchErr, fetchErr)
	}
}

func TestDiscoverNoFetchIsLocalOnly(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	g := &discoveryTestGit{fetchErr: make(map[string]error), infoErr: make(map[string]error)}
	results := []DiscoveryResult{}
	for result := range Discover(context.Background(), DiscoveryOptions{RootDir: root, Git: g}) {
		results = append(results, result)
	}
	if len(results) != 2 || !results[0].Local || results[0].Path != path || !results[1].LocalDone {
		t.Fatalf("results = %#v, want one local result and local completion", results)
	}
	if len(g.fetched) != 0 {
		t.Fatalf("fetches = %v, want none", g.fetched)
	}
}

func TestDiscoverNoFetchUsesCachedDefaultBranch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	g := &discoveryTestGit{fetchErr: make(map[string]error), infoErr: make(map[string]error)}
	results := []DiscoveryResult{}
	for result := range Discover(context.Background(), DiscoveryOptions{
		RootDir: root,
		Git:     g,
		CachedRepos: map[string]RepoInfo{
			path: {Path: path, DefaultBranch: "develop"},
		},
	}) {
		results = append(results, result)
	}
	if got := results[0].Repo; got.DefaultBranch != "develop" || got.Status != StatusNonDefault {
		t.Errorf("local repo = %#v, want cached default branch develop and non-default status", got)
	}
	if g.remoteDefaultCalls != 0 {
		t.Errorf("remote default queries = %d, want none", g.remoteDefaultCalls)
	}
}

func TestRefreshRepoUsesRemoteDefaultBranch(t *testing.T) {
	g := &discoveryTestGit{
		fetchErr:      make(map[string]error),
		infoErr:       make(map[string]error),
		remoteDefault: "develop",
	}
	result := refreshRepo(context.Background(), RepoInfo{Path: "/repo", Branch: "main", DefaultBranch: "main"}, g, nil)
	if result.FetchErr != nil {
		t.Fatalf("FetchErr = %v, want nil", result.FetchErr)
	}
	if result.Repo.DefaultBranch != "develop" || result.Repo.Status != StatusNonDefault {
		t.Errorf("repo = %#v, want remote default branch develop and non-default status", result.Repo)
	}
	if g.remoteDefaultCalls != 1 {
		t.Errorf("remote default queries = %d, want 1", g.remoteDefaultCalls)
	}
}

func TestRefreshRepoRetainsLocalDefaultWhenRemoteQueryFails(t *testing.T) {
	g := &discoveryTestGit{
		fetchErr:         make(map[string]error),
		infoErr:          make(map[string]error),
		remoteDefaultErr: errors.New("remote unavailable"),
	}
	result := refreshRepo(context.Background(), RepoInfo{Path: "/repo", Branch: "main", DefaultBranch: "develop"}, g, nil)
	if result.FetchErr != nil {
		t.Fatalf("FetchErr = %v, want nil", result.FetchErr)
	}
	if result.Repo.DefaultBranch != "develop" {
		t.Errorf("default branch = %q, want cached local value develop", result.Repo.DefaultBranch)
	}
}

func TestSkippedRefreshProducesTerminalResult(t *testing.T) {
	path := "/repo"
	coordinator := NewOperationCoordinator()
	coordinator.states[path] = &operationState{running: true, done: make(chan struct{})}
	g := &discoveryTestGit{fetchErr: make(map[string]error), infoErr: make(map[string]error)}
	resultCh := make(chan DiscoveryResult, 1)
	go func() {
		resultCh <- refreshRepo(context.Background(), RepoInfo{Path: path}, g, coordinator)
	}()

	deadline := time.After(time.Second)
	for {
		coordinator.mu.Lock()
		waiting := coordinator.states[path].lowWaiting > 0
		coordinator.mu.Unlock()
		if waiting {
			break
		}
		select {
		case <-deadline:
			t.Fatal("refresh did not queue behind the running operation")
		case <-time.After(time.Millisecond):
		}
	}
	coordinator.mu.Lock()
	state := coordinator.states[path]
	state.highWaiting = 1
	state.highGeneration++
	state.running = false
	close(state.done)
	coordinator.mu.Unlock()

	result := <-resultCh
	if !result.Skipped || result.Path != path {
		t.Fatalf("result = %#v, want skipped terminal result", result)
	}
	if len(g.fetched) != 0 {
		t.Fatalf("fetches = %v, want none", g.fetched)
	}
}

func (g *discoveryTestGit) Pull(context.Context, string) (string, error) { return "", nil }
func (g *discoveryTestGit) Push(context.Context, string) (string, error) { return "", nil }
func (g *discoveryTestGit) RunCommand(context.Context, string, ...string) (string, error) {
	return "", nil
}
func (g *discoveryTestGit) SwitchBranch(context.Context, string, string) (string, error) {
	return "", nil
}
func (g *discoveryTestGit) RunShellCommand(context.Context, string, string, ...string) (string, error) {
	return "", nil
}
