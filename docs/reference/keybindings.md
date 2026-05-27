# Keybindings

Stability: Stable

Canonical, complete keybinding reference. The README excerpts the most common ones; this page is the authoritative source.

## Main Screen

| Key       | Action                                                         |
| --------- | -------------------------------------------------------------- |
| `Enter`   | Open the highlighted repo in the manage view                   |
| `s` / `/` | Start search (substring over name, branch, status)             |
| `t`       | Cycle status filter (`All` → `Error` → … → `Up to date`)       |
| `1`–`6`   | Sort by column (`Name`, `Branch`, `Ahead`, `Behind`, `Changes`, `Status`); press again to reverse |
| `p`       | Pull selected repo                                             |
| `P`       | Pull **all** visible (filter-aware) repos                      |
| `f`       | Fetch selected repo                                            |
| `F`       | Fetch **all** visible repos                                    |
| `d`       | Switch selected repo to its default branch                     |
| `D`       | Switch **all** visible repos to their default branches         |
| `o`       | Open in external editor (only when `--open-cmd` / `$FOSSOR_OPEN_CMD` is set) |
| `j` / `↓` | Move cursor down                                               |
| `k` / `↑` | Move cursor up                                                 |
| `Esc`     | Exit search (when active)                                      |
| `q`       | Quit                                                           |

## Manage View — Status Tab

| Key            | Action                                              |
| -------------- | --------------------------------------------------- |
| `Up` / `Down`  | Navigate file list                                  |
| `PgUp` / `PgDn`| Scroll diff preview                                 |
| `p`            | Pull                                                |
| `R`            | Pull `--rebase`                                     |
| `u`            | Push                                                |
| `f`            | Fetch                                               |
| `d`            | Switch to default branch                            |
| `s`            | Stash                                               |
| `S`            | Stash pop                                           |
| `a`            | Stage all                                           |
| `i`            | Stage selected file                                 |
| `I`            | Unstage selected file                               |
| `c`            | Commit (inline editor; `Ctrl+S` to confirm)         |
| `C`            | Commit via `$EDITOR` (suspends TUI)                 |
| `x`            | Restore selected file (`git restore`)               |
| `X`            | Delete selected untracked file                      |
| `U`            | `git submodule update --init`                       |
| `b`            | Rebase                                              |
| `B`            | Interactive rebase (suspends TUI)                   |
| `m`            | Merge                                               |
| `z`            | `git reset --soft HEAD~1` (undo last commit, keep staged) |
| `Z`            | `git reset --hard HEAD~1` (destructive; confirms)   |
| `k`            | Cherry-pick                                         |
| `Tab`          | Next tab                                            |
| `1`–`4`        | Jump directly to tab `Status` / `History` / `Stash` / `Branches` |
| `Esc`          | Back to main screen                                 |

## Manage View — History Tab

| Key            | Action                |
| -------------- | --------------------- |
| `Up` / `Down`  | Scroll commit log     |
| `PgUp` / `PgDn`| Page scroll           |

## Manage View — Stash Tab

| Key            | Action                              |
| -------------- | ----------------------------------- |
| `Up` / `Down`  | Navigate stash entries              |
| `PgUp` / `PgDn`| Scroll stash diff preview           |
| `p`            | Pop selected stash                  |
| `d`            | Drop selected stash                 |

## Manage View — Branches Tab

| Key            | Action                                     |
| -------------- | ------------------------------------------ |
| `Up` / `Down`  | Navigate branches                          |
| `Enter` / `s`  | Switch to branch                           |
| `n`            | Create new branch (prompts for name)       |
| `r`            | Rename branch                              |
| `d`            | Safe delete (refuses if unmerged)          |
| `D`            | Force delete                               |

## Inline Commit Editor

| Key      | Action                                |
| -------- | ------------------------------------- |
| `Ctrl+S` | Confirm and commit                    |
| `Esc`    | Cancel                                |

## Global

| Key      | Action                                                          |
| -------- | --------------------------------------------------------------- |
| `Ctrl+C` | Quit immediately (cancels any in-flight discovery / bulk op)    |

## Destructive Actions

These prompt for confirmation before running:

- `Z` — `git reset --hard HEAD~1`
- `X` — delete untracked file
- `D` (Branches tab) — force-delete branch

## See Also

- [First run tutorial](../tutorials/first-run.md)
- [Bulk operations how-to](../how-to/bulk-operations.md)
- [Status states reference](status-states.md)
