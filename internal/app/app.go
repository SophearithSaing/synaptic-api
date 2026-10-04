// Package app is the composition root: it wires infrastructure, use cases,
// and the HTTP server together.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/config"
	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// connectTimeout bounds the initial MongoDB connection.
const connectTimeout = 15 * time.Second

// disconnectTimeout bounds the final MongoDB disconnect.
const disconnectTimeout = 10 * time.Second

// readyPingTimeout bounds each readiness probe.
const readyPingTimeout = 2 * time.Second

// throttle ambient and per-route limits match the pinned contract.
var (
	globalThrottle = web.ThrottleConfig{
		Limit: 100, TTL: time.Minute, Block: time.Minute,
	}
	registerThrottle = web.ThrottleConfig{
		Limit: 3, TTL: time.Minute, Block: 5 * time.Minute,
	}
	loginThrottle = web.ThrottleConfig{
		Limit: 5, TTL: time.Minute, Block: 5 * time.Minute,
	}
	throttleOverrides = map[string]web.ThrottleConfig{
		"POST /auth/register": registerThrottle,
		"POST /auth/login":    loginThrottle,
	}
)

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

	ready := func(ctx context.Context) error {
		pingCtx, cancel := context.WithTimeout(ctx, readyPingTimeout)
		defer cancel()

		return mongoClient.Ping(pingCtx, readpref.Primary())
	}

	resolve, err := web.ForwardedForClientIP(cfg.ThrottleTrustedProxies)
	if err != nil {
		_ = mongoClient.Disconnect(context.Background())

		return nil, fmt.Errorf("resolve trusted proxies: %w", err)
	}

	throttleStore := mongostore.NewThrottleStore(
		mongoClient.Database(cfg.MongoDatabase),
	)
	throttler := web.NewThrottler(
		globalThrottle, throttleOverrides, resolve, throttleStore,
	)

	authStore := mongostore.NewIdentityStore(
		mongoClient.Database(cfg.MongoDatabase),
	)
	if err := mongostore.EnsureIdentityIndexes(
		ctx, mongoClient.Database(cfg.MongoDatabase),
	); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, fmt.Errorf("ensure identity indexes: %w", err)
	}
	if err := mongostore.EnsureThrottleIndexes(
		ctx, mongoClient.Database(cfg.MongoDatabase),
	); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, fmt.Errorf("ensure throttle indexes: %w", err)
	}
	issuer := identity.NewTokenIssuer(
		cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTAccessTTL,
	)
	authService := identity.NewService(authStore, issuer, identity.Options{
		AccessTTL:     cfg.JWTAccessTTL,
		RefreshTTL:    cfg.JWTRefreshTTL,
		SecureCookies: cfg.SecureCookies(),
	})
	authenticator := identity.NewAuthenticator(issuer, authStore)
	authHandler := identity.NewHandler(
		authService, authenticator, identity.Options{
			AccessTTL:     cfg.JWTAccessTTL,
			RefreshTTL:    cfg.JWTRefreshTTL,
			SecureCookies: cfg.SecureCookies(),
		},
	)
	catalogStore := mongostore.NewCatalogStore(
		mongoClient.Database(cfg.MongoDatabase),
	)
	catalogHandler := catalog.NewHandler(catalogStore, authenticator)

	middleware := []web.Middleware{throttler.Middleware}
	mounters := []web.MountFunc{authHandler.Mount, catalogHandler.Mount}

	router := web.NewRouter(cfg.ClientURL, ready, middleware, mounters)

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
