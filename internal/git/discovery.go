package git

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// defaultConcurrency caps the number of in-flight per-repo git invocations
// during discovery. Picked at startup to scale with the machine: 4×NumCPU
// keeps the box busy on I/O-bound git fetches without exhausting the
// default file-descriptor budget (typically 256 on macOS, 1024 on Linux)
// when scanning hundreds of repos. Floor of 8, ceiling of 16 — past 16 we
// hit diminishing returns and start trading throughput for FD pressure.
var defaultConcurrency = discoveryConcurrency()

func discoveryConcurrency() int {
	n := runtime.NumCPU() * 4
	if n < 8 {
		n = 8
	}
	if n > 16 {
		n = 16
	}
	return n
}

// DiscoveryResult carries a discovered repo or indicates completion.
type DiscoveryResult struct {
	Repo RepoInfo
	Done bool
}

// DiscoveryOptions configures discovery behavior.
type DiscoveryOptions struct {
	RootDir   string
	Recursive bool
	NoFetch   bool
	Git       Git
}

// Discover scans for git repositories and streams results on the returned channel.
// The channel is closed when discovery is complete.
func Discover(ctx context.Context, opts DiscoveryOptions) <-chan DiscoveryResult {
	ch := make(chan DiscoveryResult, 32)

	go func() {
		defer close(ch)

		repoPaths := findRepos(ctx, opts.RootDir, opts.Recursive)

		sem := make(chan struct{}, defaultConcurrency)
		var wg sync.WaitGroup

		for _, repoPath := range repoPaths {
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			sem <- struct{}{}

			go func(rp string) {
				defer wg.Done()
				defer func() { <-sem }()

				if !opts.NoFetch {
					// Best-effort fetch; don't fail discovery on fetch error
					_ = opts.Git.Fetch(ctx, rp)
				}

				info, _ := opts.Git.GetRepoInfo(ctx, rp)

				select {
				case ch <- DiscoveryResult{Repo: info}:
				case <-ctx.Done():
				}
			}(repoPath)
		}

		wg.Wait()
	}()

	return ch
}

// findRepos returns paths to directories containing .git.
func findRepos(ctx context.Context, root string, recursive bool) []string {
	var repos []string

	if recursive {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.IsDir() && d.Name() == ".git" {
				repos = append(repos, filepath.Dir(path))
				return filepath.SkipDir
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			gitDir := filepath.Join(root, e.Name(), ".git")
			if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
				repos = append(repos, filepath.Join(root, e.Name()))
			}
		}
	}

	return repos
}
