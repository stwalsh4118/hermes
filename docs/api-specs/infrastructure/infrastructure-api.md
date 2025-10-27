# Infrastructure API

Last Updated: 2025-10-27 (Models added with GORM)

## Database Migrations

### Migration Package

Location: `internal/db/migrations.go`

**Running Migrations:**
```go
func RunMigrations(db *sql.DB, migrationsPath string) error
```

Executes database migrations using golang-migrate. Applies all pending migrations to the provided database connection.

**Parameters:**
- `db`: An open `*sql.DB` connection
- `migrationsPath`: Path to migrations directory with `file://` prefix (e.g., `"file://./migrations"`)

**Returns:**
- `error`: nil if migrations succeed or if there are no changes to apply (ErrNoChange is treated as success)

**Usage Example:**
```go
import (
    "database/sql"
    "github.com/stwalsh4118/hermes/internal/db"
    _ "github.com/mattn/go-sqlite3"
)

db, err := sql.Open("sqlite3", "./data/hermes.db")
if err != nil {
    return fmt.Errorf("failed to open database: %w", err)
}
defer db.Close()

// Run migrations
if err := db.RunMigrations(db, "file://./migrations"); err != nil {
    return fmt.Errorf("migration failed: %w", err)
}
```

**Migration Files:**
- Located in `migrations/` directory
- Numbered sequentially: `000001_name.up.sql`, `000001_name.down.sql`
- Up migrations create/modify schema
- Down migrations rollback changes

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

## Data Models

### Models Package

Location: `internal/models/`

The models package defines all domain entities with proper JSON and GORM tags for serialization and ORM mapping. GORM is used for database operations while schema management is handled by golang-migrate migrations.

### Channel Model

Location: `internal/models/channel.go`

```go
type Channel struct {
    ID        uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
    Name      string    `json:"name" gorm:"type:text;not null;column:name" validate:"required,min=1,max=255"`
    Icon      *string   `json:"icon,omitempty" gorm:"type:text;column:icon"`
    StartTime time.Time `json:"start_time" gorm:"type:datetime;not null;column:start_time" validate:"required"`
    Loop      bool      `json:"loop" gorm:"type:integer;not null;default:0;column:loop"`
    CreatedAt time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
    UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:updated_at"`
}

func NewChannel(name string, startTime time.Time, loop bool) *Channel
```

**Usage Example:**
```go
import (
    "time"
    "github.com/stwalsh4118/hermes/internal/models"
)

// Create a new channel
channel := models.NewChannel("Comedy Central", time.Now(), true)

// Manually create with specific values
channel := &models.Channel{
    ID:        uuid.New(),
    Name:      "Drama Channel",
    StartTime: time.Now().Add(2 * time.Hour),
    Loop:      false,
    CreatedAt: time.Now().UTC(),
    UpdatedAt: time.Now().UTC(),
}
```

### Media Model

Location: `internal/models/media.go`

```go
type Media struct {
    ID         uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
    FilePath   string    `json:"file_path" gorm:"type:text;not null;uniqueIndex;column:file_path" validate:"required"`
    Title      string    `json:"title" gorm:"type:text;not null;column:title" validate:"required"`
    ShowName   *string   `json:"show_name,omitempty" gorm:"type:text;column:show_name"`
    Season     *int      `json:"season,omitempty" gorm:"type:integer;column:season"`
    Episode    *int      `json:"episode,omitempty" gorm:"type:integer;column:episode"`
    Duration   int64     `json:"duration" gorm:"type:integer;not null;column:duration" validate:"required,gt=0"`
    VideoCodec *string   `json:"video_codec,omitempty" gorm:"type:text;column:video_codec"`
    AudioCodec *string   `json:"audio_codec,omitempty" gorm:"type:text;column:audio_codec"`
    Resolution *string   `json:"resolution,omitempty" gorm:"type:text;column:resolution"`
    FileSize   *int64    `json:"file_size,omitempty" gorm:"type:integer;column:file_size"`
    CreatedAt  time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
}

func NewMedia(filePath, title string, duration int64) *Media
func (m *Media) DurationString() string  // Returns HH:MM:SS format
```

**Usage Example:**
```go
import "github.com/stwalsh4118/hermes/internal/models"

// Create a new media item
media := models.NewMedia("/media/videos/movie.mp4", "Action Movie", 7200) // 2 hours

// Set optional fields
showName := "Friends"
season := 1
episode := 5
media.ShowName = &showName
media.Season = &season
media.Episode = &episode

// Get formatted duration
fmt.Println(media.DurationString()) // Output: "02:00:00"
```

### PlaylistItem Model

Location: `internal/models/playlist_item.go`

```go
type PlaylistItem struct {
    ID        uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
    ChannelID uuid.UUID `json:"channel_id" gorm:"type:text;not null;column:channel_id" validate:"required"`
    MediaID   uuid.UUID `json:"media_id" gorm:"type:text;not null;column:media_id" validate:"required"`
    Position  int       `json:"position" gorm:"type:integer;not null;column:position" validate:"gte=0"`
    CreatedAt time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
    Media     *Media    `json:"media,omitempty" gorm:"-"`  // Populated by joins
}

func NewPlaylistItem(channelID, mediaID uuid.UUID, position int) *PlaylistItem
```

**Usage Example:**
```go
import "github.com/stwalsh4118/hermes/internal/models"

