package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"kong/pkg/catalog"
	"kong/pkg/config"
)

// Run wires config + server and blocks until shutdown.
func main() {
	// Initialize zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Determine which config file to use based on environment
	configFile := "config/default.yaml"
	if os.Getenv("ENV") == "local" || os.Getenv("ENV") == "development" {
		configFile = "config/local.yaml"
	}

	err := config.ParseAndLoadConfig(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	ctx := context.Background()
	cfg := config.GetAppConfig()
	app, err := catalog.New(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to init app")
	}
	defer app.Close()

	srv := &http.Server{Addr: cfg.Addr, Handler: app.Router()}
	go func() {
		log.Info().Str("addr", cfg.Addr).Msg("Catalog service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info().Msg("Server stopped")
}
