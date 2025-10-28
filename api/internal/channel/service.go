package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
)

// Maximum allowed future time for channel start time (1 year)
const maxStartTimeFuture = 365 * 24 * time.Hour

// ChannelService handles business logic for channel operations
type ChannelService struct {
	repos *db.Repositories
}

// NewChannelService creates a new channel service instance
func NewChannelService(repos *db.Repositories) *ChannelService {
	return &ChannelService{
		repos: repos,
	}
}

// CreateChannel creates a new channel with validation
func (s *ChannelService) CreateChannel(ctx context.Context, name string, icon *string, startTime time.Time, loop bool) (*models.Channel, error) {
	// Validate name uniqueness
	if err := s.validateNameUniqueness(ctx, name, uuid.Nil); err != nil {
		logger.Log.Warn().
			Str("name", name).
			Msg("Channel creation failed: duplicate name")
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Validate start time
	if err := s.validateStartTime(startTime); err != nil {
		logger.Log.Warn().
			Time("start_time", startTime).
			Msg("Channel creation failed: invalid start time")
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Create channel model
	now := time.Now().UTC()
	channel := &models.Channel{
		ID:        uuid.New(),
		Name:      name,
		Icon:      icon,
		StartTime: startTime.UTC(),
		Loop:      loop,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save to database
	if err := s.repos.Channels.Create(ctx, channel); err != nil {
		logger.Log.Error().
			Err(err).
			Str("name", name).
			Msg("Failed to create channel in database")
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	logger.Log.Info().
		Str("channel_id", channel.ID.String()).
		Str("name", channel.Name).
		Msg("Channel created successfully")

	return channel, nil
}

// GetByID retrieves a channel by its ID
func (s *ChannelService) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	channel, err := s.repos.Channels.GetByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, ErrChannelNotFound
		}
		logger.Log.Error().
			Err(err).
			Str("channel_id", id.String()).
			Msg("Failed to get channel by ID")
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return channel, nil
}

// List retrieves all channels
func (s *ChannelService) List(ctx context.Context) ([]*models.Channel, error) {
	channels, err := s.repos.Channels.List(ctx)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to list channels")
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	logger.Log.Debug().
		Int("count", len(channels)).
		Msg("Listed channels")

	return channels, nil
}

// UpdateChannel updates an existing channel with validation
func (s *ChannelService) UpdateChannel(ctx context.Context, channel *models.Channel) error {
	// Load existing channel
	existing, err := s.GetByID(ctx, channel.ID)
	if err != nil {
		return err
	}

	// Validate name uniqueness if name changed
	if !strings.EqualFold(existing.Name, channel.Name) {
		if err := s.validateNameUniqueness(ctx, channel.Name, channel.ID); err != nil {
			logger.Log.Warn().
				Str("channel_id", channel.ID.String()).
				Str("name", channel.Name).
				Msg("Channel update failed: duplicate name")
			return fmt.Errorf("failed to update channel: %w", err)
		}
	}

	// Validate start time if changed
	if !existing.StartTime.Equal(channel.StartTime) {
		if err := s.validateStartTime(channel.StartTime); err != nil {
			logger.Log.Warn().
				Str("channel_id", channel.ID.String()).
				Time("start_time", channel.StartTime).
				Msg("Channel update failed: invalid start time")
			return fmt.Errorf("failed to update channel: %w", err)
		}
	}

	// Update timestamp
	channel.UpdatedAt = time.Now().UTC()

	// Save to database
	if err := s.repos.Channels.Update(ctx, channel); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channel.ID.String()).
			Msg("Failed to update channel in database")
		return fmt.Errorf("failed to update channel: %w", err)
	}

	logger.Log.Info().
		Str("channel_id", channel.ID.String()).
		Str("name", channel.Name).
		Msg("Channel updated successfully")

	return nil
}

// DeleteChannel deletes a channel by its ID
func (s *ChannelService) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	// Verify channel exists
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}

	// Delete from database (cascade to playlist items handled by DB)
	if err := s.repos.Channels.Delete(ctx, id); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", id.String()).
			Msg("Failed to delete channel from database")
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	logger.Log.Info().
		Str("channel_id", id.String()).
		Msg("Channel deleted successfully")

	return nil
}

// HasEmptyPlaylist checks if a channel has an empty playlist
func (s *ChannelService) HasEmptyPlaylist(ctx context.Context, channelID uuid.UUID) (bool, error) {
	items, err := s.repos.PlaylistItems.GetByChannelID(ctx, channelID)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to check playlist items")
		return false, fmt.Errorf("failed to check playlist: %w", err)
	}

	isEmpty := len(items) == 0

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Bool("is_empty", isEmpty).
		Int("item_count", len(items)).
		Msg("Checked channel playlist")

	return isEmpty, nil
}

// validateNameUniqueness checks if a channel name is unique (case-insensitive)
// excludeID allows excluding a specific channel ID (for updates)
func (s *ChannelService) validateNameUniqueness(ctx context.Context, name string, excludeID uuid.UUID) error {
	channels, err := s.repos.Channels.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate name uniqueness: %w", err)
	}

	nameLower := strings.ToLower(strings.TrimSpace(name))

	for _, channel := range channels {
		// Skip the channel being updated
		if channel.ID == excludeID {
			continue
		}

		existingNameLower := strings.ToLower(strings.TrimSpace(channel.Name))
		if existingNameLower == nameLower {
			return ErrDuplicateChannelName
		}
	}

	return nil
}

// validateStartTime checks if the start time is not more than 1 year in the future
func (s *ChannelService) validateStartTime(startTime time.Time) error {
	maxAllowed := time.Now().UTC().Add(maxStartTimeFuture)
	if startTime.After(maxAllowed) {
		return ErrInvalidStartTime
	}
	return nil
}
