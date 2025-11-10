# Streaming Engine API

Last Updated: 2025-10-31

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

### 24/7 Channel Behavior

The streaming system implements continuous 24/7 channel behavior:

**Infinite Looping:**
- Videos loop infinitely using `-stream_loop -1`
- Provides continuous playback like a TV channel
- Seeks are applied before looping starts

**Sliding Window Playlist:**
- Maintains only the most recent N segments (`hls_list_size`)
- Old segments are automatically deleted (`hls_flags delete_segments`)
- No `hls_playlist_type` specified (not "event" or "vod")
- Creates a rolling window of content

**Benefits:**
- Constant disk usage (only 10 segments per quality)
- Fast startup (clients see recent segments immediately)
- Continuous playback experience
- Automatic old segment cleanup

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
    RealtimePacing  bool          // Enable -re flag for 1x speed encoding
    EncodingPreset  string        // FFmpeg encoding preset (ultrafast, veryfast, medium, slow)
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

Encoding presets control the speed vs quality tradeoff. The `EncodingPreset` field in `StreamParams` accepts: `ultrafast`, `veryfast`, `fast`, `medium`, `slow`.

**Software (none/auto):**
```
-c:v libx264 -preset <preset>
```
Example with ultrafast: `-c:v libx264 -preset ultrafast`

**NVENC (NVIDIA):**
NVENC uses p1-p7 preset scale. Software presets are automatically mapped:
- ultrafast → p1 (fastest)
- veryfast → p2
- fast → p3
- medium → p4
- slow → p5

```
-c:v h264_nvenc -preset <mapped_preset>
```
Example with ultrafast: `-c:v h264_nvenc -preset p1`

**QSV (Intel):**
```
-c:v h264_qsv -preset <preset>
```
Example with ultrafast: `-c:v h264_qsv -preset ultrafast`

**VAAPI (AMD/Intel Linux):**
```
-c:v h264_vaapi
```
Note: VAAPI doesn't support presets in the same way

**VideoToolbox (macOS):**
```
-c:v h264_videotoolbox
```
Note: VideoToolbox doesn't support presets in the same way

### Example Commands

**1080p Software Encoding with Ultrafast Preset (24/7 Channel):**
```
ffmpeg -re -stream_loop -1 -i /media/video.mp4 \
  -c:v libx264 -preset ultrafast \
  -c:a aac -b:a 192k -ac 2 \
  -b:v 5000k -maxrate 5000k -bufsize 10000k -s 1920x1080 \
  -f hls -hls_time 2 -hls_list_size 10 \
  -hls_flags delete_segments \
  -hls_segment_filename /streams/channel1/1080p_segment_%03d.ts \
  /streams/channel1/1080p.m3u8
```
Note: 
- `-re` enables real-time pacing (1x speed)
- `-stream_loop -1` loops video infinitely for 24/7 channel
- `hls_time 2` creates 2-second segments for faster startup
- `hls_list_size 10` maintains sliding window of 10 most recent segments
- No `hls_playlist_type` allows sliding window behavior (not event/vod)

**720p with NVENC + Seeking (Fast Testing Mode):**
```
ffmpeg -ss 3600 -stream_loop -1 -i /media/video.mp4 \
  -c:v h264_nvenc -preset p1 \
  -c:a aac -b:a 192k -ac 2 \
  -b:v 3000k -maxrate 3000k -bufsize 6000k -s 1280x720 \
  -f hls -hls_time 2 -hls_list_size 10 \
  -hls_flags delete_segments \
  -hls_segment_filename /streams/channel1/720p_segment_%03d.ts \
  /streams/channel1/720p.m3u8
```
Note: 
- No `-re` flag = fastest encoding (16x+) for testing
- NVENC `p1` preset maps to software `ultrafast`
- `-ss 3600` seeks to 1 hour before looping

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

## Timeline to FFmpeg Input Conversion

Location: `internal/streaming/timeline_input.go`

### Overview

Converts a channel's virtual timeline position into FFmpeg input parameters, handling seeks, file concatenation, and playlist transitions for seamless streaming.

### Constants

