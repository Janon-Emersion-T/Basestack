package main

import (
	"fmt"
	"os"

	"github.com/Janon-Emersion-T/Basestack/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "BaseStack:", err)
		os.Exit(1)
	}
}
