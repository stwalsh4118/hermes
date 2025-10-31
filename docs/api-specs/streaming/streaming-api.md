# Streaming Engine API

Last Updated: 2025-10-30

## Status

This API specification is being populated during PBI 6 (Streaming Engine) implementation.

## Hardware Detection

Location: `internal/streaming/hardware.go`

### CheckFFmpegInstalled

```go
func CheckFFmpegInstalled() error
```

Checks if FFmpeg is available in the system PATH.

**Returns:**
- `nil` if FFmpeg is installed
- `ErrFFmpegNotFound` if FFmpeg is not found

**Usage:**
```go
if err := streaming.CheckFFmpegInstalled(); err != nil {
    log.Fatal("FFmpeg required but not found")
}
```

### HardwareAccel Type

```go
type HardwareAccel string

const (
    HardwareAccelNone         HardwareAccel = "none"
    HardwareAccelNVENC        HardwareAccel = "nvenc"
    HardwareAccelQSV          HardwareAccel = "qsv"
    HardwareAccelVAAPI        HardwareAccel = "vaapi"
    HardwareAccelVideoToolbox HardwareAccel = "videotoolbox"
    HardwareAccelAuto         HardwareAccel = "auto"
)

func (h HardwareAccel) String() string
func (h HardwareAccel) IsValid() bool
```

Typed enum for hardware acceleration methods with validation.

### DetectHardwareEncoders

```go
func DetectHardwareEncoders(ctx context.Context) ([]HardwareAccel, error)
```

Probes FFmpeg for available hardware encoders. Parses encoder names from exact field positions to prevent false positives.

**Returns:**
- `[]HardwareAccel`: Unique list of available methods (always includes "none")
- `error`: ErrFFmpegNotFound, ErrTimeout, or execution error

**Usage:**
```go
encoders, err := streaming.DetectHardwareEncoders(ctx)
// encoders: []HardwareAccel{HardwareAccelNVENC, HardwareAccelNone}
```

### ValidateHardwareAccel

```go
func ValidateHardwareAccel(method HardwareAccel, available []HardwareAccel) error
```

Validates hardware acceleration method against available encoders.

**Usage:**
```go
available, _ := streaming.DetectHardwareEncoders(ctx)
if err := streaming.ValidateHardwareAccel(streaming.HardwareAccelNVENC, available); err != nil {
    // NVENC not available
}
```

### SelectBestEncoder

```go
func SelectBestEncoder(available []HardwareAccel) HardwareAccel
```

Auto-selects best encoder. **Priority:** nvenc > qsv > videotoolbox > vaapi > none

**Usage:**
```go
best := streaming.SelectBestEncoder(available)
```

### Errors

```go
var (
    ErrFFmpegNotFound = errors.New("ffmpeg not found in PATH")
    ErrTimeout        = errors.New("ffmpeg detection timed out")
)
```

## FFmpeg Command Builder

Location: `internal/streaming/ffmpeg.go`

### Quality Constants

```go
const (
    Quality1080p = "1080p"
    Quality720p  = "720p"
    Quality480p  = "480p"
)
```

### StreamParams

```go
type StreamParams struct {
    InputFile       string        // Path to input video file
    OutputPath      string        // Full path to output .m3u8 playlist
    Quality         string        // Quality level (1080p, 720p, 480p)
    HardwareAccel   HardwareAccel // Hardware acceleration method
    SeekSeconds     int64         // Starting position in seconds (0 = beginning)
    SegmentDuration int           // HLS segment duration in seconds
    PlaylistSize    int           // Number of segments to keep in playlist
}
```

Parameters for building an FFmpeg HLS command.

### FFmpegCommand

```go
type FFmpegCommand struct {
    Args []string // Command arguments (without "ffmpeg" itself)
}
```

Represents a built FFmpeg command ready for execution.

### BuildHLSCommand

```go
func BuildHLSCommand(params StreamParams) (*FFmpegCommand, error)
```

Builds a complete FFmpeg command for HLS stream generation with specified quality and hardware acceleration.

**Returns:**
- `*FFmpegCommand`: Built command with all arguments
- `error`: Validation errors (invalid quality, hardware accel, paths, or params)

**Usage:**
```go
params := streaming.StreamParams{
    InputFile:       "/media/video.mp4",
    OutputPath:      "/streams/channel1/1080p.m3u8",
    Quality:         streaming.Quality1080p,
    HardwareAccel:   streaming.HardwareAccelNVENC,
    SeekSeconds:     3600, // Start at 1 hour
    SegmentDuration: 6,
    PlaylistSize:    10,
}

cmd, err := streaming.BuildHLSCommand(params)
if err != nil {
    return err
}

// Execute: exec.Command("ffmpeg", cmd.Args...)
```

