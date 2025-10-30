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

// BulkAddItem represents a single item to be added in a bulk operation
type BulkAddItem struct {
	MediaID  uuid.UUID
	Position int
}

// BulkAddToPlaylist adds multiple media items to a playlist in a single transaction
// This is much more efficient than calling AddToPlaylist multiple times
func (s *PlaylistService) BulkAddToPlaylist(ctx context.Context, channelID uuid.UUID, items []BulkAddItem) ([]*models.PlaylistItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	// Validate positions are non-negative
	for i, item := range items {
		if item.Position < 0 {
			logger.Log.Warn().
				Int("index", i).
				Int("position", item.Position).
				Msg("Bulk add to playlist failed: invalid position")
			return nil, fmt.Errorf("failed to bulk add to playlist: position at index %d must be non-negative: %w", i, ErrInvalidPosition)
		}
	}

	// Validate channel exists once
	_, err := s.repos.Channels.GetByID(ctx, channelID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fmt.Errorf("failed to bulk add to playlist: %w", ErrChannelNotFound)
		}
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to validate channel existence for bulk add")
		return nil, fmt.Errorf("failed to bulk add to playlist: %w", err)
	}

	// Validate all media items exist
	mediaIDs := make([]uuid.UUID, len(items))
	for i, item := range items {
		mediaIDs[i] = item.MediaID
	}

	// Batch validate media existence with single query
	existsMap, err := s.repos.Media.ExistsByIDs(ctx, mediaIDs)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to batch validate media existence")
		return nil, fmt.Errorf("failed to bulk add to playlist: %w", err)
	}

	// Check if any media IDs don't exist
	for _, mediaID := range mediaIDs {
		if !existsMap[mediaID] {
			logger.Log.Warn().
				Str("media_id", mediaID.String()).
				Msg("Bulk add to playlist failed: media not found")
			return nil, fmt.Errorf("failed to bulk add to playlist: media %s not found: %w", mediaID.String(), ErrMediaNotFound)
		}
	}

	// Create all items within a single transaction
	var newItems []*models.PlaylistItem
	err = s.db.WithTransaction(ctx, func(tx *gorm.DB) error {
		// Build all items to insert
		now := time.Now().UTC()
		itemsToInsert := make([]*models.PlaylistItem, len(items))
		for i, item := range items {
			itemsToInsert[i] = &models.PlaylistItem{
				ID:        uuid.New(),
				ChannelID: channelID,
				MediaID:   item.MediaID,
				Position:  item.Position,
				CreatedAt: now,
			}
		}

		// Single batch INSERT with GORM
		if err := tx.Create(&itemsToInsert).Error; err != nil {
			return fmt.Errorf("failed to create playlist items: %w", err)
		}

		newItems = itemsToInsert
		return nil
	})

	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Int("item_count", len(items)).
			Msg("Failed to bulk add media to playlist")
		return nil, fmt.Errorf("failed to bulk add to playlist: %w", err)
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Int("item_count", len(newItems)).
		Msg("Media items bulk added to playlist successfully")

	return newItems, nil
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

// BulkRemoveFromPlaylist removes multiple playlist items in a single transaction
func (s *PlaylistService) BulkRemoveFromPlaylist(ctx context.Context, channelID uuid.UUID, itemIDs []uuid.UUID) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Validate all items exist and belong to this channel
	for _, itemID := range itemIDs {
		item, err := s.repos.PlaylistItems.GetByID(ctx, itemID)
		if err != nil {
			if db.IsNotFound(err) {
				return fmt.Errorf("failed to bulk remove: %w", ErrPlaylistItemNotFound)
			}
			return fmt.Errorf("failed to bulk remove: %w", err)
		}

		if item.ChannelID != channelID {
			return fmt.Errorf("failed to bulk remove: item %s does not belong to channel %s", itemID, channelID)
		}
	}

	// Delete all items and renumber positions in single transaction
	err := s.db.WithTransaction(ctx, func(tx *gorm.DB) error {
		// Batch delete all items in one query
		result := tx.Where("id IN ?", itemIDs).Delete(&models.PlaylistItem{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete items: %w", result.Error)
		}

		// Renumber remaining positions sequentially in a single SQL statement
		// Uses ROW_NUMBER() window function to assign new sequential positions
		result = tx.Exec(`
			UPDATE playlist_items 
			SET position = numbered.new_pos 
			FROM (
				SELECT id, ROW_NUMBER() OVER (ORDER BY position) - 1 AS new_pos
				FROM playlist_items 
				WHERE channel_id = ?
			) AS numbered
			WHERE playlist_items.id = numbered.id
		`, channelID.String())

		if result.Error != nil {
			return fmt.Errorf("failed to renumber positions: %w", result.Error)
		}

		return nil
	})

	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Int("item_count", len(itemIDs)).
			Msg("Failed to bulk remove from playlist")
		return fmt.Errorf("failed to bulk remove from playlist: %w", err)
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Int("item_count", len(itemIDs)).
		Msg("Items bulk removed from playlist successfully")

	return nil
}

// CalculateDuration calculates the total duration from a list of playlist items
func (s *PlaylistService) CalculateDuration(items []*models.PlaylistItem) int64 {
	// Handle empty playlist
	if len(items) == 0 {
		return 0
	}

	// Sum all media durations
	var totalDuration int64
	for _, item := range items {
		if item.Media != nil {
			totalDuration += item.Media.Duration
		}
	}

	return totalDuration
}
