package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/models"
	"gorm.io/gorm"
)

// PlaylistItemRepository handles database operations for playlist items
type PlaylistItemRepository struct {
	db *DB
}

// NewPlaylistItemRepository creates a new playlist item repository
func NewPlaylistItemRepository(db *DB) *PlaylistItemRepository {
	return &PlaylistItemRepository{db: db}
}

// ReorderItem represents a playlist item position update
type ReorderItem struct {
	ID       uuid.UUID
	Position int
}

// Create inserts a new playlist item into the database
func (r *PlaylistItemRepository) Create(ctx context.Context, item *models.PlaylistItem) error {
	result := r.db.WithContext(ctx).Create(item)
	if result.Error != nil {
		return fmt.Errorf("failed to create playlist item: %w", MapGormError(result.Error))
	}
	return nil
}

// GetByID retrieves a playlist item by its UUID
func (r *PlaylistItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PlaylistItem, error) {
	var item models.PlaylistItem
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&item)
	if result.Error != nil {
		return nil, MapGormError(result.Error)
	}
	return &item, nil
}

// GetByChannelID retrieves all playlist items for a channel, ordered by position
func (r *PlaylistItemRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.PlaylistItem, error) {
	var items []*models.PlaylistItem
	result := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID.String()).
		Order("position ASC").
		Find(&items)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get playlist items by channel: %w", MapGormError(result.Error))
	}
	return items, nil
}

// GetWithMedia retrieves playlist items for a channel with joined media data
func (r *PlaylistItemRepository) GetWithMedia(ctx context.Context, channelID uuid.UUID) ([]*models.PlaylistItem, error) {
	var items []*models.PlaylistItem
	result := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID.String()).
		Preload("Media").
		Order("position ASC").
		Find(&items)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get playlist items with media: %w", MapGormError(result.Error))
	}
	return items, nil
}

// Delete deletes a playlist item by its UUID
func (r *PlaylistItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).Delete(&models.PlaylistItem{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete playlist item: %w", MapGormError(result.Error))
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteByChannelID deletes all playlist items for a channel
func (r *PlaylistItemRepository) DeleteByChannelID(ctx context.Context, channelID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("channel_id = ?", channelID.String()).Delete(&models.PlaylistItem{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete playlist items by channel: %w", MapGormError(result.Error))
	}
	return nil
}

// Reorder updates positions for multiple playlist items in a transaction
func (r *PlaylistItemRepository) Reorder(ctx context.Context, channelID uuid.UUID, items []ReorderItem) error {
	return r.db.WithTransaction(ctx, func(tx *gorm.DB) error {
		for _, item := range items {
			result := tx.Model(&models.PlaylistItem{}).
				Where("id = ? AND channel_id = ?", item.ID.String(), channelID.String()).
				Update("position", item.Position)
			if result.Error != nil {
				return fmt.Errorf("failed to update position for item %s: %w", item.ID, MapGormError(result.Error))
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("playlist item %s not found or does not belong to channel", item.ID)
			}
		}
		return nil
	})
}
