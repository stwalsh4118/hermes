package streaming

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/config"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
	"github.com/stwalsh4118/hermes/internal/streaming/playlist"
	"github.com/stwalsh4118/hermes/internal/timeline"
)

// Common errors
var (
	ErrStreamNotFound      = errors.New("stream not found")
	ErrStreamAlreadyExists = errors.New("stream already exists")
	ErrManagerStopped      = errors.New("stream manager has been stopped")
)

const (
	defaultBatchTriggerInterval = 1 * time.Second // Check more frequently to catch buffer issues earlier
)

// StreamManager orchestrates the entire streaming pipeline
type StreamManager struct {
	repos                *db.Repositories
	timelineService      *timeline.TimelineService
	sessionManager       *SessionManager
	config               *config.StreamingConfig
	cleanupTicker        *time.Ticker
	batchTicker          *time.Ticker
	batchTriggerInterval time.Duration
	stopChan             chan struct{}
	cleanupDone          chan struct{}
	batchDone            chan struct{}
	playlistManagers     map[string]playlist.Manager // key: channelID_quality (e.g., "uuid-1080p")
	playlistManagersMu   sync.RWMutex
	mu                   sync.RWMutex
	stopped              bool
}

// NewStreamManager creates a new stream manager instance
func NewStreamManager(
	repos *db.Repositories,
	timelineService *timeline.TimelineService,
	cfg *config.StreamingConfig,
) *StreamManager {
	return &StreamManager{
		repos:                repos,
		timelineService:      timelineService,
		sessionManager:       NewSessionManager(),
		config:               cfg,
		batchTriggerInterval: defaultBatchTriggerInterval,
		stopChan:             make(chan struct{}),
		cleanupDone:          make(chan struct{}),
		batchDone:            make(chan struct{}),
		playlistManagers:     make(map[string]playlist.Manager),
		stopped:              false,
	}
}

// Start initializes the stream manager and starts background cleanup
func (m *StreamManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return ErrManagerStopped
	}

	// Create cleanup ticker
	cleanupInterval := time.Duration(m.config.CleanupInterval) * time.Second
	m.cleanupTicker = time.NewTicker(cleanupInterval)

	// Start background cleanup goroutine
	go m.runCleanupLoop()

	// Create batch coordinator ticker
	m.batchTicker = time.NewTicker(m.batchTriggerInterval)

	// Start batch coordinator goroutine
	go m.runBatchCoordinator()

	logger.Log.Info().
		Int("cleanup_interval_seconds", m.config.CleanupInterval).
		Int("grace_period_seconds", m.config.GracePeriodSeconds).
		Dur("batch_trigger_interval", m.batchTriggerInterval).
		Msg("Stream manager started")

	return nil
}

// Stop gracefully shuts down the stream manager
func (m *StreamManager) Stop() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	m.mu.Unlock()

	logger.Log.Info().Msg("Stopping stream manager...")

	// Signal cleanup goroutine to stop
	close(m.stopChan)

	// Wait for cleanup goroutine to finish (only if it was started)
	if m.cleanupTicker != nil {
		<-m.cleanupDone
		m.cleanupTicker.Stop()
	}

	// Wait for batch coordinator goroutine to finish (only if it was started)
	if m.batchTicker != nil {
		<-m.batchDone
		m.batchTicker.Stop()
	}

	// Stop all active streams
	sessions := m.sessionManager.List()
	for _, session := range sessions {
		channelID := session.ChannelID.String()
		if err := m.StopStream(context.Background(), session.ChannelID); err != nil {
			logger.Log.Error().
				Err(err).
				Str("channel_id", channelID).
				Msg("Failed to stop stream during shutdown")
		}
	}

	logger.Log.Info().
		Int("stopped_streams", len(sessions)).
		Msg("Stream manager stopped")
}

// StartStream starts a new stream for a channel or returns existing stream
func (m *StreamManager) StartStream(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error) {
	m.mu.RLock()
	if m.stopped {
		m.mu.RUnlock()
		return nil, ErrManagerStopped
	}
	m.mu.RUnlock()

	channelIDStr := channelID.String()

	// Check if stream already exists
	if existingSession, ok := m.sessionManager.Get(channelIDStr); ok {
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Int("client_count", existingSession.GetClientCount()).
			Msg("Returning existing stream")
		return existingSession, nil
	}

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Msg("Starting new stream")

	// Check disk space before starting stream
	if err := checkDiskSpace(m.config.SegmentPath); err != nil {
		logger.Log.Error().
			Err(err).
			Str("segment_path", m.config.SegmentPath).
			Msg("Insufficient disk space to start stream")
		return nil, ErrInsufficientDiskSpace
	}

	// Fetch channel from database
	channel, err := m.repos.Channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// TESTING: Skip timeline calculation and start from beginning
	// Get current timeline position
	// position, err := m.timelineService.GetCurrentPosition(ctx, channelID)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get timeline position: %w", err)
	// }

	// Verify playlist has items (batch generation will handle timeline position)
	playlistItems, err := m.repos.PlaylistItems.GetWithMedia(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist items: %w", err)
	}
	if len(playlistItems) == 0 {
		return nil, fmt.Errorf("no playlist items found for channel")
	}

	// Build output directory
	outputDir := fmt.Sprintf("%s/%s", m.config.SegmentPath, channelIDStr)
	quality := Quality1080p // Start with 1080p for now

	// Create segment directories
	if err := createSegmentDirectories(outputDir, channelIDStr); err != nil {
		return nil, fmt.Errorf("failed to create segment directories: %w", err)
	}

	// Create stream session
	session := models.NewStreamSession(channelID)
	session.SetState(StateIdle.String()) // Start in idle state, batch generation will activate it
	session.SetOutputDir(outputDir)
	session.SetSegmentPath(filepath.Join(outputDir, quality))
	session.UpdateLastAccess()

	// Set quality information
	qualities := []models.StreamQuality{
		{
			Level:       quality,
			Bitrate:     5000, // 1080p bitrate
			Resolution:  "1920x1080",
			SegmentPath: session.GetSegmentPath(),
			// PlaylistPath will be set by playlist manager
		},
	}
	session.SetQualities(qualities)

	// Store session in manager
	m.sessionManager.Set(channelIDStr, session)

	// Generate and write master playlist
	if err := m.generateMasterPlaylist(outputDir, qualities); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Msg("Failed to generate master playlist (continuing anyway)")
		// Don't fail the stream start, just log the error
	}

	// Batch coordinator will automatically trigger batch generation which will:
	// 1. Launch FFmpeg with stream_segment mode
	// 2. Initialize the first batch
	// 3. Monitor batch completion

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("channel_name", channel.Name).
		Str("output_dir", outputDir).
		Msg("Stream session created, batch generation will start automatically")

	return session, nil
}

