package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/app"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return app.Run(ctx, app.DefaultOptions())
}
