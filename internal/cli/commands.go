package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// Require explicit flag/value pairs and reject duplicates, typos and empty values.
func options(args []string, allowed ...string) (map[string]string, error) {
	result := map[string]string{}
	for i := 0; i < len(args); i += 2 {
		flag := args[i]
		valid := false
		for _, name := range allowed { if flag == name { valid = true; break } }
		if !valid { return nil, fmt.Errorf("unknown option %q", flag) }
		if _, exists := result[flag]; exists { return nil, fmt.Errorf("duplicate option %s", flag) }
		if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" || strings.HasPrefix(args[i+1], "--") { return nil, fmt.Errorf("%s needs a value", flag) }
		result[flag] = args[i+1]
	}
	return result, nil
}

func sectionCommand(args []string, out io.Writer) error {
	if len(args) < 2 { return fmt.Errorf("usage: basestack %s <type-or-id> [--page slug]", args[0]) }
	allowed := []string{"--page"}
	if args[0] == "add" { allowed = append(allowed, "--variant") }
	opts, err := options(args[2:], allowed...); if err != nil { return err }
	slug := "home"; if value, ok := opts["--page"]; ok { slug = value }
	c, err := readConfig(); if err != nil { return err }
	selected, err := pageSections(&c, slug); if err != nil { return err }
	sections := *selected
	if args[0] == "add" {
		kind := args[1]
		if !known(kind) { return fmt.Errorf("unknown template %q; run basestack templates", kind) }
		variant := opts["--variant"]
		if !validVariant(kind, variant) { return fmt.Errorf("unsupported %s variant %q; use default or %s", kind, variant, variants[kind]) }
		used := map[string]bool{}; for _, s := range sections { used[s.ID] = true }
		id := kind; for n := 2; used[id]; n++ { id = fmt.Sprintf("%s-%d", kind, n) }
		s := defaults(kind, id); s.Variant = variant
		pos := len(sections)
		if kind == "navbar" { pos = 0 } else if kind != "footer" { for i, existing := range sections { if existing.Type == "footer" { pos = i; break } } }
		sections = append(sections, Section{})
		copy(sections[pos+1:], sections[pos:]); sections[pos] = s
		*selected = sections
		if err := writeConfig(c); err != nil { return err }
		fmt.Fprintf(out, "Added %s to %s.\n", id, slug)
	} else {
		found := false
		for i, s := range sections { if s.ID == args[1] { sections = append(sections[:i], sections[i+1:]...); found = true; break } }
		if !found { return fmt.Errorf("section %q not found on page %q", args[1], slug) }
		*selected = sections
		if err := writeConfig(c); err != nil { return err }
		fmt.Fprintf(out, "Removed %s from %s.\n", args[1], slug)
	}
	return nil
}

func pageCommand(args []string, out io.Writer) error {
	if len(args) == 0 { return errors.New("usage: basestack page <list|add|remove>") }
	switch args[0] {
	case "list":
		if len(args) != 1 { return errors.New("usage: basestack page list") }
		c, err := readConfig(); if err != nil { return err }
		fmt.Fprintln(out, "home\tHome\t?page=home")
		for _, page := range c.Pages { fmt.Fprintf(out, "%s\t%s\t?page=%s\n", page.Slug, page.Title, page.Slug) }
		return nil
	case "add":
		if len(args) < 2 { return errors.New("usage: basestack page add <slug> [--title \"Page title\"]") }
		opts, err := options(args[2:], "--title"); if err != nil { return err }
		slug := args[1]
		if !namePattern.MatchString(slug) || slug == "home" { return errors.New("use a lowercase page slug such as about-us; home is reserved") }
		c, err := readConfig(); if err != nil { return err }
		for _, page := range c.Pages { if page.Slug == slug { return fmt.Errorf("page %q already exists", slug) } }
		title := opts["--title"]
		if title == "" { title = strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ") }
		hero := defaults("hero", "hero"); hero.Title = title
		c.Pages = append(c.Pages, Page{Slug:slug, Title:title, Sections:[]Section{defaults("navbar", "navbar"), hero, defaults("footer", "footer")}})
		if err := writeConfig(c); err != nil { return err }
		fmt.Fprintf(out, "Created page %s at ?page=%s.\n", slug, slug)
		return nil
	case "remove":
		if len(args) != 2 { return errors.New("usage: basestack page remove <slug>") }
		if args[1] == "home" { return errors.New("home cannot be removed; remove its sections instead") }
		c, err := readConfig(); if err != nil { return err }
		for i, page := range c.Pages {
			if page.Slug == args[1] {
				c.Pages = append(c.Pages[:i], c.Pages[i+1:]...)
				if err := writeConfig(c); err != nil { return err }
				fmt.Fprintln(out, "Removed page", args[1]); return nil
			}
		}
		return fmt.Errorf("page %q not found", args[1])
	default:
		return fmt.Errorf("unknown page command %q; use list, add or remove", args[0])
	}
}
