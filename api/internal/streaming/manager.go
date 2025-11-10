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
	"github.com/stwalsh4118/hermes/internal/timeline"
)

// Common errors
var (
	ErrStreamNotFound      = errors.New("stream not found")
	ErrStreamAlreadyExists = errors.New("stream already exists")
	ErrManagerStopped      = errors.New("stream manager has been stopped")
)

const (
	defaultBatchTriggerInterval = 2 * time.Second
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

	// logger.Log.Debug().
	// 	Str("channel_id", channelIDStr).
	// 	Str("media_id", position.MediaID.String()).
	// 	Int64("offset_seconds", position.OffsetSeconds).
	// 	Msg("Timeline position calculated")

	// TESTING: Get first playlist item and start from 0
	playlistItems, err := m.repos.PlaylistItems.GetWithMedia(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist items: %w", err)
	}
	if len(playlistItems) == 0 {
		return nil, fmt.Errorf("no playlist items found for channel")
	}

	// Get the first media item
	firstItem := playlistItems[0]
	if firstItem.Media == nil {
		return nil, fmt.Errorf("first playlist item has no media")
	}

	offsetSeconds := int64(0) // Start from beginning for testing

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Str("media_id", firstItem.MediaID.String()).
		Int64("offset_seconds", offsetSeconds).
		Msg("TESTING: Starting from beginning of first playlist item")

	// Get the media file path
	media := firstItem.Media

	// Build output paths
	outputDir := fmt.Sprintf("%s/%s", m.config.SegmentPath, channelIDStr)
	quality := Quality1080p // Start with 1080p for now
	outputPath := GetOutputPath(outputDir, quality)

	// Create segment directories
	if err := createSegmentDirectories(outputDir, channelIDStr); err != nil {
		return nil, fmt.Errorf("failed to create segment directories: %w", err)
	}

	// Build FFmpeg command
	params := StreamParams{
		InputFile:       media.FilePath,
		OutputPath:      outputPath,
		Quality:         quality,
		HardwareAccel:   HardwareAccel(m.config.HardwareAccel),
		SeekSeconds:     offsetSeconds,
		SegmentDuration: m.config.SegmentDuration,
		PlaylistSize:    m.config.PlaylistSize,
		RealtimePacing:  m.config.RealtimePacing,
		EncodingPreset:  m.config.EncodingPreset,
	}

	ffmpegCmd, err := BuildHLSCommand(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build FFmpeg command: %w", err)
	}

	// Launch FFmpeg process
	execCmd, err := launchFFmpeg(ffmpegCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to launch FFmpeg: %w", err)
	}

	// Create stream session
	session := models.NewStreamSession(channelID)
	session.SetFFmpegPID(execCmd.Process.Pid)
	session.SetState(StateActive.String())
	session.SetOutputDir(outputDir)
	session.SetSegmentPath(fmt.Sprintf("%s/%s", outputDir, quality))
	session.UpdateLastAccess()

	// Set quality information
	qualities := []models.StreamQuality{
		{
			Level:        quality,
			Bitrate:      5000, // 1080p bitrate
			Resolution:   "1920x1080",
			SegmentPath:  session.GetSegmentPath(),
			PlaylistPath: outputPath,
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

	// Start monitoring FFmpeg process in background
	go m.monitorFFmpegProcess(channelID, execCmd)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("channel_name", channel.Name).
		Int("ffmpeg_pid", execCmd.Process.Pid).
		Str("quality", quality).
		Int64("seek_seconds", offsetSeconds).
		Msg("Stream started successfully")

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

	// Remove session from manager
	m.sessionManager.Delete(channelIDStr)

	// Remove circuit breaker
	m.sessionManager.DeleteCircuitBreaker(channelIDStr)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Msg("Stream stopped successfully")

	return nil
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

		// Check if next batch should be generated
		if session.ShouldGenerateNextBatch(m.config.TriggerThreshold) {
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
// This is a stub implementation - full logic will be implemented in task 12-6
func (m *StreamManager) generateNextBatch(ctx context.Context, session *models.StreamSession) error {
	currentBatch := session.GetCurrentBatch()
	if currentBatch == nil {
		logger.Log.Debug().
			Str("channel_id", session.ChannelID.String()).
			Msg("No current batch to continue from")
		return nil
	}

	logger.Log.Info().
		Str("channel_id", session.ChannelID.String()).
		Int("current_batch", currentBatch.BatchNumber).
		Int("furthest_segment", session.GetFurthestPosition()).
		Msg("Batch generation needed - stub implementation (full logic in task 12-6)")

	// Full implementation will be added in task 12-6
	return nil
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

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
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
