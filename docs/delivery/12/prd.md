# PBI-12: Just-in-Time Segment Generation

[View in Backlog](../backlog.md#user-content-12)

## Overview

Implement just-in-time batch segment generation for the streaming system, replacing the current continuous generation approach with an on-demand model that generates segments only as viewers need them. This eliminates the need for slow real-time encoding while avoiding wasteful generation of unwatched content, significantly reducing resource usage and eliminating end-of-video restart disruptions.

## Problem Statement

The current streaming system has two operating modes, neither of which is satisfactory:

**Real-time Mode (`-re` flag enabled)**:
- Encodes at 1x speed (real-time pacing)
- Results in slow stream startup (10+ seconds)
- Segments become available slowly, causing initial buffering
- Wastes resources staying at 1x when hardware can encode faster

**Fast Mode (no `-re` flag)**:
- Encodes at ~20x speed with hardware acceleration
- Generates all segments rapidly, then reaches end of video
- When reaching the end, FFmpeg restarts the entire stream with `-stream_loop -1`
- Generates hundreds of segments that may never be watched
- Wastes significant CPU/GPU resources and disk I/O
- Stream restart causes potential disruption for connected clients

Both approaches are inefficient:
- Real-time mode is too slow for good user experience
- Fast mode wastes 80%+ of encoding effort on unwatched content
- Neither approach provides optimal balance of startup speed and resource efficiency

Users need streams that:
- Start quickly (under 10 seconds)
- Stay ahead of viewer position without wasting resources
- Handle end-of-video transitions smoothly without restarts
- Scale efficiently with multiple channels

## User Stories

**As a user, I want:**
- Streams to start playing quickly without long waits
- Smooth playback without interruptions when videos loop
- The system to use resources efficiently so I can run more channels
- Consistent performance regardless of video length

**As a system operator, I want:**
- Encoding resources used only for segments that will be watched
- Automatic segment generation triggered by viewer position
- No manual intervention when streams reach end of content
- Predictable resource usage patterns for capacity planning

**As the system, I need to:**
- Track where viewers are in the stream
- Generate segments just ahead of viewer position
- Stop generating when sufficient buffer exists
- Resume generation when viewers approach buffer end
- Handle video transitions at batch boundaries seamlessly

## Technical Approach

### Architecture Overview

Replace continuous FFmpeg execution with a batch-based generation system:

1. **Batch-Based Generation**: FFmpeg generates N segments, then exits cleanly
2. **Position Tracking**: Clients report current playback position to server
3. **Batch Coordinator**: Monitors positions and triggers next batch generation
4. **Seamless Continuation**: New batches start exactly where previous batch ended

```
Current State:                    New State:
┌──────────────┐                 ┌──────────────┐
│   FFmpeg     │                 │  FFmpeg      │
│  (infinite)  │                 │  (Batch 1)   │
│ -stream_loop │                 │  20 segments │
│     -1       │                 │  then EXIT   │
└──────────────┘                 └──────────────┘
       │                                │
       │ Generates forever              │ Generates batch
       │ until killed                   │ then stops
       ↓                                ↓
  All segments                   Only needed segments
                                        
                                 Client position → Threshold
                                        ↓
                                 ┌──────────────┐
                                 │  FFmpeg      │
                                 │  (Batch 2)   │
                                 │  20 segments │
                                 └──────────────┘
```

### 1. Batch-Based FFmpeg Execution

**Current Behavior**:
- FFmpeg runs with `-stream_loop -1` (infinite loop)
- Generates segments continuously until process killed
- Restarts video automatically when reaching end

**New Behavior**:
- Remove `-stream_loop -1` flag
- Add segment count limiting (generate exactly N segments)
- FFmpeg exits cleanly after batch completes
- Use `-ss` (seek) flag for starting position

**Implementation in `api/internal/streaming/ffmpeg.go`**:
```go
type StreamParams struct {
    // ... existing fields ...
    BatchMode      bool   // Enable batch generation mode
    BatchSize      int    // Number of segments to generate in batch
    SeekSeconds    int64  // Starting position for this batch
}

func buildInputArgs(params StreamParams) []string {
    args := []string{}
    
    // Add seeking BEFORE input file (faster)
    if params.SeekSeconds > 0 {
        args = append(args, "-ss", strconv.FormatInt(params.SeekSeconds, 10))
    }
    
    // Add real-time pacing only if explicitly enabled (legacy mode)
    if params.RealtimePacing {
        args = append(args, "-re")
    }
    
    args = append(args, "-i", params.InputFile)
    
    // DO NOT add -stream_loop in batch mode
    
    return args
}

func buildHLSArgs(params StreamParams) []string {
    args := []string{
        "-f", "hls",
        "-hls_time", strconv.Itoa(params.SegmentDuration),
        "-hls_list_size", strconv.Itoa(params.PlaylistSize),
    }
    
    if params.BatchMode && params.BatchSize > 0 {
        // Generate exactly BatchSize segments then exit
        // hls_list_size controls playlist window
        // Calculate total segment time to determine when to stop
        totalSeconds := params.BatchSize * params.SegmentDuration
        args = append(args, "-t", strconv.Itoa(totalSeconds))
    }
    
    args = append(args, "-hls_flags", "delete_segments")
    args = append(args, "-hls_segment_filename", getSegmentFilenamePattern(params.OutputPath))
    args = append(args, params.OutputPath)
    
    return args
}
```

### 2. Client Position Tracking

**New API Endpoint**: `POST /api/stream/:channel_id/position`

**Purpose**: Frontend reports current playback position so backend knows when to generate next batch.

**Request Body**:
```json
{
  "session_id": "uuid-v4-string",
  "segment_number": 42,
  "quality": "1080p",
  "timestamp": "2025-10-31T12:34:56Z"
}
```

**Response**:
```json
{
  "acknowledged": true,
  "current_batch": 2,
  "segments_remaining": 8
}
```

**Implementation in `api/internal/api/stream.go`**:
```go
type UpdatePositionRequest struct {
    SessionID     string `json:"session_id" binding:"required"`
    SegmentNumber int    `json:"segment_number" binding:"required,min=0"`
    Quality       string `json:"quality" binding:"required"`
    Timestamp     string `json:"timestamp"`
}

func (h *StreamHandler) UpdatePosition(c *gin.Context) {
    var req UpdatePositionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{Error: "invalid_request", Message: err.Error()})
        return
    }
    
    channelID, _ := uuid.Parse(c.Param("channel_id"))
    
    session, found := h.streamManager.GetStream(channelID)
    if !found {
        c.JSON(404, ErrorResponse{Error: "stream_not_found"})
        return
    }
    
    // Update client position in session
    session.UpdateClientPosition(req.SessionID, req.SegmentNumber, req.Quality)
    
    c.JSON(200, gin.H{
        "acknowledged": true,
        "current_batch": session.GetCurrentBatchNumber(),
        "segments_remaining": session.GetSegmentsUntilBatchEnd(),
    })
}
```

### 3. Batch State Tracking

**Extend `StreamSession` model** (`api/internal/models/stream_session.go`):

```go
type BatchState struct {
    BatchNumber       int       // Current batch number (0, 1, 2, ...)
    StartSegment      int       // First segment number in batch
    EndSegment        int       // Last segment number in batch
    VideoSourcePath   string    // Media file being encoded
    VideoStartOffset  int64     // Starting position in source video (seconds)
    GenerationStarted time.Time // When batch generation began
    GenerationEnded   time.Time // When batch generation completed
    IsComplete        bool      // Whether batch finished generating
}

type ClientPosition struct {
    SessionID     string
    SegmentNumber int
    Quality       string
    LastUpdated   time.Time
}

type StreamSession struct {
    // ... existing fields ...
    CurrentBatch     *BatchState
    ClientPositions  map[string]*ClientPosition // key: session_id
    FurthestSegment  int                        // Furthest segment any client has reached
    mu               sync.RWMutex
}

func (s *StreamSession) UpdateClientPosition(sessionID string, segment int, quality string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.ClientPositions[sessionID] = &ClientPosition{
        SessionID:     sessionID,
        SegmentNumber: segment,
        Quality:       quality,
        LastUpdated:   time.Now(),
    }
    
    // Track furthest position across all clients
    if segment > s.FurthestSegment {
        s.FurthestSegment = segment
    }
}

func (s *StreamSession) GetFurthestPosition() int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.FurthestSegment
}

func (s *StreamSession) ShouldGenerateNextBatch(threshold int) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if s.CurrentBatch == nil || !s.CurrentBatch.IsComplete {
        return false // Wait for current batch to finish
    }
    
    segmentsRemaining := s.CurrentBatch.EndSegment - s.FurthestSegment
    return segmentsRemaining <= threshold
}
```

### 4. Batch Coordinator

**Add to `StreamManager`** (`api/internal/streaming/manager.go`):

```go
type StreamManager struct {
    // ... existing fields ...
    batchTriggerInterval time.Duration
}

func (m *StreamManager) Start() error {
    // ... existing startup code ...
    
    // Start batch monitoring goroutine
    go m.runBatchCoordinator()
    
    return nil
}

func (m *StreamManager) runBatchCoordinator() {
    ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            m.checkAndTriggerBatches()
        case <-m.stopChan:
            return
        }
    }
}

func (m *StreamManager) checkAndTriggerBatches() {
    sessions := m.sessionManager.List()
    
    for _, session := range sessions {
        // Skip if no active clients
        if session.GetClientCount() == 0 {
            continue
        }
        
        // Check if next batch should be generated
        if session.ShouldGenerateNextBatch(m.config.TriggerThreshold) {
            go m.generateNextBatch(context.Background(), session)
        }
    }
}

func (m *StreamManager) generateNextBatch(ctx context.Context, session *models.StreamSession) error {
    session.mu.Lock()
    currentBatch := session.CurrentBatch
    session.mu.Unlock()
    
    // Calculate next batch parameters
    nextBatchNumber := currentBatch.BatchNumber + 1
    nextStartSegment := currentBatch.EndSegment + 1
    nextEndSegment := nextStartSegment + m.config.BatchSize - 1
    
    // Calculate video position for next batch
    videoDuration := getVideoDuration(currentBatch.VideoSourcePath)
    nextOffset := currentBatch.VideoStartOffset + int64(m.config.BatchSize * m.config.SegmentDuration)
    
    // Handle video looping
    nextVideoPath := currentBatch.VideoSourcePath
    if nextOffset >= videoDuration {
        // Need to loop or move to next video
        nextOffset = nextOffset % videoDuration
        // TODO: Query timeline service for next media file if needed
    }
    
    // Build FFmpeg command for next batch
    params := StreamParams{
        InputFile:       nextVideoPath,
        OutputPath:      session.GetSegmentPath(),
        Quality:         "1080p", // TODO: Handle multiple qualities
        HardwareAccel:   HardwareAccel(m.config.HardwareAccel),
        SeekSeconds:     nextOffset,
        SegmentDuration: m.config.SegmentDuration,
        PlaylistSize:    m.config.PlaylistSize,
        BatchMode:       true,
        BatchSize:       m.config.BatchSize,
        RealtimePacing:  false, // Batch mode never uses real-time pacing
        EncodingPreset:  m.config.EncodingPreset,
    }
    
    ffmpegCmd, err := BuildHLSCommand(params)
    if err != nil {
        return fmt.Errorf("failed to build FFmpeg command: %w", err)
    }
    
    // Launch FFmpeg for this batch
    execCmd, err := launchFFmpeg(ffmpegCmd)
    if err != nil {
        return fmt.Errorf("failed to launch FFmpeg: %w", err)
    }
    
    // Update session with new batch info
    newBatch := &models.BatchState{
        BatchNumber:       nextBatchNumber,
        StartSegment:      nextStartSegment,
        EndSegment:        nextEndSegment,
        VideoSourcePath:   nextVideoPath,
        VideoStartOffset:  nextOffset,
        GenerationStarted: time.Now(),
        IsComplete:        false,
    }
    
    session.SetCurrentBatch(newBatch)
    
    // Wait for batch to complete
    go m.monitorBatchCompletion(session, execCmd, newBatch)
    
    logger.Log.Info().
        Str("channel_id", session.ChannelID.String()).
        Int("batch_number", nextBatchNumber).
        Int64("video_offset", nextOffset).
        Msg("Started generating next batch")
    
    return nil
}

func (m *StreamManager) monitorBatchCompletion(session *models.StreamSession, cmd *exec.Cmd, batch *models.BatchState) {
    // Wait for FFmpeg process to exit
    err := cmd.Wait()
    
    batch.GenerationEnded = time.Now()
    batch.IsComplete = true
    
    if err != nil {
        logger.Log.Error().
            Err(err).
            Int("batch_number", batch.BatchNumber).
            Msg("Batch generation failed")
        // TODO: Retry logic with circuit breaker
    } else {
        logger.Log.Info().
            Int("batch_number", batch.BatchNumber).
            Dur("generation_time", batch.GenerationEnded.Sub(batch.GenerationStarted)).
            Msg("Batch generation completed")
    }
}
```

### 5. Configuration

**Add to `StreamingConfig`** (`api/internal/config/config.go`):

```go
type StreamingConfig struct {
    // ... existing fields ...
    BatchSize        int  // Number of segments per batch (default: 20)
    TriggerThreshold int  // Generate next batch when N segments remain (default: 5)
}

const (
    // ... existing defaults ...
    defaultStreamingBatchSize        = 20
    defaultStreamingTriggerThreshold = 5
)

func setDefaults(v *viper.Viper) {
    // ... existing defaults ...
    v.SetDefault("streaming.batchsize", defaultStreamingBatchSize)
    v.SetDefault("streaming.triggerthreshold", defaultStreamingTriggerThreshold)
}

func (c *Config) Validate() error {
    // ... existing validation ...
    
    if c.Streaming.BatchSize <= 0 {
        return fmt.Errorf("batch size must be positive")
    }
    
    if c.Streaming.TriggerThreshold <= 0 || c.Streaming.TriggerThreshold >= c.Streaming.BatchSize {
        return fmt.Errorf("trigger threshold must be positive and less than batch size")
    }
    
    return nil
}
```

**Update `api/config.yaml.example`**:
```yaml
streaming:
  # ... existing config ...
  
  # Just-in-time batch generation
  batchsize: 20              # Generate 20 segments per batch
  triggerthreshold: 5        # Start next batch when 5 segments remain
```

### 6. Batch-Aware Cleanup

**Update `api/internal/streaming/cleanup.go`**:

Current cleanup deletes segments immediately with `delete_segments` flag. For batches:

- Keep N-1 batch (previous batch) available during N batch generation
- Delete N-2 batch once N batch is verified complete
- Prevents gaps if generation fails

```go
func (m *StreamManager) cleanupOldBatches(session *models.StreamSession) {
    currentBatch := session.GetCurrentBatch()
    if currentBatch == nil || currentBatch.BatchNumber < 2 {
        return // Need at least 2 batches before cleanup
    }
    
    // Delete segments from 2 batches ago
    batchToDelete := currentBatch.BatchNumber - 2
    startSegment := batchToDelete * m.config.BatchSize
    endSegment := startSegment + m.config.BatchSize - 1
    
    for i := startSegment; i <= endSegment; i++ {
        segmentPath := filepath.Join(
            session.GetOutputDir(),
            fmt.Sprintf("1080p_segment_%03d.ts", i % 1000), // Handle wrap-around
        )
        
        if err := os.Remove(segmentPath); err != nil && !os.IsNotExist(err) {
            logger.Log.Warn().
                Err(err).
                Str("segment", segmentPath).
                Msg("Failed to delete old segment")
        }
    }
}
```

### 7. Client Synchronization Strategy

**All clients start at oldest available segment** (beginning of current batch):

- Ensures clients stay within same batch window
- Simplifies batch management (single buffer for all clients)
- Prevents edge case where new client joins mid-batch

**Implementation**:
- When client requests master playlist, include `EXT-X-START` directive
- Points to first segment in current batch
- Clients automatically seek to that position on load

### 8. Timeline Service Integration

When generating next batch:
1. Query timeline service: "What should play at timestamp T?"
2. Handle media file transitions if batch crosses video boundaries
3. Adjust input file and seek position accordingly

**Example**: If batch 5 spans across two videos:
- Generate partial batch from video A
- Continue with video B for remaining segments
- Or: Align batch boundaries with video boundaries

### 9. Migration Path

**Remove Legacy Features**:
- Delete `RealtimePacing` config option
- Remove `-stream_loop -1` from FFmpeg commands
- Remove continuous monitoring (replaced by batch completion monitoring)

**Compatibility**:
- Frontend must be updated to report position (required)
- Old frontend without position reporting won't trigger batches (streams stall)
- Deploy backend and frontend together

## UX/UI Considerations

### User Experience

**Startup Time**:
- First batch generates in ~5-10 seconds (20 segments × 2s each = 40s of content)
- Encoding happens at 10-20x speed = 2-4 seconds generation time
- Client can start playing once first few segments ready

**Playback Continuity**:
- Client never knows batches exist (seamless to viewer)
- No interruption when transitioning between batches
- Segment numbering continuous across batches

**Error Handling**:
- If batch generation fails, retry with circuit breaker
- Keep previous batch available as fallback
- Display "buffering" if client catches up to generation

### Frontend Changes

**Position Reporting**:
```typescript
// In video player component
useEffect(() => {
  const reportPosition = () => {
    const currentSegment = Math.floor(player.currentTime() / segmentDuration);
    
    fetch(`/api/stream/${channelId}/position`, {
      method: 'POST',
      body: JSON.stringify({
        session_id: sessionId,
        segment_number: currentSegment,
        quality: currentQuality,
        timestamp: new Date().toISOString()
      })
    });
  };
  
  // Report position every 5 seconds
  const interval = setInterval(reportPosition, 5000);
  return () => clearInterval(interval);
}, [channelId, sessionId]);
```

**Client Sync on Join**:
```typescript
// When loading master playlist, respect EXT-X-START directive
// HLS.js automatically handles this
```

## Acceptance Criteria

Core Functionality:
- [ ] FFmpeg generates exactly N segments per batch then exits cleanly
- [ ] Batch size configurable via config file (default: 20 segments)
- [ ] Trigger threshold configurable via config file (default: 5 segments)
- [ ] `-stream_loop -1` flag removed from all FFmpeg commands
- [ ] Position tracking API endpoint implemented and functional
- [ ] Frontend reports current segment position every 5 seconds
- [ ] StreamSession model tracks batch state (number, range, video position)
- [ ] StreamSession model tracks client positions per session ID
- [ ] Batch coordinator monitors positions and triggers generation
- [ ] Next batch starts when furthest client reaches threshold
- [ ] Batch generation uses `-ss` seek for precise continuation
- [ ] Seamless continuation between batches (no gaps or overlaps)
- [ ] Segment numbering continuous across batch boundaries
- [ ] Multiple clients synchronized within same batch window

Performance & Resources:
- [ ] Initial batch generates in under 10 seconds
- [ ] Resource usage reduced by >70% compared to fast mode
- [ ] No encoding waste on unwatched segments
- [ ] CPU/GPU usage scales with number of active viewers, not video length

Video Transitions:
- [ ] Batch boundaries handle video loops correctly
- [ ] Timeline service queried for media transitions
- [ ] Multi-video playlists handled seamlessly

Cleanup:
- [ ] Batch-aware cleanup keeps N-1 batch during generation
- [ ] N-2 batch deleted after N batch completes
- [ ] No segment gaps during cleanup operations

Error Handling:
- [ ] Failed batch generation triggers retry
- [ ] Circuit breaker prevents repeated failures
- [ ] Previous batch remains available during failures
- [ ] Error messages logged with batch context

Configuration:
- [ ] Config validation ensures BatchSize > 0
- [ ] Config validation ensures 0 < TriggerThreshold < BatchSize
- [ ] Invalid config values provide clear error messages
- [ ] Config changes require service restart

Testing:
- [ ] Integration tests verify batch generation completes
- [ ] Integration tests verify position tracking updates
- [ ] Integration tests verify automatic triggering
- [ ] Integration tests verify seamless continuation
- [ ] Integration tests verify multi-client synchronization
- [ ] Unit tests for batch state calculations
- [ ] Unit tests for position threshold detection

Documentation:
- [ ] API specification updated with position tracking endpoint
- [ ] Streaming API docs updated with batch architecture
- [ ] Configuration options documented in config.yaml.example
- [ ] StreamSession model documented with batch fields

Migration:
- [ ] RealtimePacing config option removed
- [ ] Continuous monitoring code removed
- [ ] All references to `-stream_loop -1` removed
- [ ] Frontend updated with position reporting
- [ ] Deployment guide includes frontend/backend coordination

## Dependencies

**PBI Dependencies**:
- PBI-6: Streaming Engine (REQUIRED) - Provides base streaming infrastructure
- PBI-4: Virtual Timeline Calculation (REQUIRED) - Needed for video position calculations
- PBI-10: Video Player & Streaming UI (REQUIRED) - Frontend must be updated for position reporting

**Technical Dependencies**:
- FFmpeg with HLS support
- Go 1.21+ for generic type support
- Frontend with HLS.js or Video.js

**No Breaking Changes For**:
- Channel management
- Media library
- Database schema
- Existing API endpoints (only additions)

## Open Questions

1. **Batch size optimization**: Should batch size adapt based on video length or stay fixed?
   - Fixed: Simpler, predictable
   - Adaptive: Could optimize for very short/long videos

2. **Multiple quality support**: How to handle batch generation for 1080p, 720p, 480p?
   - Generate all qualities in same batch process (3x resources)
   - Stagger generation (complexity)
   - Generate on-demand per quality (current approach)

3. **Very long videos**: For 2-hour movies, should batches align with chapters/scenes?
   - Current approach: Fixed batch size regardless of content
   - Alternative: Content-aware batching

4. **Network interruptions**: If client disconnects during batch, stop generation?
   - Keep generating (client might reconnect)
   - Stop after timeout (save resources)

5. **Pre-warming**: Should first batch pre-generate when channel created (even without viewers)?
   - Pro: Instant playback for first viewer
   - Con: Wastes resources if no one watches

6. **Position reporting frequency**: Every 5 seconds optimal?
   - Too frequent: Network overhead
   - Too infrequent: Delayed batch triggering

7. **Segment wrapping**: FFmpeg uses %03d (wraps at 1000). Long-running channels?
   - Current: Logical segment tracking separate from filename
   - Alternative: Custom segment naming

8. **Batch overlap**: Should batches overlap by 1-2 segments for safety?
   - Pro: Prevents gaps on timing issues
   - Con: Slight resource waste

## Related Tasks

Tasks for this PBI are defined in [tasks.md](./tasks.md).



