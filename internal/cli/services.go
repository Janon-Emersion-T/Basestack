package cli

import (
	"context"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/migrations"
	"github.com/Janon-Emersion-T/Basestack/internal/services"
	"io"
	"os"
)

func servicesCommand(ctx context.Context, args []string, out io.Writer) error {
	if len(args) != 1 || (args[0] != "list" && args[0] != "start" && args[0] != "stop" && args[0] != "status") {
		return fmt.Errorf("usage: basestack services list|start|stop|status")
	}
	if _, err := project.Read(); err != nil {
		return err
	}
	if args[0] == "list" {
		c, err := config.Read(".")
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "database: %t (PostgreSQL; local Compose or external)\napi: %t (foreground: basestack api)\nauth: %t\nstorage: %t\nfunctions: %t\nauthorization: %t\n", config.Enabled(c.Database.Enabled), config.Enabled(c.API.Enabled), c.Auth != nil && config.Enabled(c.Auth.Enabled), c.Storage != nil && config.Enabled(c.Storage.Enabled), c.Functions != nil && config.Enabled(c.Functions.Enabled), c.Authorization != nil && config.Enabled(c.Authorization.Enabled))
		return nil
	}
	if args[0] == "start" {
		if err := project.EnsurePrivateIgnore(); err != nil {
			return err
		}
	}
	if args[0] == "status" {
		if err := frontendServiceStatus(out); err != nil {
			return err
		}
	}
	return services.Execute(ctx, ".", args[0], out, nil)
}
func migrationCommand(args []string, out io.Writer) error {
	if len(args) != 2 || args[0] != "new" {
		return fmt.Errorf("usage: basestack migration new <lowercase_name>")
	}
	if _, err := project.Read(); err != nil {
		return err
	}
	c, err := config.Read(".")
	if err != nil {
		return err
	}
	if !config.Enabled(c.Database.Enabled) {
		return fmt.Errorf("database is disabled")
	}
	info, err := os.Lstat("basestack")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("basestack must be a real project directory")
	}
	path, err := migrations.Create(migrations.Directory, args[1])
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Created", path)
	return nil
}
func dbCommand(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 2 && args[0] == "create-migration" {
		return migrationCommand([]string{"new", args[1]}, out)
	}
	if len(args) != 1 || (args[0] != "migrate" && args[0] != "status" && args[0] != "rollback") {
		return fmt.Errorf("usage: basestack db migrate|status|rollback or db create-migration <name>")
	}
	return runRuntime(ctx, args[0], out)
}
