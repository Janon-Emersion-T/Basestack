package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/Janon-Emersion-T/Basestack/internal/registry"
)

func templatesCommand(args []string, out io.Writer) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: basestack templates [type]")
	}
	defs := registry.Builtin.Components
	if len(args) == 1 {
		c, ok := registry.Find(args[0])
		if !ok {
			return fmt.Errorf("unknown template %q; run basestack templates", args[0])
		}
		defs = []registry.Component{c}
	}
	for _, c := range defs {
		fmt.Fprintf(out, "%s: %s (default: %s)\n", c.Type, strings.Join(c.Variants, ", "), c.DefaultVariant)
	}
	return nil
}
