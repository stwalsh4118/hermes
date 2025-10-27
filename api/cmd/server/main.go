package main

import (
	"os"

	// Core dependencies for subsequent tasks
	_ "github.com/gin-gonic/gin"                              // Task 1-8: HTTP routing
	_ "github.com/golang-migrate/migrate/v4"                  // Task 1-5: Database migrations
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3" // Task 1-5: SQLite migration driver
	_ "github.com/golang-migrate/migrate/v4/source/file"      // Task 1-5: File-based migrations
	_ "github.com/google/uuid"                                // Tasks 1-5, 1-6, 1-7: UUID generation
	_ "github.com/mattn/go-sqlite3"                           // Tasks 1-5, 1-6, 1-7: SQLite driver

	"github.com/stwalsh4118/hermes/internal/config"
	"github.com/stwalsh4118/hermes/internal/logger"
)

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

	logger.Log.Info().Msg("Foundation setup complete")

	// TODO: Server initialization will be implemented in task 1-8
	// This will include:
	// - Database connection (Task 1-7)
	// - Gin router setup (Task 1-8)
	// - Graceful shutdown handling (Task 1-8)
}
