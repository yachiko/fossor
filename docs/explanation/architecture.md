# Architecture Overview

Stability: Stable

High-level flow (happy path):

1. `cmd.Execute()` resolves the root directory and constructs the `ui.App` Bubble Tea model.
2. `App.Init()` starts the discovery pipeline in a goroutine, kicks the spinner, and (unless `--no-auto-refresh`) schedules the 30-second refresh tick.
3. Discovery walks the selected directory, validates Git worktrees (including linked worktrees), excludes submodules, fans out `git` invocations per checkout, and streams `RepoDiscoveredMsg` back through a Bubble Tea command.
4. Each message lands in `App.Update`, which forwards to `mainscreen.UpdateRepo` to grow the table live.
5. The user navigates with the keybindings; pressing `Enter` constructs a `manageview.Model` and switches screens.
6. The manage view loads commits / stash / branches lazily as tabs are entered, and renders the Status tab from `git status --porcelain` + `git diff` output.
7. Actions that need terminal access (commit editor, `rebase -i`, conflict resolution) suspend the TUI via `tea.ExecProcess` and hand the terminal to `git`.

## Components

| Component                    | File(s)                                | Responsibility                                                                              |
| ---------------------------- | -------------------------------------- | ------------------------------------------------------------------------------------------- |
| Root command                 | `cmd/root.go`                          | Cobra wiring, flag definitions, root-dir validation, `--version`.                            |
| App model                    | `internal/ui/app.go`                   | Top-level Bubble Tea model. Switches between main screen and manage view. Owns discovery.    |
| Main screen                  | `internal/ui/mainscreen/`              | Checkout table, worktree-family grouping, sort/filter, status counts, bulk actions.          |
| Manage view                  | `internal/ui/manageview/`              | Four-tab per-repo workspace. Action registry, inline commit editor.                          |
| Common UI                    | `internal/ui/common/`                  | Theme colors, shared messages, key constants.                                                 |
| Status bar                   | `internal/ui/components/statusbar.go`  | Reusable bottom-of-screen help/status line.                                                  |
| Git layer                    | `internal/git/`                        | Wraps the `git` CLI behind a `Git` interface. Discovery, status, log, fetch/pull/push.       |

## Update / View Cycle

Fossor follows the Elm-architecture pattern that Bubble Tea encodes:

- `Init() tea.Cmd` — kick off async work at startup (`startDiscovery`, spinner tick, refresh tick).
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)` — pure transition: take a message, mutate model, optionally return a new command.
- `View() string` — render the current model to a string.

Cross-screen state flows through messages in `internal/ui/common/messages.go`:

- `RepoDiscoveredMsg` — one repo's initial info is in.
- `DiscoveryCompleteMsg` — channel closed; scan done.
- `RepoUpdatedMsg` — a repo's state changed (post-refresh or post-action).
- `SwitchToManageMsg` / `SwitchToMainMsg` — screen transitions.
- `OperationResultMsg` / `BulkOperationTickMsg` — bulk action progress.
- `StatusMsg` / `StatusClearMsg` / `RefreshTickMsg` — status bar / periodic refresh plumbing.

## Discovery Pipeline

`internal/git/discovery.go`:

```
Walk directory ──► per-repo goroutine ──► channel ──► tea.Msg
                  (parallelism = NumCPU)
```

Each goroutine constructs a `RepoInfo` by calling `Git.GetRepoInfo`, which under the hood runs:

- `git rev-parse --abbrev-ref HEAD` (branch)
- `git remote get-url origin` (remote URL)
- Optional `git fetch` (skipped with `--no-fetch`)
- After a successful fetch, `git ls-remote --symref origin HEAD` refreshes the cached default branch
- `git rev-list --left-right --count` (ahead/behind)
- `git status --porcelain=v1` (changes)

The Tea `Cmd` that wraps the channel is re-issued on every received message, so the UI streams one repo per turn until the channel closes.

## Action Registry

`internal/ui/manageview/registry.go` defines a list of declarative `Action` entries — each with a key, label, condition for enabling, optional confirmation gate, and a handler that returns a `tea.Cmd`. The action grid view reads this registry to render the bottom half of the Status tab. Adding a new repo action is a one-entry registry change plus the handler.

## Concurrency

- Discovery: fan-out goroutines + buffered channel.
- Fetch and pull operations sharing a resolved common Git directory are serialized by `OperationCoordinator`; local checkout operations remain independent.
- Background refresh: a 30-second `tea.Tick` posts `RefreshTickMsg`; only the main screen acts on it, and only for the selected repo.
- All `git` calls go through `exec.CommandContext` so they respect cancellation when the app shuts down.

## Stale Lock Recovery

Inside the `git` wrapper, every command is run through `runGitOnce`. On failure, the error message is matched against a small set of well-known lock-error markers; if any match, `tryClearStaleLocks` checks each well-known `.git/*.lock` file's mtime and (when `lsof` is available) holder process, removes the lock, and retries the original command once. See `SECURITY.md` for the threat-model framing.

## Discovery Cache

Fossor has no configuration file. It stores the last complete, non-cancelled discovery snapshot in `~/.cache/fossor/repositories.json`, keyed by absolute root and recursive mode. The versioned JSON file is private and atomically replaced. Read and write failures are ignored, so the cache never prevents startup or discovery.

Cached rows are hydrated synchronously as unverified and change to checking only while their live refresh runs. They are replaced progressively. A completed scan replaces the scoped snapshot, removing repositories that no longer exist. Cache contents include resolved common-Git-dir identity so linked worktrees can be grouped immediately; remote state is always refreshed by the live scan unless `--no-fetch` is used.

## See Also

- [Design choices](design-choices.md) — the *why* behind each architectural pick
- [Status states reference](../reference/status-states.md) — how `Status` is computed
- [Troubleshooting](../reference/troubleshooting.md) — debug log and stale-lock recovery
