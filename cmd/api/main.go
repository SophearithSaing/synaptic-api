// Command api runs the Synaptic HTTP API server.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SophearithSaing/synaptic-api/internal/app"
	"github.com/SophearithSaing/synaptic-api/internal/config"
)

func main() {
	slog.SetDefault(newLogger())

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	if err := application.Run(ctx); err != nil {
		slog.Error("application stopped with an error", "error", err)
		os.Exit(1)
	}
}

// newLogger returns a JSON logger in production and a text logger otherwise.
func newLogger() *slog.Logger {
	if os.Getenv("APP_ENV") == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