// StopStream stops a stream and cleans up resources
func (m *StreamManager) StopStream(_ context.Context, channelID uuid.UUID) error {
	channelIDStr := channelID.String()

	// Get stream session
	session, ok := m.sessionManager.Get(channelIDStr)
	if !ok {
		return ErrStreamNotFound
	}

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Int("ffmpeg_pid", session.GetFFmpegPID()).
		Msg("Stopping stream")

	// Set state to stopping
	session.SetState(StateStopping.String())

	// Terminate FFmpeg process
	pid := session.GetFFmpegPID()
	if pid > 0 {
		if err := terminateProcess(pid); err != nil {
			logger.Log.Warn().
				Err(err).
				Str("channel_id", channelIDStr).
				Int("pid", pid).
				Msg("Failed to terminate FFmpeg process")
		}
	}

	// Clean up segment files
	outputDir := session.GetOutputDir()
	if outputDir != "" {
		if err := cleanupSegments(outputDir); err != nil {
			logger.Log.Warn().
				Err(err).
				Str("channel_id", channelIDStr).
				Str("output_dir", outputDir).
				Msg("Failed to cleanup segments")
		}
	}

	// Close all playlist managers for this channel
	m.closePlaylistManagersForChannel(channelIDStr)

	// Remove session from manager
	m.sessionManager.Delete(channelIDStr)

	// Remove circuit breaker
	m.sessionManager.DeleteCircuitBreaker(channelIDStr)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Msg("Stream stopped successfully")

	return nil
}

// closePlaylistManagersForChannel closes all playlist managers for a channel
func (m *StreamManager) closePlaylistManagersForChannel(channelIDStr string) {
	m.playlistManagersMu.Lock()
	defer m.playlistManagersMu.Unlock()

	prefix := channelIDStr + "_"
	for key, pm := range m.playlistManagers {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			if err := pm.Close(); err != nil {
				logger.Log.Warn().
					Err(err).
					Str("channel_id", channelIDStr).
					Str("manager_key", key).
					Msg("Failed to close playlist manager")
			}
			delete(m.playlistManagers, key)
		}
	}
}

// GetStream retrieves a stream session by channel ID
func (m *StreamManager) GetStream(channelID uuid.UUID) (*models.StreamSession, bool) {
	return m.sessionManager.Get(channelID.String())
}

// RegisterClient registers a client connection for a channel
func (m *StreamManager) RegisterClient(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error) {
	// Start stream if it doesn't exist, or get existing stream
	session, err := m.StartStream(ctx, channelID)
	if err != nil {
		return nil, err
	}

	// Increment client count
	session.IncrementClients()
	session.UpdateLastAccess()

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Int("client_count", session.GetClientCount()).
		Msg("Client registered")

	return session, nil
}

// UnregisterClient unregisters a client connection from a channel
func (m *StreamManager) UnregisterClient(_ context.Context, channelID uuid.UUID) error {
	channelIDStr := channelID.String()

	// Get stream session
	session, ok := m.sessionManager.Get(channelIDStr)
	if !ok {
		return ErrStreamNotFound
	}

	// Decrement client count
	session.DecrementClients()
	session.UpdateLastAccess()

	clientCount := session.GetClientCount()

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Int("client_count", clientCount).
		Msg("Client unregistered")

	// Grace period will be handled by cleanup goroutine
	if clientCount == 0 {
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Int("grace_period_seconds", m.config.GracePeriodSeconds).
			Msg("Last client disconnected, grace period started")
	}

	return nil
}

// runCleanupLoop runs periodic cleanup of idle streams
func (m *StreamManager) runCleanupLoop() {
	defer close(m.cleanupDone)

	logger.Log.Debug().Msg("Cleanup loop started")

	for {
		select {
		case <-m.stopChan:
			logger.Log.Debug().Msg("Cleanup loop stopping")
			return
		case <-m.cleanupTicker.C:
			m.performCleanup()
		}
	}
}

// runBatchCoordinator runs periodic batch generation checks
func (m *StreamManager) runBatchCoordinator() {
	defer close(m.batchDone)

	logger.Log.Debug().Msg("Batch coordinator started")

	for {
		select {
		case <-m.stopChan:
			logger.Log.Debug().Msg("Batch coordinator stopping")
			return
		case <-m.batchTicker.C:
			logger.Log.Debug().Msg("Checking and triggering batches")
			m.checkAndTriggerBatches()
		}
	}
}

