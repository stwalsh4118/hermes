package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stwalsh4118/hermes/internal/config"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/server"
)

const shutdownTimeout = 10 * time.Second

func main() {
	// Load configuration from .env, config files, environment variables, and defaults
	cfg, err := config.Load()
	if err != nil {
		// Can't use logger yet, so use stderr
		os.Stderr.WriteString("Failed to load configuration: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialize logger with configuration values
	logger.Init(cfg.Logging.Level, cfg.Logging.Pretty)

	// Log application startup with configuration details
	logger.Log.Info().Msg("Hermes Virtual TV Channel Service starting...")

	logger.Log.Info().
		Str("log_level", cfg.Logging.Level).
		Bool("pretty_logging", cfg.Logging.Pretty).
		Msg("Logger initialized successfully")

	logger.Log.Info().
		Str("server_host", cfg.Server.Host).
		Int("server_port", cfg.Server.Port).
		Str("database_path", cfg.Database.Path).
		Bool("database_wal", cfg.Database.EnableWAL).
		Msg("Configuration loaded successfully")

	// Log media configuration if library path is set
	if cfg.Media.LibraryPath != "" {
		logger.Log.Info().
			Str("media_library", cfg.Media.LibraryPath).
			Strs("supported_formats", cfg.Media.SupportedFormats).
			Msg("Media library configured")
	} else {
		logger.Log.Debug().
			Msg("Media library path not configured (optional at this stage)")
	}

	// Connect to database
	database, err := db.New(cfg.Database.Path)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	logger.Log.Info().Str("path", cfg.Database.Path).Msg("Connected to database")

	// Get underlying sql.DB for migrations
	sqlDB, err := database.GetSQLDB()
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to get sql.DB for migrations")
	}

	// Run migrations
	if err := db.RunMigrations(sqlDB, "file://migrations"); err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	logger.Log.Info().Msg("Database migrations completed")

	// Create and start server
	srv := server.New(cfg, database)

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start server in goroutine
	go func() {
		serverErrors <- srv.Start()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive an error or interrupt signal
	select {
	case err := <-serverErrors:
		logger.Log.Fatal().Err(err).Msg("Server error")

	case sig := <-shutdown:
		logger.Log.Info().
			Str("signal", sig.String()).
			Msg("Shutdown signal received")

		// Give outstanding requests time to complete
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Log.Error().Err(err).Msg("Graceful shutdown failed")
			os.Exit(1)
		}
	}

	logger.Log.Info().Msg("Server stopped successfully")
}
