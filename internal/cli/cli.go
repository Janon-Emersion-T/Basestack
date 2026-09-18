// Package cli implements the BaseStack command-line interface.
package cli

import (
	"context"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"io"
	"os"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
)

const Version = "0.3.2"

func Run(args []string, out io.Writer) error { return RunContext(context.Background(), args, out) }
func RunContext(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		args = []string{"help"}
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "help") {
		usage := map[string]string{
			"env":      "basestack env list | show <environment> | set <environment> <KEY> (stdin) | unset <environment> <KEY>\nEnvironments: development, test, production. Values are always redacted.",
			"db":       "basestack db status | migrate | rollback | create-migration <name>\nRollback reverses only the latest applied migration with a recorded .down.sql file.",
			"services": "basestack services list | status | start | stop\nstart/stop manage local PostgreSQL. Run the API separately with basestack api; volumes are retained.",
		}
		if text, ok := usage[args[0]]; ok {
			fmt.Fprintln(out, text)
			return nil
		}
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Fprintln(out, help)
		return nil
	case "version", "--version":
		if len(args) != 1 {
			return fmt.Errorf("usage: basestack version")
		}
		fmt.Fprintln(out, Version)
		return nil
	case "auth":
		return authCommand(args[1:], out)
	case "env":
		return envCommand(args[1:], os.Stdin, out)
	case "functions", "storage", "roles":
		return applicationCommand(ctx, args, out)
	case "services":
		return servicesCommand(ctx, args[1:], out)
	case "migration":
		return migrationCommand(args[1:], out)
	case "db":
		return dbCommand(ctx, args[1:], out)
	case "api":
		if len(args) != 1 {
			return fmt.Errorf("usage: basestack api")
		}
		return runRuntime(ctx, "serve", out)
	case "init":
		return initCommand(args[1:], out)
	case "templates":
		return templatesCommand(args[1:], out)
	case "add":
		return addCommand(args[1:], out)
	case "remove":
		return removeCommand(args[1:], out)
	case "page":
		return pageCommand(args[1:], out)
	case "theme":
		return themeCommand(args[1:], out)
	case "check", "dev", "build":
		if len(args) != 1 {
			return fmt.Errorf("usage: basestack %s", args[0])
		}
		if _, err := project.Read(); err != nil {
			return err
		}
		if _, err := os.Stat(config.File); err == nil {
			if _, err := config.Read("."); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if args[0] == "check" {
			fmt.Fprintln(out, "Configuration is valid.")
			return nil
		}
		if args[0] == "dev" {
			if err := frontendServiceStatus(out); err != nil {
				return err
			}
		}
		return runFrontend(ctx, args[0], out)
	default:
		return fmt.Errorf("unknown command %q; run basestack help", args[0])
	}
}

const help = `BaseStack — build from reusable components, without AI.

Commands:
  init <name>                       Create a React + TypeScript project
  templates [type]                  List components and their variants
  add <type> [--page id] [--variant name]
                                    Insert a section before the first footer
  remove <id> [--page id]            Remove a section from a page
  page list                         List pages, paths and titles
  page add <id> [--path /path] [--title "Page title"]
  page remove <id>                  Remove a page (at least one must remain)
  theme list                        List themes
  theme set <name>                   Set the application theme
  check                             Validate basestack.json
  dev                               Start the local Vite server
  build                             Validate and build for production
  services list|start|stop|status        Manage local PostgreSQL (Docker Compose)
  migration new <name>              Create an ordered SQL migration
  db migrate|status|rollback                 Apply migrations or inspect database history
  db create-migration <name>        Create a forward SQL migration
  functions list|inspect|run         Discover or execute compiled server functions
  storage buckets|create-bucket|list|put|get|delete
                                    Manage private local objects (put reads stdin)
  roles create|grant|revoke|assign|unassign|check
                                    Manage persistent role permissions
  api                               Run the generated Go API (separate terminal)
  auth status                       Show Auth configuration without secrets
  env list                          List supported environments
  env show <environment>            Show stored profile keys, always redacted
  env set <environment> <KEY>        Store a private value read from stdin
  env unset <environment> <KEY>      Remove a stored profile override
  version                           Print CLI version

Run project commands inside a generated project.
Omitting --page selects home, or the only page; otherwise specify --page.
Install frontend dependencies with npm install before dev/build.
Schema v1 requires manual migration; see docs/CONFIGURATION.md.`
