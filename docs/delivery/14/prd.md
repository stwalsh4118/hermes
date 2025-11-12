# PBI-14: Custom HLS Playlist Writer/Reader

[View in Backlog](../backlog.md#user-content-14)

## Overview

Replace the buggy `hls-m3u8` library with a custom, reliable HLS playlist writer/reader implementation that we fully control. This will eliminate the index out of range errors and other library bugs we've been experiencing, providing a simple, maintainable solution.

## Problem Statement

The current `hls-m3u8` library has several critical issues:
- Index out of range panics when encoding playlists
- Internal state inconsistencies between `Count()` and array length
- Complex workarounds needed for sliding window functionality
- Lack of control over internal implementation details
- Difficult to debug and maintain

The HLS playlist format is simple text-based (RFC 8216), making it feasible to implement our own reliable solution that we can fully understand and control.

## User Stories

**As a developer, I want:**
- A reliable playlist writer that doesn't panic
- Full control over playlist generation logic
- Simple, maintainable code we can debug
- Proper sliding window implementation
- Atomic file writes for consistency

**As a viewer, I want:**
- Stable HLS playback without interruptions
- Consistent segment availability
- Proper media sequence numbering

## Technical Approach

### Implementation Strategy

1. **Replace existing `playlistManager`** in `api/internal/streaming/playlist/manager.go`
   - Remove dependency on `hls-m3u8` library
   - Use simple Go data structures (slice of segments)
   - Manual sliding window logic with explicit control

2. **Core Data Structure:**
```go
type playlistManager struct {
    mu                  sync.RWMutex
    segments            []SegmentMeta  // Simple slice, we control pruning
    outputPath          string
    segmentDir          string
    windowSize          uint
    maxDuration         float64
    mediaSequence       uint64  // Track manually, increments on prune
    totalSegments       uint64  // Track total segments ever added
    discontinuityNext   bool
    lastSuccessfulWrite *time.Time
}
```

3. **Improved Manager Interface:**
   - Add `GetMediaSequence()` and `GetSegmentCount()` methods for better observability
   - Keep existing interface methods for compatibility
   - Clear separation of concerns

4. **Playlist Format Generation:**
   - Write m3u8 format directly as text using `strings.Builder`
   - Format: `#EXTM3U`, `#EXT-X-VERSION:3`, `#EXT-X-MEDIA-SEQUENCE`, `#EXT-X-TARGETDURATION`, `#EXTINF`, `#EXT-X-PROGRAM-DATE-TIME`, `#EXT-X-DISCONTINUITY`
   - Proper formatting: durations with 3 decimal places, RFC3339 timestamps

5. **Sliding Window:**
   - When `len(segments) >= windowSize`, remove from front: `segments = segments[1:]`
   - Increment `mediaSequence` when pruning
   - Return list of removed segment URIs for file deletion
   - Clean and predictable behavior

6. **Error Handling:**
   - Return errors instead of panicking
   - Validate inputs (empty URI, invalid duration, etc.)
   - Log warnings for non-fatal issues (file deletion failures)

## UX/UI Considerations

No UI changes required. This is a backend implementation change that maintains API compatibility.

## Acceptance Criteria

- [ ] New `playlistManager` implements improved `Manager` interface
- [ ] Sliding window works correctly (prunes old segments, maintains windowSize exactly)
- [ ] Atomic writes prevent partial reads (temp file + rename pattern)
- [ ] Media sequence increments correctly when segments pruned
- [ ] Segment files deleted when removed from playlist
- [ ] ProgramDateTime and Discontinuity tags handled correctly
- [ ] No panics or index out of range errors
- [ ] Proper error handling (no panics, return errors instead)
- [ ] Thread-safe (concurrent AddSegment/Write calls)
- [ ] Unit tests with >80% coverage
- [ ] Integration test with real HLS playback (HLS.js can play the playlist)
- [ ] Performance comparable or better than library version
- [ ] All existing streaming functionality works without changes
- [ ] `hls-m3u8` library dependency removed from go.mod

## Dependencies

- PBI-13 (current streaming implementation) - will replace playlist manager from this PBI

## Open Questions

- Should we also implement playlist reading/parsing, or just writing? (Starting with writing only)
- Do we need to support all HLS tags, or just the ones we currently use? (Starting with current tags)
- Should we keep the library as fallback, or remove it completely? (Remove completely)

## Related Tasks

Tasks for this PBI will be tracked in `docs/delivery/14/tasks.md` once defined.

