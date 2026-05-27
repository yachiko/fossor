# Design Choices & Trade-offs

Stability: Stable (narrative) / Some implementation details Provisional where noted.

Explains *why* Fossor adopts its current patterns and rejects alternatives.

## Stateless, No Config File

**Choice**: No on-disk config. All behavior is driven by CLI flags and environment variables.

- Pros: Install/uninstall is `go install` / `rm`. Two machines with the same flags behave identically. No "where do I put the config file?" question.
- Cons: Persistent preferences (favorite sort order, default `--open-cmd`) must be set in your shell init.
- Mitigation: Environment variables (`FOSSOR_OPEN_CMD`, `FOSSOR_DEBUG`) give a stable, shell-rc-friendly persistence story.
- Future: A config file may appear if persistent UI preferences accumulate — but only when we have at least three of them.

## Shell Out to `git`

**Choice**: Every git operation is `exec.Command("git", …)`. No `go-git` or libgit2 binding.

- Pros: 100% compatibility with the user's installed git. Credential helpers, hooks, SSH config, partial clone, sparse checkout, LFS — all work because they're handled by the real `git` binary.
- Cons: One process spawn per call. Overhead is real on directories with hundreds of repos.
- Mitigations: Discovery parallelizes per-repo. Bulk operations cap concurrency at 8. The Bubble Tea reactor lets the UI stay responsive while git processes run.
- Rejected: `go-git` doesn't implement enough of the git surface (no fetch with credential helpers, no LFS, partial protocol support).

## Interactive Commands Suspend the TUI

**Choice**: `C` (commit via editor), `B` (rebase -i), and merge-conflict states call `tea.ExecProcess` to release the terminal to `git`, then resume.

- Pros: Users get the full git interactive experience (`git commit -v`, `git rebase -i` with their preferred editor, conflict markers visible in the working tree).
- Cons: The TUI flickers off and back on. Bubble Tea has to redraw from scratch.
- Rejected: Embedding an editor or conflict resolver in the TUI — too much surface to maintain, and worse than the user's editor.

## Inline Commit Editor

**Choice**: `c` opens a one-line in-TUI input for short commit messages, with the staged diff still visible above.

- Pros: Fast path for the 90% of commits that are one short line.
- Cons: No multi-line, no commit template support.
- Mitigation: `C` (capital) hands off to `$EDITOR` for anything more.

## Bubble Tea over a Custom TUI Stack

**Choice**: Bubble Tea + Bubbles + Lip Gloss.

- Pros: Elm-style update loop maps cleanly onto async git operations. Mature components (text input, viewport, table). Active ecosystem.
- Cons: Frequent breaking changes in the v0/v1 era; constrains us to its update-loop discipline.
- Mitigation: Dependabot groups `charmbracelet/*` updates so we move in lockstep, not piecemeal.

## Cobra for Flags

**Choice**: `spf13/cobra` even though Fossor has no subcommands.

- Pros: Free `--version` plumbing, `--help` formatting, future-proofs for adding subcommands.
- Cons: Slight binary-size bloat versus stdlib `flag`.
- Future: If we never grow subcommands, switching to `pflag` directly is a possible cleanup. Low priority.

## Discovery Streams to the UI

**Choice**: Each discovered repo becomes a `tea.Msg` immediately rather than waiting for the full scan to complete.

- Pros: User sees their repos appear in the table as discovery progresses; the TUI never blocks. Indispensable on directories with hundreds of repos.
- Cons: The header status counts shift while you're still on the main screen.
- Mitigation: The status bar shows a spinner + count while discovery runs.

## Bulk Operation Concurrency Capped at 8

**Choice**: Bulk pull/fetch/switch issue at most 8 in-flight git processes.

- Rationale: Beyond ~8, you hit diminishing returns on most networks while consuming file descriptors and producing log spam. Picked empirically.
- Trade-off: Slower than unbounded parallelism on extremely high-bandwidth networks with hundreds of small repos.
- Future: A `--bulk-concurrency` flag could expose this. Not a priority until someone asks.

## Stale-Lock Recovery

**Choice**: Auto-remove `.git/*.lock` files older than 5s **and** not held by any live process per `lsof`. Logged to `~/.cache/fossor/debug.log` when `FOSSOR_DEBUG=1`.

- Pros: A crashed git process used to permanently break a repo in the table until the user manually removed the lock. The recovery is conservative — both age and holder checks must pass.
- Cons: If `lsof` isn't installed, only the mtime check runs (still safe in practice; 5s is well past any normal git operation).
- Mitigation: Debug log captures every recovery so users can audit.

## Single Binary, No Plugin System

**Choice**: One static binary. No external scripts or plugins.

- Pros: GoReleaser ships one artifact per platform. No "did you install the plugin?" support tickets.
- Cons: New features need a release.
- Future: If the action grid grows past ~20 entries, a "user-defined actions" YAML may be worth adding.

## Submodule Handling

**Choice**: Detect submodules and label them in the file list, with submodule-aware diffs. Don't recurse into them.

- Pros: Users see their submodules. Diffs make sense (don't try to compute the diff of a submodule pointer as if it were a file).
- Cons: No bulk actions across submodules.
- Future: If demand emerges, a `--recursive-submodules` style flag could open the door.

## Trade-offs Summary

| Area                | Decision                                | Primary Benefit                  | Main Trade-off               | Status      |
| ------------------- | --------------------------------------- | -------------------------------- | ---------------------------- | ----------- |
| State               | None on disk (apart from debug log)     | Trivial install/uninstall        | No persistent preferences    | Stable      |
| Git layer           | Shell out to `git`                      | Full compatibility               | Process overhead             | Stable      |
| Interactive ops     | Suspend & hand off                      | Full git experience              | TUI redraw cost              | Stable      |
| Commit UX           | Inline editor for short, `$EDITOR` for long | Fast common path             | Two keybinds, two flows      | Stable      |
| TUI framework       | Bubble Tea                              | Mature reactor model             | Breaking changes occasional  | Stable      |
| CLI framework       | Cobra                                   | Free `--version` / `--help`      | Binary-size cost             | Stable      |
| Discovery           | Streaming via channel                   | Live feedback                    | More UI churn                | Stable      |
| Bulk parallelism    | Capped at 8                             | Predictable on big trees         | Slower on huge networks      | Provisional |
| Stale-lock recovery | Auto, behind mtime + `lsof` gates       | Self-heals after git crashes     | Requires `lsof` for full safety | Stable    |

## Rejected Alternatives

| Alternative                              | Reason Rejected                                                | Revisit Trigger                                                      |
| ---------------------------------------- | -------------------------------------------------------------- | -------------------------------------------------------------------- |
| `go-git` instead of shelling out         | Missing protocol/credential/LFS coverage                       | If `go-git` reaches parity with credential helpers                   |
| Plugin/extension system                  | Significant maintenance for unclear demand                     | When the action grid grows past ~20 entries                          |
| Persistent config file                   | Premature for the current preference surface                   | When ≥3 persistent UI preferences accumulate                         |
| Embedded merge-conflict resolver         | Worse than the user's editor; large surface                    | Probably never                                                       |
| Cross-repo "supercommit"                 | Encourages bad cross-repo coupling; surprising semantics       | When users credibly request it with concrete use cases               |

## See Also

- [Architecture overview](architecture.md)
- [CLI reference](../reference/cli.md)
- [Bulk operations how-to](../how-to/bulk-operations.md)