// checkAndTriggerBatches checks all active streams and triggers batch generation when needed
func (m *StreamManager) checkAndTriggerBatches() {
	sessions := m.sessionManager.List()

	for _, session := range sessions {
		// Skip if no active clients
		if session.GetClientCount() == 0 {
			continue
		}

		currentBatch := session.GetCurrentBatch()

		// Check if first batch needs to be initialized (no batch exists yet)
		if currentBatch == nil {
			logger.Log.Debug().
				Str("channel_id", session.ChannelID.String()).
				Msg("No batch exists, initializing first batch")
			// Launch batch generation in goroutine to avoid blocking coordinator
			go func(sess *models.StreamSession) {
				if err := m.generateNextBatch(context.Background(), sess); err != nil {
					logger.Log.Error().
						Err(err).
						Str("channel_id", sess.ChannelID.String()).
						Msg("Failed to initialize first batch")
				}
			}(session)
			continue
		}

		// Check if next batch should be generated (for subsequent batches)
		// Increased trigger threshold (7) and faster coordinator interval (1s) help prevent buffer stalls
		if session.ShouldGenerateNextBatch(m.config.TriggerThreshold) {
			logger.Log.Debug().
				Str("channel_id", session.ChannelID.String()).
				Int("trigger_threshold", m.config.TriggerThreshold).
				Msg("Triggering next batch generation")
			// Launch batch generation in goroutine to avoid blocking coordinator
			go func(sess *models.StreamSession) {
				if err := m.generateNextBatch(context.Background(), sess); err != nil {
					logger.Log.Error().
						Err(err).
						Str("channel_id", sess.ChannelID.String()).
						Msg("Failed to generate next batch")
				}
			}(session)
		}
	}
}

// generateNextBatch generates the next batch of segments for a stream session
func (m *StreamManager) generateNextBatch(ctx context.Context, session *models.StreamSession) error {
	channelID := session.ChannelID
	channelIDStr := channelID.String()

	// Get current batch from session
	currentBatch := session.GetCurrentBatch()

	// Handle first batch initialization (when currentBatch is nil)
	if currentBatch == nil {
		return m.initializeFirstBatch(ctx, session)
	}

	// Safety check: prevent concurrent batch generation
	// If current batch is not complete, another goroutine is already generating
	if !currentBatch.IsComplete {
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Int("batch_number", currentBatch.BatchNumber).
			Msg("Batch generation already in progress, skipping")
		return nil // Not an error, just skip
	}

	// Calculate next batch parameters
	nextBatchNumber := currentBatch.BatchNumber + 1
	nextStartSegment := currentBatch.EndSegment + 1
	nextEndSegment := nextStartSegment + m.config.BatchSize - 1

	// Get playlist items to handle video transitions
	playlistItems, err := m.repos.PlaylistItems.GetWithMedia(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get playlist items: %w", err)
	}

	// Find current video in playlist and determine next video/offset
	previousBatchVideoPath := currentBatch.VideoSourcePath
	currentVideoPath := currentBatch.VideoSourcePath
	// VideoStartOffset already points to where the next segment should start (set at end of previous batch)
	// So we can use it directly without adding batch duration
	currentOffset := currentBatch.VideoStartOffset
	currentPlaylistIndex := 0

	// Find current video index in playlist
	for i, item := range playlistItems {
		if item.Media != nil && item.Media.FilePath == currentVideoPath {
			currentPlaylistIndex = i
			break
		}
	}

	// Check if we've moved to a new video file since the last batch
	// This handles video transitions that occur between batches
	currentItem := playlistItems[currentPlaylistIndex]
	if currentItem.Media != nil {
		// Check if currentOffset exceeds current video duration (moved to next video)
		if currentOffset >= int64(currentItem.Media.Duration) {
			// Calculate which video we should be on now
			remainingOffset := currentOffset
			for remainingOffset >= int64(currentItem.Media.Duration) && currentPlaylistIndex < len(playlistItems) {
				remainingOffset -= int64(currentItem.Media.Duration)
				currentPlaylistIndex++
				if currentPlaylistIndex >= len(playlistItems) {
					// Loop back to beginning if channel loops
					channel, err := m.repos.Channels.GetByID(ctx, channelID)
					if err == nil && channel.Loop {
						currentPlaylistIndex = 0
					} else {
						return fmt.Errorf("reached end of playlist and channel does not loop")
					}
				}
				currentItem = playlistItems[currentPlaylistIndex]
				if currentItem.Media == nil {
					return fmt.Errorf("playlist item at index %d has no media", currentPlaylistIndex)
				}
			}
			currentVideoPath = currentItem.Media.FilePath
			currentOffset = remainingOffset

			// Mark discontinuity if video file changed between batches
			if currentVideoPath != previousBatchVideoPath {
				// Get playlist manager (quality is defined below, but we can use the constant here)
				pm, err := m.getPlaylistManager(session, Quality1080p)
				if err == nil {
					pm.SetDiscontinuityNext()
					logger.Log.Debug().
						Str("channel_id", channelIDStr).
						Str("previous_video", previousBatchVideoPath).
						Str("new_video", currentVideoPath).
						Int("batch_number", nextBatchNumber).
						Msg("Video switch detected between batches, marking discontinuity")
				}
			}
		}
	}

	// Build output directory
	outputDir := session.GetOutputDir()
	quality := Quality1080p
	qualityDir := filepath.Join(outputDir, quality)

	// Ensure playlist manager is initialized for this quality
	if err := m.ensurePlaylistManager(session, quality, qualityDir); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Str("quality", quality).
			Msg("Failed to initialize playlist manager (continuing anyway)")
		// Continue anyway - segments will still be generated
	}

	// Create new BatchState
	newBatch := &models.BatchState{
		BatchNumber:       nextBatchNumber,
		StartSegment:      nextStartSegment,
		EndSegment:        nextEndSegment,
		VideoSourcePath:   currentVideoPath, // Will be updated as we progress
		VideoStartOffset:  currentOffset,    // Will be updated as we progress
		GenerationStarted: time.Now(),
		IsComplete:        false,
	}

	// Update session with new batch (atomic update with lock)
	session.SetCurrentBatch(newBatch)

	// Generate batch by looping BatchSize times, generating one segment at a time
	previousVideoPath := currentVideoPath // Track previous video to detect switches
	for segmentNum := 0; segmentNum < m.config.BatchSize; segmentNum++ {
		// Get current playlist item
		currentItem := playlistItems[currentPlaylistIndex]
		if currentItem.Media == nil {
			return fmt.Errorf("playlist item at index %d has no media", currentPlaylistIndex)
		}

		// Check if we need to advance to next video
		if currentOffset >= int64(currentItem.Media.Duration) {
			// Move to next playlist item
			currentPlaylistIndex++
			if currentPlaylistIndex >= len(playlistItems) {
				// Loop back to beginning if channel loops
				channel, err := m.repos.Channels.GetByID(ctx, channelID)
				if err == nil && channel.Loop {
					currentPlaylistIndex = 0
				} else {
					return fmt.Errorf("reached end of playlist and channel does not loop")
				}
			}
			currentItem = playlistItems[currentPlaylistIndex]
			if currentItem.Media == nil {
				return fmt.Errorf("playlist item at index %d has no media", currentPlaylistIndex)
			}
			currentVideoPath = currentItem.Media.FilePath
			currentOffset = 0

			// Update batch state with new video
			newBatch.VideoSourcePath = currentVideoPath
			newBatch.VideoStartOffset = currentOffset

			// Mark discontinuity when switching videos (different source file)
			if currentVideoPath != previousVideoPath {
				pm, err := m.getPlaylistManager(session, quality)
				if err == nil {
					pm.SetDiscontinuityNext()
					logger.Log.Debug().
						Str("channel_id", channelIDStr).
						Str("previous_video", previousVideoPath).
						Str("new_video", currentVideoPath).
						Int("segment_number", nextStartSegment+segmentNum).
						Msg("Video switch detected, marking discontinuity")
				}
				previousVideoPath = currentVideoPath
			}
		}

		// Generate single segment synchronously
		if err := m.generateSingleSegment(
			ctx,
			session,
			currentVideoPath,
			currentOffset,
			quality,
			qualityDir,
			nextStartSegment+segmentNum,
		); err != nil {
			logger.Log.Error().
				Err(err).
				Str("channel_id", channelIDStr).
				Int("segment_number", nextStartSegment+segmentNum).
				Int("batch_number", nextBatchNumber).
				Msg("Failed to generate segment in batch")
			return fmt.Errorf("failed to generate segment %d: %w", nextStartSegment+segmentNum, err)
		}

		// Advance offset for next segment
		currentOffset += int64(m.config.StreamSegmentDuration)
	}

	// Update final batch state
	newBatch.VideoSourcePath = currentVideoPath
	// currentOffset now points to where the NEXT segment should start (after the last segment in this batch)
	// This is the correct value to use for the next batch's starting offset
	newBatch.VideoStartOffset = currentOffset

	// Mark batch as complete
	generationEnded := time.Now()
	session.UpdateBatchCompletion(generationEnded, true)

	// Calculate generation metrics
	batchGenerationTime := generationEnded.Sub(newBatch.GenerationStarted)
	batchContentDuration := time.Duration(m.config.BatchSize*m.config.StreamSegmentDuration) * time.Second
	batchSpeedRatio := float64(batchGenerationTime) / float64(batchContentDuration)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Int("batch_number", nextBatchNumber).
		Int("start_segment", nextStartSegment).
		Int("end_segment", nextEndSegment).
		Int("segments_generated", m.config.BatchSize).
		Dur("generation_time", batchGenerationTime).
		Dur("content_duration", batchContentDuration).
		Float64("generation_speed_ratio", batchSpeedRatio).
		Msg("Next batch completed successfully")

	// Warn if generation is slower than real-time
	if batchSpeedRatio > 1.0 {
		logger.Log.Warn().
			Str("channel_id", channelIDStr).
			Int("batch_number", nextBatchNumber).
			Float64("speed_ratio", batchSpeedRatio).
			Msg("Batch generation is slower than real-time playback - buffer may drain")
	}

	// Proactively check if we should start the next batch immediately
	// This prevents gaps in segment generation by starting the next batch right away
	// instead of waiting for the coordinator to check
	if session.ShouldGenerateNextBatch(m.config.TriggerThreshold) {
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Msg("Proactively starting next batch immediately after completion")
		// Start next batch in goroutine to avoid blocking
		go func() {
			if err := m.generateNextBatch(context.Background(), session); err != nil {
				logger.Log.Error().
					Err(err).
					Str("channel_id", channelIDStr).
					Msg("Failed to proactively generate next batch")
			}
		}()
	}

	return nil
}