### Quality Specifications

**1080p:**
- Video bitrate: 5000 kbps
- Resolution: 1920x1080
- Buffer size: 10000k

**720p:**
- Video bitrate: 3000 kbps
- Resolution: 1280x720
- Buffer size: 6000k

**480p:**
- Video bitrate: 1500 kbps
- Resolution: 854x480
- Buffer size: 3000k

**Audio (all qualities):**
- Codec: AAC
- Bitrate: 192 kbps
- Channels: 2 (stereo)

### Hardware Encoder Mapping

**Software (none/auto):**
```
-c:v libx264 -preset veryfast
```

**NVENC (NVIDIA):**
```
-c:v h264_nvenc -preset p1
```

**QSV (Intel):**
```
-c:v h264_qsv -preset veryfast
```

**VAAPI (AMD/Intel Linux):**
```
-c:v h264_vaapi
```

**VideoToolbox (macOS):**
```
-c:v h264_videotoolbox
```

### Example Commands

**1080p Software Encoding:**
```
ffmpeg -i /media/video.mp4 \
  -c:v libx264 -preset veryfast \
  -c:a aac -b:a 192k -ac 2 \
  -b:v 5000k -maxrate 5000k -bufsize 10000k -s 1920x1080 \
  -f hls -hls_time 6 -hls_list_size 10 \
  -hls_flags delete_segments \
  -hls_segment_filename /streams/channel1/1080p_segment_%03d.ts \
  -hls_playlist_type event \
  /streams/channel1/1080p.m3u8
```

**720p with NVENC + Seeking:**
```
ffmpeg -ss 3600 -i /media/video.mp4 \
  -c:v h264_nvenc -preset p1 \
  -c:a aac -b:a 192k -ac 2 \
  -b:v 3000k -maxrate 3000k -bufsize 6000k -s 1280x720 \
  -f hls -hls_time 6 -hls_list_size 10 \
  -hls_flags delete_segments \
  -hls_segment_filename /streams/channel1/720p_segment_%03d.ts \
  -hls_playlist_type event \
  /streams/channel1/720p.m3u8
```

### Path Helpers

**GetOutputPath:**
```go
func GetOutputPath(baseDir, channelID, quality string) string
```

Generates consistent output path for channel and quality.

```go
path := streaming.GetOutputPath("/streams", "channel1", "1080p")
// Returns: /streams/channel1/1080p.m3u8
```

**GetSegmentPattern:**
```go
func GetSegmentPattern(channelID, quality string) string
```

Generates segment naming pattern.

```go
pattern := streaming.GetSegmentPattern("channel1", "1080p")
// Returns: channel1_1080p_segment_%03d.ts
```

**GetPlaylistFilename:**
```go
func GetPlaylistFilename(quality string) string
```

Returns playlist filename for quality.

```go
filename := streaming.GetPlaylistFilename("1080p")
// Returns: 1080p.m3u8
```

### Errors

```go
var (
    ErrInvalidQuality         = errors.New("invalid quality level")
    ErrInvalidHardwareAccel   = errors.New("invalid hardware acceleration method")
    ErrEmptyInputFile         = errors.New("input file cannot be empty")
    ErrEmptyOutputPath        = errors.New("output path cannot be empty")
    ErrInvalidSegmentDuration = errors.New("segment duration must be positive")
    ErrInvalidPlaylistSize    = errors.New("playlist size must be positive")
)
```

## HLS Playlist Generation

Location: `internal/streaming/playlist.go`

### Playlist Type Constants

```go
const (
    PlaylistTypeEvent = "EVENT"  // Live streaming playlist
    PlaylistTypeVOD   = "VOD"    // Video on demand playlist
)
```

### Data Structures

#### MasterPlaylist

```go
type MasterPlaylist struct {
    Variants []PlaylistVariant
}

type PlaylistVariant struct {
    Bandwidth  int    // Video + audio bitrate in bits per second
    Resolution string // Format: "1920x1080"
    Path       string // Relative path to media playlist
}
```

Represents an HLS master playlist with multiple quality variants for adaptive bitrate streaming.

#### MediaPlaylist

