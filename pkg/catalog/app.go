package catalog

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"kong/pkg/catalog/middleware"
	"kong/pkg/catalog/routes"
	"kong/pkg/config"
	"kong/pkg/models"
)

// App is the main application struct
type App struct {
	cfg   *config.AppConfig
	pool  *pgxpool.Pool
	store *models.Store
	r     *chi.Mux
}

// New creates a new App instance
func New(ctx context.Context, cfg *config.AppConfig) (*App, error) {
	// Configure database connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Apply custom connection pool settings
	poolConfig.MaxConns = int32(cfg.DBMaxConnections)
	poolConfig.MinConns = int32(cfg.DBMinConnections)
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	poolConfig.ConnConfig.ConnectTimeout = cfg.DBConnectTimeout
	poolConfig.HealthCheckPeriod = cfg.DBHealthCheckPeriod

	// Log database configuration
	log.Info().
		Int("max_connections", cfg.DBMaxConnections).
		Int("min_connections", cfg.DBMinConnections).
		Dur("max_conn_lifetime", cfg.DBMaxConnLifetime).
		Dur("max_conn_idle_time", cfg.DBMaxConnIdleTime).
		Dur("connect_timeout", cfg.DBConnectTimeout).
		Dur("health_check_period", cfg.DBHealthCheckPeriod).
		Msg("Database pool configuration")

	// Create connection pool with custom configuration
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	store := models.NewStore(pool, cfg.MaxPageSize)

	// Create a new router
	r := chi.NewRouter()

	// Setup global middleware in the correct order
	middleware.SetupGlobalMiddleware(r, cfg.ValidAPIKeys)

	// Use the new routes system with middleware
	routes.SetupRoutes(store, r)

	app := &App{cfg: cfg, pool: pool, store: store, r: r}
	return app, nil
}

// Router returns the router for the app
func (a *App) Router() http.Handler { return a.r }

// Pool returns the database pool
func (a *App) Pool() *pgxpool.Pool { return a.pool }

// Close closes the app
func (a *App) Close() { a.pool.Close() }