// generateSingleSegment generates exactly one segment synchronously
// This function launches FFmpeg, waits for it to complete, adds the segment to the playlist, and returns
func (m *StreamManager) generateSingleSegment(
	ctx context.Context,
	session *models.StreamSession,
	videoPath string,
	offsetSeconds int64,
	quality string,
	qualityDir string,
	segmentNumber int,
) error {
	channelIDStr := session.ChannelID.String()

	// Build StreamParams for single segment (1 segment = SegmentDuration seconds)
	// Calculate cumulative stream position for PTS timestamps and ProgramDateTime
	streamPositionSeconds := int64(segmentNumber) * int64(m.config.StreamSegmentDuration)
	params := StreamParams{
		InputFile:              videoPath,
		Quality:                quality,
		HardwareAccel:          HardwareAccel(m.config.HardwareAccel),
		SeekSeconds:            offsetSeconds,         // Position within current video file (for FFmpeg -ss)
		StreamPositionSeconds:  streamPositionSeconds, // Cumulative stream position (for -output_ts_offset and ProgramDateTime)
		EncodingPreset:         m.config.EncodingPreset,
		BatchMode:              false, // Not batch mode - generate exactly 1 segment
		StreamSegmentMode:      true,
		SegmentOutputDir:       qualityDir,
		SegmentFilenamePattern: m.config.StreamSegmentFilenamePattern,
		SegmentDuration:        m.config.StreamSegmentDuration,
		FPS:                    m.config.FPS,
	}

	// Build FFmpeg command
	ffmpegCmd, err := BuildHLSCommand(params)
	if err != nil {
		return fmt.Errorf("failed to build FFmpeg command for segment %d: %w", segmentNumber, err)
	}

	// Extract output filename from command (last argument is the output path)
	var segmentFilename string
	if len(ffmpegCmd.Args) > 0 {
		outputPath := ffmpegCmd.Args[len(ffmpegCmd.Args)-1]
		segmentFilename = filepath.Base(outputPath)
	} else {
		return fmt.Errorf("FFmpeg command has no output path")
	}

	// Launch FFmpeg process
	execCmd, err := launchFFmpeg(ffmpegCmd)
	if err != nil {
		return fmt.Errorf("failed to launch FFmpeg for segment %d: %w", segmentNumber, err)
	}

	segmentStartTime := time.Now()
	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Int("segment_number", segmentNumber).
		Int64("offset_seconds", offsetSeconds).
		Str("video_path", videoPath).
		Str("segment_filename", segmentFilename).
		Int("ffmpeg_pid", execCmd.Process.Pid).
		Msg("Generating single segment")

	// Wait for FFmpeg to complete (synchronous - blocks until segment is created)
	err = execCmd.Wait()
	segmentGenerationTime := time.Since(segmentStartTime)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Int("segment_number", segmentNumber).
			Int64("offset_seconds", offsetSeconds).
			Msg("Failed to generate segment")
		return fmt.Errorf("FFmpeg failed for segment %d: %w", segmentNumber, err)
	}

	// Segment generated successfully - add it directly to the playlist
	pm, err := m.getPlaylistManager(session, quality)
	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("channel_id", channelIDStr).
			Str("quality", quality).
			Str("segment_filename", segmentFilename).
			Msg("Failed to get playlist manager, segment generated but not added to playlist")
		// Don't fail - segment was generated successfully
		return nil
	}

	// Add segment to playlist
	seg := playlist.SegmentMeta{
		URI:      segmentFilename,
		Duration: float64(m.config.StreamSegmentDuration),
	}
	// Set ProgramDateTime based on cumulative stream position (already calculated above)
	// This tells HLS.js the absolute timeline position of this segment in the overall stream
	// Use session start time as base and add cumulative stream position to create sequential timestamps
	// Note: offsetSeconds is position within current video file, not stream position
	// streamPositionSeconds is already calculated above as segmentNumber * segmentDuration
	sessionStartTime := session.StartedAt.UTC()
	segmentProgramTime := sessionStartTime.Add(time.Duration(streamPositionSeconds) * time.Second)
	seg.ProgramDateTime = &segmentProgramTime

	if err := pm.AddSegment(seg); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Str("segment_filename", segmentFilename).
			Msg("Failed to add segment to playlist")
		return fmt.Errorf("failed to add segment to playlist: %w", err)
	}

	// Write playlist to disk
	if err := pm.Write(); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Str("segment_filename", segmentFilename).
			Msg("Failed to write playlist")
		return fmt.Errorf("failed to write playlist: %w", err)
	}

	// Calculate generation speed ratio (generation_time / content_duration)
	// If ratio > 1.0, generation is slower than real-time (bad)
	// If ratio < 1.0, generation is faster than real-time (good)
	segmentContentDuration := time.Duration(m.config.StreamSegmentDuration) * time.Second
	generationSpeedRatio := float64(segmentGenerationTime) / float64(segmentContentDuration)

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Int("segment_number", segmentNumber).
		Int64("offset_seconds", offsetSeconds).
		Str("segment_filename", segmentFilename).
		Dur("generation_time", segmentGenerationTime).
		Float64("generation_speed_ratio", generationSpeedRatio).
		Msg("Segment generated and added to playlist successfully")

	return nil
}

