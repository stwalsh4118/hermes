package streaming

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
)

// Recovery configuration constants
const (
	// MinDiskSpaceBytes is the minimum required disk space (5GB)
	MinDiskSpaceBytes = 5 * 1024 * 1024 * 1024
	// WarnDiskSpaceBytes is the warning threshold for disk space (10GB)
	WarnDiskSpaceBytes = 10 * 1024 * 1024 * 1024
	// MaxRestartAttempts is the maximum number of restart attempts
	MaxRestartAttempts = 3
	// CircuitBreakerThreshold is the number of failures before circuit opens
	CircuitBreakerThreshold = 3
	// CircuitBreakerResetTimeout is how long to wait before trying again after circuit opens
	CircuitBreakerResetTimeout = 60 * time.Second
	// InitialBackoff is the initial backoff duration for retries
	InitialBackoff = 1 * time.Second
	// MaxBackoff is the maximum backoff duration
	MaxBackoff = 8 * time.Second
)

// calculateBackoffDuration calculates exponential backoff duration based on attempt count
func calculateBackoffDuration(attemptCount int) time.Duration {
	if attemptCount <= 0 {
		return InitialBackoff
	}

	// Exponential backoff: 1s, 2s, 4s, 8s
	backoff := InitialBackoff
	for i := 0; i < attemptCount && backoff < MaxBackoff; i++ {
		backoff *= 2
	}

	if backoff > MaxBackoff {
		return MaxBackoff
	}

	return backoff
}

// checkDiskSpace checks if there's sufficient disk space at the given path
func checkDiskSpace(path string) error {
	available, err := getAvailableSpace(path)
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	if available < MinDiskSpaceBytes {
		return fmt.Errorf("insufficient disk space: %d bytes available, %d bytes required",
			available, MinDiskSpaceBytes)
	}

	// Log warning if below warning threshold
	if available < WarnDiskSpaceBytes {
		logger.Log.Warn().
			Uint64("available_bytes", available).
			Uint64("warning_threshold", WarnDiskSpaceBytes).
			Str("path", path).
			Msg("Disk space below warning threshold")
	}

	return nil
}

// getAvailableSpace returns available disk space in bytes for the given path
// Platform-specific implementations are in recovery_unix.go and recovery_windows.go

// restartStream attempts to restart a stream with exponential backoff
func (m *StreamManager) restartStream(ctx context.Context, channelID uuid.UUID, reason string) error {
	channelIDStr := channelID.String()

	// Get existing session
	session, ok := m.sessionManager.Get(channelIDStr)
	if !ok {
		return fmt.Errorf("session not found for channel %s", channelIDStr)
	}

	// Get or create circuit breaker
	circuitBreaker := m.sessionManager.GetOrCreateCircuitBreaker(channelIDStr)

	// Check circuit breaker
	if !circuitBreaker.CanAttempt() {
		logger.Log.Error().
			Str("channel_id", channelIDStr).
			Str("circuit_state", circuitBreaker.GetState().String()).
			Msg("Circuit breaker is open, cannot restart stream")
		return ErrCircuitOpen
	}

	// Get restart count
	restartCount := session.GetRestartCount()

	// Check if we've exceeded max restart attempts
	if restartCount >= MaxRestartAttempts {
		logger.Log.Error().
			Str("channel_id", channelIDStr).
			Int("restart_count", restartCount).
			Int("max_attempts", MaxRestartAttempts).
			Msg("Exceeded maximum restart attempts")
		return fmt.Errorf("exceeded maximum restart attempts (%d)", MaxRestartAttempts)
	}

	// Calculate backoff duration
	backoff := calculateBackoffDuration(restartCount)

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("reason", reason).
		Int("restart_count", restartCount).
		Dur("backoff", backoff).
		Msg("Attempting stream restart")

	// Increment restart count
	session.IncrementRestartCount()

	// Wait for backoff period
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(backoff):
		// Continue with restart
	}

	// Stop current stream (cleanup resources)
	if err := m.StopStream(ctx, channelID); err != nil {
		logger.Log.Warn().
			Err(err).
			Str("channel_id", channelIDStr).
			Msg("Failed to stop stream before restart")
		// Continue anyway - we'll try to start fresh
	}

	// Start new stream
	newSession, err := m.StartStream(ctx, channelID)
	if err != nil {
		// Record failure in circuit breaker
		circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to restart stream: %w", err)
	}

	// Success - reset error count and record success
	newSession.ResetErrors()
	newSession.ResetRestartCount()
	circuitBreaker.RecordSuccess()

	// Discontinuities are handled automatically in generateNextBatch() and initializeFirstBatch()
	// when videos switch. If recovery restarts the stream, discontinuities are handled by
	// the batch generation logic.

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Int("restart_count", restartCount+1).
		Msg("Stream restarted successfully")

	return nil
}

