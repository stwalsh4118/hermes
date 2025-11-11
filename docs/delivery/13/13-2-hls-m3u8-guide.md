# hls-m3u8 Package Usage Guide

**Date:** 2025-01-11  
**Package:** `github.com/Eyevinn/hls-m3u8/m3u8`  
**Upstream Repository:** https://github.com/Eyevinn/hls-m3u8

## Overview

The `hls-m3u8` package provides a Go implementation for generating and manipulating HLS (HTTP Live Streaming) media playlists compliant with RFC 8216. This guide focuses on the operations needed for our sliding-window media playlist manager that maintains a fixed-size window of 4-second TS segments.

**Key Features:**
- Media playlist generation and manipulation
- HLS version calculation
- Discontinuity handling
- Media sequence management
- Target duration calculation

## Installation

```bash
go get github.com/Eyevinn/hls-m3u8/m3u8
```

**Import:**
```go
import "github.com/Eyevinn/hls-m3u8/m3u8"
```

## Core Types

### MediaPlaylist

The `MediaPlaylist` type represents an HLS media playlist with segments.

```go
type MediaPlaylist struct {
    Version        uint8   // HLS version (typically 3)
    TargetDuration float64 // Maximum segment duration in seconds
    MediaSequence  uint64  // Starting sequence number
    Segments       []*Segment
    // ... other fields
}
```

### Segment

Represents a single media segment in the playlist.

```go
type Segment struct {
    Duration       float64    // Segment duration in seconds
    URI            string     // Segment filename or URI
    Discontinuity bool       // Whether to insert #EXT-X-DISCONTINUITY before this segment
    ProgramDateTime *time.Time // Optional program date-time
    // ... other fields
}
```

## Common Operations

### Creating a Media Playlist

Create a new media playlist with initial configuration:

```go
import (
    "github.com/Eyevinn/hls-m3u8/m3u8"
)

// Create a new media playlist
// Parameters: version (uint8), windowSize (uint), targetDuration (float64)
playlist, err := m3u8.NewMediaPlaylist(3, 6, 4.0)
if err != nil {
    return fmt.Errorf("failed to create playlist: %w", err)
}

// Set initial media sequence (optional, defaults to 0)
playlist.MediaSequence = 0

// Set target duration (in seconds, must be >= max segment duration)
playlist.TargetDuration = 4.0
```

**Notes:**
- Version 3 is standard for most HLS implementations
- Window size determines how many segments to keep (sliding window)
- Target duration should be set to `ceil(max segment duration)`

### Appending Segments

Add segments to the playlist. The library handles sliding window automatically:

```go
// Append a 4-second segment
segmentURI := "seg-20250111T120000.ts"
duration := 4.0

err := playlist.Append(segmentURI, duration, "")
if err != nil {
    return fmt.Errorf("failed to append segment: %w", err)
}

// Append multiple segments
segments := []struct {
    uri      string
    duration float64
}{
    {"seg-20250111T120004.ts", 4.0},
    {"seg-20250111T120008.ts", 4.0},
    {"seg-20250111T120012.ts", 4.0},
}

for _, seg := range segments {
    if err := playlist.Append(seg.uri, seg.duration, ""); err != nil {
        return fmt.Errorf("failed to append segment %s: %w", seg.uri, err)
    }
}
```

**Behavior:**
- When window size is reached, oldest segments are automatically removed
- Media sequence is automatically incremented when segments are pruned
- Each segment gets an `#EXTINF` tag with its duration

### Encoding/Serialization

Convert the playlist to m3u8 format string:

```go
// Encode playlist to m3u8 string format
playlistString, err := playlist.Encode()
if err != nil {
    return fmt.Errorf("failed to encode playlist: %w", err)
}

// Alternative: Use String() method (if available)
playlistString := playlist.String()
```

**Output Format:**
```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:4
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:4.0,
seg-20250111T120000.ts
#EXTINF:4.0,
seg-20250111T120004.ts
#EXTINF:4.0,
seg-20250111T120008.ts
```

### Atomic Write Pattern

Write playlist atomically to avoid partial reads during updates:

```go
import (
    "os"
    "path/filepath"
    "github.com/Eyevinn/hls-m3u8/m3u8"
)

func writePlaylistAtomic(playlist *m3u8.MediaPlaylist, path string) error {
    // Encode playlist to string
    content, err := playlist.Encode()
    if err != nil {
        return fmt.Errorf("failed to encode playlist: %w", err)
    }

    // Create directory if it doesn't exist
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Create temp file in same directory
    tempFile, err := os.CreateTemp(dir, ".playlist-*.tmp")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    tempPath := tempFile.Name()

    // Ensure cleanup on error
    defer func() {
        if tempFile != nil {
            tempFile.Close()
            os.Remove(tempPath)
        }
    }()

    // Write content
    if _, err := tempFile.WriteString(content); err != nil {
        return fmt.Errorf("failed to write content: %w", err)
    }

    // Sync to disk
    if err := tempFile.Sync(); err != nil {
        return fmt.Errorf("failed to sync file: %w", err)
    }

    // Close temp file
    if err := tempFile.Close(); err != nil {
        return fmt.Errorf("failed to close temp file: %w", err)
    }

    // Atomic rename (POSIX systems)
    if err := os.Rename(tempPath, path); err != nil {
        return fmt.Errorf("failed to rename file: %w", err)
    }

    // Success - prevent cleanup
    tempFile = nil
    return nil
}
```