// initializeFirstBatch initializes the first batch when currentBatch is nil
func (m *StreamManager) initializeFirstBatch(ctx context.Context, session *models.StreamSession) error {
	channelID := session.ChannelID
	channelIDStr := channelID.String()

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Msg("Initializing first batch")

	// Get playlist items to find first media file
	playlistItems, err := m.repos.PlaylistItems.GetWithMedia(ctx, channelID)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Msg("Failed to get playlist items for first batch")
		return fmt.Errorf("failed to get playlist items: %w", err)
	}

	if len(playlistItems) == 0 {
		logger.Log.Error().
			Str("channel_id", channelIDStr).
			Msg("No playlist items found for first batch")
		return fmt.Errorf("no playlist items found for channel")
	}

	// Get first playlist item (start from beginning of playlist)
	firstItem := playlistItems[0]
	if firstItem.Media == nil {
		logger.Log.Error().
			Str("channel_id", channelIDStr).
			Msg("First playlist item has no media")
		return fmt.Errorf("first playlist item has no media")
	}

	mediaFilePath := firstItem.Media.FilePath

	// Initialize first batch - start from beginning of first video
	nextBatchNumber := 0
	nextStartSegment := 0
	nextEndSegment := m.config.BatchSize - 1
	nextOffset := int64(0) // Start from beginning of the media file

	// Build output directory
	outputDir := session.GetOutputDir()
	quality := Quality1080p
	qualityDir := filepath.Join(outputDir, quality)

	// Initialize playlist manager for this quality (if not already done)
	if err := m.ensurePlaylistManager(session, quality, qualityDir); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Str("quality", quality).
			Msg("Failed to initialize playlist manager (continuing anyway)")
		// Continue anyway - segments will still be generated
	}

	// Create first BatchState
	newBatch := &models.BatchState{
		BatchNumber:       nextBatchNumber,
		StartSegment:      nextStartSegment,
		EndSegment:        nextEndSegment,
		VideoSourcePath:   mediaFilePath,
		VideoStartOffset:  nextOffset,
		GenerationStarted: time.Now(),
		IsComplete:        false,
	}

	// Update session with first batch
	session.SetCurrentBatch(newBatch)

	// Generate batch by looping BatchSize times, generating one segment at a time
	currentVideoPath := mediaFilePath
	currentOffset := nextOffset
	currentPlaylistIndex := 0
	previousVideoPath := currentVideoPath // Track previous video to detect switches

	for segmentNum := 0; segmentNum < m.config.BatchSize; segmentNum++ {
		// Check if we need to advance to next video
		if firstItem.Media != nil && currentOffset >= int64(firstItem.Media.Duration) {
			// Move to next playlist item
			currentPlaylistIndex++
			if currentPlaylistIndex >= len(playlistItems) {
				// Loop back to beginning if channel loops
				channel, err := m.repos.Channels.GetByID(ctx, channelID)
				if err == nil && channel.Loop {
					currentPlaylistIndex = 0
				} else {
					return fmt.Errorf("reached end of playlist and channel does not loop")
				}
			}
			firstItem = playlistItems[currentPlaylistIndex]
			if firstItem.Media == nil {
				return fmt.Errorf("playlist item at index %d has no media", currentPlaylistIndex)
			}
			previousVideoPath = currentVideoPath
			currentVideoPath = firstItem.Media.FilePath
			currentOffset = 0

			// Mark discontinuity when switching videos (different source file)
			if currentVideoPath != previousVideoPath {
				pm, err := m.getPlaylistManager(session, quality)
				if err == nil {
					pm.SetDiscontinuityNext()
					logger.Log.Debug().
						Str("channel_id", channelIDStr).
						Str("previous_video", previousVideoPath).
						Str("new_video", currentVideoPath).
						Int("segment_number", segmentNum).
						Msg("Video switch detected in first batch, marking discontinuity")
				}
			}
		}

		// Generate single segment synchronously
		if err := m.generateSingleSegment(
			ctx,
			session,
			currentVideoPath,
			currentOffset,
			quality,
			qualityDir,
			segmentNum,
		); err != nil {
			logger.Log.Error().
				Err(err).
				Str("channel_id", channelIDStr).
				Int("segment_number", segmentNum).
				Int("batch_number", nextBatchNumber).
				Msg("Failed to generate segment in batch")
			return fmt.Errorf("failed to generate segment %d: %w", segmentNum, err)
		}

		// Advance offset for next segment
		currentOffset += int64(m.config.StreamSegmentDuration)
	}

	// Update final batch state
	newBatch.VideoSourcePath = currentVideoPath
	// currentOffset now points to where the NEXT segment should start (after the last segment in this batch)
	// This is the correct value to use for the next batch's starting offset
	newBatch.VideoStartOffset = currentOffset

	// Mark batch as complete
	generationEnded := time.Now()
	session.UpdateBatchCompletion(generationEnded, true)

	// Calculate generation metrics
	batchGenerationTime := generationEnded.Sub(newBatch.GenerationStarted)
	batchContentDuration := time.Duration(m.config.BatchSize*m.config.StreamSegmentDuration) * time.Second
	batchSpeedRatio := float64(batchGenerationTime) / float64(batchContentDuration)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Int("batch_number", nextBatchNumber).
		Int("start_segment", nextStartSegment).
		Int("end_segment", nextEndSegment).
		Int("segments_generated", m.config.BatchSize).
		Dur("generation_time", batchGenerationTime).
		Dur("content_duration", batchContentDuration).
		Float64("generation_speed_ratio", batchSpeedRatio).
		Msg("First batch completed successfully")

	// Warn if generation is slower than real-time
	if batchSpeedRatio > 1.0 {
		logger.Log.Warn().
			Str("channel_id", channelIDStr).
			Int("batch_number", nextBatchNumber).
			Float64("speed_ratio", batchSpeedRatio).
			Msg("Batch generation is slower than real-time playback - buffer may drain")
	}

	// Proactively check if we should start the next batch immediately
	// This prevents gaps in segment generation by starting the next batch right away
	// instead of waiting for the coordinator to check
	if session.ShouldGenerateNextBatch(m.config.TriggerThreshold) {
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Msg("Proactively starting next batch immediately after first batch completion")
		// Start next batch in goroutine to avoid blocking
		go func() {
			if err := m.generateNextBatch(context.Background(), session); err != nil {
				logger.Log.Error().
					Err(err).
					Str("channel_id", channelIDStr).
					Msg("Failed to proactively generate next batch")
			}
		}()
	}

	return nil
}

