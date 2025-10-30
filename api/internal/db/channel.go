// Package db provides database connection management and repository interfaces.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/models"
)

// ChannelRepository handles database operations for channels
type ChannelRepository struct {
	db *DB
}

// NewChannelRepository creates a new channel repository
func NewChannelRepository(db *DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// Create inserts a new channel into the database
func (r *ChannelRepository) Create(ctx context.Context, channel *models.Channel) error {
	result := r.db.WithContext(ctx).Create(channel)
	if result.Error != nil {
		return fmt.Errorf("failed to create channel: %w", MapGormError(result.Error))
	}
	return nil
}

// GetByID retrieves a channel by its UUID
func (r *ChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	var channel models.Channel
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&channel)
	if result.Error != nil {
		return nil, MapGormError(result.Error)
	}
	return &channel, nil
}

// List retrieves all channels ordered by creation date (newest first)
func (r *ChannelRepository) List(ctx context.Context) ([]*models.Channel, error) {
	var channels []*models.Channel
	result := r.db.WithContext(ctx).Order("created_at DESC").Find(&channels)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list channels: %w", MapGormError(result.Error))
	}
	return channels, nil
}

// Update updates an existing channel
func (r *ChannelRepository) Update(ctx context.Context, channel *models.Channel) error {
	// Update the UpdatedAt timestamp
	channel.UpdatedAt = time.Now().UTC()

	// Use Select to explicitly update all fields including zero values
	result := r.db.WithContext(ctx).
		Where("id = ?", channel.ID.String()).
		Select("name", "icon", "start_time", "loop", "updated_at").
		Updates(channel)
	if result.Error != nil {
		return fmt.Errorf("failed to update channel: %w", MapGormError(result.Error))
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete deletes a channel by its UUID (cascade delete to playlist items)
func (r *ChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).Delete(&models.Channel{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete channel: %w", MapGormError(result.Error))
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
