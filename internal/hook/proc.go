package hook

import (
	"os/exec"
	"syscall"
)

// isolate puts the hook in its own process group and kills the whole group on timeout,
// so a script's children cannot outlive it holding the output open.
func isolate(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