```go
type MediaPlaylist struct {
    TargetDuration int       // Maximum segment duration in seconds
    MediaSequence  int       // Starting sequence number
    Segments       []Segment // List of segments
    PlaylistType   string    // "EVENT" or "VOD"
}

type Segment struct {
    Duration float64 // Segment duration in seconds
    Path     string  // Filename or relative path to segment
}
```

Represents an HLS media playlist containing segment information.

#### MediaPlaylistConfig

```go
type MediaPlaylistConfig struct {
    TargetDuration int    // Maximum segment duration in seconds
    MediaSequence  int    // Starting sequence number
    PlaylistType   string // "EVENT" or "VOD"
    MaxSegments    int    // Maximum number of segments to keep (sliding window)
}
```

Configuration for media playlist generation.

### GenerateMasterPlaylist

```go
func GenerateMasterPlaylist(variants []PlaylistVariant) (string, error)
```

Generates an HLS-compliant master playlist from quality variants.

**Parameters:**
- `variants`: List of quality variants with bandwidth, resolution, and playlist path

**Returns:**
- `string`: Formatted HLS master playlist content
- `error`: Validation errors (empty variants, invalid bandwidth/resolution/path)

**Generated Format:**
```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=5192000,RESOLUTION=1920x1080
1080p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=3192000,RESOLUTION=1280x720
720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1692000,RESOLUTION=854x480
480p.m3u8
```

**Usage:**
```go
variants := []streaming.PlaylistVariant{
    {
        Bandwidth:  5192000,
        Resolution: "1920x1080",
        Path:       "1080p.m3u8",
    },
    {
        Bandwidth:  3192000,
        Resolution: "1280x720",
        Path:       "720p.m3u8",
    },
    {
        Bandwidth:  1692000,
        Resolution: "854x480",
        Path:       "480p.m3u8",
    },
}

playlist, err := streaming.GenerateMasterPlaylist(variants)
if err != nil {
    return err
}
```

### GenerateMediaPlaylist

```go
func GenerateMediaPlaylist(segments []Segment, config MediaPlaylistConfig) (string, error)
```

Generates an HLS-compliant media playlist from segments.

**Parameters:**
- `segments`: List of video segments with duration and path
- `config`: Configuration including target duration, media sequence, playlist type, max segments

**Returns:**
- `string`: Formatted HLS media playlist content
- `error`: Validation errors (invalid playlist type)

**Features:**
- Auto-calculates target duration if not provided (ceiling of max segment duration)
- Supports sliding window (keeps only last N segments if MaxSegments > 0)
- Automatically updates MediaSequence when segments are dropped (MediaSequence += droppedSegments)
- Adds `#EXT-X-ENDLIST` tag for VOD playlists
- EVENT playlists remain open for new segments

**Generated Format (EVENT):**
```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:EVENT
#EXTINF:6.0,
segment_000.ts
#EXTINF:6.0,
segment_001.ts
#EXTINF:6.0,
segment_002.ts
```

**Generated Format (VOD):**
```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD
#EXTINF:6.0,
segment_000.ts
#EXTINF:6.0,
segment_001.ts
#EXT-X-ENDLIST
```

**Usage:**
```go
segments := []streaming.Segment{
    {Duration: 6.0, Path: "segment_000.ts"},
    {Duration: 6.0, Path: "segment_001.ts"},
    {Duration: 6.0, Path: "segment_002.ts"},
}

config := streaming.MediaPlaylistConfig{
    TargetDuration: 6,
    MediaSequence:  0,
    PlaylistType:   streaming.PlaylistTypeEvent,
    MaxSegments:    10, // Sliding window size
}

playlist, err := streaming.GenerateMediaPlaylist(segments, config)
if err != nil {
    return err
}
```

### DiscoverSegments

```go
func DiscoverSegments(directory string) ([]Segment, error)
```

Scans a directory and discovers all HLS segment files (.ts).

**Parameters:**
- `directory`: Path to directory containing segment files

**Returns:**
- `[]Segment`: Sorted list of discovered segments (by sequence number)
- `error`: Directory access errors (invalid directory, not a directory)

**Behavior:**
- Returns empty list (not error) if directory doesn't exist
- Parses segment filenames matching pattern: `*_segment_NNN.ts`
- Sorts segments by sequence number
- Assigns default duration (6 seconds) to each segment
- Ignores non-segment files

**Usage:**
```go
segments, err := streaming.DiscoverSegments("/streams/channel1/1080p")
if err != nil {
    return err
}

// Use discovered segments to generate playlist
config := streaming.MediaPlaylistConfig{
    TargetDuration: 6,
    MediaSequence:  0,
    PlaylistType:   streaming.PlaylistTypeEvent,
}
playlist, err := streaming.GenerateMediaPlaylist(segments, config)
```

