package main

import (
	// Core dependencies for subsequent tasks
	_ "github.com/gin-gonic/gin"                              // Task 1-8: HTTP routing
	_ "github.com/golang-migrate/migrate/v4"                  // Task 1-5: Database migrations
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3" // Task 1-5: SQLite migration driver
	_ "github.com/golang-migrate/migrate/v4/source/file"      // Task 1-5: File-based migrations
	_ "github.com/google/uuid"                                // Tasks 1-5, 1-6, 1-7: UUID generation
	_ "github.com/mattn/go-sqlite3"                           // Tasks 1-5, 1-6, 1-7: SQLite driver
	_ "github.com/spf13/viper"                                // Task 1-4: Configuration management

	"github.com/stwalsh4118/hermes/internal/logger"
)

const (
	defaultLogLevel = "info"
	prettyLogging   = true
)

func main() {
	// Initialize logger early in application startup
	logger.Init(defaultLogLevel, prettyLogging)

	// Log application startup
	logger.Log.Info().Msg("Hermes Virtual TV Channel Service starting...")

	// Demonstrate different log levels with structured fields
	logger.Log.Debug().
		Str("component", "main").
		Msg("Debug logging enabled")

	logger.Log.Info().
		Str("status", "ready").
		Bool("pretty_logging", prettyLogging).
		Msg("Logger initialized successfully")

	logger.Log.Warn().
		Str("feature", "server").
		Msg("Server initialization will be added in task 1-8")

	// Example of structured logging with multiple fields
	logger.Log.Info().
		Str("log_level", defaultLogLevel).
		Bool("development_mode", prettyLogging).
		Msg("Foundation setup complete")

	// TODO: Server initialization will be implemented in task 1-8
	// This will include:
	// - Configuration loading (Task 1-4)
	// - Database connection (Task 1-7)
	// - Gin router setup (Task 1-8)
	// - Graceful shutdown handling (Task 1-8)
}
