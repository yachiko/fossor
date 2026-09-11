package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type discoveryTestGit struct {
	mu       sync.Mutex
	fetched  []string
	fetchErr map[string]error
	infoErr  map[string]error
}

func (g *discoveryTestGit) GetRepoInfo(_ context.Context, path string) (RepoInfo, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return RepoInfo{Name: filepath.Base(path), Path: path, Status: StatusUpToDate}, g.infoErr[path]
}
func (g *discoveryTestGit) DetectDefaultBranch(context.Context, string) string { return "main" }
func (g *discoveryTestGit) GetBranch(context.Context, string) (string, error)  { return "main", nil }
func (g *discoveryTestGit) GetRemote(context.Context, string) (string, error)  { return "origin", nil }
func (g *discoveryTestGit) GetAheadBehind(context.Context, string, string) (int, int, error) {
	return 0, 0, nil
}
func (g *discoveryTestGit) GetChanges(context.Context, string) ([]ChangeInfo, error) { return nil, nil }
func (g *discoveryTestGit) GetLog(context.Context, string, int) ([]CommitInfo, error) {
	return nil, nil
}
func (g *discoveryTestGit) Fetch(_ context.Context, path string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fetched = append(g.fetched, path)
	return g.fetchErr[path]
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

func TestDiscoverCollectsLocalStatusBeforeRefresh(t *testing.T) {
	root := t.TempDir()
	paths := make([]string, 0, 2)
	for _, name := range []string{"one", "two"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(path, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	g := &discoveryTestGit{fetchErr: map[string]error{paths[1]: errors.New("offline")}, infoErr: make(map[string]error)}
	var local []DiscoveryResult
	for result := range Discover(context.Background(), DiscoveryOptions{RootDir: root, Git: g}) {
		local = append(local, result)
	}
	if len(local) != len(paths) {
		t.Fatalf("local results = %d, want %d", len(local), len(paths))
	}
	if len(g.fetched) != 0 {
		t.Fatalf("fetches started during local discovery: %v", g.fetched)
	}
	g.infoErr[paths[0]] = errors.New("status unavailable")
	localRepos := make([]RepoInfo, 0, len(local))
	for _, result := range local {
		localRepos = append(localRepos, result.Repo)
	}

	refreshing := 0
	refreshed := make(map[string]DiscoveryResult)
	for result := range RefreshDiscovered(context.Background(), localRepos, g, NewOperationCoordinator()) {
		if result.Refreshing {
			refreshing++
			continue
		}
		refreshed[result.Path] = result
	}
	if refreshing != len(paths) {
		t.Errorf("refresh markers = %d, want %d", refreshing, len(paths))
	}
	if len(g.fetched) != len(paths) {
		t.Errorf("fetches = %d, want %d", len(g.fetched), len(paths))
	}
	if !errors.Is(refreshed[paths[1]].FetchErr, g.fetchErr[paths[1]]) {
		t.Errorf("refresh error = %v, want %v", refreshed[paths[1]].FetchErr, g.fetchErr[paths[1]])
	}
	if refreshed[paths[0]].Repo.Path != paths[0] || !errors.Is(refreshed[paths[0]].FetchErr, g.infoErr[paths[0]]) {
		t.Errorf("status-read failure did not retain local result: %#v", refreshed[paths[0]])
	}
}
