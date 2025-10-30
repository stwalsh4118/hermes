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

## Planned Features

- Stream lifecycle management
- Multi-client support with stream sharing
- Stream cleanup and resource management

## REST Endpoints

To be defined during implementation.

## Service Interfaces

To be defined during implementation.

## Data Contracts

To be defined during implementation.

## Stream Session (In-Memory)

The `StreamSession` model (not persisted to database) will be defined during implementation.

