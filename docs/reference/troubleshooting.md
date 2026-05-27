# Troubleshooting

Stability: Stable

Diagnostics and recovery for the most common failure modes.

## A Repo Is Stuck on `…`

A repo whose status stays at `…` (loading) usually has a stale `.git/*.lock` file from a crashed git process.

Fossor auto-recovers locks that are:

1. Older than 5 seconds (`staleLockThreshold` in `internal/git/git.go`).
2. Not held by any live process per `lsof`. If `lsof` isn't on `PATH`, this check is skipped and the lock is removed based on mtime alone.

The lock files Fossor will consider:

- `.git/index.lock`
- `.git/HEAD.lock`
- `.git/config.lock`
- `.git/packed-refs.lock`
- `.git/shallow.lock`

If a repo *still* stays stuck after a refresh, enable debug logging:

```bash
FOSSOR_DEBUG=1 fossor ~/code
```

Recoveries are appended to `~/.cache/fossor/debug.log` with the repo path, lock file, and age at removal. Inspect that file to see what happened.

## `fossor: command not found` After `go install`

`go install` puts binaries in `$(go env GOBIN)`, which defaults to `~/go/bin`. Ensure that directory is on your `PATH`:

```bash
export PATH="$(go env GOBIN):$PATH"
# or
export PATH="$HOME/go/bin:$PATH"
```

## `fossor --version` Reports `dev`

A version of `dev` means neither the build-time `-ldflags` injection nor `debug.ReadBuildInfo()` returned a usable version. Possible causes:

- Built via plain `go build` from a working tree without `-ldflags`. Use `make build` instead.
- Installed via `go install <module>@<commit-without-a-tag>` long enough ago that the Go proxy returned a generic version. Re-install with `@latest` once the proxy refreshes.

## `--version` Lags After a New Tag

`go install github.com/yachiko/fossor@latest` may resolve to the previous tag for up to an hour after a new tag is pushed — the Go module proxy lazily updates its `@latest` index. Pin the version explicitly to bypass the cache:

```bash
go install github.com/yachiko/fossor@v0.1.2
```

## The Manage View Action Suspended the TUI and Didn't Come Back

This happens when Fossor hands off to `git` for an interactive command (`C`, `B`, merge conflicts). Either:

- The git command is still waiting for your input — finish or abort it.
- `$EDITOR` is misconfigured. Try `EDITOR=vi fossor ~/code` and retry.
- If you accidentally backgrounded the foreground git process (`Ctrl+Z`), bring it back with `fg`.

If all else fails, `Ctrl+C` will exit the suspended git command, then Fossor will redraw.

## Permission Denied on `git fetch`

Fossor doesn't manage credentials — it inherits your environment. Confirm your credentials work outside the TUI:

```bash
git -C <repo> fetch
```

Fix the underlying credential issue (SSH key, credential helper, `$GIT_ASKPASS`, etc.) and the TUI will pick it up on the next run or refresh.

## Bulk Pull Is Slow

By design, bulk operations cap concurrency at 8 in-flight git processes. On directories with hundreds of repos, this is a deliberate trade-off — higher concurrency hits diminishing returns and saturates file descriptors / network. If a small subset is the bottleneck, filter (`t`) before triggering `P`.

## Colors Look Off / Status Column Invisible

Fossor uses `lipgloss` color defaults. On terminals with custom palettes, `Up to date` (green) and `Behind` / `Dirty` (yellow) may render differently. Check that your `$TERM` is set to a 256-color value (`xterm-256color`, `screen-256color`, …):

```bash
echo $TERM
```

## See Also

- [CLI reference](cli.md) (for env vars including `FOSSOR_DEBUG`)
- [Status states reference](status-states.md)
- [Architecture explanation](../explanation/architecture.md)
