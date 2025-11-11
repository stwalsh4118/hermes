# PBI-13: Reliable HLS live (4s TS) with Go-managed playlists

[View in Backlog](../backlog.md#user-content-13)

## Overview
Improve the reliability of live HLS by decoupling segment production from playlist generation. Use FFmpeg only to emit 4-second MPEG-TS segments, and manage a robust sliding-window media playlist in Go. This gives full control over window size, atomic updates, and recovery behavior.

## Problem Statement
The current usage of FFmpeg’s built-in HLS muxer is not consistently generating segments or maintaining the sliding window as desired. This causes playlist instability and playback issues during live streaming.

## User Stories
- As a viewer, I want reliable live playback with consistent 4s segments and a stable sliding window so that playback is smooth and resilient.
- As an operator, I want deterministic playlist updates and cleanup so that the system is predictable and easy to monitor.

## Technical Approach
- Segments: FFmpeg `stream_segment` producing 4s MPEG-TS files (no playlist output).
- Keyframe alignment: GOP length = `fps * 4`, `-sc_threshold 0`, and `-force_key_frames "expr:gte(t,n_forced*4)"`.
- Playlist: A Go service maintains a fixed-size sliding window and uses `hls-m3u8` to render the media playlist.
- Atomicity: Write playlist to a temp file and rename to avoid partial reads.
- Cleanup: Delete segments older than `(window + safety buffer)` to keep storage bounded.
- Discontinuity handling: Insert `#EXT-X-DISCONTINUITY` when encoder restarts or timestamp regressions are detected.
- Observability: Log segment cadence, drift, and last playlist update time.

## UX/UI Considerations
- No UI changes required in this PBI. Player compatibility maintained via standard HLS with TS segments. Optional status/metrics may be surfaced later.

## Acceptance Criteria
- Generate ~4s TS segments with aligned keyframes and minimal drift.
- Playlist shows a sliding window of N segments (configurable) and updates atomically on each new segment.
- `#EXT-X-TARGETDURATION` equals `ceil(max observed segment duration)`.
- `#EXTINF` accurate within ±100ms (or constant 4.0s if strictly enforced).
- `#EXT-X-MEDIA-SEQUENCE` increases monotonically; segments older than `(window + safety buffer)` are removed from disk.
- On encoder restarts, playlist includes `#EXT-X-DISCONTINUITY` and playback recovers automatically.
- Playback can run ≥5 minutes in a standard HLS player without stalls.

## Dependencies
- FFmpeg (stream_segment, H.264/AAC).
- Go library: `github.com/Eyevinn/hls-m3u8/m3u8` for playlist generation.
- Optional: `fsnotify` (or equivalent) for file watching.

## Open Questions
- Exact window size and safety buffer defaults (e.g., window=6, safety=2).
- Whether to calculate precise `#EXTINF` via probe vs. constant target.
- Future ABR support and master playlist management.

## Related Tasks
- Tasks for this PBI will be tracked in `docs/delivery/13/tasks.md` once defined.


