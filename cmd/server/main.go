// Command server runs the Synaptic HTTP API server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// main runs the API and reports fatal startup or server errors.
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run loads configuration and serves the HTTP API with a health endpoint.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		return errors.New("invalid database configuration")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return errors.New("database connection failed")
	}
	slog.Info("database connected")

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         fmt.Sprintf(":%v", cfg.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("listening on port:" + cfg.Port)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("unexpected error: %w", err)
	case <-ctx.Done():
		stop()
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		server.Close()
		return err
	}

	return nil
}
