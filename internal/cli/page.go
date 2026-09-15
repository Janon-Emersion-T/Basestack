package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
)

func pageCommand(args []string, out io.Writer) error {
	usage := fmt.Errorf("usage: basestack page list | add <id> [--path /path] [--title title] | remove <id>")
	if len(args) == 0 {
		return usage
	}
	var opts map[string]string
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return usage
		}
	case "remove":
		if len(args) != 2 {
			return usage
		}
	case "add":
		if len(args) < 2 {
			return usage
		}
		var err error
		opts, err = options(args[2:], "path", "title")
		if err != nil {
			return err
		}
	default:
		return usage
	}
	c, err := project.Read()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		for _, p := range c.Pages {
			fmt.Fprintf(out, "%s\t%s\t%s (%d sections)\n", p.ID, p.Path, p.Title, len(p.Sections))
		}
		return nil
	case "add":
		id := args[1]
		if !project.NamePattern.MatchString(id) {
			return fmt.Errorf("invalid page id %q", id)
		}
		path := opts["path"]
		if path == "" {
			path = "/" + id
		}
		title := opts["title"]
		if title == "" {
			words := strings.Split(id, "-")
			for i, w := range words {
				if w != "" {
					words[i] = strings.ToUpper(w[:1]) + w[1:]
				}
			}
			title = strings.Join(words, " ")
		}
		c.Pages = append(c.Pages, project.Page{ID: id, Path: path, Title: title, Sections: []project.Section{}})
		if err := project.Write(c); err != nil {
			return err
		}
		fmt.Fprintf(out, "Added page %s at %s.\n", id, path)
		return nil
	case "remove":
		for i, p := range c.Pages {
			if p.ID == args[1] {
				if len(c.Pages) == 1 {
					return fmt.Errorf("cannot remove the final remaining page")
				}
				c.Pages = append(c.Pages[:i], c.Pages[i+1:]...)
				if err := project.Write(c); err != nil {
					return err
				}
				fmt.Fprintf(out, "Removed page %s.\n", p.ID)
				return nil
			}
		}
		return fmt.Errorf("page %q not found", args[1])
	}
	return usage
}
