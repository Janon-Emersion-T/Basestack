package cli

import (
	"fmt"
	"io"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
)

func addCommand(args []string, out io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: basestack add <type> [--page id] [--variant name]")
	}
	opts, err := options(args[1:], "page", "variant")
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
	used := map[string]bool{}
	for _, s := range p.Sections {
		used[s.ID] = true
	}
	kind := args[0]
	id := kind
	for n := 2; used[id]; n++ {
		id = fmt.Sprintf("%s-%d", kind, n)
	}
	s, err := project.DefaultSection(kind, id, opts["variant"])
	if err != nil {
		return err
	}
	pos := len(p.Sections)
	if kind != "footer" {
		for i, s := range p.Sections {
			if s.Type == "footer" {
				pos = i
				break
			}
		}
	}
	p.Sections = append(p.Sections, project.Section{})
	copy(p.Sections[pos+1:], p.Sections[pos:])
	p.Sections[pos] = s
	if err := project.Write(c); err != nil {
		return err
	}
	fmt.Fprintf(out, "Added %s to page %s (variant: %s).\n", id, p.ID, s.Variant)
	return nil
}
