# Documentation Index

Single hub for all Fossor documentation following the Diátaxis framework.

## Quadrants

| Need                 | Category    | Start Here                                       | Description                                            |
| -------------------- | ----------- | ------------------------------------------------ | ------------------------------------------------------ |
| First success        | Tutorial    | [`tutorials/first-run.md`](tutorials/first-run.md)         | Install, scan a directory, drive your first repo      |
| Perform a task       | How-To      | [`how-to/bulk-operations.md`](how-to/bulk-operations.md)   | Keep many repos in sync                                |
| Look up a key        | Reference   | [`reference/keybindings.md`](reference/keybindings.md)     | Full keybinding cheatsheet                             |
| Understand rationale | Explanation | [`explanation/architecture.md`](explanation/architecture.md) | How Bubble Tea + discovery + git wire together         |

## All Pages

### Tutorials
- [First run on a directory of repos](tutorials/first-run.md)

### How-To
- [Bulk operations across many repos](how-to/bulk-operations.md)
- [Manage stashes and branches](how-to/manage-stashes-and-branches.md)
- [Open a repo in an external editor](how-to/open-in-editor.md)

### Reference
- [Keybindings](reference/keybindings.md) — all shortcuts on every screen
- [CLI flags and environment variables](reference/cli.md)
- [Repository status states](reference/status-states.md) — meanings of `Up to date`, `Behind`, …
- [Troubleshooting](reference/troubleshooting.md) — stuck repos, debug log, common errors

### Explanation
- [Architecture overview](explanation/architecture.md) — model/update/view, discovery pipeline
- [Design choices and trade-offs](explanation/design-choices.md)

## Stability Legend

| Marker       | Meaning                                                       |
| ------------ | ------------------------------------------------------------- |
| Stable       | Backwards compatibility expected; changes rare and documented |
| Provisional  | Behavior may evolve; avoid automation hard coupling           |
| Experimental | Early preview; subject to removal or breaking changes         |

## Conventions

- Fossor is **stateless**: no config file, no on-disk state beyond the optional `~/.cache/fossor/debug.log`. Every run rediscovers.
- Fossor shells out to the system `git` binary. Your `~/.gitconfig`, credential helpers, and `$EDITOR` are all honored as-is.
- Bulk operations are concurrency-capped at 8 to keep behavior predictable on directories with hundreds of repos.

## See Also

- Root README: [`../README.md`](../README.md)
- Contribution guidelines: [`../CONTRIBUTING.md`](../CONTRIBUTING.md)
- Changelog: [`../CHANGELOG.md`](../CHANGELOG.md)
- Security policy: [`../SECURITY.md`](../SECURITY.md)
