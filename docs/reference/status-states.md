# Repository Status States

Stability: Stable

The `Status` column on the main screen summarizes each repo into one of the states below. Each state has a color and a precedence in the status filter cycle.

## States

| State         | Color   | Meaning                                                                                       |
| ------------- | ------- | --------------------------------------------------------------------------------------------- |
| `Up to date`  | Green   | On the default branch, no ahead/behind versus remote, no uncommitted changes.                 |
| `Ahead`       | Blue    | On the default branch, has local commits not yet pushed.                                      |
| `Behind`      | Yellow  | On the default branch, has remote commits not yet pulled.                                     |
| `Dirty`       | Yellow  | On the default branch, has uncommitted changes (staged, unstaged, or untracked).              |
| `Diverged`    | Yellow  | On the default branch, both ahead **and** behind the remote.                                  |
| `Non-default` | Red     | Currently checked out to a branch that isn't the default. Overrides any other state.          |
| `Error`       | Red     | Discovery failed (e.g. corrupted `.git`, permission denied). Inspect with `FOSSOR_DEBUG=1`.   |
| `…` (dots)    | Muted   | Discovery is still in progress for this repo.                                                 |
| `stale`       | Muted   | A row restored from the discovery cache that has not been read yet (usually a fraction of a second). Not counted, filterable or actionable. |
| `remote error` | Red | Local inspection worked but `git fetch` failed. Pull and fetch remain available; it is not counted or filterable. |

While a repository's discovery fetch is still running, its whole row is dimmed while still showing its live status and status colour: local state (branch, changes, non-default) is current, and only Ahead / Behind may change once the fetch lands. Such rows are counted, filterable and fully actionable, including in the manage view and bulk operations; pull, fetch and push perform their own network round-trip and take priority over the pending discovery fetch.

## Precedence

When multiple conditions apply, the higher-precedence state wins. From most → least urgent:

1. `Error`
2. `Non-default`
3. `Diverged`
4. `Behind`
5. `Ahead`
6. `Dirty`
7. `Up to date`

This ordering drives the header summary line on the main screen and the `t` filter cycle order.

## How the Status Is Computed

- **Branch, ahead / behind, changes**: one `git status --porcelain=v2 --branch`; ahead / behind compare against the branch's upstream.
- **Default branch detection**: from `refs/remotes/origin/HEAD`; falls back to `main`, then `master`. After a successful fetch, discovery confirms it with `git ls-remote --symref origin HEAD` at most once every 24 hours.

If `--no-fetch` is passed, remote refs are not refreshed before this computation; `Ahead` / `Behind` will reflect the last fetch.

## Status Filter

Press `t` on the main screen to cycle through the filter. The filter only includes verified states that have at least one matching repo, plus `All` at the end. Bulk actions (`P`, `F`, `D`) operate on verified rows in the filtered view, not the entire directory.

## See Also

- [Keybindings reference](keybindings.md)
- [Bulk operations how-to](../how-to/bulk-operations.md)
- [Architecture explanation](../explanation/architecture.md)