// monitorBatchCompletion monitors FFmpeg process completion for a batch
func (m *StreamManager) monitorBatchCompletion(session *models.StreamSession, cmd *exec.Cmd, batch *models.BatchState) {
	channelIDStr := session.ChannelID.String()

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Int("batch_number", batch.BatchNumber).
		Int("ffmpeg_pid", cmd.Process.Pid).
		Int("start_segment", batch.StartSegment).
		Int("end_segment", batch.EndSegment).
		Int("batch_size", batch.EndSegment-batch.StartSegment+1).
		Time("generation_started", batch.GenerationStarted).
		Msg("Monitoring batch completion")

	// Wait for FFmpeg process to exit
	err := cmd.Wait()

	// Update batch state atomically
	generationEnded := time.Now()
	generationDuration := generationEnded.Sub(batch.GenerationStarted)
	generationDurationMs := generationDuration.Milliseconds()
	session.UpdateBatchCompletion(generationEnded, true)

	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Int("batch_number", batch.BatchNumber).
			Int("start_segment", batch.StartSegment).
			Int("end_segment", batch.EndSegment).
			Int64("generation_time_ms", generationDurationMs).
			Dur("generation_time", generationDuration).
			Time("generation_started", batch.GenerationStarted).
			Time("generation_ended", generationEnded).
			Msg("Batch generation failed")

		// Update session error state
		session.IncrementErrorCount()
		session.SetLastError(err)
	} else {
		logger.Log.Info().
			Str("channel_id", channelIDStr).
			Int("batch_number", batch.BatchNumber).
			Int("start_segment", batch.StartSegment).
			Int("end_segment", batch.EndSegment).
			Int("batch_size", batch.EndSegment-batch.StartSegment+1).
			Int64("generation_time_ms", generationDurationMs).
			Dur("generation_time", generationDuration).
			Time("generation_started", batch.GenerationStarted).
			Time("generation_ended", generationEnded).
			Msg("Batch generation completed successfully")

		// Clean up old batches (N-2) after successful completion
		// This keeps N-1 batch available during N batch generation
		outputDir := session.GetOutputDir()
		quality := Quality1080p // Default to 1080p for now (matches current implementation)
		cleanupOldBatches(session, m.config.BatchSize, outputDir, quality)
	}
}