```go
const (
    SeekOptimizationThreshold = 10   // Skip seeks < 10 seconds for faster startup
    ConcatThreshold           = 30   // Use concat if < 30s remaining for smooth transitions
    MaxStreamDuration         = 7200 // Max 2 hours of content
    MaxConcatFiles            = 10   // Limit concat list size
)
```

### Data Structures

#### TimelineInput

```go
type TimelineInput struct {
    PrimaryFile    string       // Main input file path
    SeekSeconds    int64        // Seek position in primary file (0 = start)
    UseConcatFile  bool         // Whether to use concat protocol
    ConcatFilePath string       // Path to generated concat.txt (if used)
    ConcatItems    []ConcatItem // Files to concatenate
    TotalDuration  int64        // Total duration to stream (seconds)
}
```

Represents the FFmpeg input configuration derived from a timeline position. Either a simple file+seek or a concat-based multi-file input for seamless transitions.

#### ConcatItem

```go
type ConcatItem struct {
    FilePath string // Absolute path to media file
    InPoint  int64  // Start time within file (seconds, 0 = start)
    OutPoint int64  // End time within file (0 = use all)
}
```

Represents a single file in an FFmpeg concat demuxer list.

### BuildTimelineInput

```go
func BuildTimelineInput(
    ctx context.Context,
    channelID uuid.UUID,
    timelineService *timeline.TimelineService,
    repos *db.Repositories,
) (*TimelineInput, error)
```

Main integration function that converts a channel's timeline position into FFmpeg input parameters.

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `channelID` - UUID of the channel to build input for
- `timelineService` - Timeline service for current position calculation
- `repos` - Database repositories for channel and playlist access

**Returns:**
- `*TimelineInput` - Complete FFmpeg input configuration
- `error` - Timeline, database, or validation errors

**Process:**
1. Gets current timeline position via Timeline Service
2. Fetches channel and playlist from database
3. Calculates remaining duration in current media item
4. Determines strategy (simple seek vs concat)
5. Validates all file paths exist
6. Returns appropriate input configuration

**Strategies:**

**Simple Input** (remaining > 30s):
- Single file with seek position
- Optimization: skips seek if offset < 10s (faster startup)
- Used for most normal playback scenarios

**Concat Input** (remaining < 30s):
- Current file from offset + next N playlist items
- Generates FFmpeg concat demuxer file
- Ensures seamless transitions between playlist items
- Handles playlist looping correctly

**Usage:**
```go
input, err := streaming.BuildTimelineInput(ctx, channelID, timelineService, repos)
if err != nil {
    return fmt.Errorf("failed to build input: %w", err)
}

if input.UseConcatFile {
    // Use concat protocol
    params := streaming.StreamParams{
        InputFile:   input.ConcatFilePath,
        SeekSeconds: 0, // Seeking handled in concat file
        // ... other params
    }
    // Remember to cleanup concat file after use
    defer os.Remove(input.ConcatFilePath)
} else {
    // Simple seek
    params := streaming.StreamParams{
        InputFile:   input.PrimaryFile,
        SeekSeconds: input.SeekSeconds,
        // ... other params
    }
}
```

### GetNextPlaylistItems

```go
func GetNextPlaylistItems(
    playlist []*models.PlaylistItem,
    currentPosition int,
    count int,
    loop bool,
) []*models.PlaylistItem
```

Returns the next N items from a playlist, handling looping behavior.

**Parameters:**
- `playlist` - Full playlist ordered by position
- `currentPosition` - Index of current item
- `count` - Number of next items to retrieve
- `loop` - Whether playlist loops

**Returns:**
- `[]*models.PlaylistItem` - Next items (may be fewer than count if non-looping)

**Behavior:**
- With looping: wraps around to start when reaching end
- Without looping: stops at last item
- Returns empty list for invalid inputs

**Usage:**
```go
nextItems := streaming.GetNextPlaylistItems(playlist, 2, 5, true)
// Returns items at positions 3, 4, 5, 6, 7 (wrapping if needed)
```

### CalculateStreamDuration

```go
func CalculateStreamDuration(
    remainingCurrent int64,
    nextItems []*models.PlaylistItem,
    maxDuration int64,
) int64
```

Calculates total streaming duration by summing remaining time in current item plus next items, capped at maximum.

