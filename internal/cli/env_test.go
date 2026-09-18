package cli

import (
	"bytes"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"os"
	"strings"
	"testing"
)

func TestEnvironmentCLIAndOldProject(t *testing.T) {
	inTemp(t)
	run(t, "init", "app")
	os.Chdir("app")
	os.Remove(config.File)
	os.WriteFile(".gitignore", []byte("dist/\n"), 0644)
	var out bytes.Buffer
	if err := envCommand([]string{"set", "test", "API_SECRET"}, strings.NewReader("PRIVATE_VALUE\n"), &out); err != nil {
		t.Fatal(err)
	}
	if err := envCommand([]string{"show", "test"}, nil, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "PRIVATE_VALUE") || !strings.Contains(out.String(), "API_SECRET=[redacted]") {
		t.Fatal("unsafe CLI output")
	}
	ignore, _ := os.ReadFile(".gitignore")
	if !bytes.Contains(ignore, []byte("/.basestack/")) {
		t.Fatal("legacy project secrets not ignored")
	}
	run(t, "check")
	run(t, "env", "list")
	run(t, "env", "unset", "test", "API_SECRET")
	for _, args := range [][]string{{"env"}, {"env", "show", "../escape"}, {"env", "set", "test", "KEY", "PRIVATE_VALUE"}, {"env", "unset"}, {"functions", "run", "../escape"}, {"storage", "delete", "bucket", "../escape"}, {"roles", "assign", "bad-id", "admin"}, {"services", "list", "extra"}, {"db", "rollback", "extra"}} {
		out.Reset()
		err := Run(args, &out)
		if err == nil {
			t.Fatalf("accepted %v", args)
		}
		if strings.Contains(err.Error()+out.String(), "PRIVATE_VALUE") {
			t.Fatal("argument error leaked secret")
		}
	}
}