// performCleanup checks all sessions and stops idle ones past grace period
func (m *StreamManager) performCleanup() {
	gracePeriod := time.Duration(m.config.GracePeriodSeconds) * time.Second
	sessions := m.sessionManager.List()

	stoppedCount := 0
	for _, session := range sessions {
		if session.ShouldCleanup(gracePeriod) {
			channelID := session.ChannelID
			logger.Log.Info().
				Str("channel_id", channelID.String()).
				Dur("idle_duration", session.IdleDuration()).
				Msg("Cleaning up idle stream")

			if err := m.StopStream(context.Background(), channelID); err != nil {
				logger.Log.Error().
					Err(err).
					Str("channel_id", channelID.String()).
					Msg("Failed to stop idle stream during cleanup")
			} else {
				stoppedCount++
			}
		}
	}

	if stoppedCount > 0 {
		logger.Log.Info().
			Int("stopped_count", stoppedCount).
			Int("active_count", len(sessions)-stoppedCount).
			Msg("Cleanup cycle completed")
	}

	// Cleanup orphaned directories
	activeSessions := m.sessionManager.List()
	if err := cleanupOrphanedDirectories(m.config.SegmentPath, activeSessions); err != nil {
		logger.Log.Warn().
			Err(err).
			Msg("Failed to cleanup orphaned directories")
	}
}

// monitorFFmpegProcess monitors an FFmpeg process and handles crashes
func (m *StreamManager) monitorFFmpegProcess(channelID uuid.UUID, cmd interface{}) {
	channelIDStr := channelID.String()
	monitorStartTime := time.Now()

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Time("monitor_start_time", monitorStartTime).
		Msg("FFmpeg process monitor started")

	// Type assert to exec.Cmd
	execCmd, ok := cmd.(*exec.Cmd)
	if !ok {
		logger.Log.Error().
			Str("channel_id", channelIDStr).
			Msg("Invalid command type in process monitor")
		return
	}

	// Wait for process to exit
	err := execCmd.Wait()
	processExitTime := time.Now()
	processRuntimeMs := processExitTime.Sub(monitorStartTime).Milliseconds()

	// Get session to check if stop was intentional
	session, exists := m.sessionManager.Get(channelIDStr)
	if !exists {
		// Session was already cleaned up, likely intentional stop
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Msg("FFmpeg process exited, session already cleaned up")
		return
	}

	// Check if we're in stopping state (intentional stop)
	currentState := StreamState(session.GetState())
	if currentState == StateStopping || currentState == StateIdle {
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Str("state", currentState.String()).
			Msg("FFmpeg process exited during intentional shutdown")
		return
	}

	// Process crashed unexpectedly
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelIDStr).
			Int("ffmpeg_pid", session.GetFFmpegPID()).
			Int64("process_runtime_ms", processRuntimeMs).
			Time("process_exit_time", processExitTime).
			Msg("FFmpeg process crashed unexpectedly")

		// Update session state and error tracking
		session.SetState(StateFailed.String())
		session.IncrementErrorCount()
		session.SetLastError(err)

		// Check if we have any active clients before attempting recovery
		clientCount := session.GetClientCount()
		if clientCount == 0 {
			logger.Log.Info().
				Str("channel_id", channelIDStr).
				Msg("FFmpeg crashed with no active clients, cleaning up session")

			// Clean up session
			m.sessionManager.Delete(channelIDStr)
			m.sessionManager.DeleteCircuitBreaker(channelIDStr)
			return
		}

		// Classify the error
		streamErr := ClassifyError(err)

		// Attempt recovery only if we have active clients
		ctx := context.Background()
		if recoveryErr := m.attemptRecovery(ctx, channelID, streamErr); recoveryErr != nil {
			logger.Log.Error().
				Err(recoveryErr).
				Str("channel_id", channelIDStr).
				Str("error_type", streamErr.Type.String()).
				Msg("Failed to recover from FFmpeg crash")

			// If recovery failed, set state to failed
			if session, ok := m.sessionManager.Get(channelIDStr); ok {
				session.SetState(StateFailed.String())
			}
		}
	} else {
		// Process exited cleanly - video finished playing
		logger.Log.Info().
			Str("channel_id", channelIDStr).
			Int("ffmpeg_pid", session.GetFFmpegPID()).
			Int64("process_runtime_ms", processRuntimeMs).
			Time("process_exit_time", processExitTime).
			Msg("FFmpeg process completed video successfully")

		// Check if we have any active clients
		clientCount := session.GetClientCount()
		if clientCount == 0 {
			logger.Log.Info().
				Str("channel_id", channelIDStr).
				Msg("No active clients, cleaning up stream")

			// Clean up session
			m.sessionManager.Delete(channelIDStr)
			m.sessionManager.DeleteCircuitBreaker(channelIDStr)
			return
		}

		// Advance to next video in playlist for active clients
		logger.Log.Info().
			Str("channel_id", channelIDStr).
			Int("client_count", clientCount).
			Msg("Video finished, advancing to next video")

		ctx := context.Background()
		if err := m.advanceToNextVideo(ctx, channelID); err != nil {
			logger.Log.Error().
				Err(err).
				Str("channel_id", channelIDStr).
				Msg("Failed to advance to next video")

			if session, ok := m.sessionManager.Get(channelIDStr); ok {
				session.SetState(StateFailed.String())
			}
		}
	}
}