**Parameters:**
- `remainingCurrent` - Seconds remaining in current item
- `nextItems` - Following playlist items
- `maxDuration` - Maximum duration cap (typically MaxStreamDuration)

**Returns:**
- `int64` - Total duration in seconds, capped at maxDuration

**Behavior:**
- Sums durations until reaching max
- Skips items with nil Media
- Used to determine how much content to prepare

**Usage:**
```go
duration := streaming.CalculateStreamDuration(600, nextItems, 7200)
// Returns min(600 + sum(nextItems.durations), 7200)
```

### BuildConcatFile

```go
func BuildConcatFile(items []ConcatItem, outputPath string) error
```

Generates an FFmpeg concat demuxer format file for seamless multi-file playback.

**Parameters:**
- `items` - List of files and time ranges to concatenate
- `outputPath` - Where to write the concat file

**Returns:**
- `error` - Write errors or validation failures

**Generated Format:**
```
file '/absolute/path/to/video1.mp4'
inpoint 120
file '/absolute/path/to/video2.mp4'
file '/absolute/path/to/video3.mp4'
```

**Features:**
- Atomic write (temp file + rename)
- Includes inpoint/outpoint directives when specified
- Paths must be absolute
- Compatible with `ffmpeg -f concat -safe 0 -i concat.txt`

**Usage:**
```go
items := []streaming.ConcatItem{
    {FilePath: "/media/video1.mp4", InPoint: 120, OutPoint: 0},
    {FilePath: "/media/video2.mp4", InPoint: 0, OutPoint: 0},
}

concatPath := filepath.Join(os.TempDir(), "concat.txt")
if err := streaming.BuildConcatFile(items, concatPath); err != nil {
    return err
}
defer os.Remove(concatPath)

// Use with FFmpeg
cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", concatPath, ...)
```

### ValidateFilePaths

```go
func ValidateFilePaths(items []ConcatItem) error
```

Validates that all file paths in concat items exist and are accessible.

**Parameters:**
- `items` - List of concat items to validate

**Returns:**
- `error` - First validation error encountered, nil if all valid

**Checks:**
- File exists
- Path is absolute
- Path is a file (not directory)
- File is readable

**Usage:**
```go
if err := streaming.ValidateFilePaths(concatItems); err != nil {
    log.Error("Invalid file paths: %v", err)
    return err
}
```

### Errors

```go
var (
    ErrInvalidOffset = errors.New("seek offset exceeds media duration")
    ErrFileNotFound  = errors.New("media file not found")
    ErrEmptyPlaylist = errors.New("playlist is empty")
)
```

### Optimization Notes

**Seek Optimization (<10s):**
- Skips seeking for positions near start of file
- Improves startup time by ~100-200ms
- Minimal impact on user experience (< 10s difference)

**Concat for Smooth Transitions (<30s remaining):**
- Prepares next files in advance
- Eliminates gaps between playlist items
- FFmpeg handles transition seamlessly
- Essential for continuous channel experience

**Duration Capping:**
- Limits prepared content to 2 hours
- Prevents excessive memory/disk usage
- Sufficient for any streaming session
- New input built as needed for continuation

### Performance

- BuildTimelineInput: < 50ms typical (includes DB queries)
- GetNextPlaylistItems: O(n) where n = count
- CalculateStreamDuration: O(n) where n = items
- BuildConcatFile: < 10ms for 10 files
- ValidateFilePaths: O(n) where n = files

### Integration Example

Complete workflow for starting a stream:

```go
// Get timeline input configuration
input, err := streaming.BuildTimelineInput(ctx, channelID, timelineService, repos)
if err != nil {
    return fmt.Errorf("timeline input failed: %w", err)
}

// Build FFmpeg command with appropriate parameters
var params streaming.StreamParams
if input.UseConcatFile {
    params = streaming.StreamParams{
        InputFile:       input.ConcatFilePath,
        OutputPath:      outputPath,
        Quality:         streaming.Quality1080p,
        HardwareAccel:   hwAccel,
        SeekSeconds:     0, // Seeking in concat file
        SegmentDuration: 6,
        PlaylistSize:    10,
    }
    // Cleanup concat file when stream stops
    defer os.Remove(input.ConcatFilePath)
} else {
    params = streaming.StreamParams{
        InputFile:       input.PrimaryFile,
        OutputPath:      outputPath,
        Quality:         streaming.Quality1080p,
        HardwareAccel:   hwAccel,
        SeekSeconds:     input.SeekSeconds,
        SegmentDuration: 6,
        PlaylistSize:    10,
    }
}

// Build and launch FFmpeg command
cmd, err := streaming.BuildHLSCommand(params)
if err != nil {
    return fmt.Errorf("failed to build command: %w", err)
}

// Execute streaming...
```

