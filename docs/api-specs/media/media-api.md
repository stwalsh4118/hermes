# Media Service API

Last Updated: 2025-10-27

## Status

In Progress - PBI 2 implementation underway. FFprobe integration and filename parser completed.

## Utility Functions

### FFprobe Integration

Location: `internal/media/ffprobe.go`

```go
func ProbeFile(ctx context.Context, filePath string) (*VideoMetadata, error)
func CheckFFprobeInstalled() error
```

**VideoMetadata:**
```go
type VideoMetadata struct {
    Duration   int64  // Duration in seconds
    VideoCodec string // e.g., "h264", "hevc"
    AudioCodec string // e.g., "aac", "mp3"
    Resolution string // e.g., "1920x1080"
    FileSize   int64  // File size in bytes
    Width      int
    Height     int
}
```

**Usage:**
```go
metadata, err := media.ProbeFile(ctx, "/path/to/video.mp4")
// Returns duration, codecs, resolution, file size
```

**Errors:**
- `ErrFFprobeNotFound` - FFprobe not installed
- `ErrFileNotFound` - File not found/readable
- `ErrInvalidFile` - Corrupted or invalid video
- `ErrTimeout` - Execution timeout (30s)

### Filename Parser

Location: `internal/media/parser.go`

```go
func ParseFilename(fullPath string) ParseResult
```

**ParseResult:**
```go
type ParseResult struct {
    ShowName    *string // Extracted show name
    Season      *int    // Season number
    Episode     *int    // Episode number
    Title       string  // Display title (e.g., "Show - S01E05")
    RawFilename string  // Original filename
}
```

**Supported Patterns:**
- `Show.Name.S01E05.mp4`
- `Show Name - S01E05 - Title.mkv`
- `Show Name 1x05.avi`
- `Show/Season 1/05 - Title.mp4`

**Usage:**
```go
result := media.ParseFilename("/media/Friends.S01E05.mp4")
// ShowName: "Friends", Season: 1, Episode: 5
// Title: "Friends - S01E05"
```

## REST Endpoints

To be defined.

## Data Contracts

See database schema in `docs/api-specs/database/database-api.md` for the `Media` model.