### WritePlaylistAtomic

```go
func WritePlaylistAtomic(path string, content string) error
```

Writes a playlist to a file atomically to prevent partial writes.

**Parameters:**
- `path`: Full path to playlist file
- `content`: Playlist content to write

**Returns:**
- `error`: File operation errors (directory creation, write, sync, rename)

**Atomic Write Process:**
1. Creates directory if it doesn't exist
2. Writes content to temporary file in same directory
3. Syncs temp file to disk
4. Atomically renames temp file to final path
5. Cleans up temp file on error

**Thread Safety:**
- Safe for concurrent writes to different files
- Atomic rename ensures clients never read partial content

**Usage:**
```go
playlist, _ := streaming.GenerateMasterPlaylist(variants)
if err := streaming.WritePlaylistAtomic("/streams/channel1/master.m3u8", playlist); err != nil {
    return fmt.Errorf("failed to write playlist: %w", err)
}
```

### ValidateMasterPlaylist

```go
func ValidateMasterPlaylist(content string) error
```

Validates an HLS master playlist for RFC 8216 compliance.

**Parameters:**
- `content`: Master playlist content to validate

**Returns:**
- `error`: Validation errors (missing required tags, invalid format)

**Checks:**
- Presence of `#EXTM3U` header
- Presence of `#EXT-X-VERSION` tag
- At least one `#EXT-X-STREAM-INF` tag
- Each stream-inf has BANDWIDTH and RESOLUTION attributes

**Usage:**
```go
if err := streaming.ValidateMasterPlaylist(playlistContent); err != nil {
    log.Warn("Invalid master playlist: %v", err)
}
```

### ValidateMediaPlaylist

```go
func ValidateMediaPlaylist(content string) error
```

Validates an HLS media playlist for RFC 8216 compliance.

**Parameters:**
- `content`: Media playlist content to validate

**Returns:**
- `error`: Validation errors (missing required tags)

**Checks:**
- Presence of `#EXTM3U` header
- Presence of `#EXT-X-VERSION` tag
- Presence of `#EXT-X-TARGETDURATION` tag
- Presence of `#EXT-X-MEDIA-SEQUENCE` tag

**Usage:**
```go
if err := streaming.ValidateMediaPlaylist(playlistContent); err != nil {
    log.Warn("Invalid media playlist: %v", err)
}
```

### GetBandwidthForQuality

```go
func GetBandwidthForQuality(quality string) (int, error)
```

Returns the total bandwidth in bits per second for a quality level.

**Parameters:**
- `quality`: Quality level (Quality1080p, Quality720p, Quality480p)

**Returns:**
- `int`: Bandwidth in bps (video + audio)
- `error`: Invalid quality error

**Bandwidths:**
- 1080p: 5,192,000 bps (5000k video + 192k audio)
- 720p: 3,192,000 bps (3000k video + 192k audio)
- 480p: 1,692,000 bps (1500k video + 192k audio)

**Usage:**
```go
bandwidth, err := streaming.GetBandwidthForQuality(streaming.Quality1080p)
// bandwidth: 5192000
```

### GetResolutionForQuality

```go
func GetResolutionForQuality(quality string) (string, error)
```

Returns the resolution string for a quality level.

**Parameters:**
- `quality`: Quality level (Quality1080p, Quality720p, Quality480p)

**Returns:**
- `string`: Resolution in "WIDTHxHEIGHT" format
- `error`: Invalid quality error

**Resolutions:**
- 1080p: "1920x1080"
- 720p: "1280x720"
- 480p: "854x480"

**Usage:**
```go
resolution, err := streaming.GetResolutionForQuality(streaming.Quality720p)
// resolution: "1280x720"
```

### Errors

```go
var (
    ErrEmptyVariants       = errors.New("playlist must have at least one variant")
    ErrInvalidBandwidth    = errors.New("bandwidth must be positive")
    ErrInvalidResolution   = errors.New("invalid resolution format")
    ErrInvalidPlaylistType = errors.New("invalid playlist type (must be EVENT or VOD)")
    ErrMissingRequiredTag  = errors.New("missing required HLS tag")
    ErrInvalidDirectory    = errors.New("invalid directory path")
)
```

## Stream State Machine

Location: `internal/streaming/types.go`

### StreamState Type