// Create a playlist item
item := models.NewPlaylistItem(channelID, mediaID, 0)

// The Media field is populated when fetching with JOIN queries
// It has gorm:"-" tag so it's not stored in the database
```

### Settings Model

Location: `internal/models/settings.go`

```go
type Settings struct {
    ID               int       `json:"id" gorm:"type:integer;primaryKey;default:1;column:id"`
    MediaLibraryPath string    `json:"media_library_path" gorm:"type:text;not null;column:media_library_path" validate:"required"`
    TranscodeQuality string    `json:"transcode_quality" gorm:"type:text;default:medium;column:transcode_quality" validate:"oneof=high medium low"`
    HardwareAccel    string    `json:"hardware_accel" gorm:"type:text;default:none;column:hardware_accel" validate:"oneof=none nvenc qsv vaapi videotoolbox"`
    ServerPort       int       `json:"server_port" gorm:"type:integer;default:8080;column:server_port" validate:"gte=1,lte=65535"`
    UpdatedAt        time.Time `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:updated_at"`
}

func DefaultSettings() *Settings
```

**Constants:**
```go
// Quality constants
const (
    QualityHigh   = "high"
    QualityMedium = "medium"
    QualityLow    = "low"
)

// Hardware acceleration constants
const (
    HardwareAccelNone         = "none"
    HardwareAccelNVENC        = "nvenc"
    HardwareAccelQSV          = "qsv"
    HardwareAccelVAAPI        = "vaapi"
    HardwareAccelVideoToolbox = "videotoolbox"
)
```

**Usage Example:**
```go
import "github.com/stwalsh4118/hermes/internal/models"

// Create settings with defaults
settings := models.DefaultSettings()

// Modify as needed
settings.MediaLibraryPath = "/path/to/media"
settings.TranscodeQuality = models.QualityHigh
settings.HardwareAccel = models.HardwareAccelNVENC
settings.ServerPort = 9090
```

### StreamSession Model

Location: `internal/models/stream_session.go`

**Note:** This model is NOT persisted to the database. It's used for runtime streaming state management only.

```go
type StreamSession struct {
    ID          uuid.UUID `json:"id"`
    ChannelID   uuid.UUID `json:"channel_id"`
    StartedAt   time.Time `json:"started_at"`
    ClientCount int       `json:"client_count"`
    FFmpegPID   int       `json:"ffmpeg_pid"`
    mu          sync.RWMutex  // Internal mutex for thread-safety
}

func NewStreamSession(channelID uuid.UUID) *StreamSession
func (s *StreamSession) IncrementClients()
func (s *StreamSession) DecrementClients()
func (s *StreamSession) GetClientCount() int
func (s *StreamSession) SetFFmpegPID(pid int)
func (s *StreamSession) GetFFmpegPID() int
func (s *StreamSession) IsActive() bool
```

**Usage Example:**
```go
import "github.com/stwalsh4118/hermes/internal/models"

// Create a new stream session
session := models.NewStreamSession(channelID)

// Track client connections (thread-safe)
session.IncrementClients()
fmt.Println(session.GetClientCount())  // Output: 1
fmt.Println(session.IsActive())        // Output: true

session.DecrementClients()
fmt.Println(session.GetClientCount())  // Output: 0
fmt.Println(session.IsActive())        // Output: false

// Manage FFmpeg process (thread-safe)
session.SetFFmpegPID(12345)
pid := session.GetFFmpegPID()
fmt.Println(pid)  // Output: 12345
```

### Model Usage Notes

**UUIDs:**
- All entity IDs use `uuid.UUID` from `github.com/google/uuid`
- Constructor functions automatically generate new UUIDs

**Timestamps:**
- All timestamps stored as `time.Time` in UTC
- Constructor functions automatically set current UTC time

**Nullable Fields:**
- Use pointer types (*string, *int, *int64) for nullable database columns
- JSON tags include `omitempty` to exclude null fields from JSON output
- Distinguishes between null and zero values

**Tags:**
- `json`: Field name in JSON serialization
- `gorm`: GORM field configuration (type, constraints, column name)
- `validate`: Validation rules (prepared for future validator integration)
- `gorm:"-"`: Excludes field from database operations (e.g., joined data)

**GORM Tag Options:**
- `type:text` or `type:integer`: SQLite column type
- `primaryKey`: Marks field as primary key
- `not null`: Field cannot be NULL
- `uniqueIndex`: Creates unique index on the field
- `default:value`: Sets default value
- `column:name`: Explicitly sets column name
- `-`: Excludes field from database mapping

**Database Operations:**
GORM is used for all database operations (queries, inserts, updates, deletes), while schema management is handled by golang-migrate to maintain explicit control over migrations.

**Thread Safety (StreamSession):**
StreamSession uses `sync.RWMutex` for concurrent access protection. Always use provided accessor methods (`GetClientCount()`, `GetFFmpegPID()`) instead of direct field access.

## Best Practices

1. Initialize logger once at application startup
2. Use structured fields with `.Str()`, `.Int()`, `.Err()`, etc.
3. Always end with `.Msg()` to output the log
4. Use JSON format in production for machine parsing
5. Use pretty format in development for readability

