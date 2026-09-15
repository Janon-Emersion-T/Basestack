package cli

import (
	"fmt"
	"io"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/registry"
)

func themeCommand(args []string, out io.Writer) error {
	if len(args) == 1 && args[0] == "list" {
		for _, t := range registry.Builtin.Themes {
			fmt.Fprintln(out, t.Name)
		}
		return nil
	}
	if len(args) != 2 || args[0] != "set" {
		return fmt.Errorf("usage: basestack theme list | set <name>")
	}
	c, err := project.Read()
	if err != nil {
		return err
	}
	c.Theme = args[1]
	if err := project.Write(c); err != nil {
		return err
	}
	fmt.Fprintf(out, "Theme set to %s.\n", c.Theme)
	return nil
}
