package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoveryCacheReplacesScopeAtomically(t *testing.T) {
	old := CacheDir
	CacheDir = t.TempDir()
	t.Cleanup(func() { CacheDir = old })

	root := filepath.Join(string(os.PathSeparator), "repos")
	SaveDiscoveryCache(root, false, NextDiscoveryCacheSeq(), []RepoInfo{{Name: "one", Path: "/repos/one"}})
	SaveDiscoveryCache(root, false, NextDiscoveryCacheSeq(), []RepoInfo{{Name: "two", Path: "/repos/two"}})
	got := LoadDiscoveryCache(root, false)
	if len(got) != 1 || got[0].Path != "/repos/two" {
		t.Fatalf("LoadDiscoveryCache() = %#v, want replacement snapshot", got)
	}
	if got := LoadDiscoveryCache(root, true); got != nil {
		t.Fatalf("recursive scope returned %#v, want nil", got)
	}
	info, err := os.Stat(filepath.Join(CacheDir, "repositories.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("cache permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestDiscoveryCacheIgnoresInvalidData(t *testing.T) {
	old := CacheDir
	CacheDir = t.TempDir()
	t.Cleanup(func() { CacheDir = old })

	if err := os.WriteFile(filepath.Join(CacheDir, "repositories.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := LoadDiscoveryCache("/repos", false); got != nil {
		t.Fatalf("LoadDiscoveryCache() = %#v, want nil", got)
	}
}

func TestDiscoveryCachePersistsDefaultBranchCheckTime(t *testing.T) {
	old := CacheDir
	CacheDir = t.TempDir()
	t.Cleanup(func() { CacheDir = old })

	checkedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	SaveDiscoveryCache("/repos", false, NextDiscoveryCacheSeq(), []RepoInfo{{Path: "/repos/one", DefaultBranch: "develop", DefaultBranchCheckedAt: checkedAt}})
	got := LoadDiscoveryCache("/repos", false)
	if len(got) != 1 || !got[0].DefaultBranchCheckedAt.Equal(checkedAt) {
		t.Fatalf("LoadDiscoveryCache() = %#v, want check time %v", got, checkedAt)
	}
}

func TestDiscoveryCacheDropsStaleSnapshot(t *testing.T) {
	old := CacheDir
	CacheDir = t.TempDir()
	t.Cleanup(func() { CacheDir = old })

	root := filepath.Join(string(os.PathSeparator), "repos")
	older, newer := NextDiscoveryCacheSeq(), NextDiscoveryCacheSeq()
	SaveDiscoveryCache(root, false, newer, []RepoInfo{{Name: "new", Path: "/repos/new"}})
	SaveDiscoveryCache(root, false, older, []RepoInfo{{Name: "old", Path: "/repos/old"}})
	got := LoadDiscoveryCache(root, false)
	if len(got) != 1 || got[0].Path != "/repos/new" {
		t.Fatalf("LoadDiscoveryCache() = %#v, want newer snapshot", got)
	}

	// Sequences are tracked per scope.
	SaveDiscoveryCache(root, true, older, []RepoInfo{{Name: "old", Path: "/repos/old"}})
	if got := LoadDiscoveryCache(root, true); len(got) != 1 {
		t.Fatalf("recursive scope = %#v, want its own snapshot", got)
	}
}
