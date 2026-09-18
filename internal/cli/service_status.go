package cli

import (
	"encoding/json"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func frontendServiceStatus(out io.Writer) error {
	if _, err := os.Stat(config.File); os.IsNotExist(err) {
		return nil
	}
	c, err := config.Load(".")
	if err != nil {
		return err
	}
	if !config.Enabled(c.Services.API.Enabled) {
		fmt.Fprintln(out, "API: disabled in basestack/services.json.")
		return nil
	}
	host := c.Host
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	if host == "::" {
		host = "::1"
	}
	client := http.Client{Timeout: 800 * time.Millisecond, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	resp, err := client.Get("http://" + net.JoinHostPort(host, strconv.Itoa(c.Port)) + "/api/health")
	if err == nil {
		defer resp.Body.Close()
		var health struct {
			Status   string `json:"status"`
			Service  string `json:"service"`
			Database string `json:"database"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&health) == nil && resp.StatusCode == 200 && health.Status == "ok" && health.Service == "basestack" && (health.Database == "connected" || health.Database == "disabled") {
			fmt.Fprintf(out, "API: healthy; database: %s.\n", health.Database)
			return nil
		}
	}
	fmt.Fprintln(out, "API: unavailable or unhealthy. Run basestack services start, basestack db migrate, then basestack api in another terminal. Frontend will continue.")
	return nil
}
