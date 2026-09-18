package app

import (
	"context"
	"encoding/json"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestRunWithoutDatabaseAndShutdown(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "basestack"), 0755)
	c := config.Default()
	*c.Database.Enabled = false
	*c.Auth.Enabled = false
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	c.API.Port = listener.Addr().(*net.TCPAddr).Port
	addr := listener.Addr().String()
	listener.Close()
	// Explicit test overrides isolate this test from developer API settings.
	t.Setenv("BASESTACK_API_HOST", "127.0.0.1")
	t.Setenv("BASESTACK_API_PORT", fmtPort(c.API.Port))
	t.Setenv("BASESTACK_DATABASE_PORT", "54322")
	t.Setenv("BASESTACK_CORS_ORIGINS", "")
	b, _ := json.Marshal(c)
	os.WriteFile(filepath.Join(dir, config.File), b, 0644)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, dir, []string{"serve"}, io.Discard) }()
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(2 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://" + addr + "/api/health")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			ready = resp.StatusCode == 200
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	client.CloseIdleConnections()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("app did not shut down")
	}
	if !ready {
		t.Fatal("app never became ready")
	}
	if Run(context.Background(), dir, []string{"migrate"}, io.Discard) == nil {
		t.Fatal("disabled database accepted")
	}
	*c.API.Enabled = false
	b, _ = json.Marshal(c)
	os.WriteFile(filepath.Join(dir, config.File), b, 0644)
	if Run(context.Background(), dir, nil, io.Discard) == nil {
		t.Fatal("disabled API started")
	}
	if Run(context.Background(), dir, []string{"unknown"}, io.Discard) == nil {
		t.Fatal("unknown command accepted")
	}
}
func fmtPort(port int) string { return strconv.Itoa(port) }
