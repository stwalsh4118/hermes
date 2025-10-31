package streaming

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
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

// StreamManager orchestrates the entire streaming pipeline
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

// NewStreamManager creates a new stream manager instance
func NewStreamManager(
	repos *db.Repositories,
	timelineService *timeline.TimelineService,
	cfg *config.StreamingConfig,
) *StreamManager {
	return &StreamManager{
		repos:           repos,
		timelineService: timelineService,
		sessionManager:  NewSessionManager(),
		config:          cfg,
		stopChan:        make(chan struct{}),
		cleanupDone:     make(chan struct{}),
		stopped:         false,
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

	logger.Log.Info().
		Int("cleanup_interval_seconds", m.config.CleanupInterval).
		Int("grace_period_seconds", m.config.GracePeriodSeconds).
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

	// Wait for cleanup goroutine to finish
	<-m.cleanupDone

	// Stop cleanup ticker
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
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

	// Get current timeline position
	position, err := m.timelineService.GetCurrentPosition(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline position: %w", err)
	}

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Str("media_id", position.MediaID.String()).
		Int64("offset_seconds", position.OffsetSeconds).
		Msg("Timeline position calculated")

	// Get the media file path
	media, err := m.repos.Media.GetByID(ctx, position.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	// Build output paths
	outputDir := fmt.Sprintf("%s/%s", m.config.SegmentPath, channelIDStr)
	quality := Quality1080p // Start with 1080p for now
	outputPath := GetOutputPath(outputDir, channelIDStr, quality)

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
		SeekSeconds:     position.OffsetSeconds,
		SegmentDuration: m.config.SegmentDuration,
		PlaylistSize:    m.config.PlaylistSize,
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

	// Start monitoring FFmpeg process in background
	go m.monitorFFmpegProcess(channelID, execCmd)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("channel_name", channel.Name).
		Int("ffmpeg_pid", execCmd.Process.Pid).
		Str("quality", quality).
		Int64("seek_seconds", position.OffsetSeconds).
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

		// Classify the error
		streamErr := ClassifyError(err)

		// Attempt recovery
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
		// Process exited cleanly but unexpectedly
		logger.Log.Warn().
			Str("channel_id", channelIDStr).
			Msg("FFmpeg process exited cleanly but unexpectedly")

		// Treat as a crash and attempt restart
		streamErr := NewStreamError(ErrorTypeFFmpegCrash, "FFmpeg exited unexpectedly", nil)

		ctx := context.Background()
		if recoveryErr := m.attemptRecovery(ctx, channelID, streamErr); recoveryErr != nil {
			logger.Log.Error().
				Err(recoveryErr).
				Str("channel_id", channelIDStr).
				Msg("Failed to restart stream after unexpected exit")

			if session, ok := m.sessionManager.Get(channelIDStr); ok {
				session.SetState(StateFailed.String())
			}
		}
	}
}