// ensurePlaylistManager ensures a playlist manager exists for a quality
func (m *StreamManager) ensurePlaylistManager(session *models.StreamSession, quality, qualityDir string) error {
	channelIDStr := session.ChannelID.String()
	managerKey := fmt.Sprintf("%s_%s", channelIDStr, quality)

	// Check if playlist manager already exists
	m.playlistManagersMu.RLock()
	if pm, exists := m.playlistManagers[managerKey]; exists {
		m.playlistManagersMu.RUnlock()
		// Update session quality with playlist path
		playlistPath := filepath.Join(qualityDir, fmt.Sprintf("%s.m3u8", quality))
		qualities := session.GetQualities()
		for i := range qualities {
			if qualities[i].Level == quality {
				qualities[i].PlaylistPath = playlistPath
				break
			}
		}
		session.SetQualities(qualities)
		logger.Log.Debug().
			Str("channel_id", channelIDStr).
			Str("quality", quality).
			Msg("Playlist manager already exists")
		_ = pm // Use existing manager
		return nil
	}
	m.playlistManagersMu.RUnlock()

	// Create playlist manager
	// Use window size 0 to disable sliding window and keep all segments (for debugging)
	playlistPath := filepath.Join(qualityDir, fmt.Sprintf("%s.m3u8", quality))
	windowSize := uint(0) // 0 = VOD/EVENT mode (no sliding window, keep all segments)
	segmentDuration := float64(m.config.StreamSegmentDuration)

	pm, err := playlist.NewManager(windowSize, playlistPath, segmentDuration)
	if err != nil {
		return fmt.Errorf("failed to create playlist manager: %w", err)
	}

	// Store playlist manager
	m.playlistManagersMu.Lock()
	m.playlistManagers[managerKey] = pm
	m.playlistManagersMu.Unlock()

	// Update session quality with playlist path
	qualities := session.GetQualities()
	for i := range qualities {
		if qualities[i].Level == quality {
			qualities[i].PlaylistPath = playlistPath
			break
		}
	}
	session.SetQualities(qualities)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("quality", quality).
		Str("segment_dir", qualityDir).
		Str("playlist_path", playlistPath).
		Msg("Playlist manager initialized")

	return nil
}

// getPlaylistManager retrieves the playlist manager for a quality
func (m *StreamManager) getPlaylistManager(session *models.StreamSession, quality string) (playlist.Manager, error) {
	channelIDStr := session.ChannelID.String()
	managerKey := fmt.Sprintf("%s_%s", channelIDStr, quality)

	m.playlistManagersMu.RLock()
	pm, exists := m.playlistManagers[managerKey]
	m.playlistManagersMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("playlist manager not found for %s", managerKey)
	}

	return pm, nil
}

// generateMasterPlaylist generates the HLS master playlist file for a stream
func (m *StreamManager) generateMasterPlaylist(outputDir string, qualities []models.StreamQuality) error {
	// Convert StreamQuality to PlaylistVariant
	variants := make([]PlaylistVariant, 0, len(qualities))
	for _, q := range qualities {
		bandwidth, err := GetBandwidthForQuality(q.Level)
		if err != nil {
			return fmt.Errorf("failed to get bandwidth for quality %s: %w", q.Level, err)
		}

		resolution, err := GetResolutionForQuality(q.Level)
		if err != nil {
			return fmt.Errorf("failed to get resolution for quality %s: %w", q.Level, err)
		}

		variants = append(variants, PlaylistVariant{
			Bandwidth:  bandwidth,
			Resolution: resolution,
			Path:       fmt.Sprintf("%s.m3u8", q.Level),
		})
	}

	// Generate master playlist content
	content, err := GenerateMasterPlaylist(variants)
	if err != nil {
		return fmt.Errorf("failed to generate master playlist: %w", err)
	}

	// Write to file
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")
	if err := WritePlaylistAtomic(masterPlaylistPath, content); err != nil {
		return fmt.Errorf("failed to write master playlist: %w", err)
	}

	logger.Log.Debug().
		Str("path", masterPlaylistPath).
		Int("variants", len(variants)).
		Msg("Master playlist generated")

	return nil
}

// advanceToNextVideo stops the current stream and starts the next video in the playlist
func (m *StreamManager) advanceToNextVideo(ctx context.Context, channelID uuid.UUID) error {
	channelIDStr := channelID.String()

	// Get current session to preserve client count
	session, ok := m.sessionManager.Get(channelIDStr)
	if !ok {
		return fmt.Errorf("session not found for channel %s", channelIDStr)
	}

	clientCount := session.GetClientCount()

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Int("client_count", clientCount).
		Msg("Advancing to next video in playlist")

	// Stop current stream (cleans up FFmpeg process and files)
	if err := m.StopStream(ctx, channelID); err != nil {
		logger.Log.Warn().
			Err(err).
			Str("channel_id", channelIDStr).
			Msg("Error stopping stream before advancing (continuing anyway)")
	}

	// Start new stream (timeline service will calculate current position)
	// This automatically handles looping back to first video if at end
	newSession, err := m.StartStream(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to start next video: %w", err)
	}

	// Restore client count from previous session
	// (StartStream creates session with 0 clients)
	for i := 0; i < clientCount; i++ {
		newSession.IncrementClients()
	}
	newSession.UpdateLastAccess()

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Int("client_count", clientCount).
		Msg("Successfully advanced to next video")

	return nil
}
