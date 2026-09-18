package functions

import (
	"encoding/json"
	"errors"
	"io"
)

const Usage = "usage: basestack functions list | inspect <name> | run <name> (JSON from stdin; BASESTACK_FUNCTION_TOKEN for private functions)"

func ValidCommand(args []string) bool {
	return (len(args) == 1 && args[0] == "list") || (len(args) == 2 && (args[0] == "inspect" || args[0] == "run") && namePattern.MatchString(args[1]))
}
func (r *Registry) Describe(args []string, out io.Writer) error {
	if !ValidCommand(args) {
		return errors.New(Usage)
	}
	if args[0] == "list" {
		return json.NewEncoder(out).Encode(r.List())
	}
	m, ok := r.Inspect(args[1])
	if !ok {
		return errors.New("function not found")
	}
	return json.NewEncoder(out).Encode(m)
}