## REST Endpoints

Location: `internal/api/stream.go`

### GET /api/stream/:channel_id/master.m3u8

Serves the master playlist listing all quality variants for adaptive bitrate streaming. Automatically registers the client and starts the stream if not already active.

**Parameters:**
- `channel_id` (path) - UUID of the channel

**Response (200 OK):**
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

**Headers:**
- `Content-Type: application/vnd.apple.mpegurl`
- `Cache-Control: public, max-age=60`

**Error Responses:**
- `400 Bad Request` - Invalid channel UUID format
- `404 Not Found` - Channel not found
- `503 Service Unavailable` - Stream starting (retry in a moment) or service unavailable

**Notes:**
- First request to this endpoint starts the stream
- Increments client count in stream session
- Master playlist can be cached briefly (60 seconds)
- CORS headers handled globally by server middleware

### GET /api/stream/:channel_id/:quality

Serves quality-specific media playlist containing segment references. The quality parameter should include the .m3u8 extension (e.g., "1080p.m3u8").

**Parameters:**
- `channel_id` (path) - UUID of the channel
- `quality` (path) - Quality level with .m3u8 extension: "1080p.m3u8", "720p.m3u8", or "480p.m3u8"

**Response (200 OK):**
```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:EVENT
#EXTINF:6.0,
channel_id_1080p_segment_000.ts
#EXTINF:6.0,
channel_id_1080p_segment_001.ts
```

**Headers:**
- `Content-Type: application/vnd.apple.mpegurl`
- `Cache-Control: no-cache, no-store, must-revalidate`

**Error Responses:**
- `400 Bad Request` - Invalid channel UUID or invalid quality
- `404 Not Found` - Stream not active
- `503 Service Unavailable` - Playlist not yet generated

**Notes:**
- Updates last access time for stream
- Media playlists MUST NOT be cached (live content)
- HLS clients typically request this every few seconds
- CORS headers handled globally by server middleware

### GET /api/stream/:channel_id/:quality/:segment

Serves individual video segment files.

**Parameters:**
- `channel_id` (path) - UUID of the channel
- `quality` (path) - Quality level: "1080p", "720p", or "480p"
- `segment` (path) - Segment filename (must end with .ts)

**Response (200 OK):**
Binary video segment data

**Headers:**
- `Content-Type: video/MP2T`
- `Cache-Control: public, max-age=31536000, immutable`

**Error Responses:**
- `400 Bad Request` - Invalid parameters or directory traversal attempt
- `404 Not Found` - Stream not active or segment not found
- `500 Internal Server Error` - Stream configuration error

**Security:**
- Validates segment filename contains no directory traversal characters (.., /, \)
- Verifies resolved path is within expected directory
- Only serves .ts files
- Explicit error handling for filepath.Abs to prevent security bypass

**Notes:**
- Updates last access time for stream
- Segments can be cached permanently (immutable content)
- Filename format: `channel_id_quality_segment_NNN.ts`
- CORS headers handled globally by server middleware

### DELETE /api/stream/:channel_id/client

Explicitly unregisters a client from a stream.

**Parameters:**
- `channel_id` (path) - UUID of the channel

**Response (200 OK):**
```json
{
  "message": "Client unregistered successfully"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid channel UUID format
- `404 Not Found` - Stream not found or already stopped
- `500 Internal Server Error` - Failed to unregister

**Notes:**
- Decrements client count
- Grace period starts when client count reaches zero
- Optional endpoint - cleanup handles automatic expiration
- Stream stops after grace period (default 30s) if no clients reconnect

### Usage Example

**With curl:**
```bash
# Get master playlist (starts stream)
curl http://localhost:8080/api/stream/550e8400-e29b-41d4-a716-446655440000/master.m3u8

