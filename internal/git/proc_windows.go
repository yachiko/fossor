//go:build windows

package git

import "os/exec"

// detach is a no-op on Windows, where Git prompts through credential helper
// windows rather than the console the TUI occupies.
func detach(*exec.Cmd) {}
