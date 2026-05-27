# How-To: Bulk Operations Across Many Repos

Stability: Stable

Goal: Keep many repositories in sync — pull, fetch, or switch to the default branch — without opening each one.

## When To Use

You manage 10+ repos cloned side-by-side and want to bring them all to a known-good state at the start of the day, or after a tooling-wide change announcement.

## Pull or Fetch Everything

From the **main screen**:

- `p` — pull the selected repo.
- `P` — pull **all** repos.
- `f` — fetch the selected repo.
- `F` — fetch **all** repos.

Bulk operations stream status into the bottom status bar (`bulk pull repo-x done`, etc.) and finish with `Bulk pull complete`. Fossor caps concurrency at 8 in-flight git processes — sufficient throughput without exhausting file descriptors or your network's connection budget.

## Switch Everyone to the Default Branch

After landing a merge across a project graph, you often want every checkout back on `main` (or `master`, or whatever each repo declares as default):

- `d` — switch the selected repo to its default branch.
- `D` — switch **all** repos to their default branches.

Per-repo defaults are detected from `refs/remotes/origin/HEAD`. Repos with uncommitted changes will refuse the switch and the bulk operation will continue past them; check the status messages.

## Skip the Initial Fetch

By default, discovery fetches every repo so the `Ahead` / `Behind` counts are fresh. On a directory of 200 repos this can take a minute or more. To start instantly with cached refs:

```bash
fossor ~/code --no-fetch
```

You can still fetch on demand with `f` / `F` once the TUI is up.

## Recursive Discovery

If your repos live under nested directories (e.g. `~/code/<org>/<repo>`), enable recursive scan:

```bash
fossor ~/code --recursive
```

Without `--recursive`, only the top level is scanned.

## Background Refresh of the Highlighted Repo

While you sit on the main screen, Fossor re-runs status checks on the currently-selected repo every 30 seconds in the background. You'll see counts update without any keystroke. Disable with:

```bash
fossor ~/code --no-auto-refresh
```

## Tip: Filter, Then Act

Combine filters with bulk actions to narrow the blast radius:

1. `t` until the filter shows `[Behind]`.
2. `P` to pull only the behind repos. (The bulk action operates on the *visible* filtered set, not the entire directory.)

## See Also

- [First run tutorial](../tutorials/first-run.md)
- [Keybindings reference](../reference/keybindings.md)
- [CLI flags](../reference/cli.md)
- [Status states](../reference/status-states.md)