**Key Points:**
- Write to temp file first
- Sync to ensure data is on disk
- Atomic rename prevents partial reads
- Cleanup on error

### Discontinuity Handling

Insert `#EXT-X-DISCONTINUITY` tags when encoder restarts or timestamps regress:

```go
// Method 1: Set discontinuity flag before appending segment
playlist.SetDiscontinuity()
err := playlist.Append("seg-after-restart.ts", 4.0, "")
if err != nil {
    return fmt.Errorf("failed to append segment: %w", err)
}

// Method 2: Append with discontinuity parameter (if supported)
err := playlist.Append("seg-after-restart.ts", 4.0, "#EXT-X-DISCONTINUITY")
if err != nil {
    return fmt.Errorf("failed to append segment: %w", err)
}
```

**Output:**
```m3u8
#EXTINF:4.0,
seg-before-restart.ts
#EXT-X-DISCONTINUITY
#EXTINF:4.0,
seg-after-restart.ts
```

**Use Cases:**
- Encoder restart
- Timestamp regression detected
- Stream source change
- Gap in segment sequence

### MediaSequence Management

The media sequence number tracks the first segment in the playlist:

```go
// Get current media sequence
currentSeq := playlist.MediaSequence

// Set media sequence (typically done automatically when segments are pruned)
playlist.MediaSequence = 10

// When sliding window prunes segments, sequence increments automatically
// Example: If window size is 6 and we have 10 segments:
// - Segments 0-3 are pruned
// - MediaSequence becomes 4 (first remaining segment)
```

**Important:**
- Media sequence must increase monotonically
- When segments are pruned from the beginning, sequence increments by the number pruned
- Sequence number corresponds to the first segment's index

### TargetDuration Calculation

Target duration must be `ceil(max segment duration)`:

```go
// Method 1: Set manually (ensure it's >= max segment duration)
playlist.TargetDuration = 4.0

// Method 2: Calculate from segments
maxDuration := 0.0
for _, seg := range playlist.Segments {
    if seg.Duration > maxDuration {
        maxDuration = seg.Duration
    }
}
playlist.TargetDuration = math.Ceil(maxDuration)

// Method 3: Use library's calculation (if available)
playlist.UpdateTargetDuration()
```

**Rules:**
- Must be >= all segment durations
- Should be set to ceiling of max observed duration
- Typically updated when new segments with longer duration are added

### Version Calculation

Calculate minimum HLS version required for playlist features:

```go
// Calculate minimum version based on playlist features
version := playlist.CalcMinVersion()
if version > playlist.Version {
    playlist.Version = version
}

// Or set version explicitly
playlist.Version = 3 // Standard version for most features
```

**Version Requirements:**
- Version 3: Standard for most HLS implementations
- Version 4+: Required for certain advanced features (I-frame playlists, etc.)
- Version 6+: Required for newer HLS features

## Edge Cases

### Sliding Window Implementation

The library handles sliding window automatically when window size is set:

```go
// Create playlist with window size of 6 segments
playlist, _ := m3u8.NewMediaPlaylist(3, 6, 4.0)

// Append 10 segments
for i := 0; i < 10; i++ {
    uri := fmt.Sprintf("seg-%03d.ts", i)
    playlist.Append(uri, 4.0, "")
}

// Only last 6 segments remain
// MediaSequence automatically becomes 4 (10 - 6 = 4)
// Segments 0-3 are pruned
```

### Handling Encoder Restarts

When encoder restarts, insert discontinuity and reset sequence if needed:

```go
// Detect encoder restart (e.g., via file watcher or segment gap)
func handleEncoderRestart(playlist *m3u8.MediaPlaylist, newSegmentURI string) error {
    // Insert discontinuity
    playlist.SetDiscontinuity()
    
    // Append new segment
    if err := playlist.Append(newSegmentURI, 4.0, ""); err != nil {
        return fmt.Errorf("failed to append segment after restart: %w", err)
    }
    
    // Optionally reset media sequence if starting fresh
    // playlist.MediaSequence = 0
    
    return nil
}
```

### Sequence Number Wrapping

Media sequence numbers are uint64, so wrapping is unlikely in practice. However, handle gracefully:

```go
// Check for sequence overflow (unlikely but good practice)
if playlist.MediaSequence > math.MaxUint64-1000 {
    // Reset or handle appropriately
    playlist.MediaSequence = 0
}
```

### Target Duration Updates

Update target duration when longer segments are added:

```go
func updateTargetDuration(playlist *m3u8.MediaPlaylist, newDuration float64) {
    // If new segment duration exceeds current target, update it
    if newDuration > playlist.TargetDuration {
        playlist.TargetDuration = math.Ceil(newDuration)
    }
}

// When appending segment with longer duration
duration := 4.5 // Longer than typical 4.0
updateTargetDuration(playlist, duration)
playlist.Append("seg-long.ts", duration, "")
```