# Get media playlist for 1080p
curl http://localhost:8080/api/stream/550e8400-e29b-41d4-a716-446655440000/1080p.m3u8

# Get a segment
curl http://localhost:8080/api/stream/550e8400-e29b-41d4-a716-446655440000/1080p/channel_id_1080p_segment_000.ts -o segment.ts

# Unregister client
curl -X DELETE http://localhost:8080/api/stream/550e8400-e29b-41d4-a716-446655440000/client
```

**With VLC:**
```bash
vlc http://localhost:8080/api/stream/550e8400-e29b-41d4-a716-446655440000/master.m3u8
```

**With HLS.js (JavaScript):**
```javascript
const video = document.querySelector('video');
const hls = new Hls();
hls.loadSource('http://localhost:8080/api/stream/550e8400-e29b-41d4-a716-446655440000/master.m3u8');
hls.attachMedia(video);
```

### Streaming Flow

1. **Client requests master playlist** → Stream Manager starts stream if needed
2. **Client parses master playlist** → Selects quality based on bandwidth
3. **Client requests media playlist** → Gets current segment list
4. **Client downloads segments** → Plays video
5. **Client polls media playlist** → Gets updated segment list (every ~6s)
6. **Client disconnects** → Grace period starts
7. **After grace period** → Stream stops if no clients reconnected

### Performance Considerations

- **Master playlist**: Rarely changes, can cache for 60s
- **Media playlists**: Update frequently, no caching
- **Segments**: Immutable once generated, cache permanently
- **Client tracking**: Updates last access time on each request
- **Cleanup**: Automatic after 30s grace period (configurable)

### CORS

All endpoints include `Access-Control-Allow-Origin: *` header for browser-based players. For production, configure more restrictive CORS policies.

## Error Handling & Recovery

Location: `internal/streaming/errors.go`, `internal/streaming/recovery.go`, `internal/streaming/circuit_breaker.go`

### Error Classification

```go
type ErrorType int

const (
    ErrorTypeFFmpegCrash      // FFmpeg process crashed unexpectedly
    ErrorTypeFileMissing      // Input file doesn't exist
    ErrorTypeFileCorrupt      // Input file is corrupted or invalid
    ErrorTypeHardwareEncoder  // Hardware encoder failed
    ErrorTypeDiskSpace        // Insufficient disk space
    ErrorTypePlaylistEnd      // Playlist reached end (non-looping)
    ErrorTypeTimeout          // Operation timed out
)

type ErrorSeverity int

const (
    SeverityInfo       // Informational events
    SeverityWarning    // Recoverable issues
    SeverityError      // Errors that may be recoverable with retry
    SeverityCritical   // Critical errors requiring immediate action
)
```

### StreamError Type

```go
type StreamError struct {
    Type        ErrorType
    Severity    ErrorSeverity
    Message     string
    Cause       error
    Recoverable bool
}

func NewStreamError(errorType ErrorType, message string, cause error) *StreamError
func ClassifyError(err error) *StreamError
func ParseFFmpegError(stderr string) *StreamError
```

Structured error type with classification and recoverability information.

**Usage:**
```go
streamErr := streaming.NewStreamError(
    streaming.ErrorTypeFFmpegCrash,
    "FFmpeg process crashed",
    originalErr,
)

// Or classify from generic error
streamErr := streaming.ClassifyError(err)

// Parse from FFmpeg stderr
streamErr := streaming.ParseFFmpegError(stderrOutput)
```

### Error Classification Rules

**Critical Errors (Not Recoverable):**
- `ErrorTypeDiskSpace` - Stop stream, cannot continue without disk space
- `ErrorTypePlaylistEnd` (non-looping) - Expected termination

**Recoverable Errors (Automatic Retry):**
- `ErrorTypeFFmpegCrash` - Restart stream with exponential backoff
- `ErrorTypeFileMissing` - Skip to next playlist item
- `ErrorTypeFileCorrupt` - Skip to next playlist item
- `ErrorTypeHardwareEncoder` - Fallback to software encoding
- `ErrorTypeTimeout` - Retry with backoff

### Circuit Breaker Pattern

Prevents infinite restart loops by tracking consecutive failures.

```go
type CircuitBreaker struct {
    failureThreshold int           // Number of failures before opening (default: 3)
    resetTimeout     time.Duration // Time before trying again (default: 60s)
    state            CircuitState  // Current state
}

