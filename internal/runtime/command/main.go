package main

import (
	"context"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/app"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, ".", os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "BaseStack:", err)
		os.Exit(1)
	}
}
