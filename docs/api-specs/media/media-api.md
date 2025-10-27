# Media Service API

Last Updated: 2025-10-27

## Status

In Progress - PBI 2 implementation underway. Scanner, FFprobe, parser, and validator completed.

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

### Media Validator

Location: `internal/media/validator.go`

```go
func ValidateMedia(metadata *VideoMetadata) ValidationResult
func ValidateFile(filePath string) ValidationResult
```

**ValidationResult:**
```go
type ValidationResult struct {
    Compatible        bool     // true if H.264 + AAC (no transcode needed)
    RequiresTranscode bool     // true if transcoding required
    Reasons           []string // Human-readable incompatibility reasons
    Readable          bool     // File exists and is accessible
}
```

**Codec Compatibility Rules:**
- **Compatible** (no transcode): H.264 video + AAC audio
- **Requires transcode**: Any other codec combination
- Case-insensitive codec matching
- Generates specific reasons for each incompatibility

**Common Compatible Codecs:**
- Video: `h264` (also known as AVC, MPEG-4 Part 10)
- Audio: `aac` (Advanced Audio Coding)

**Common Incompatible Codecs:**
- Video: `hevc`, `vp9`, `av1`, `mpeg2`, `mpeg4`
- Audio: `flac`, `dts`, `opus`, `mp3`, `vorbis`

**Usage:**
```go
// Validate codec compatibility from metadata
metadata, _ := media.ProbeFile(ctx, "/path/to/video.mp4")
result := media.ValidateMedia(metadata)
if result.RequiresTranscode {
    fmt.Println("Transcoding required:", result.Reasons)
}

// Check file accessibility
result := media.ValidateFile("/path/to/video.mp4")
if !result.Readable {
    fmt.Println("File not accessible:", result.Reasons)
}
```

### Media Scanner

Location: `internal/media/scanner.go`

```go
func NewScanner(repos *db.Repositories) *Scanner
func (s *Scanner) StartScan(ctx context.Context, dirPath string) (string, error)
func (s *Scanner) GetScanProgress(scanID string) (*ScanProgress, error)
func (s *Scanner) CancelScan(scanID string) error
func (s *Scanner) Stop() // Graceful shutdown
```

**ScanProgress:**
```go
type ScanProgress struct {
    ScanID         string     `json:"scan_id"`
    Status         ScanStatus `json:"status"` // running, completed, cancelled, failed
    TotalFiles     int        `json:"total_files"`
    ProcessedFiles int        `json:"processed_files"`
    SuccessCount   int        `json:"success_count"`
    FailedCount    int        `json:"failed_count"`
    CurrentFile    string     `json:"current_file"`
    StartTime      time.Time  `json:"start_time"`
    EndTime        *time.Time `json:"end_time,omitempty"`
    Errors         []string   `json:"errors,omitempty"`
}
```

**Features:**
- Async directory scanning with progress tracking
- Integrates FFprobe, parser, and validator
- Thread-safe with `sync.RWMutex`
- Context-based cancellation support
- Auto-cleanup of old scans (1 hour retention)
- Prevents concurrent scans (atomic check-and-insert)
- Optimistic upsert to database (no TOCTOU races)

**Usage:**
```go
scanner := media.NewScanner(repos)
defer scanner.Stop()

// Start scan
scanID, err := scanner.StartScan(ctx, "/media/videos")

// Check progress
progress, err := scanner.GetScanProgress(scanID)

// Cancel if needed
err := scanner.CancelScan(scanID)
```

**Errors:**
- `ErrScanNotFound` - Scan ID not found
- `ErrScanAlreadyRunning` - Another scan is running
- `ErrInvalidDirectory` - Directory invalid/not accessible

## REST Endpoints

To be defined.

## Data Contracts

See database schema in `docs/api-specs/database/database-api.md` for the `Media` model.

