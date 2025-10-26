# PBI-6: Streaming Engine

[View in Backlog](../backlog.md#user-content-6)

## Overview

Implement the core streaming engine that generates HLS (HTTP Live Streaming) video streams from the virtual timeline position, manages FFmpeg transcoding processes, supports hardware acceleration, and handles multi-client scenarios with stream lifecycle management.

## Problem Statement

This is the most technically complex component of the system. Users need to:
- Start watching a channel immediately from its current position
- Stream smoothly without buffering
- Have synchronized playback across multiple devices
- Get the best video quality their device can handle (adaptive bitrate)
- Have the system efficiently use hardware acceleration when available

The system must:
- Convert the virtual timeline position into an HLS stream starting at the correct point
- Transcode video to web-compatible formats (H.264 + AAC)
- Generate multiple quality levels (1080p, 720p, 480p)
- Share transcoding processes across clients watching the same channel
- Clean up resources when clients disconnect
- Recover gracefully from FFmpeg crashes
- Maintain synchronization across clients (±2 seconds)

## User Stories

**As a user, I want to:**
- Start streaming within 10 seconds of clicking play
- Have smooth playback without buffering (on stable connection)
- Watch on any device (phone, tablet, computer, TV)
- Have video quality automatically adapt to my connection
- See the exact same content on all my devices simultaneously

**As a system administrator, I want to:**
- Use hardware acceleration when available to save CPU
- Support multiple concurrent streams without overloading the server
- Have streams automatically clean up to prevent resource leaks
- Monitor active streams and resource usage

## Technical Approach

### Components to Implement

1. **Stream Manager**
   - Track active streams per channel
   - Coordinate client connections
   - Manage FFmpeg processes
   - Handle stream lifecycle (start, keep-alive, cleanup)

2. **HLS Generator**
   - Generate master playlist (.m3u8)
   - Generate media playlists for each quality level
   - Generate video segments (.ts files)
   - Manage segment cleanup

3. **FFmpeg Integration**
   - Calculate starting position from timeline
   - Build FFmpeg command with appropriate parameters
   - Support hardware acceleration (NVENC, QSV, VAAPI, VideoToolbox)
   - Handle multi-input playlists (concat protocol)
   - Generate HLS segments in real-time

4. **Stream Endpoints**
   - `GET /stream/:channel_id/master.m3u8` - Master playlist
   - `GET /stream/:channel_id/:quality.m3u8` - Media playlist (1080p, 720p, 480p)
   - `GET /stream/:channel_id/:segment.ts` - Video segment

### FFmpeg Pipeline

**Basic Command Structure:**
```bash
ffmpeg -ss [START_TIME] \
       -hwaccel [auto|nvenc|qsv|vaapi|videotoolbox] \
       -i [INPUT_FILE] \
       -c:v libx264 \
       -preset veryfast \
       -b:v 5000k \
       -maxrate 5000k \
       -bufsize 10000k \
       -c:a aac \
       -b:a 192k \
       -f hls \
       -hls_time 6 \
       -hls_list_size 10 \
       -hls_flags delete_segments \
       -hls_segment_filename 'segment_%03d.ts' \
       output.m3u8
```

**Hardware Acceleration Options:**
- **NVENC** (NVIDIA): `-hwaccel cuda -c:v h264_nvenc`
- **QSV** (Intel): `-hwaccel qsv -c:v h264_qsv`
- **VAAPI** (AMD/Intel Linux): `-hwaccel vaapi -c:v h264_vaapi`
- **VideoToolbox** (macOS): `-hwaccel videotoolbox -c:v h264_videotoolbox`

**Quality Presets:**
- **1080p**: 5000k video, 192k audio
- **720p**: 3000k video, 128k audio
- **480p**: 1500k video, 128k audio

### Stream Lifecycle

1. **Client Requests Stream:**
   - Check if stream exists for channel
   - If exists: increment client count, return existing stream
   - If not: calculate timeline position, start FFmpeg, return new stream

2. **Client Disconnects:**
   - Decrement client count
   - If count > 0: keep stream alive
   - If count = 0: start 30-second grace timer
   - If no reconnect: stop FFmpeg, cleanup segments

3. **Stream Sync:**
   - Use server timestamps in playlists
   - Clients request latest segments
   - Segment duration: 6 seconds (balance between latency and efficiency)

### Error Handling

- FFmpeg crash: attempt restart, notify clients
- Invalid input file: skip to next playlist item
- Hardware encoder failure: fallback to software
- Disk space: cleanup old segments aggressively

## UX/UI Considerations

From user perspective:
- Stream should start quickly (< 10 seconds)
- Playback should be smooth (no buffering on stable connection)
- Quality switching should be seamless
- Error messages should be helpful ("Video unavailable, skipping to next program")

## Acceptance Criteria

- [ ] HLS stream generation from current timeline position works correctly
- [ ] FFmpeg transcoding pipeline functional (H.264 + AAC output)
- [ ] Adaptive bitrate streams (1080p, 720p, 480p) generated correctly
- [ ] Hardware acceleration detection implemented for NVENC, QSV, VAAPI, VideoToolbox
- [ ] Hardware acceleration can be selected via configuration
- [ ] Fallback to software encoding when hardware unavailable
- [ ] Stream lifecycle management (start, stop, cleanup) working correctly
- [ ] Multi-client support: clients watching same channel share transcoding process
- [ ] Client count tracking per stream accurate
- [ ] Stream cleanup when last client disconnects (30-second grace period)
- [ ] Segment file cleanup prevents disk space exhaustion
- [ ] Synchronization across clients within ±2 seconds verified
- [ ] Stream startup time < 10 seconds
- [ ] System supports 5+ concurrent streams on test hardware
- [ ] Graceful error handling for:
  - FFmpeg crashes (restart attempt)
  - Missing/corrupted video files (skip to next)
  - Hardware encoder failures (fallback)
- [ ] Master playlist correctly lists all quality variants
- [ ] Media playlists update with new segments in real-time
- [ ] Segment files accessible via HTTP
- [ ] Integration tests with real video files
- [ ] Load tests demonstrating concurrent stream handling
- [ ] Resource cleanup verified (no zombie FFmpeg processes)

## Dependencies

**PBI Dependencies:**
- PBI-1: Project Setup & Database Foundation (REQUIRED)
- PBI-3: Channel Management Backend (REQUIRED - needs channel data)
- PBI-4: Virtual Timeline Calculation (REQUIRED - determines stream start position)

**External Dependencies:**
- FFmpeg 4.4+ installed with required codecs
- FFprobe for media analysis
- Hardware drivers for GPU encoding (optional)

**System Resources:**
- Sufficient CPU for software encoding
- Sufficient GPU for hardware encoding
- Disk space for segment storage (minimum 5GB recommended)
- Memory for multiple FFmpeg processes

## Open Questions

- What's the optimal segment duration? (6 seconds recommended)
- Should we support multiple quality presets configurable by user?
- How many segments should we keep in the playlist? (10 recommended)
- Should we support different audio bitrates?
- Do we need to support subtitle tracks?
- Should we implement adaptive bitrate ladder logic or use fixed presets?
- How should we handle very short media items (< 6 seconds)?
- Should we pre-generate thumbnails for seeking?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.

