package git

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const cacheVersion = 2

// CacheDir is overrideable by tests. An empty value uses the user's cache dir.
var CacheDir string

type discoveryCache struct {
	Version int                   `json:"version"`
	Entries []discoveryCacheEntry `json:"entries"`
}

type discoveryCacheEntry struct {
	RootDir   string     `json:"root_dir"`
	Recursive bool       `json:"recursive"`
	Repos     []RepoInfo `json:"repos"`
}

func cachePath() (string, error) {
	if CacheDir != "" {
		return filepath.Join(CacheDir, "repositories.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "fossor", "repositories.json"), nil
}

// LoadDiscoveryCache returns the last complete snapshot for this scan scope.
// Cache failures are intentionally invisible to callers.
func LoadDiscoveryCache(root string, recursive bool) []RepoInfo {
	path, err := cachePath()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cache discoveryCache
	if json.Unmarshal(b, &cache) != nil || cache.Version != cacheVersion {
		return nil
	}
	for _, entry := range cache.Entries {
		if entry.RootDir == root && entry.Recursive == recursive {
			return entry.Repos
		}
	}
	return nil
}

// SaveDiscoveryCache atomically replaces the completed snapshot for this scope.
func SaveDiscoveryCache(root string, recursive bool, repos []RepoInfo) {
	path, err := cachePath()
	if err != nil {
		return
	}
	var cache discoveryCache
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &cache)
	}
	if cache.Version != cacheVersion {
		cache = discoveryCache{Version: cacheVersion}
	}
	// Errors are process-local values and cannot be reliably round-tripped as JSON.
	// Retain the status and all display data, but omit them from persisted rows.
	cachedRepos := append([]RepoInfo(nil), repos...)
	for i := range cachedRepos {
		cachedRepos[i].Error = nil
	}
	entry := discoveryCacheEntry{RootDir: root, Recursive: recursive, Repos: cachedRepos}
	found := false
	for i := range cache.Entries {
		if cache.Entries[i].RootDir == root && cache.Entries[i].Recursive == recursive {
			cache.Entries[i] = entry
			found = true
		}
	}
	if !found {
		cache.Entries = append(cache.Entries, entry)
	}
	b, err := json.Marshal(cache)
	if err != nil || os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".repositories-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err = tmp.Write(b); err != nil || tmp.Chmod(0600) != nil || tmp.Close() != nil {
		return
	}
	_ = os.Rename(tmpName, path)
}
