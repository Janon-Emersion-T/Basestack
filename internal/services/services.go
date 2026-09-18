// Package services operates the project's inspectable local PostgreSQL Compose service.
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Runner func(context.Context, string, []string, []string) (string, error)

func command(ctx context.Context, dir string, args, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir
	cmd.Env = env
	data, err := cmd.CombinedOutput()
	return string(data), err
}
func Execute(ctx context.Context, dir, action string, out io.Writer, runner Runner) error {
	if action != "start" && action != "stop" && action != "status" {
		return fmt.Errorf("usage: basestack services start|stop|status")
	}
	c, err := config.Read(dir)
	if err != nil {
		return err
	}
	if action == "start" && !config.Enabled(c.Database.Enabled) {
		return fmt.Errorf("local database is disabled in basestack/services.json")
	}
	info, err := os.Lstat(filepath.Join(dir, "compose.yaml"))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("compose.yaml is missing or not a regular file")
	}
	if action == "start" {
		if err := Credentials(dir); err != nil {
			return err
		}
	}
	values, err := config.Environment(dir)
	if err != nil {
		return err
	}
	resolved, err := config.Resolve(c, func(key string) (string, bool) { v, ok := values[key]; return v, ok })
	if err != nil {
		return err
	}
	values["BASESTACK_DATABASE_PORT"] = fmt.Sprint(resolved.DatabasePort)
	if values["BASESTACK_DB_PASSWORD"] == "" {
		if action == "start" {
			return fmt.Errorf("BASESTACK_DB_PASSWORD must not be empty")
		}
		values["BASESTACK_DB_PASSWORD"] = "unused-for-status-or-stop"
	}
	// Explicit env files prevent Compose from implicitly parsing .env with different expansion rules.
	args := []string{"compose", "--env-file", os.DevNull, "-f", "compose.yaml"}
	switch action {
	case "start":
		args = append(args, "up", "--quiet-pull", "--detach", "--wait", "--wait-timeout", "60", "postgres")
	case "stop":
		args = append(args, "stop", "postgres")
	case "status":
		args = append(args, "ps", "--all", "postgres")
	}
	env := []string{}
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	sort.Strings(env)
	if runner == nil {
		runner = command
	}
	deadline, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if action == "start" {
		fmt.Fprintln(out, "Starting local PostgreSQL with Docker Compose...")
	}
	output, err := runner(deadline, dir, args, env)
	for _, key := range []string{"BASESTACK_DB_PASSWORD", "BASESTACK_DATABASE_URL"} {
		if value := values[key]; value != "" {
			output = strings.ReplaceAll(output, value, "[redacted]")
		}
	}
	if err != nil {
		return fmt.Errorf("Docker Compose %s failed; ensure Docker is running and Compose v2 is installed (use services status to inspect containers)", action)
	}
	fmt.Fprint(out, output)
	switch action {
	case "start":
		fmt.Fprintln(out, "Local PostgreSQL is healthy. Next: basestack db migrate, then basestack api in another terminal.")
	case "stop":
		fmt.Fprintln(out, "Local PostgreSQL stopped. Database volumes are retained.")
	}
	return nil
}

// Credentials creates only ignored local state and never overwrites existing credentials.
func Credentials(dir string) error {
	local := filepath.Join(dir, ".basestack")
	if err := os.Mkdir(local, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("cannot create local services directory")
	}
	info, err := os.Lstat(local)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf(".basestack must be a real directory")
	}
	path := filepath.Join(dir, config.LocalEnv)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("local.env must be a regular file")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot inspect local.env")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("cannot generate local credentials")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("cannot create local credentials; retry if another start is running")
	}
	complete := false
	defer func() {
		f.Close()
		if !complete {
			os.Remove(path)
		}
	}()
	if _, err := fmt.Fprintf(f, "# Local development only. Never commit this file.\nBASESTACK_DB_PASSWORD=%s\n", hex.EncodeToString(secret)); err != nil {
		return fmt.Errorf("cannot write local credentials")
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("cannot persist local credentials")
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("cannot close local credentials")
	}
	complete = true
	return nil
}
