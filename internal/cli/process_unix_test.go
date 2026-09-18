//go:build !windows

package cli

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestCancellationStopsProcessTree(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cmd := exec.Command("sh", "-c", "sleep 60 & wait")
	start := time.Now()
	if err := runChild(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("child process tree did not stop promptly")
	}
}
