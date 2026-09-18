package cli

import (
	"context"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Compile the owner's generated source, so local API edits are honored by the CLI.
func runRuntime(ctx context.Context, action string, out io.Writer) error {
	if _, err := project.Read(); err != nil {
		return err
	}
	if _, err := config.Read("."); err != nil {
		return err
	}
	if info, err := os.Stat("cmd/server/main.go"); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("generated runtime source is missing; see docs/SERVICES.md")
	}
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go 1.23 or newer is required to run the application runtime")
	}
	if err := os.Mkdir(".basestack", 0700); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := os.Lstat(".basestack")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf(".basestack must be a real directory")
	}
	// Unique build directories allow concurrent API/status commands without replacing a running executable.
	dir, err := os.MkdirTemp(".basestack", "runtime-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	name := "server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary, err := filepath.Abs(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/server")
	build.Stdout = out
	build.Stderr = out
	if err := build.Run(); err != nil {
		return fmt.Errorf("build application runtime failed; check Go version and generated source")
	}
	cmd := exec.Command(binary, action)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	return runChild(ctx, cmd)
}

func runChild(ctx context.Context, cmd *exec.Cmd) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	prepareChild(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start application runtime failed")
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if err := stopChild(cmd, false); err != nil {
			_ = stopChild(cmd, true)
		}
		select {
		case <-done:
			return nil
		case <-time.After(15 * time.Second):
			_ = stopChild(cmd, true)
			<-done
			return fmt.Errorf("child process did not shut down within 15 seconds")
		}
	}
}
