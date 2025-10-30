// Package config provides configuration management using Viper.
// It loads configuration from environment variables, .env files, and config files.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	defaultServerPort                = 8080
	defaultServerHost                = "0.0.0.0"
	defaultReadTimeout               = 30 * time.Second
	defaultWriteTimeout              = 30 * time.Second
	defaultDatabasePath              = "./data/hermes.db"
	defaultDatabaseConnectionTimeout = 5 * time.Second
	defaultLogLevel                  = "info"
	defaultLogPretty                 = false
	defaultDatabaseEnableWAL         = true
	envPrefix                        = "HERMES"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logging  LoggingConfig
	Media    MediaConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Path              string
	ConnectionTimeout time.Duration
	EnableWAL         bool
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Pretty bool
}

// MediaConfig holds media library configuration
type MediaConfig struct {
	LibraryPath      string
	SupportedFormats []string
}

// Load reads configuration from .env file, config files, environment variables, and defaults
func Load() (*Config, error) {
	// Load .env file if present (optional, won't error if missing)
	// .env files are optional in production and CI where env vars are set directly
	_ = godotenv.Load() // nolint:errcheck // .env file is optional

	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/hermes")

	// Environment variable settings
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
		// Config file not found is OK, we'll use defaults and env vars
	}

	// Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults configures default values for all configuration options
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", defaultServerPort)
	v.SetDefault("server.host", defaultServerHost)
	v.SetDefault("server.readtimeout", defaultReadTimeout)
	v.SetDefault("server.writetimeout", defaultWriteTimeout)

	// Database defaults
	v.SetDefault("database.path", defaultDatabasePath)
	v.SetDefault("database.connectiontimeout", defaultDatabaseConnectionTimeout)
	v.SetDefault("database.enablewal", defaultDatabaseEnableWAL)

	// Logging defaults
	v.SetDefault("logging.level", defaultLogLevel)
	v.SetDefault("logging.pretty", defaultLogPretty)

	// Media defaults
	v.SetDefault("media.supportedformats", []string{"mp4", "mkv", "avi", "mov"})
}

// Validate checks that configuration values are valid
func (c *Config) Validate() error {
	// Validate server port
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d (must be between 1 and 65535)", c.Server.Port)
	}

	// Validate timeout durations
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("invalid read timeout: %v (must be > 0)", c.Server.ReadTimeout)
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("invalid write timeout: %v (must be > 0)", c.Server.WriteTimeout)
	}
	if c.Database.ConnectionTimeout <= 0 {
		return fmt.Errorf("invalid database connection timeout: %v (must be > 0)", c.Database.ConnectionTimeout)
	}

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, c.Logging.Level) {
		return fmt.Errorf("invalid log level: %s (must be one of: %s)", c.Logging.Level, strings.Join(validLevels, ", "))
	}

	// Database path validation will be done when opening DB
	// Media library path is optional at this stage (will be required when media features are implemented)

	return nil
}

// contains checks if a string slice contains a specific value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
