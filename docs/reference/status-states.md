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

- **Branch detection**: `git rev-parse --abbrev-ref HEAD`.
- **Default branch detection**: from `refs/remotes/origin/HEAD`; falls back to `main`, then `master`.
- **Ahead / behind**: `git rev-list --left-right --count <default>...<current>`.
- **Changes**: count of porcelain entries from `git status --porcelain=v1`.

If `--no-fetch` is passed, remote refs are not refreshed before this computation; `Ahead` / `Behind` will reflect the last fetch.

## Status Filter

Press `t` on the main screen to cycle through the filter. The filter only includes states that have at least one matching repo, plus `All` at the end. Bulk actions (`P`, `F`, `D`) operate on the filtered view, not the entire directory.

## See Also

- [Keybindings reference](keybindings.md)
- [Bulk operations how-to](../how-to/bulk-operations.md)
- [Architecture explanation](../explanation/architecture.md)
