# Database API

Last Updated: 2025-10-26

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

### Channel
```go
type Channel struct {
    ID        string    `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Icon      string    `json:"icon" db:"icon"`
    StartTime time.Time `json:"startTime" db:"start_time"`
    Loop      bool      `json:"loop" db:"loop"`
    CreatedAt time.Time `json:"createdAt" db:"created_at"`
    UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}
```

### Media
```go
type Media struct {
    ID         string    `json:"id" db:"id"`
    FilePath   string    `json:"filePath" db:"file_path"`
    Title      string    `json:"title" db:"title"`
    ShowName   *string   `json:"showName,omitempty" db:"show_name"`
    Season     *int      `json:"season,omitempty" db:"season"`
    Episode    *int      `json:"episode,omitempty" db:"episode"`
    Duration   int       `json:"duration" db:"duration"`
    VideoCodec *string   `json:"videoCodec,omitempty" db:"video_codec"`
    AudioCodec *string   `json:"audioCodec,omitempty" db:"audio_codec"`
    Resolution *string   `json:"resolution,omitempty" db:"resolution"`
    FileSize   *int64    `json:"fileSize,omitempty" db:"file_size"`
    CreatedAt  time.Time `json:"createdAt" db:"created_at"`
}
```

### PlaylistItem
```go
type PlaylistItem struct {
    ID        string    `json:"id" db:"id"`
    ChannelID string    `json:"channelId" db:"channel_id"`
    MediaID   string    `json:"mediaId" db:"media_id"`
    Position  int       `json:"position" db:"position"`
    CreatedAt time.Time `json:"createdAt" db:"created_at"`
}
```

### Settings
```go
type Settings struct {
    ID              int       `json:"id" db:"id"`
    MediaLibraryPath string   `json:"mediaLibraryPath" db:"media_library_path"`
    TranscodeQuality string   `json:"transcodeQuality" db:"transcode_quality"`
    HardwareAccel   string   `json:"hardwareAccel" db:"hardware_accel"`
    ServerPort      int       `json:"serverPort" db:"server_port"`
    UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}
```

## Indexes

To be added as needed based on query patterns.

