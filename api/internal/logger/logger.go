// Package logger provides structured logging using zerolog.
package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const (
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
)

// Log is the global logger instance
var Log zerolog.Logger

// Init initializes the global logger with the specified level and output format
func Init(level string, pretty bool) {
	// Configure timestamp format
	zerolog.TimeFieldFormat = time.RFC3339

	// Setup output writer
	var output io.Writer = os.Stdout
	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// Parse and set global log level
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// Initialize logger with timestamp and caller information
	Log = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()
}

// parseLogLevel converts a string log level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case logLevelDebug:
		return zerolog.DebugLevel
	case logLevelInfo:
		return zerolog.InfoLevel
	case logLevelWarn:
		return zerolog.WarnLevel
	case logLevelError:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
