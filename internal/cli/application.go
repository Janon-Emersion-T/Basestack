package cli

import (
	"context"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/functions"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/rbac"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/storage"
	"io"
)

func applicationCommand(ctx context.Context, args []string, out io.Writer) error {
	valid := false
	usage := ""
	switch args[0] {
	case "functions":
		valid = functions.ValidCommand(args[1:])
		usage = functions.Usage
	case "storage":
		valid = storage.ValidCommand(args[1:])
		usage = storage.Usage
	case "roles":
		valid = rbac.ValidCommand(args[1:])
		usage = rbac.Usage
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "help") {
		fmt.Fprintln(out, usage)
		return nil
	}
	if !valid {
		return fmt.Errorf("%s", usage)
	}
	return runRuntimeArgs(ctx, args, out)
}