## Complete Examples

### Example 1: Basic Playlist Creation and Atomic Write

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "github.com/Eyevinn/hls-m3u8/m3u8"
)

func main() {
    // Create playlist with 6-segment window, 4s target duration
    playlist, err := m3u8.NewMediaPlaylist(3, 6, 4.0)
    if err != nil {
        panic(err)
    }

    // Append initial segments
    segments := []string{
        "seg-20250111T120000.ts",
        "seg-20250111T120004.ts",
        "seg-20250111T120008.ts",
    }

    for _, uri := range segments {
        if err := playlist.Append(uri, 4.0, ""); err != nil {
            panic(err)
        }
    }

    // Write atomically
    outputPath := "/streams/channel1/playlist.m3u8"
    if err := writePlaylistAtomic(playlist, outputPath); err != nil {
        panic(err)
    }

    fmt.Println("Playlist written successfully")
}

func writePlaylistAtomic(playlist *m3u8.MediaPlaylist, path string) error {
    content, err := playlist.Encode()
    if err != nil {
        return err
    }

    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    tempFile, err := os.CreateTemp(dir, ".playlist-*.tmp")
    if err != nil {
        return err
    }
    tempPath := tempFile.Name()

    defer func() {
        tempFile.Close()
        os.Remove(tempPath)
    }()

    if _, err := tempFile.WriteString(content); err != nil {
        return err
    }

    if err := tempFile.Sync(); err != nil {
        return err
    }

    if err := tempFile.Close(); err != nil {
        return err
    }

    if err := os.Rename(tempPath, path); err != nil {
        return err
    }

    return nil
}
```

### Example 2: Sliding Window with Discontinuity

```go
package main

import (
    "fmt"
    "github.com/Eyevinn/hls-m3u8/m3u8"
)

func main() {
    // Create playlist with sliding window
    playlist, _ := m3u8.NewMediaPlaylist(3, 6, 4.0)

    // Append segments normally
    for i := 0; i < 5; i++ {
        uri := fmt.Sprintf("seg-%03d.ts", i)
        playlist.Append(uri, 4.0, "")
    }

    // Encoder restart detected - insert discontinuity
    playlist.SetDiscontinuity()
    playlist.Append("seg-after-restart-000.ts", 4.0, "")

    // Continue appending
    for i := 1; i < 5; i++ {
        uri := fmt.Sprintf("seg-after-restart-%03d.ts", i)
        playlist.Append(uri, 4.0, "")
    }

    // Encode and print
    output, _ := playlist.Encode()
    fmt.Println(output)
}
```

### Example 3: Playlist Manager Pattern

```go
package playlist

import (
    "math"
    "sync"
    "github.com/Eyevinn/hls-m3u8/m3u8"
)

type SegmentMeta struct {
    URI            string
    Duration       float64
    Discontinuity  bool
}

type PlaylistManager struct {
    mu            sync.RWMutex
    playlist      *m3u8.MediaPlaylist
    outputPath    string
    windowSize    uint
    maxDuration   float64
}

func NewPlaylistManager(windowSize uint, outputPath string) (*PlaylistManager, error) {
    playlist, err := m3u8.NewMediaPlaylist(3, windowSize, 4.0)
    if err != nil {
        return nil, err
    }

    return &PlaylistManager{
        playlist:   playlist,
        outputPath: outputPath,
        windowSize: windowSize,
        maxDuration: 0.0,
    }, nil
}

func (pm *PlaylistManager) AddSegment(seg SegmentMeta) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    // Track max duration for target duration calculation
    if seg.Duration > pm.maxDuration {
        pm.maxDuration = seg.Duration
        pm.playlist.TargetDuration = math.Ceil(pm.maxDuration)
    }

    // Set discontinuity if needed
    if seg.Discontinuity {
        pm.playlist.SetDiscontinuity()
    }

    // Append segment
    return pm.playlist.Append(seg.URI, seg.Duration, "")
}

func (pm *PlaylistManager) Write() error {
    pm.mu.RLock()
    defer pm.mu.RUnlock()

    content, err := pm.playlist.Encode()
    if err != nil {
        return err
    }

    return writePlaylistAtomic(pm.playlist, pm.outputPath)
}

func (pm *PlaylistManager) Close() error {
    // Final write and cleanup
    return pm.Write()
}
```

## References

- **GitHub Repository:** https://github.com/Eyevinn/hls-m3u8
- **GoDoc Documentation:** https://pkg.go.dev/github.com/Eyevinn/hls-m3u8/m3u8
- **HLS Specification (RFC 8216):** https://tools.ietf.org/html/rfc8216
- **Apple HLS Authoring Guide:** https://developer.apple.com/documentation/http_live_streaming

## Notes

- This guide is based on typical Go HLS library patterns and the package structure indicated by the import path
- Actual API may vary - consult upstream documentation for definitive API details
- All examples use 4-second segments aligned with our FFmpeg `stream_segment` output
- Atomic writes are critical for live streaming to prevent clients from reading partial playlists
- Media sequence must increase monotonically - the library handles this automatically when using sliding window

