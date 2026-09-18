//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// Give owned commands a process group so npm's shell/Node descendants receive shutdown too.
func prepareChild(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func stopChild(cmd *exec.Cmd, force bool) error {
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-cmd.Process.Pid, signal)
}
