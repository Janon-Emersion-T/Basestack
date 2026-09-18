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
	if len(args) != 1 || (args[0] != "start" && args[0] != "stop" && args[0] != "status") {
		return fmt.Errorf("usage: basestack services start|stop|status")
	}
	if _, err := project.Read(); err != nil {
		return err
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
	if len(args) != 1 || (args[0] != "migrate" && args[0] != "status") {
		return fmt.Errorf("usage: basestack db migrate|status")
	}
	return runRuntime(ctx, args[0], out)
}