type CircuitState int

const (
    StateClosed    // Normal operation, failures counted
    StateOpen      // Blocking calls, failure threshold exceeded
    StateHalfOpen  // Testing if system recovered
)
```

**State Transitions:**
```
Closed ─(failures >= threshold)─> Open
Open ─(timeout elapsed)─> HalfOpen
HalfOpen ─(success)─> Closed
HalfOpen ─(failure)─> Open
```

**Per-Channel Circuit Breakers:**
- Each channel has its own circuit breaker
- Stored in SessionManager, keyed by channel ID
- Automatically created when needed
- Cleaned up when stream stops

**Usage:**
```go
// Get or create circuit breaker for channel
cb := sessionManager.GetOrCreateCircuitBreaker(channelID)

// Check if can attempt operation
if !cb.CanAttempt() {
    return ErrCircuitOpen
}

// Execute with circuit breaker
err := cb.Call(func() error {
    return riskyOperation()
})

// Manual recording
cb.RecordSuccess()
cb.RecordFailure()
```

### Recovery Strategies

#### Automatic Stream Restart

Used for transient FFmpeg crashes.

**Process:**
1. Check circuit breaker state (fail fast if open)
2. Check restart count (max 3 attempts)
3. Calculate exponential backoff (1s, 2s, 4s, 8s)
4. Wait for backoff period
5. Stop current stream (cleanup resources)
6. Start new stream
7. Reset errors on success, or trip circuit breaker on failure

**Configuration:**
```go
const (
    MaxRestartAttempts = 3
    InitialBackoff     = 1 * time.Second
    MaxBackoff         = 8 * time.Second
)
```

#### File Error Handling

Skip to next playlist item when current file fails.

**Process:**
1. Log file error with path
2. Fetch channel playlist from database
3. Find current position in playlist
4. Get next valid playlist items
5. Restart stream (timeline service calculates new position)
6. Return `ErrPlaylistEnded` if no more items (non-looping)

**Applicable Errors:**
- `ErrorTypeFileMissing`
- `ErrorTypeFileCorrupt`

#### Hardware Encoder Fallback

Disable hardware acceleration and use software encoding.

**Process:**
1. Detect hardware encoder failure from FFmpeg stderr
2. Log fallback event (warning level)
3. Mark hardware acceleration as failed in session
4. Update global config to `HardwareAccelNone`
5. Restart stream with software encoding
6. Configuration persists for future streams

**Trigger Patterns:**
- "Cannot load nvcuda"
- "QSV not available"
- "VAAPI failed"
- "VideoToolbox failed"

#### Disk Space Monitoring

Prevent stream startup when disk space insufficient.

**Implementation:**
```go
func checkDiskSpace(path string) error
func getAvailableSpace(path string) (uint64, error)
```

**Thresholds:**
- Minimum required: 5GB (blocks stream start)
- Warning threshold: 10GB (logs warning but allows)

**Integration Points:**
- Checked before starting stream (`StartStream`)
- Periodic check during cleanup cycle
- Triggers cleanup if space low

### Recovery Constants

```go
const (
    MinDiskSpaceBytes          = 5 * 1024 * 1024 * 1024  // 5GB
    WarnDiskSpaceBytes         = 10 * 1024 * 1024 * 1024 // 10GB
    MaxRestartAttempts         = 3
    CircuitBreakerThreshold    = 3
    CircuitBreakerResetTimeout = 60 * time.Second
    InitialBackoff             = 1 * time.Second
    MaxBackoff                 = 8 * time.Second
)
```

### Stream State Tracking

Enhanced `StreamSession` model tracks recovery state:

```go
type StreamSession struct {
    // ... existing fields ...
    ErrorCount          int    // Consecutive errors
    LastError           string // Last error message
    RestartCount        int    // Restart attempts
    HardwareAccelFailed bool   // Hardware fallback applied
}
```

**Methods:**
```go
session.IncrementErrorCount()
session.GetErrorCount()
session.ResetErrors()
session.SetLastError(err)
session.GetLastError()
session.IncrementRestartCount()
session.GetRestartCount()
session.ResetRestartCount()
session.SetHardwareAccelFailed(bool)
session.GetHardwareAccelFailed()
```

### Monitoring & Logging

All recovery events are logged with structured data:

**Example Logs:**

```go
// FFmpeg crash detected
logger.Log.Error().
    Err(err).
    Str("channel_id", channelID).
    Int("ffmpeg_pid", pid).
    Int("error_count", errorCount).
    Msg("FFmpeg process crashed unexpectedly")

