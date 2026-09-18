package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
)

func envCommand(args []string, in io.Reader, out io.Writer) error {
	usage := fmt.Errorf("usage: basestack env list | show <environment> | set <environment> <KEY> (value from stdin) | unset <environment> <KEY>")
	if len(args) == 0 {
		return usage
	}
	if _, err := project.Read(); err != nil {
		return err
	}
	store := config.FileSecrets{Dir: "."}
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return usage
		}
		fmt.Fprintln(out, "development\ntest\nproduction")
		return nil
	case "show":
		if len(args) != 2 {
			return usage
		}
		values, err := store.Read(args[1])
		if err != nil {
			return err
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Fprintln(out, "Stored profile keys (all values redacted; process/.env overrides are not displayed):")
		for _, key := range keys {
			fmt.Fprintf(out, "%s=[redacted]\n", key)
		}
		return nil
	case "set", "unset":
		if len(args) != 3 {
			return usage
		}
		if err := project.EnsurePrivateIgnore(); err != nil {
			return err
		}
		var err error
		if args[0] == "set" {
			// Never accept values in argv or echo them. Refuse terminal input to avoid echo.
			if f, ok := in.(*os.File); ok {
				info, e := f.Stat()
				if e != nil || info.Mode()&os.ModeCharDevice != 0 {
					return fmt.Errorf("pipe or redirect the private value into stdin")
				}
			}
			b, e := io.ReadAll(io.LimitReader(in, 65538))
			if e != nil || len(b) > 65537 {
				return fmt.Errorf("cannot read private value or value exceeds 65536 bytes")
			}
			value := strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
			err = store.Set(args[1], args[2], value)
		} else {
			err = store.Unset(args[1], args[2])
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "Environment profile updated.")
		return nil
	default:
		return usage
	}
}
