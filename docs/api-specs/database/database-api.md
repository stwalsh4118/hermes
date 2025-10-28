# Database API

Last Updated: 2025-10-27

## Overview

The database layer uses GORM ORM with SQLite for all database operations. Schema management is handled by golang-migrate, while GORM provides the query interface and repository pattern implementation.

## Database Schema

### channels table
- id (TEXT, PRIMARY KEY) - UUID
- name (TEXT, NOT NULL)
- icon (TEXT) - Icon URL or path
- start_time (DATETIME, NOT NULL) - Channel start time
- loop (BOOLEAN, NOT NULL, DEFAULT 0) - Whether to loop playlist
- created_at (DATETIME, DEFAULT CURRENT_TIMESTAMP)
- updated_at (DATETIME, DEFAULT CURRENT_TIMESTAMP)

### media table
- id (TEXT, PRIMARY KEY) - UUID
- file_path (TEXT, NOT NULL, UNIQUE) - Absolute path to media file
- title (TEXT, NOT NULL)
- show_name (TEXT) - Optional show/series name
- season (INTEGER) - Optional season number
- episode (INTEGER) - Optional episode number
- duration (INTEGER, NOT NULL) - Duration in seconds
- video_codec (TEXT) - e.g., "h264", "hevc"
- audio_codec (TEXT) - e.g., "aac", "mp3"
- resolution (TEXT) - e.g., "1920x1080"
- file_size (INTEGER) - Size in bytes
- created_at (DATETIME, DEFAULT CURRENT_TIMESTAMP)

### playlist_items table
- id (TEXT, PRIMARY KEY) - UUID
- channel_id (TEXT, NOT NULL, FK → channels.id)
- media_id (TEXT, NOT NULL, FK → media.id)
- position (INTEGER, NOT NULL) - Order in playlist (0-indexed)
- created_at (DATETIME, DEFAULT CURRENT_TIMESTAMP)

**Constraints:**
- FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
- FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE
- UNIQUE (channel_id, position)

### settings table
- id (INTEGER, PRIMARY KEY, DEFAULT 1) - Singleton settings
- media_library_path (TEXT, NOT NULL) - Path to media library
- transcode_quality (TEXT, DEFAULT 'medium') - "low", "medium", "high"
- hardware_accel (TEXT, DEFAULT 'none') - "none", "nvenc", "qsv", "vaapi", "videotoolbox"
- server_port (INTEGER, DEFAULT 8080)
- updated_at (DATETIME, DEFAULT CURRENT_TIMESTAMP)

## Data Models (Go)

Models are defined in `internal/models/` with GORM struct tags. See `docs/api-specs/infrastructure/infrastructure-api.md` for full model definitions.

### Channel
```go
type Channel struct {
    ID        uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
    Name      string    `json:"name" gorm:"type:text;not null;column:name"`
    Icon      *string   `json:"icon,omitempty" gorm:"type:text;column:icon"`
    StartTime time.Time `json:"start_time" gorm:"type:datetime;not null;column:start_time"`
    Loop      bool      `json:"loop" gorm:"type:integer;not null;default:0;column:loop"`
    CreatedAt time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
    UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:updated_at"`
}
```

### Media
```go
type Media struct {
    ID         uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
    FilePath   string    `json:"file_path" gorm:"type:text;not null;uniqueIndex;column:file_path"`
    Title      string    `json:"title" gorm:"type:text;not null;column:title"`
    ShowName   *string   `json:"show_name,omitempty" gorm:"type:text;column:show_name"`
    Season     *int      `json:"season,omitempty" gorm:"type:integer;column:season"`
    Episode    *int      `json:"episode,omitempty" gorm:"type:integer;column:episode"`
    Duration   int64     `json:"duration" gorm:"type:integer;not null;column:duration"`
    VideoCodec *string   `json:"video_codec,omitempty" gorm:"type:text;column:video_codec"`
    AudioCodec *string   `json:"audio_codec,omitempty" gorm:"type:text;column:audio_codec"`
    Resolution *string   `json:"resolution,omitempty" gorm:"type:text;column:resolution"`
    FileSize   *int64    `json:"file_size,omitempty" gorm:"type:integer;column:file_size"`
    CreatedAt  time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
}
```

### PlaylistItem
```go
type PlaylistItem struct {
    ID        uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
    ChannelID uuid.UUID `json:"channel_id" gorm:"type:text;not null;column:channel_id"`
    MediaID   uuid.UUID `json:"media_id" gorm:"type:text;not null;column:media_id"`
    Position  int       `json:"position" gorm:"type:integer;not null;column:position"`
    CreatedAt time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
    Media     *Media    `json:"media,omitempty" gorm:"foreignKey:MediaID;references:ID"`
}
```

### Settings
```go
type Settings struct {
    ID               int       `json:"id" gorm:"type:integer;primaryKey;default:1;column:id"`
    MediaLibraryPath string    `json:"media_library_path" gorm:"type:text;not null;column:media_library_path"`
    TranscodeQuality string    `json:"transcode_quality" gorm:"type:text;default:medium;column:transcode_quality"`
    HardwareAccel    string    `json:"hardware_accel" gorm:"type:text;default:none;column:hardware_accel"`
    ServerPort       int       `json:"server_port" gorm:"type:integer;default:8080;column:server_port"`
    UpdatedAt        time.Time `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:updated_at"`
}
```

## Repository API

All repositories are accessed through the `Repositories` struct created by `NewRepositories(db *DB)`.

### Channel Repository

```go
Create(ctx, *models.Channel) error
GetByID(ctx, uuid.UUID) (*models.Channel, error)
List(ctx) ([]*models.Channel, error)
Update(ctx, *models.Channel) error
Delete(ctx, uuid.UUID) error
```

### Media Repository

```go
Create(ctx, *models.Media) error
GetByID(ctx, uuid.UUID) (*models.Media, error)
GetByPath(ctx, string) (*models.Media, error)
List(ctx, limit, offset int) ([]*models.Media, error)
ListByShow(ctx, string, limit, offset int) ([]*models.Media, error)
Count(ctx) (int64, error)
CountByShow(ctx, string) (int64, error)
Update(ctx, *models.Media) error
Delete(ctx, uuid.UUID) error
```

### PlaylistItem Repository

```go
Create(ctx, *models.PlaylistItem) error
GetByID(ctx, uuid.UUID) (*models.PlaylistItem, error)
GetByChannelID(ctx, uuid.UUID) ([]*models.PlaylistItem, error)
GetWithMedia(ctx, uuid.UUID) ([]*models.PlaylistItem, error)
Delete(ctx, uuid.UUID) error
DeleteByChannelID(ctx, uuid.UUID) error
Reorder(ctx, uuid.UUID, []ReorderItem) error
```

### Settings Repository

```go
Get(ctx) (*models.Settings, error)
Update(ctx, *models.Settings) error
```

## Database Connection

```go
// Open connection
database, err := db.New("./data/hermes.db")

// Create repositories
repos := db.NewRepositories(database)

// Use repositories
channel, err := repos.Channels.GetByID(ctx, channelID)
```

## Indexes

Indexes are managed by golang-migrate migrations, not GORM.

