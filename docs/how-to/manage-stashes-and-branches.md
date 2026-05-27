# How-To: Manage Stashes and Branches

Stability: Stable

Goal: Stash work-in-progress and drive branch operations (switch / create / rename / delete) from inside Fossor.

## Stash a Dirty Working Tree

From the **Status tab** of the manage view (press `1` if you're not there):

- `s` — `git stash push` your working tree. The TUI returns to a clean state.
- `S` — `git stash pop` the most recent entry.

For finer-grained stash control (multiple entries, apply without pop, drop a specific entry), use the **Stash tab** (press `3`):

| Key            | Action                              |
| -------------- | ----------------------------------- |
| `Up` / `Down`  | Navigate stash entries              |
| `PgUp` / `PgDn`| Scroll the diff preview             |
| `p`            | Pop the selected stash              |
| `d`            | Drop the selected stash             |

The Stash tab shows a diff preview of the highlighted entry on the right.

## Switch Branches

Press `4` for the **Branches tab**. You'll see every local branch with ahead/behind counts relative to the default branch and a merged indicator.

| Key                  | Action                                     |
| -------------------- | ------------------------------------------ |
| `Up` / `Down`        | Navigate the branch list                   |
| `Enter` or `s`       | `git switch` to the selected branch        |
| `n`                  | Create a new branch (prompts for name)     |
| `r`                  | Rename the selected branch                 |
| `d`                  | Safe delete (refuses if unmerged)          |
| `D`                  | Force delete                               |

**Note:** If your working tree is dirty, switching will fail. Either stash (`s` on the Status tab) or commit first.

## Common Recipes

### Create a feature branch from `main`

1. `4` to open the Branches tab.
2. Cursor to `main`, `Enter` to switch.
3. `n` to create, type the branch name, `Enter`.

You're now on the new branch.

### Throw away a local-only experiment

1. `4` to open the Branches tab.
2. Cursor to the experiment branch.
3. `D` for force delete. Confirms before deleting.

### Recover from "oh no I started a commit on `main`"

1. `1` to return to the Status tab.
2. `z` — `git reset --soft HEAD~1`. The commit is unmade but your changes stay staged.
3. `4` to the Branches tab.
4. `n`, type a feature branch name, `Enter`.
5. Back to `1`, commit normally (`c` or `C`).

## See Also

- [Keybindings reference](../reference/keybindings.md)
- [Bulk operations how-to](../how-to/bulk-operations.md)
- [Architecture explanation](../explanation/architecture.md) — why some actions stay in-TUI and some hand off to git
