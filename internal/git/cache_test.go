package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoveryCacheReplacesScopeAtomically(t *testing.T) {
	old := CacheDir
	CacheDir = t.TempDir()
	t.Cleanup(func() { CacheDir = old })

	root := filepath.Join(string(os.PathSeparator), "repos")
	SaveDiscoveryCache(root, false, []RepoInfo{{Name: "one", Path: "/repos/one"}})
	SaveDiscoveryCache(root, false, []RepoInfo{{Name: "two", Path: "/repos/two"}})
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
