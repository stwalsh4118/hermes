# Infrastructure API

Last Updated: 2025-10-27

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