// handleFileError handles missing or corrupt file errors by skipping to next playlist item
func (m *StreamManager) handleFileError(ctx context.Context, channelID uuid.UUID, filePath string, errorType ErrorType) error {
	channelIDStr := channelID.String()

	logger.Log.Warn().
		Str("channel_id", channelIDStr).
		Str("file_path", filePath).
		Str("error_type", errorType.String()).
		Msg("Handling file error, attempting to skip to next item")

	// Get channel and playlist
	channel, err := m.repos.Channels.GetByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	playlist, err := m.repos.PlaylistItems.GetByChannelID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get playlist: %w", err)
	}

	if len(playlist) == 0 {
		return fmt.Errorf("playlist is empty for channel %s", channelIDStr)
	}

	// Find current position in playlist
	currentPosition := -1
	for i, item := range playlist {
		if item.Media != nil && item.Media.FilePath == filePath {
			currentPosition = i
			break
		}
	}

	// Get next playlist items
	nextItems := GetNextPlaylistItems(playlist, currentPosition, 5, channel.Loop)
	if len(nextItems) == 0 {
		if !channel.Loop {
			return ErrPlaylistEnded
		}
		return fmt.Errorf("no valid items found in playlist")
	}

	// Find first valid (accessible) file
	var validItem *models.PlaylistItem
	for _, item := range nextItems {
		if item.Media != nil {
			// TODO: Add file existence check here
			validItem = item
			break
		}
	}

	if validItem == nil {
		return fmt.Errorf("no valid media items found in playlist")
	}

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("skipped_file", filePath).
		Str("next_file", validItem.Media.FilePath).
		Msg("Skipping to next valid playlist item")

	// Restart stream will pick up from current timeline position
	// The timeline service will calculate the next position
	return m.restartStream(ctx, channelID, "file error - skipping to next item")
}

// fallbackToSoftwareEncoding disables hardware acceleration and restarts with software encoding
func (m *StreamManager) fallbackToSoftwareEncoding(ctx context.Context, channelID uuid.UUID) error {
	channelIDStr := channelID.String()

	logger.Log.Warn().
		Str("channel_id", channelIDStr).
		Msg("Hardware encoder failed, falling back to software encoding")

	// Get session
	session, ok := m.sessionManager.Get(channelIDStr)
	if !ok {
		return fmt.Errorf("session not found for channel %s", channelIDStr)
	}

	// Mark hardware acceleration as failed in session
	session.SetHardwareAccelFailed(true)

	// Update global config to use software encoding for future streams
	m.mu.Lock()
	previousHwAccel := m.config.HardwareAccel
	m.config.HardwareAccel = HardwareAccelNone.String()
	m.mu.Unlock()

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("previous_hw_accel", previousHwAccel).
		Msg("Updated configuration to use software encoding")

	// Restart stream with software encoding
	return m.restartStream(ctx, channelID, "hardware encoder failed - using software fallback")
}

// attemptRecovery routes errors to appropriate recovery handlers
func (m *StreamManager) attemptRecovery(ctx context.Context, channelID uuid.UUID, streamErr *StreamError) error {
	channelIDStr := channelID.String()

	// Check if error is recoverable
	if !streamErr.Recoverable {
		logger.Log.Error().
			Str("channel_id", channelIDStr).
			Str("error_type", streamErr.Type.String()).
			Str("severity", streamErr.Severity.String()).
			Msg("Error is not recoverable")
		return streamErr
	}

	logger.Log.Info().
		Str("channel_id", channelIDStr).
		Str("error_type", streamErr.Type.String()).
		Msg("Attempting error recovery")

	// Route to appropriate recovery strategy based on error type
	switch streamErr.Type {
	case ErrorTypeFFmpegCrash:
		return m.restartStream(ctx, channelID, "FFmpeg crash")

	case ErrorTypeFileMissing, ErrorTypeFileCorrupt:
		// Extract file path from error if available
		filePath := ""
		// For now, restart stream and let timeline service handle it
		return m.handleFileError(ctx, channelID, filePath, streamErr.Type)

	case ErrorTypeHardwareEncoder:
		return m.fallbackToSoftwareEncoding(ctx, channelID)

	case ErrorTypeTimeout:
		return m.restartStream(ctx, channelID, "timeout")

	default:
		// Generic restart for unknown recoverable errors
		return m.restartStream(ctx, channelID, fmt.Sprintf("error: %s", streamErr.Type.String()))
	}
}
