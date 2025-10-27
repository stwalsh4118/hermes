# Infrastructure API

Last Updated: 2025-10-27

## Configuration Management

### Config Package

Location: `internal/config/config.go`

**Loading Configuration:**
```go
func Load() (*Config, error)
```

Loads configuration from multiple sources in priority order:
1. Environment variables (with `HERMES_` prefix)
2. `.env` file (in `api/` directory)
3. `config.yaml` or `config.json` file
4. Default values

**Configuration Search Paths:**
- `./config.yaml` (current directory)
- `./config/config.yaml`
- `/etc/hermes/config.yaml`

**Usage Example:**
```go
import "github.com/stwalsh4118/hermes/internal/config"

cfg, err := config.Load()
if err != nil {
    log.Fatal("Failed to load configuration:", err)
}

// Access configuration values
port := cfg.Server.Port
logLevel := cfg.Logging.Level
dbPath := cfg.Database.Path
```

### Configuration Structure

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Logging  LoggingConfig
    Media    MediaConfig
}

type ServerConfig struct {
    Port         int           // Default: 8080
    Host         string        // Default: "0.0.0.0"
    ReadTimeout  time.Duration // Default: 30s
    WriteTimeout time.Duration // Default: 30s
}

type DatabaseConfig struct {
    Path              string        // Default: "./data/hermes.db"
    ConnectionTimeout time.Duration // Default: 5s
    EnableWAL         bool          // Default: true
}

type LoggingConfig struct {
    Level  string // "debug", "info", "warn", "error" - Default: "info"
    Pretty bool   // true=console, false=JSON - Default: false
}

type MediaConfig struct {
    LibraryPath      string
    SupportedFormats []string // Default: ["mp4", "mkv", "avi", "mov"]
}
```

### Environment Variables

All configuration can be set via environment variables with the `HERMES_` prefix. Nested keys use underscores.

**Examples:**
```bash
# Server configuration
HERMES_SERVER_PORT=9090
HERMES_SERVER_HOST=127.0.0.1
HERMES_SERVER_READTIMEOUT=60s

# Database configuration
HERMES_DATABASE_PATH=/var/lib/hermes/hermes.db
HERMES_DATABASE_ENABLEWAL=true

# Logging configuration
HERMES_LOGGING_LEVEL=debug
HERMES_LOGGING_PRETTY=true

# Media configuration
HERMES_MEDIA_LIBRARYPATH=/media/videos
HERMES_MEDIA_SUPPORTEDFORMATS=mp4,mkv,avi
```

### .env File Support

Place a `.env` file in the `api/` directory for local development:

```bash
# api/.env
HERMES_SERVER_PORT=8080
HERMES_LOGGING_LEVEL=debug
HERMES_LOGGING_PRETTY=true
HERMES_DATABASE_PATH=./data/hermes-dev.db
```

### YAML Configuration File

Example `config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  readtimeout: 30s
  writetimeout: 30s

database:
  path: "./data/hermes.db"
  enablewal: true

logging:
  level: "info"
  pretty: false

media:
  librarypath: "/path/to/media"
  supportedformats:
    - mp4
    - mkv
    - avi
    - mov
```

### Configuration Validation

The `Load()` function automatically validates configuration:
- Server port must be between 1-65535
- All timeout values (ReadTimeout, WriteTimeout, ConnectionTimeout) must be > 0
- Log level must be: debug, info, warn, error
- Returns error if validation fails

**Example Error Handling:**
```go
cfg, err := config.Load()
if err != nil {
    // Validation or loading error
    log.Fatal(err)
}
```

## Logging (zerolog)

### Logger Package

Location: `internal/logger/logger.go`

**Global Logger Variable:**
```go
var Log zerolog.Logger
```

**Initialization:**
```go
func Init(level string, pretty bool)
```
- `level`: "debug", "info", "warn", "error" (defaults to "info")
- `pretty`: true for console output (dev), false for JSON (production)

**Usage Example:**
```go
import "github.com/stwalsh4118/hermes/internal/logger"

// Initialize once at application startup
logger.Init("info", true)

// Use throughout application
logger.Log.Info().
    Str("component", "example").
    Msg("Application started")

logger.Log.Error().
    Err(err).
    Str("operation", "database").
    Msg("Operation failed")
```

### HTTP Request Logging Middleware

Location: `internal/middleware/logging.go`

**Gin Middleware:**
```go
func RequestLogger() gin.HandlerFunc
```

Logs HTTP requests with structured fields:
- method, path, status, duration, client_ip
- Separate error logging if request has errors

**Usage Example:**
```go
import "github.com/stwalsh4118/hermes/internal/middleware"

router := gin.New()
router.Use(middleware.RequestLogger())
```

**Output Format:**
- Production (JSON): `{"level":"info","method":"GET","path":"/api/health","status":200,"duration":0.5,"client_ip":"127.0.0.1","time":"2025-10-27T12:00:00Z","message":"HTTP request"}`
- Development (Pretty): `2025-10-27T12:00:00-05:00 INF middleware/logging.go:27 > HTTP request method=GET path=/api/health status=200 duration=0.5ms client_ip=127.0.0.1`

## Log Levels

- **debug**: Detailed diagnostic information
- **info**: General informational messages
- **warn**: Warning messages for potentially harmful situations
- **error**: Error messages for failures

## Best Practices

1. Initialize logger once at application startup
2. Use structured fields with `.Str()`, `.Int()`, `.Err()`, etc.
3. Always end with `.Msg()` to output the log
4. Use JSON format in production for machine parsing
5. Use pretty format in development for readability

