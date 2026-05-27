# CLI Reference

Stability: Stable

Authoritative reference for the `fossor` command, its arguments, flags, and environment variables.

## Synopsis

```
fossor [path] [flags]
```

## Argument

| Name   | Default                | Description                                              |
| ------ | ---------------------- | -------------------------------------------------------- |
| `path` | current directory      | Directory containing git repositories to scan.           |

If `path` doesn't exist or isn't a directory, Fossor exits with a non-zero status and a message.

## Flags

| Flag                       | Default | Description                                                                                                                                             |
| -------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `-r`, `--recursive`        | `false` | Recursively scan for git repositories under `path`. Without this, only direct subdirectories are checked.                                                |
| `--no-fetch`               | `false` | Skip `git fetch` during discovery. Ahead / behind counts will reflect cached refs.                                                                       |
| `--no-auto-refresh`        | `false` | Disable the 30-second background refresh of the highlighted repo on the main screen.                                                                     |
| `--open-cmd <cmd>`         | unset   | Command used to open the selected repo (`o` key on main screen). Falls back to `$FOSSOR_OPEN_CMD` when empty. Hidden from `--help` when neither is set.  |
| `--version`                | —       | Print the version and exit. Source: `-ldflags` injection at build time, then `debug.ReadBuildInfo()` fallback for `go install`-built binaries.           |
| `-h`, `--help`             | —       | Print help and exit.                                                                                                                                     |

## Environment Variables

| Variable             | Effect                                                                                                              |
| -------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `FOSSOR_OPEN_CMD`    | Fallback for `--open-cmd` when the flag is unset. See [open-in-editor how-to](../how-to/open-in-editor.md).         |
| `FOSSOR_DEBUG`       | When set to any non-empty value, enables the stale-lock debug log at `~/.cache/fossor/debug.log`.                   |
| `EDITOR`             | Used by `C` (commit via editor) and `B` (interactive rebase) when Fossor suspends the TUI and hands off to `git`.   |
| `PATH`               | Must contain the `git` binary.                                                                                       |

## Exit Codes

| Code | Meaning                                            |
| ---- | -------------------------------------------------- |
| 0    | Normal termination (`q` or `Ctrl+C`).              |
| 1    | Startup error (bad path, missing `git`, etc.).     |

## Examples

```bash
# Scan the current directory.
fossor

# Scan a specific directory, skip the initial fetch.
fossor ~/code --no-fetch

# Recursive scan of an org tree, no background refresh.
fossor ~/work --recursive --no-auto-refresh

# Open repos in VS Code with `o`.
fossor ~/code --open-cmd code

# Same, via env var.
FOSSOR_OPEN_CMD=cursor fossor ~/code

# Print the version.
fossor --version
```

## See Also

- [Keybindings reference](keybindings.md)
- [Status states reference](status-states.md)
- [Troubleshooting reference](troubleshooting.md)
- [Open-in-editor how-to](../how-to/open-in-editor.md)
