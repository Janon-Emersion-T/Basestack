package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// dev and build share the same checked npm invocation.
func runFrontend(ctx context.Context, command string, out io.Writer) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm is required; install Node.js 22 or newer")
	}
	cmd := exec.Command("npm", "run", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	return runChild(ctx, cmd)
}
