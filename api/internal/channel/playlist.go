package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
	"gorm.io/gorm"
)

// PlaylistService handles business logic for playlist operations
type PlaylistService struct {
	repos *db.Repositories
	db    *db.DB
}

// NewPlaylistService creates a new playlist service instance
func NewPlaylistService(database *db.DB, repos *db.Repositories) *PlaylistService {
	return &PlaylistService{
		repos: repos,
		db:    database,
	}
}

// AddToPlaylist adds a media item to a channel's playlist at a specific position
func (s *PlaylistService) AddToPlaylist(ctx context.Context, channelID, mediaID uuid.UUID, position int) (*models.PlaylistItem, error) {
	// Validate position is non-negative
	if position < 0 {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Str("media_id", mediaID.String()).
			Int("position", position).
			Msg("Add to playlist failed: invalid position")
		return nil, fmt.Errorf("failed to add media to playlist: %w", ErrInvalidPosition)
	}

	// Validate channel exists
	_, err := s.repos.Channels.GetByID(ctx, channelID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fmt.Errorf("failed to add media to playlist: %w", ErrChannelNotFound)
		}
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to validate channel existence")
		return nil, fmt.Errorf("failed to add media to playlist: %w", err)
	}

	// Validate media exists
	_, err = s.repos.Media.GetByID(ctx, mediaID)
	if err != nil {
		if db.IsNotFound(err) {
			logger.Log.Warn().
				Str("media_id", mediaID.String()).
				Msg("Add to playlist failed: media not found")
			return nil, fmt.Errorf("failed to add media to playlist: %w", ErrMediaNotFound)
		}
		logger.Log.Error().
			Err(err).
			Str("media_id", mediaID.String()).
			Msg("Failed to validate media existence")
		return nil, fmt.Errorf("failed to add media to playlist: %w", err)
	}

	// Create new playlist item within a transaction
	var newItem *models.PlaylistItem
	err = s.db.WithTransaction(ctx, func(tx *gorm.DB) error {
		// Shift existing items at or after the target position
		result := tx.Model(&models.PlaylistItem{}).
			Where("channel_id = ? AND position >= ?", channelID.String(), position).
			Update("position", gorm.Expr("position + 1"))
		if result.Error != nil {
			return fmt.Errorf("failed to shift playlist positions: %w", result.Error)
		}

		// Create the new item
		newItem = &models.PlaylistItem{
			ID:        uuid.New(),
			ChannelID: channelID,
			MediaID:   mediaID,
			Position:  position,
			CreatedAt: time.Now().UTC(),
		}

		if err := tx.Create(newItem).Error; err != nil {
			return fmt.Errorf("failed to create playlist item: %w", err)
		}

		return nil
	})

	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("media_id", mediaID.String()).
			Int("position", position).
			Msg("Failed to add media to playlist")
		return nil, fmt.Errorf("failed to add media to playlist: %w", err)
	}

	logger.Log.Info().
		Str("playlist_item_id", newItem.ID.String()).
		Str("channel_id", channelID.String()).
		Str("media_id", mediaID.String()).
		Int("position", position).
		Msg("Media added to playlist successfully")

	return newItem, nil
}

