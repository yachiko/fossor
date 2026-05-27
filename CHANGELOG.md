# Changelog

All notable changes to Fossor are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`.
- Diátaxis-structured documentation under `docs/` (tutorials, how-to, reference, explanation, plus `index.md` and `style.md`).

### Security
- Strip C0/C1 control characters from repo-supplied strings (commit subjects, author names, branch refs, file paths, branch listings) before they reach the terminal. Prevents ANSI/escape-sequence injection that could spoof TUI elements or hijack the cursor. Exported `git.Sanitize` so the `manageview` branch parser can use the same helper.
- **Fix RCE in rebase action via poisoned `DefaultBranch` (F1/F2).** Every git command that passes a repo- or user-controlled refspec now inserts a `--` separator before the value. Previously, a poisoned `.git/refs/remotes/origin/HEAD` containing `ref: refs/remotes/origin/--exec=<cmd>` allowed silent command execution when the user pressed `b` (rebase) or `B` (rebase -i) on a tracking branch with commits ahead — `git rebase --exec=<cmd>` runs `<cmd>` after every replayed commit. Same defense applied to: `git switch`, `git rev-list`, `git merge`, `git cherry-pick`, `git branch --merged`, `git branch -m / -d / -D`, and the `git branch <name>` branch-create path. Added a regression test exercising every affected action.

## [0.1.2] - 2026-05-26

Hardening pass: CI now exercises the race detector and runs `golangci-lint`; the codebase was scrubbed clean against the latter. README gains a project logo.

### Added
- Race detector enabled in CI (`go test -race ./...`).
- `golangci-lint` workflow with `.golangci.yml` (standard linter set + misspell + gofmt).
- Dependabot config covering Go modules and GitHub Actions, with grouped charmbracelet/* and golang.org/x/* updates.
- Project logo (`logo.png`) shown in README header.

### Changed
- `golangci-lint-action` bumped from v6 → v7 to match the v2 linter pin.

### Fixed
- All errcheck findings (`Close`/`Fprintf`/`Sscanf`/`Mkdir` return values are now explicitly handled or discarded).
- All staticcheck QF1012 findings (`b.WriteString(fmt.Sprintf(...))` → `fmt.Fprintf(&b, ...)`).

## [0.1.1] - 2026-05-25

Patch release fixing the `--version` display for users installing via `go install`.

### Fixed
- `--version` now reports the module version when installed via `go install <module>@<version>` (falls back to `debug.ReadBuildInfo()` when `-ldflags` did not inject a value). Plain `go build` from a working tree still reports `dev`.

## [0.1.0] - 2026-05-25

Initial public release.

### Added
- Terminal UI for managing multiple Git repositories from a single screen, built on Bubble Tea.
- Repository discovery with parallel fetching and live streaming results to the UI.
- Main screen: sortable, filterable table of all repos with status counts header.
- Manage view: four-tab workspace (Status / History / Stash / Branches) per repo.
- Status tab with file list, pretty diffs, staged/unstaged indicators, submodule detection, and a full action grid (pull, push, fetch, rebase, stash, commit, restore, delete, submodule update, …).
- Inline commit editor (write commit message in-app with staged diff visible, or hand off to `$EDITOR`).
- Interactive git commands (commit editor, `rebase -i`, merge conflicts) suspend the TUI and hand the terminal to `git`.
- Branches tab: switch / create / rename / delete (safe + force) with ahead-behind indicators relative to the default branch.
- Stash tab: list, diff preview, pop, drop.
- History tab: scrollable commit log.
- Periodic background refresh of the selected repo every 30s; `--no-auto-refresh` to disable.
- Bulk operations from the main screen: pull all, fetch all, switch all to default branch (parallelism capped at 8).
- Optional `--open-cmd` (or `$FOSSOR_OPEN_CMD`) to open the selected repo in an external editor with `o`.
- Stale `.git/*.lock` recovery: locks older than 5s and not held by any process per `lsof` are removed automatically; `FOSSOR_DEBUG=1` logs the recoveries to `~/.cache/fossor/debug.log`.
- `--version` flag (Cobra-wired), `--recursive` (-r), `--no-fetch`, `--no-auto-refresh`, `--open-cmd`.
- CI workflow (`go vet`, `go test`, `go build` on `ubuntu-latest`).
- GoReleaser release workflow producing archives for linux/darwin (amd64 + arm64) and windows/amd64 plus a `checksums.txt` per tag.
- MIT License.
- README with screenshot, install instructions, keyboard cheatsheet, and a development section.

[Unreleased]: https://github.com/yachiko/fossor/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/yachiko/fossor/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/yachiko/fossor/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/yachiko/fossor/releases/tag/v0.1.0
