// Package app is the composition root: it wires infrastructure, use cases,
// and the HTTP server together.
package app

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/SophearithSaing/synaptic-api/internal/config"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// connectTimeout bounds the initial MongoDB connection.
const connectTimeout = 15 * time.Second

// disconnectTimeout bounds the final MongoDB disconnect.
const disconnectTimeout = 10 * time.Second

// readyPingTimeout bounds each readiness probe.
const readyPingTimeout = 2 * time.Second

// App is the wired application.
type App struct {
	server *web.Server
	mongo  *mongo.Client
}

// New builds the application from the validated configuration.
func New(cfg config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	mongoClient, err := mongostore.Connect(ctx, cfg.MongoURI)
	if err != nil {
		return nil, err
	}

	router := web.NewRouter(cfg.ClientURL, func(ctx context.Context) error {
		pingCtx, cancel := context.WithTimeout(ctx, readyPingTimeout)
		defer cancel()

		return mongoClient.Ping(pingCtx, readpref.Primary())
	})

	return &App{
		server: web.NewServer(cfg.Port, router),
		mongo:  mongoClient,
	}, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	defer a.disconnect()

	return a.server.Start(ctx)
}

// disconnect closes the MongoDB client during shutdown.
func (a *App) disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
	defer cancel()

	if err := a.mongo.Disconnect(ctx); err != nil {
		slog.Error("failed to disconnect mongo", "error", err)
	}
}
