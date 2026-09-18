package cli

import (
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"io"
)

func authCommand(args []string, out io.Writer) error {
	if len(args) != 1 || args[0] != "status" {
		return fmt.Errorf("usage: basestack auth status")
	}
	if _, err := project.Read(); err != nil {
		return err
	}
	c, err := config.Load(".")
	if err != nil {
		return err
	}
	enabled := c.Services.Auth != nil && config.Enabled(c.Services.Auth.Enabled)
	fmt.Fprintf(out, "Auth enabled: %t\nEnvironment: %s\nDelivery mode: %s\n", enabled, c.Environment, c.AuthDelivery)
	if enabled {
		fmt.Fprintf(out, "Email verification required: %t\n", config.Enabled(c.Services.Auth.RequireEmailVerification))
		if c.AuthDelivery == "none" || c.AuthDelivery == "external" {
			fmt.Fprintln(out, "An external delivery provider must be wired into the generated runtime before serving Auth.")
		}
	}
	fmt.Fprintln(out, "Credential verification only; no sessions or access tokens are issued.")
	return nil
}