// RemoveFromPlaylist removes a playlist item and reorders remaining items
func (s *PlaylistService) RemoveFromPlaylist(ctx context.Context, itemID uuid.UUID) error {
	// Fetch the item to get its position and channel ID
	item, err := s.repos.PlaylistItems.GetByID(ctx, itemID)
	if err != nil {
		if db.IsNotFound(err) {
			logger.Log.Warn().
				Str("item_id", itemID.String()).
				Msg("Remove from playlist failed: item not found")
			return fmt.Errorf("failed to remove from playlist: %w", ErrPlaylistItemNotFound)
		}
		logger.Log.Error().
			Err(err).
			Str("item_id", itemID.String()).
			Msg("Failed to fetch playlist item")
		return fmt.Errorf("failed to remove from playlist: %w", err)
	}

	deletedPosition := item.Position
	channelID := item.ChannelID

	// Delete item and reorder within a transaction
	err = s.db.WithTransaction(ctx, func(tx *gorm.DB) error {
		// Delete the item
		result := tx.Where("id = ?", itemID.String()).Delete(&models.PlaylistItem{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete playlist item: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrPlaylistItemNotFound
		}

		// Shift down items after the deleted position
		result = tx.Model(&models.PlaylistItem{}).
			Where("channel_id = ? AND position > ?", channelID.String(), deletedPosition).
			Update("position", gorm.Expr("position - 1"))
		if result.Error != nil {
			return fmt.Errorf("failed to reorder playlist items: %w", result.Error)
		}

		return nil
	})

	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("item_id", itemID.String()).
			Str("channel_id", channelID.String()).
			Msg("Failed to remove from playlist")
		return fmt.Errorf("failed to remove from playlist: %w", err)
	}

	logger.Log.Info().
		Str("item_id", itemID.String()).
		Str("channel_id", channelID.String()).
		Int("position", deletedPosition).
		Msg("Item removed from playlist successfully")

	return nil
}

// ReorderPlaylist reorders multiple playlist items atomically
func (s *PlaylistService) ReorderPlaylist(ctx context.Context, channelID uuid.UUID, items []db.ReorderItem) error {
	// Validate all items belong to the same channel
	for _, item := range items {
		existingItem, err := s.repos.PlaylistItems.GetByID(ctx, item.ID)
		if err != nil {
			if db.IsNotFound(err) {
				logger.Log.Warn().
					Str("item_id", item.ID.String()).
					Msg("Reorder failed: playlist item not found")
				return fmt.Errorf("failed to reorder playlist: %w", ErrPlaylistItemNotFound)
			}
			logger.Log.Error().
				Err(err).
				Str("item_id", item.ID.String()).
				Msg("Failed to fetch playlist item for validation")
			return fmt.Errorf("failed to reorder playlist: %w", err)
		}

		if existingItem.ChannelID != channelID {
			logger.Log.Warn().
				Str("item_id", item.ID.String()).
				Str("expected_channel_id", channelID.String()).
				Str("actual_channel_id", existingItem.ChannelID.String()).
				Msg("Reorder failed: item does not belong to channel")
			return fmt.Errorf("failed to reorder playlist: item %s does not belong to channel %s", item.ID, channelID)
		}
	}

	// Use repository's reorder method (handles transaction)
	err := s.repos.PlaylistItems.Reorder(ctx, channelID, items)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Int("item_count", len(items)).
			Msg("Failed to reorder playlist")
		return fmt.Errorf("failed to reorder playlist: %w", err)
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Int("item_count", len(items)).
		Msg("Playlist reordered successfully")

	return nil
}

// GetPlaylist retrieves all playlist items for a channel with media details
func (s *PlaylistService) GetPlaylist(ctx context.Context, channelID uuid.UUID) ([]*models.PlaylistItem, error) {
	// Validate channel exists
	_, err := s.repos.Channels.GetByID(ctx, channelID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get playlist: %w", ErrChannelNotFound)
		}
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to validate channel existence")
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	// Get playlist items with media details
	items, err := s.repos.PlaylistItems.GetWithMedia(ctx, channelID)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to get playlist items")
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Int("item_count", len(items)).
		Msg("Retrieved playlist items")

	return items, nil
}

// CalculateDuration calculates the total duration of all media in a channel's playlist
func (s *PlaylistService) CalculateDuration(ctx context.Context, channelID uuid.UUID) (int64, error) {
	// Get playlist with media details
	items, err := s.GetPlaylist(ctx, channelID)
	if err != nil {
		return 0, err
	}

	// Handle empty playlist
	if len(items) == 0 {
		return 0, nil
	}

	// Sum all media durations
	var totalDuration int64
	for _, item := range items {
		if item.Media != nil {
			totalDuration += item.Media.Duration
		}
	}

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Int64("total_duration", totalDuration).
		Int("item_count", len(items)).
		Msg("Calculated playlist duration")

	return totalDuration, nil
}
