package cli

import (
	"fmt"
	"io"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
)

func removeCommand(args []string, out io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: basestack remove <id> [--page id]")
	}
	opts, err := options(args[1:], "page")
	if err != nil {
		return err
	}
	c, err := project.Read()
	if err != nil {
		return err
	}
	p, err := c.SelectPage(opts["page"])
	if err != nil {
		return err
	}
	for i, s := range p.Sections {
		if s.ID == args[0] {
			p.Sections = append(p.Sections[:i], p.Sections[i+1:]...)
			if err := project.Write(c); err != nil {
				return err
			}
			fmt.Fprintf(out, "Removed %s from page %s.\n", s.ID, p.ID)
			return nil
		}
	}
	return fmt.Errorf("section %q not found on page %q", args[0], p.ID)
}