```go
type StreamState string

const (
    StateIdle     StreamState = "idle"     // No active stream
    StateStarting StreamState = "starting" // FFmpeg process launching
    StateActive   StreamState = "active"   // Stream running, generating segments
    StateStopping StreamState = "stopping" // Graceful shutdown in progress
    StateFailed   StreamState = "failed"   // Stream failed, needs recovery
)

func (s StreamState) String() string
func (s StreamState) IsValid() bool
func (s StreamState) CanTransitionTo(newState StreamState) bool
```

### Valid State Transitions

```
idle ──────> starting ──────> active ──────> stopping ──────> idle
               │                 │                              ▲
               │                 │                              │
               │                 ▼                              │
               └─────────────> failed ─────────────────────────┘
```

**From Idle:**
- → Starting (only valid transition)

**From Starting:**
- → Active (successful start)
- → Failed (startup failure)
- → Stopping (early abort)

**From Active:**
- → Stopping (graceful shutdown)
- → Failed (runtime error)

**From Stopping:**
- → Idle (cleanup complete)

**From Failed:**
- → Starting (retry)
- → Idle (give up)

**Usage:**
```go
currentState := streaming.StateIdle
if currentState.CanTransitionTo(streaming.StateStarting) {
    // Transition is valid
}
```

## Stream Quality Variants

Location: `internal/streaming/types.go`

### StreamQuality Struct

```go
type StreamQuality struct {
    Level        string `json:"level"`         // "1080p", "720p", "480p"
    Bitrate      int    `json:"bitrate"`       // Video bitrate in kbps
    Resolution   string `json:"resolution"`    // "1920x1080"
    SegmentPath  string `json:"segment_path"`  // Path to segments
    PlaylistPath string `json:"playlist_path"` // Path to .m3u8 file
}
```

**Usage:**
```go
qualities := []streaming.StreamQuality{
    {
        Level:        streaming.Quality1080p,
        Bitrate:      5000,
        Resolution:   "1920x1080",
        SegmentPath:  "/streams/channel1/1080p",
        PlaylistPath: "/streams/channel1/1080p.m3u8",
    },
    {
        Level:        streaming.Quality720p,
        Bitrate:      3000,
        Resolution:   "1280x720",
        SegmentPath:  "/streams/channel1/720p",
        PlaylistPath: "/streams/channel1/720p.m3u8",
    },
}
```

## Session Manager

Location: `internal/streaming/types.go`

### SessionManager Type

```go
type SessionManager struct {
    sessions map[string]*models.StreamSession
    mu       sync.RWMutex
}

func NewSessionManager() *SessionManager
func (m *SessionManager) Get(channelID string) (*models.StreamSession, bool)
func (m *SessionManager) Set(channelID string, session *models.StreamSession)
func (m *SessionManager) Delete(channelID string)
func (m *SessionManager) List() []*models.StreamSession
func (m *SessionManager) GetAll(filter func(*models.StreamSession) bool) []*models.StreamSession
```

Thread-safe collection for managing active streaming sessions.

**Usage:**
```go
manager := streaming.NewSessionManager()

// Set a session
channelID := uuid.New()
session := models.NewStreamSession(channelID)
manager.Set(channelID.String(), session)

// Get a session
session, ok := manager.Get(channelID.String())
if ok {
    // Session exists
}

// Delete a session
manager.Delete(channelID.String())

// List all sessions
allSessions := manager.List()

// Filter sessions
activeSessions := manager.GetAll(func(s *models.StreamSession) bool {
    return s.IsActive()
})
```

## Stream Session (In-Memory)

Location: `internal/models/stream_session.go`

**Note:** This model is NOT persisted to the database. It's used for runtime streaming state management only.

### StreamSession Struct

```go
type StreamSession struct {
    ID             uuid.UUID       `json:"id"`
    ChannelID      uuid.UUID       `json:"channel_id"`
    StartedAt      time.Time       `json:"started_at"`
    ClientCount    int             `json:"client_count"`
    FFmpegPID      int             `json:"ffmpeg_pid"`
    State          string          `json:"state"`
    Qualities      []StreamQuality `json:"qualities"`
    LastAccessTime time.Time       `json:"last_access_time"`
    ErrorCount     int             `json:"error_count"`
    LastError      string          `json:"last_error"`
    SegmentPath    string          `json:"segment_path"`
    OutputDir      string          `json:"output_dir"`
    mu             sync.RWMutex
}

func NewStreamSession(channelID uuid.UUID) *StreamSession
```

### Client Management Methods

