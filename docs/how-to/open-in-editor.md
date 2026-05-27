# How-To: Open a Repo in an External Editor

Stability: Stable

Goal: Jump from the Fossor main screen straight into your editor with the selected repo as its workspace.

## Wire It Up

Pick **one** of the following:

### Option A — Pass it on the command line

```bash
fossor ~/code --open-cmd code      # VS Code
fossor ~/code --open-cmd cursor    # Cursor
fossor ~/code --open-cmd zed       # Zed
fossor ~/code --open-cmd "idea --no-wait"  # JetBrains via CLI
```

Whatever you pass is invoked as `<cmd> <repo-path>` when you press `o` on the main screen.

### Option B — Set an environment variable

```bash
export FOSSOR_OPEN_CMD=code     # in your shell rc
fossor ~/code
```

The `--open-cmd` flag wins if both are present. When neither is set, the `o` key does nothing and is hidden from the on-screen help bar.

## Use It

From the **main screen**, with a repo highlighted: press `o`. Fossor invokes `$FOSSOR_OPEN_CMD <selected-repo-path>` and returns control to the TUI immediately (it does not wait for the editor to exit).

## Notes

- The command runs with your shell's `PATH`. Quote arguments inside the value if you need flags: `--open-cmd "code --reuse-window"`.
- Tip: `--open-cmd "open -a Terminal"` on macOS will spawn a terminal in the repo directory if you prefer drilling in with a shell.
- The `o` key is **main screen only**. It is not bound inside the manage view.

## See Also

- [CLI reference](../reference/cli.md)
- [Keybindings reference](../reference/keybindings.md)