// Recovery attempt
logger.Log.Info().
    Str("channel_id", channelID).
    Str("reason", "FFmpeg crash").
    Int("restart_count", restartCount).
    Dur("backoff", backoff).
    Msg("Attempting stream restart")

// Circuit breaker tripped
logger.Log.Error().
    Str("channel_id", channelID).
    Str("circuit_state", "open").
    Int("failures", failureCount).
    Msg("Circuit breaker is open, cannot restart stream")

// Hardware fallback
logger.Log.Warn().
    Str("channel_id", channelID).
    Str("previous_hw_accel", "nvenc").
    Msg("Hardware encoder failed, falling back to software encoding")
```

**Log Levels:**
- `Debug`: Recovery attempts, state transitions, circuit breaker checks
- `Info`: Successful recovery, circuit breaker state changes
- `Warn`: Hardware fallback, file skipping, disk space warnings
- `Error`: Failed recovery, critical errors, circuit breaker trips

### Error Codes

**Manager Errors:**
```go
var (
    ErrStreamNotFound         = errors.New("stream not found")
    ErrManagerStopped         = errors.New("stream manager has been stopped")
    ErrCircuitOpen            = errors.New("circuit breaker is open")
    ErrInsufficientDiskSpace  = StreamError // Critical severity
    ErrPlaylistEnded          = StreamError // Info severity
)
```

### Recovery Workflow Example

Complete workflow for handling FFmpeg crash:

```go
// 1. Process monitor detects crash
err := execCmd.Wait()

// 2. Classify error
streamErr := streaming.ClassifyError(err)

// 3. Update session state
session.SetState(streaming.StateFailed.String())
session.IncrementErrorCount()
session.SetLastError(err)

// 4. Attempt recovery
if recoveryErr := manager.attemptRecovery(ctx, channelID, streamErr); recoveryErr != nil {
    // Recovery failed - log and mark as failed
    logger.Log.Error().
        Err(recoveryErr).
        Str("channel_id", channelID).
        Msg("Failed to recover from FFmpeg crash")
    session.SetState(streaming.StateFailed.String())
}

// 5. Recovery routes to appropriate handler based on error type
switch streamErr.Type {
case ErrorTypeFFmpegCrash:
    return manager.restartStream(ctx, channelID, "FFmpeg crash")
case ErrorTypeHardwareEncoder:
    return manager.fallbackToSoftwareEncoding(ctx, channelID)
case ErrorTypeFileMissing, ErrorTypeFileCorrupt:
    return manager.handleFileError(ctx, channelID, filePath, streamErr.Type)
// ... other error types
}

// 6. Restart process
// - Check circuit breaker
// - Calculate backoff
// - Stop current stream
// - Start new stream
// - Reset errors on success
```

### Testing Recovery

**Unit Tests:** (`errors_test.go`, `circuit_breaker_test.go`, `recovery_test.go`)
- Error classification logic
- Circuit breaker state transitions
- Backoff duration calculation
- Disk space checks

**Integration Tests:** (task 6-9)
- FFmpeg crash and restart
- Missing file handling
- Circuit breaker behavior under load
- Hardware fallback scenarios

### Best Practices

1. **Always check disk space before starting streams**
2. **Monitor circuit breaker state to detect patterns**
3. **Log all recovery attempts with context**
4. **Set reasonable backoff intervals to avoid overwhelming system**
5. **Test recovery paths with real failure scenarios**
6. **Document error patterns for operations team**
7. **Alert on circuit breaker trips (production)**
8. **Track recovery success rate metrics**

## Service Interfaces

To be defined during implementation.