```go
func (s *StreamSession) IncrementClients()
func (s *StreamSession) DecrementClients()
func (s *StreamSession) GetClientCount() int
func (s *StreamSession) IsActive() bool  // Returns true if ClientCount > 0
```

### State Management Methods

```go
func (s *StreamSession) GetState() string
func (s *StreamSession) SetState(state string)
```

**Note:** State validation should be done by caller using `streaming.StreamState` type:
```go
currentState := streaming.StreamState(session.GetState())
if currentState.CanTransitionTo(streaming.StateActive) {
    session.SetState(streaming.StateActive.String())
}
```

### Quality Management Methods

```go
func (s *StreamSession) SetQualities(qualities []StreamQuality)
func (s *StreamSession) GetQualities() []StreamQuality
```

### Error Tracking Methods

```go
func (s *StreamSession) IncrementErrorCount()
func (s *StreamSession) GetErrorCount() int
func (s *StreamSession) SetLastError(err error)
func (s *StreamSession) GetLastError() string
func (s *StreamSession) ResetErrors()
```

### Access Time and Cleanup Methods

```go
func (s *StreamSession) UpdateLastAccess()
func (s *StreamSession) GetLastAccessTime() time.Time
func (s *StreamSession) IdleDuration() time.Duration
func (s *StreamSession) ShouldCleanup(gracePeriod time.Duration) bool
```

**Cleanup Logic:**
`ShouldCleanup` returns true if BOTH conditions are met:
1. `ClientCount == 0` (no active clients)
2. `IdleDuration() > gracePeriod` (grace period expired)

### FFmpeg Process Methods

```go
func (s *StreamSession) SetFFmpegPID(pid int)
func (s *StreamSession) GetFFmpegPID() int
```

### Path Management Methods

```go
func (s *StreamSession) GetSegmentPath() string
func (s *StreamSession) SetSegmentPath(path string)
func (s *StreamSession) GetOutputDir() string
func (s *StreamSession) SetOutputDir(dir string)
```

### Usage Example

```go
// Create session
channelID := uuid.New()
session := models.NewStreamSession(channelID)

// Track clients
session.IncrementClients()
fmt.Println(session.IsActive())  // true

// Manage state
session.SetState(streaming.StateStarting.String())

// Set quality variants
qualities := []models.StreamQuality{
    {Level: "1080p", Bitrate: 5000, Resolution: "1920x1080"},
    {Level: "720p", Bitrate: 3000, Resolution: "1280x720"},
}
session.SetQualities(qualities)

// Track errors
if err := startFFmpeg(); err != nil {
    session.IncrementErrorCount()
    session.SetLastError(err)
}

// Update access time
session.UpdateLastAccess()

// Check cleanup eligibility
gracePeriod := 30 * time.Second
if session.ShouldCleanup(gracePeriod) {
    // Cleanup the stream
}
```

## Thread Safety

All session operations are thread-safe:
- `StreamSession` uses `sync.RWMutex` for all field access
- `SessionManager` uses `sync.RWMutex` for collection operations
- Safe for concurrent access from multiple goroutines
- All tests pass with `-race` flag

## Stream Manager Service

Location: `internal/streaming/manager.go`

### StreamManager Type

```go
type StreamManager struct {
    repos           *db.Repositories
    timelineService *timeline.TimelineService
    sessionManager  *SessionManager
    config          *config.StreamingConfig
    cleanupTicker   *time.Ticker
    stopChan        chan struct{}
    cleanupDone     chan struct{}
    mu              sync.RWMutex
    stopped         bool
}

func NewStreamManager(
    repos *db.Repositories,
    timelineService *timeline.TimelineService,
    cfg *config.StreamingConfig,
) *StreamManager

func (m *StreamManager) Start() error
func (m *StreamManager) Stop()
func (m *StreamManager) StartStream(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error)
func (m *StreamManager) StopStream(ctx context.Context, channelID uuid.UUID) error
func (m *StreamManager) GetStream(channelID uuid.UUID) (*models.StreamSession, bool)
func (m *StreamManager) RegisterClient(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error)
func (m *StreamManager) UnregisterClient(ctx context.Context, channelID uuid.UUID) error
```

Central orchestrator for the streaming pipeline. Manages stream lifecycle, coordinates FFmpeg processes, tracks client connections with grace periods, and ensures proper resource cleanup.

### NewStreamManager

Creates a new stream manager instance with the required dependencies.

**Parameters:**
- `repos` - Database repositories for accessing channels and media
- `timelineService` - Timeline service for calculating current playback positions
- `cfg` - Streaming configuration (hardware accel, segment paths, grace periods)

