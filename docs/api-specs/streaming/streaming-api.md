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

## REST Endpoints

To be defined during implementation.

## Service Interfaces

To be defined during implementation.

