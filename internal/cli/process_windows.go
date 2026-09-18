//go:build windows

package cli

import (
	"os"
	"os/exec"
	"strconv"
)

func prepareChild(cmd *exec.Cmd) {}
func stopChild(cmd *exec.Cmd, force bool) error {
	if !force {
		if err := cmd.Process.Signal(os.Interrupt); err == nil {
			return nil
		}
	}
	// Native Windows does not support os.Interrupt delivery. Terminate the owned process tree.
	return exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
}