**Returns:**
- `*StreamManager` - Initialized stream manager (not yet started)

**Usage:**
```go
repos := db.NewRepositories(database)
timelineService := timeline.NewTimelineService(repos)
streamManager := streaming.NewStreamManager(repos, timelineService, &cfg.Streaming)
```

### Start

Initializes the stream manager and starts background cleanup goroutine.

**Returns:**
- `error` - ErrManagerStopped if already stopped, nil on success

**Process:**
1. Creates cleanup ticker based on configuration
2. Starts background cleanup goroutine
3. Logs startup with cleanup interval and grace period settings

**Usage:**
```go
if err := streamManager.Start(); err != nil {
    return fmt.Errorf("failed to start stream manager: %w", err)
}
```

### Stop

Gracefully shuts down the stream manager.

**Process:**
1. Signals cleanup goroutine to stop
2. Waits for cleanup goroutine to finish
3. Stops cleanup ticker
4. Stops all active streams
5. Logs shutdown with count of stopped streams

**Thread Safety:**
- Safe to call multiple times
- Idempotent (subsequent calls are no-ops)

**Usage:**
```go
streamManager.Stop()
```

### StartStream

Starts a new stream for a channel or returns existing stream if already active.

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `channelID` - UUID of the channel to stream

**Returns:**
- `*models.StreamSession` - Active stream session
- `error` - One of:
  - `ErrManagerStopped` - Manager has been stopped
  - Channel not found errors
  - Timeline calculation errors
  - FFmpeg launch errors

**Process:**
1. Checks if stream already exists → returns immediately if found
2. Fetches channel from database
3. Gets current timeline position (what should be playing now)
4. Fetches media file information
5. Creates output directories for segments
6. Builds FFmpeg command with timeline seek position
7. Launches FFmpeg process
8. Creates StreamSession with process info
9. Stores session in SessionManager
10. Starts background process monitor
11. Returns session

**Concurrent Behavior:**
- Multiple clients can call StartStream for same channel
- First call creates stream, subsequent calls return existing
- Thread-safe via SessionManager

**Usage:**
```go
session, err := streamManager.StartStream(ctx, channelID)
if err != nil {
    return fmt.Errorf("failed to start stream: %w", err)
}

fmt.Printf("Stream started: %s\n", session.ID)
fmt.Printf("FFmpeg PID: %d\n", session.GetFFmpegPID())
```

### StopStream

Stops a stream and cleans up all resources.

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `channelID` - UUID of the channel stream to stop

**Returns:**
- `error` - ErrStreamNotFound if stream doesn't exist

**Process:**
1. Gets stream session from manager
2. Sets state to Stopping
3. Terminates FFmpeg process (SIGTERM, then SIGKILL if needed)
4. Cleans up segment files from disk
5. Removes session from SessionManager

**Resource Cleanup:**
- FFmpeg process terminated gracefully (5s timeout) then forcefully
- All segment files and directories removed
- Session removed from memory

**Usage:**
```go
if err := streamManager.StopStream(ctx, channelID); err != nil {
    if errors.Is(err, streaming.ErrStreamNotFound) {
        // Stream wasn't running
        return nil
    }
    return err
}
```

### GetStream

Retrieves a stream session by channel ID.

**Parameters:**
- `channelID` - UUID of the channel

**Returns:**
- `*models.StreamSession` - Stream session if found
- `bool` - true if session exists, false otherwise

**Thread Safety:**
- Read-only operation
- Safe for concurrent access

**Usage:**
```go
session, found := streamManager.GetStream(channelID)
if !found {
    // Stream not active
    return nil
}

clientCount := session.GetClientCount()
state := session.GetState()
```

### RegisterClient

Registers a client connection for a channel (starts stream if needed).

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `channelID` - UUID of the channel to stream

**Returns:**
- `*models.StreamSession` - Active stream session
- `error` - Same errors as StartStream

**Process:**
1. Calls StartStream (creates new or returns existing)
2. Increments client count on session
3. Updates last access time
4. Returns session

**Client Tracking:**
- Client count used for cleanup decisions
- Multiple clients share same stream instance
- Last access time updated for grace period calculation

**Usage:**
```go
session, err := streamManager.RegisterClient(ctx, channelID)
if err != nil {
    return fmt.Errorf("failed to register client: %w", err)
}

// Use session for streaming
playlistPath := session.GetQualities()[0].PlaylistPath
```

