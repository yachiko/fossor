//go:build !windows

package git

import (
	"os"
	"os/exec"
	"syscall"
)

// detach starts Git in its own session. Without a controlling terminal, Git
// and SSH cannot prompt on /dev/tty underneath the TUI; they fail instead,
// while agents, credential helpers and askpass programs keep working.
// Cancellation kills the whole process group so helpers such as ssh do not
// outlive Git and hold its output pipes open.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != syscall.ESRCH {
			return err
		}
		return os.ErrProcessDone
	}
}
