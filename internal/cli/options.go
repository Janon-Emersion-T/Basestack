package cli

import (
	"fmt"
	"strings"
)

// options deliberately accepts options only after the command's positional argument.
func options(args []string, allowed ...string) (map[string]string, error) {
	result := map[string]string{}
	for i := 0; i < len(args); i += 2 {
		key := strings.TrimPrefix(args[i], "--")
		ok := false
		for _, a := range allowed {
			if args[i] == "--"+a {
				ok = true
			}
		}
		if !ok {
			return nil, fmt.Errorf("unknown option %q", args[i])
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("duplicate option --%s", key)
		}
		if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
			return nil, fmt.Errorf("--%s requires a value", key)
		}
		result[key] = args[i+1]
	}
	return result, nil
}
