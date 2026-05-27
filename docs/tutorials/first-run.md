# Tutorial: First Run on a Directory of Repos

Stability: Stable

Stated Goal: Install Fossor, scan a directory of git repos, navigate the main screen, and drive a manage view.

Estimated Time: 5 minutes.

## Prerequisites

- Go 1.25+ installed (`go version` works), **or** a prebuilt binary from the [Releases page](https://github.com/yachiko/fossor/releases).
- `git` 2.20+ on your `PATH`.
- A directory containing at least a handful of git repositories. For a curated set, the project ships a generator — see step 1b.

## 1a. Install

```bash
go install github.com/yachiko/fossor@latest
```

The binary lands in `$(go env GOBIN)` (usually `~/go/bin`). Confirm:

```bash
fossor --version
```

## 1b. (Optional) Generate a Demo Directory

If you don't have a directory of repos handy, the project's `make testdata` target creates 20 mock repos in `testdata/repos/` covering every status state (clean, dirty, ahead, behind, diverged, stashed, submodules, …):

```bash
git clone https://github.com/yachiko/fossor.git
cd fossor
make testdata
./testdata/repos    # this is the directory you'll point fossor at
```

## 2. Launch

```bash
fossor ~/code            # or wherever your repos live
```

For first-run speed, skip the network fetch on startup:

```bash
fossor ~/code --no-fetch
```

You'll land on the **main screen** — a table of every repo Fossor discovered, with columns for `Name`, `Branch`, `Ahead`, `Behind`, `Changes`, `Status`. The header line summarizes how many repos sit in each state.

## 3. Navigate the Main Screen

- `j` / `k` (or `↓` / `↑`) — move the cursor.
- `1`–`6` — sort by that column (press again to reverse).
- `s` or `/` — start a substring search; `Esc` exits search.
- `t` — cycle the status filter (`All` → `Error` → `Non-default` → `Diverged` → `Behind` → `Ahead` → `Dirty` → `Up to date`).
- `Enter` — open the highlighted repo in the manage view.

Full reference: [`reference/keybindings.md`](../reference/keybindings.md).

## 4. Drive the Manage View

Select a repo and press `Enter`. The manage view has four tabs:

| Tab        | Press     | What you see                                                     |
| ---------- | --------- | ---------------------------------------------------------------- |
| Status     | `1`       | File list + diff preview + action grid (pull, push, commit, …)   |
| History    | `2`       | Scrollable `git log`                                             |
| Stash      | `3`       | Stash entries with diff preview                                  |
| Branches   | `4`       | Branch table with ahead/behind indicators                        |

`Tab` cycles tabs forward. `Esc` returns to the main screen.

On the **Status tab**, try:

- `i` to stage the selected file, `I` to unstage.
- `c` to write a commit message inline (staged diff stays visible above), then `Ctrl+S` to commit.
- `p` to pull, `u` to push.

If you trigger an interactive operation — `C` (commit via editor), `B` (rebase -i), or you hit a merge conflict — Fossor suspends, hands the terminal to `git`, and resumes when you exit.

## 5. Quit

`q` from the main screen, or `Esc` from any nested view followed by `q`.

## See Also

- [Keybindings reference](../reference/keybindings.md)
- [Bulk operations how-to](../how-to/bulk-operations.md)
- [Status states reference](../reference/status-states.md)
