package cli

import (
	"fmt"
	"io"

	"github.com/Janon-Emersion-T/Basestack/internal/scaffold"
)

func initCommand(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: basestack init <name>")
	}
	if err := scaffold.Create(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(out, "Created %s.\n\nNext:\n  cd %s\n  npm install\n  basestack dev\n\nEdit basestack.json to customise pages, content, variants and theme.\n", args[0], args[0])
	return nil
}
