// Package cli implements the BaseStack command-line interface.
package cli

import (
	"fmt"
	"io"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
)

const Version = "0.2.0"

func Run(args []string, out io.Writer) error {
	if len(args) == 0 {
		args = []string{"help"}
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
		if args[0] == "check" {
			fmt.Fprintln(out, "Configuration is valid.")
			return nil
		}
		return runFrontend(args[0], out)
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
  version                           Print CLI version

Run project commands inside a generated project.
Omitting --page selects home, or the only page; otherwise specify --page.
Install frontend dependencies with npm install before dev/build.
Schema v1 requires manual migration; see docs/CONFIGURATION.md.`
