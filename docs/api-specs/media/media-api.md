# Media Service API

Last Updated: 2025-10-30

## Repository Methods

### MediaRepository (Go)

Location: `internal/db/media.go`

```go
func (r *MediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Media, error)
func (r *MediaRepository) ExistsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error)
```

**ExistsByIDs** - Batch validation checking if multiple media IDs exist. Returns map[uuid.UUID]bool where true = exists. Used by bulk playlist operations to avoid N+1 queries.

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

### POST /api/media/scan
Triggers a media library scan

**Request:**
```json
{
  "path": "/path/to/media"
}
```

**Response (201 Created):**
```json
{
  "scan_id": "uuid-here",
  "message": "Scan started"
}
```

**Errors:**
- `400 Bad Request` - Missing or invalid path
- `409 Conflict` - Scan already running
- `500 Internal Server Error` - Failed to start scan

**Usage:**
```bash
curl -X POST http://localhost:8080/api/media/scan \
  -H "Content-Type: application/json" \
  -d '{"path": "/media/videos"}'
```

### GET /api/media/scan/:scanId/status
Get scan progress

**Response (200 OK):**
```json
{
  "scan_id": "uuid-here",
  "status": "running",
  "total_files": 100,
  "processed_files": 50,
  "success_count": 48,
  "failed_count": 2,
  "current_file": "/media/videos/video.mp4",
  "start_time": "2025-10-27T12:00:00Z",
  "end_time": null,
  "errors": ["error message 1", "error message 2"]
}
```

**Status Values:**
- `running` - Scan in progress
- `completed` - Scan finished successfully
- `cancelled` - Scan was cancelled
- `failed` - Scan failed

**Errors:**
- `404 Not Found` - Scan ID not found

**Usage:**
```bash
curl http://localhost:8080/api/media/scan/{scanId}/status
```

### GET /api/media
List all media items with pagination and filtering

**Query Parameters:**
- `limit` (optional) - Items per page (default: 20, max: 10000, use -1 for unlimited)
- `offset` (optional) - Number of items to skip (default: 0)
- `show` (optional) - Filter by show name

**Special limit values:**
- `-1` - Fetch all items (unlimited). Useful for tree views with virtual scrolling
- `1-10000` - Specific page size (values over 10000 are capped at 10000)
- Default: `20` (for backward compatibility)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "uuid-here",
      "file_path": "/media/videos/video.mp4",
      "title": "Show Name - S01E05",
      "show_name": "Show Name",
      "season": 1,
      "episode": 5,
      "duration": 3600,
      "video_codec": "h264",
      "audio_codec": "aac",
      "resolution": "1920x1080",
      "file_size": 1073741824,
      "created_at": "2025-10-27T12:00:00Z"
    }
  ],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

**Errors:**
- `400 Bad Request` - Invalid query parameters
- `500 Internal Server Error` - Query failed

**Usage:**
```bash
# List all media (default pagination)
curl http://localhost:8080/api/media

# With pagination
curl http://localhost:8080/api/media?limit=10&offset=20

# Fetch all items (unlimited) - for tree views
curl http://localhost:8080/api/media?limit=-1

# Filter by show
curl "http://localhost:8080/api/media?show=Friends"

# Filter by show with unlimited fetch
curl "http://localhost:8080/api/media?show=Friends&limit=-1"
```

### GET /api/media/:id
Get single media item details

**Response (200 OK):**
```json
{
  "id": "uuid-here",
  "file_path": "/media/videos/video.mp4",
  "title": "Show Name - S01E05",
  "show_name": "Show Name",
  "season": 1,
  "episode": 5,
  "duration": 3600,
  "video_codec": "h264",
  "audio_codec": "aac",
  "resolution": "1920x1080",
  "file_size": 1073741824,
  "created_at": "2025-10-27T12:00:00Z"
}
```

**Errors:**
- `400 Bad Request` - Invalid UUID format
- `404 Not Found` - Media not found
- `500 Internal Server Error` - Query failed

**Usage:**
```bash
curl http://localhost:8080/api/media/{uuid}
```

### PUT /api/media/:id
Update media metadata (partial update)

**Request:**
```json
{
  "title": "Updated Title",
  "show_name": "Updated Show",
  "season": 2,
  "episode": 10
}
```

All fields are optional - only provided fields will be updated.

**Response (200 OK):**
```json
{
  "id": "uuid-here",
  "file_path": "/media/videos/video.mp4",
  "title": "Updated Title",
  "show_name": "Updated Show",
  "season": 2,
  "episode": 10,
  "duration": 3600,
  "video_codec": "h264",
  "audio_codec": "aac",
  "resolution": "1920x1080",
  "file_size": 1073741824,
  "created_at": "2025-10-27T12:00:00Z"
}
```

**Errors:**
- `400 Bad Request` - Invalid UUID or request body
- `404 Not Found` - Media not found
- `500 Internal Server Error` - Update failed

**Usage:**
```bash
curl -X PUT http://localhost:8080/api/media/{uuid} \
  -H "Content-Type: application/json" \
  -d '{"title": "New Title", "season": 2}'
```

### DELETE /api/media/:id
Delete media from library

**Response (200 OK):**
```json
{
  "message": "Media deleted successfully"
}
```

**Errors:**
- `400 Bad Request` - Invalid UUID format
- `404 Not Found` - Media not found
- `500 Internal Server Error` - Delete failed

**Usage:**
```bash
curl -X DELETE http://localhost:8080/api/media/{uuid}
```

## Data Contracts

See database schema in `docs/api-specs/database/database-api.md` for the `Media` model.

