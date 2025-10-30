// Package timeline provides business logic for timeline calculation operations.
package timeline

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/channel"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
)

// TimelineService handles business logic for timeline calculation operations
//
//nolint:revive // Service name matches established patterns in codebase
type TimelineService struct {
	repos *db.Repositories
}

// NewTimelineService creates a new timeline service instance
func NewTimelineService(repos *db.Repositories) *TimelineService {
	return &TimelineService{
		repos: repos,
	}
}

// GetCurrentPosition calculates and returns the current timeline position for a channel.
// It fetches the channel and its playlist from the database, then uses the calculator
// to determine what should be playing at the current moment.
//
// Returns:
//   - TimelinePosition: The current playback position with all fields populated
//   - error: channel.ErrChannelNotFound, ErrChannelNotStarted, ErrEmptyPlaylist,
//     ErrPlaylistFinished, or wrapped database errors
func (s *TimelineService) GetCurrentPosition(ctx context.Context, channelID uuid.UUID) (*TimelinePosition, error) {
	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Msg("Starting timeline calculation")

	// Fetch channel from database
	ch, err := s.repos.Channels.GetByID(ctx, channelID)
	if err != nil {
		if db.IsNotFound(err) {
			logger.Log.Warn().
				Str("channel_id", channelID.String()).
				Msg("Timeline calculation failed: channel not found")
			return nil, channel.ErrChannelNotFound
		}
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to fetch channel from database")
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Fetch playlist with media details
	playlist, err := s.repos.PlaylistItems.GetWithMedia(ctx, channelID)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to fetch playlist from database")
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	// Validate playlist is not empty
	if len(playlist) == 0 {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Msg("Timeline calculation failed: empty playlist")
		return nil, ErrEmptyPlaylist
	}

	// Calculate current position using the calculator
	currentTime := time.Now().UTC()
	position, err := CalculatePosition(ch.StartTime, currentTime, playlist, ch.Loop)
	if err != nil {
		// Calculator errors are already well-defined, pass them through
		logger.Log.Warn().
			Err(err).
			Str("channel_id", channelID.String()).
			Time("start_time", ch.StartTime).
			Bool("loop", ch.Loop).
			Int("playlist_items", len(playlist)).
			Msg("Timeline calculation failed")
		return nil, err
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Str("media_id", position.MediaID.String()).
		Str("media_title", position.MediaTitle).
		Int64("offset_seconds", position.OffsetSeconds).
		Int64("duration", position.Duration).
		Msg("Timeline calculation successful")

	return position, nil
}