### UnregisterClient

Unregisters a client connection from a channel.

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `channelID` - UUID of the channel

**Returns:**
- `error` - ErrStreamNotFound if stream doesn't exist

**Process:**
1. Gets stream session
2. Decrements client count
3. Updates last access time
4. Grace period starts automatically if client count reaches zero

**Grace Period Behavior:**
- When last client disconnects, grace period timer starts
- Stream continues running during grace period
- If no client reconnects before grace period expires, cleanup goroutine stops stream
- If client reconnects during grace period, stream continues without interruption

**Usage:**
```go
if err := streamManager.UnregisterClient(ctx, channelID); err != nil {
    // Log warning but don't fail - client already disconnected
    logger.Log.Warn().Err(err).Msg("Failed to unregister client")
}
```

### Background Cleanup

The stream manager runs a background goroutine that periodically checks for idle streams.

**Cleanup Process:**
1. Runs every `CleanupInterval` seconds (from config)
2. Iterates through all active sessions
3. Checks if session should be cleaned up:
   - Client count == 0 (no active clients)
   - Idle duration > grace period (from config)
4. Calls StopStream for eligible sessions
5. Cleans up orphaned segment directories

**Configuration:**
- `CleanupInterval` - How often cleanup runs (default: 60 seconds)
- `GracePeriodSeconds` - How long to keep idle streams (default: 30 seconds)

**Graceful Shutdown:**
- Stop() signals cleanup goroutine to exit
- Waits for cleanup to finish before proceeding
- Ensures no resource leaks on shutdown

### Errors

```go
var (
    ErrStreamNotFound      = errors.New("stream not found")
    ErrStreamAlreadyExists = errors.New("stream already exists")
    ErrManagerStopped      = errors.New("stream manager has been stopped")
)
```

### Process Management (process.go)

**launchFFmpeg:**
```go
func launchFFmpeg(cmd *FFmpegCommand) (*exec.Cmd, error)
```

Launches an FFmpeg process with stdout/stderr capture for logging.

**terminateProcess:**
```go
func terminateProcess(pid int) error
```

Terminates a process gracefully (SIGTERM) then forcefully (SIGKILL) if needed.
- Timeout: 5 seconds for graceful termination
- Falls back to SIGKILL after timeout

**captureFFmpegOutput:**
```go
func captureFFmpegOutput(pid int, reader io.Reader, streamName string)
```

Captures and logs FFmpeg output, detecting errors for alerting.

### Resource Cleanup (cleanup.go)

**createSegmentDirectories:**
```go
func createSegmentDirectories(baseDir, channelID string) error
```

Creates directory structure for stream segments (1080p, 720p, 480p subdirectories).

**cleanupSegments:**
```go
func cleanupSegments(outputDir string) error
```

Removes all segment files and directories for a channel.

**cleanupOrphanedDirectories:**
```go
func cleanupOrphanedDirectories(baseDir string, activeSessions []*models.StreamSession) error
```

Removes segment directories for channels that no longer have active streams.

### Complete Usage Example

```go
package main

import (
    "context"
    "time"
    
    "github.com/stwalsh4118/hermes/internal/config"
    "github.com/stwalsh4118/hermes/internal/db"
    "github.com/stwalsh4118/hermes/internal/streaming"
    "github.com/stwalsh4118/hermes/internal/timeline"
)

func main() {
    // Load configuration
    cfg, _ := config.Load()
    
    // Initialize database
    database, _ := db.New(cfg.Database.Path)
    repos := db.NewRepositories(database)
    
    // Create timeline service
    timelineService := timeline.NewTimelineService(repos)
    
    // Create and start stream manager
    streamManager := streaming.NewStreamManager(repos, timelineService, &cfg.Streaming)
    if err := streamManager.Start(); err != nil {
        panic(err)
    }
    defer streamManager.Stop()
    
    // Register a client (starts stream automatically)
    ctx := context.Background()
    channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
    
    session, err := streamManager.RegisterClient(ctx, channelID)
    if err != nil {
        panic(err)
    }
    
    // Get stream information
    qualities := session.GetQualities()
    playlistPath := qualities[0].PlaylistPath
    
    // Client watches stream...
    time.Sleep(10 * time.Minute)
    
    // Unregister client
    streamManager.UnregisterClient(ctx, channelID)
    
    // Stream continues for grace period (30s default)
    // Then automatically stops via cleanup goroutine
}
```

## REST Endpoints

To be defined during implementation.

## Service Interfaces

To be defined during implementation.

