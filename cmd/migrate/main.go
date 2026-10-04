// Command migrate applies required MongoDB schema migrations.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SophearithSaing/synaptic-api/internal/config"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
)

func main() {
	cfg, err := config.LoadMongo()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	client, err := mongostore.Connect(ctx, cfg.URI)
	if err != nil {
		slog.Error("failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			slog.Error("failed to disconnect MongoDB", "error", err)
		}
	}()

	if err := mongostore.RunMigrations(
		ctx,
		client.Database(cfg.Database),
	); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations complete")
}
