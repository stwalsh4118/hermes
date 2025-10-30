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

## Planned Features

- HLS stream generation from timeline position
- FFmpeg transcoding pipeline (H.264 + AAC)
- Adaptive bitrate streams (1080p, 720p, 480p)
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

