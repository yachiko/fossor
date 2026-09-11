package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const (
	localStatusWorkers = 8
	fetchWorkers       = 16
)

// DiscoveryResult carries a discovered repo or indicates completion.
type DiscoveryResult struct {
	Repo           RepoInfo
	FetchErr       error
	Path           string
	Skipped        bool
	Revision       uint64
	CoordinatorKey string
	Local          bool
	LocalDone      bool
}

// DiscoveryOptions configures discovery behavior.
type DiscoveryOptions struct {
	RootDir     string
	Recursive   bool
	Git         Git
	Fetch       bool
	Coordinator *OperationCoordinator
	CachedRepos map[string]RepoInfo
}

// Discover streams each local status before queuing its optional remote refresh.
// LocalDone marks the complete local snapshot; the channel closes only after all
// queued refreshes have produced a terminal result.
func Discover(ctx context.Context, opts DiscoveryOptions) <-chan DiscoveryResult {
	ch := make(chan DiscoveryResult, 32)

	go func() {
		defer close(ch)
		repoPaths := findRepos(ctx, opts.RootDir, opts.Recursive, opts.Git)
		paths := make(chan string)
		var refreshInput chan RepoInfo
		refreshes := make(chan RepoInfo)
		var localWG, fetchWG, schedulerWG sync.WaitGroup

		for range localStatusWorkers {
			localWG.Add(1)
			go func() {
				defer localWG.Done()
				for path := range paths {
					info, err := opts.Git.GetRepoInfo(ctx, path)
					if err != nil {
						continue
					}
					if cached, ok := opts.CachedRepos[path]; ok && cached.DefaultBranch != "" {
						info.DefaultBranch = cached.DefaultBranch
						info.Status = computeStatus(info)
					}
					if !sendDiscovery(ctx, ch, DiscoveryResult{Repo: info, Path: path, Local: true}) {
						return
					}
					if opts.Fetch && !sendRepo(ctx, refreshInput, info) {
						return
					}
				}
			}()
		}

		if opts.Fetch {
			refreshInput = make(chan RepoInfo)
			for range fetchWorkers {
				fetchWG.Add(1)
				go func() {
					defer fetchWG.Done()
					for local := range refreshes {
						result := refreshRepo(ctx, local, opts.Git, opts.Coordinator)
						if !sendDiscovery(ctx, ch, result) {
							return
						}
					}
				}()
			}
			schedulerWG.Add(1)
			go func() {
				defer schedulerWG.Done()
				dispatchRefreshes(ctx, refreshInput, refreshes)
			}()
		}

		for _, path := range repoPaths {
			if !sendPath(ctx, paths, path) {
				close(paths)
				localWG.Wait()
				if opts.Fetch {
					close(refreshInput)
					schedulerWG.Wait()
					fetchWG.Wait()
				}
				return
			}
		}
		close(paths)
		localWG.Wait()
		if !sendDiscovery(ctx, ch, DiscoveryResult{LocalDone: true}) {
			if opts.Fetch {
				close(refreshInput)
				schedulerWG.Wait()
				fetchWG.Wait()
			}
			return
		}
		if opts.Fetch {
			close(refreshInput)
			schedulerWG.Wait()
			fetchWG.Wait()
		}
	}()

	return ch
}

// dispatchRefreshes decouples local status workers from slow remote operations
// while keeping the number of active fetches bounded by fetchWorkers.
func dispatchRefreshes(ctx context.Context, input <-chan RepoInfo, workers chan<- RepoInfo) {
	defer close(workers)
	var pending []RepoInfo
	for input != nil || len(pending) > 0 {
		// Prefer accepting an already-waiting local result over dispatching another
		// fetch so remote backpressure cannot starve local discovery.
		select {
		case repo, ok := <-input:
			if !ok {
				input = nil
			} else {
				pending = append(pending, repo)
			}
			continue
		default:
		}

		var output chan<- RepoInfo
		var next RepoInfo
		if len(pending) > 0 {
			output = workers
			next = pending[0]
		}
		select {
		case repo, ok := <-input:
			if !ok {
				input = nil
			} else {
				pending = append(pending, repo)
			}
		case output <- next:
			pending = pending[1:]
		case <-ctx.Done():
			return
		}
	}
}

func refreshRepo(ctx context.Context, local RepoInfo, g Git, coordinator *OperationCoordinator) DiscoveryResult {
	rp := local.Path
	var revision uint64
	var fetchErr error
	if coordinator != nil {
		var ran bool
		key := local.CoordinatorKey()
		revision, ran, fetchErr = coordinator.Run(ctx, key, false, func(ctx context.Context) error {
			return g.Fetch(ctx, rp)
		})
		if !ran {
			return DiscoveryResult{Path: rp, Skipped: true, Revision: revision, CoordinatorKey: key}
		}
	} else {
		fetchErr = g.Fetch(ctx, rp)
	}
	info, err := g.GetRepoInfo(ctx, rp)
	if err != nil {
		info = local
		fetchErr = errors.Join(fetchErr, err)
	}
	if local.DefaultBranch != "" {
		info.DefaultBranch = local.DefaultBranch
		info.Status = computeStatus(info)
	}
	if fetchErr == nil {
		if branch, err := g.GetRemoteDefaultBranch(ctx, rp); err == nil && branch != "" {
			info.DefaultBranch = branch
			info.Status = computeStatus(info)
		}
	}
	return DiscoveryResult{Repo: info, FetchErr: fetchErr, Path: rp, Revision: revision, CoordinatorKey: local.CoordinatorKey()}
}

func sendDiscovery(ctx context.Context, ch chan<- DiscoveryResult, result DiscoveryResult) bool {
	select {
	case ch <- result:
		return true
	case <-ctx.Done():
		return false
	}
}

func sendPath(ctx context.Context, ch chan<- string, path string) bool {
	select {
	case ch <- path:
		return true
	case <-ctx.Done():
		return false
	}
}

func sendRepo(ctx context.Context, ch chan<- RepoInfo, repo RepoInfo) bool {
	select {
	case ch <- repo:
		return true
	case <-ctx.Done():
		return false
	}
}

// findRepos returns worktree roots physically within root. A detector validates
// .git directories and pointer files when the Git implementation supports it.
func findRepos(ctx context.Context, root string, recursive bool, g Git) []string {
	var repos []string
	valid := func(path string) bool {
		if detector, ok := g.(interface {
			IsWorktree(context.Context, string) bool
		}); ok {
			return detector.IsWorktree(ctx, path)
		}
		return true
	}

	if recursive {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.Name() == ".git" && (d.IsDir() || d.Type().IsRegular()) {
				if repo := filepath.Dir(path); valid(repo) {
					repos = append(repos, repo)
				}
				if d.IsDir() {
					return filepath.SkipDir
				}
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
			if info, err := os.Stat(gitDir); err == nil && (info.IsDir() || info.Mode().IsRegular()) && valid(filepath.Join(root, e.Name())) {
				repos = append(repos, filepath.Join(root, e.Name()))
			}
		}
	}

	return repos
}
