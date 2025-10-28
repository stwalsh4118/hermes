package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/models"
)

// MediaRepository handles database operations for media
type MediaRepository struct {
	db *DB
}

// NewMediaRepository creates a new media repository
func NewMediaRepository(db *DB) *MediaRepository {
	return &MediaRepository{db: db}
}

// Create inserts a new media item into the database
func (r *MediaRepository) Create(ctx context.Context, media *models.Media) error {
	result := r.db.WithContext(ctx).Create(media)
	if result.Error != nil {
		return fmt.Errorf("failed to create media: %w", MapGormError(result.Error))
	}
	return nil
}

// GetByID retrieves a media item by its UUID
func (r *MediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	var media models.Media
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&media)
	if result.Error != nil {
		return nil, MapGormError(result.Error)
	}
	return &media, nil
}

// GetByPath retrieves a media item by its file path (for duplicate checking)
func (r *MediaRepository) GetByPath(ctx context.Context, path string) (*models.Media, error) {
	var media models.Media
	result := r.db.WithContext(ctx).Where("file_path = ?", path).First(&media)
	if result.Error != nil {
		return nil, MapGormError(result.Error)
	}
	return &media, nil
}

// List retrieves all media items with pagination
func (r *MediaRepository) List(ctx context.Context, limit, offset int) ([]*models.Media, error) {
	var mediaList []*models.Media
	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	result := query.Find(&mediaList)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list media: %w", MapGormError(result.Error))
	}
	return mediaList, nil
}

// ListByShow retrieves media items filtered by show name with pagination
// Orders by season and episode with NULLs sorted last using COALESCE
func (r *MediaRepository) ListByShow(ctx context.Context, showName string, limit, offset int) ([]*models.Media, error) {
	var mediaList []*models.Media
	// Use COALESCE to sort NULLs last (SQLite sorts NULLs first by default)
	query := r.db.WithContext(ctx).
		Where("show_name = ?", showName).
		Order("COALESCE(season, 9999999) ASC, COALESCE(episode, 9999999) ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	result := query.Find(&mediaList)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list media by show: %w", MapGormError(result.Error))
	}
	return mediaList, nil
}

// Count returns the total number of media items
func (r *MediaRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Media{}).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count media: %w", MapGormError(result.Error))
	}
	return count, nil
}

// CountByShow returns the total number of media items for a specific show
func (r *MediaRepository) CountByShow(ctx context.Context, showName string) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Media{}).Where("show_name = ?", showName).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count media by show: %w", MapGormError(result.Error))
	}
	return count, nil
}

// Update updates an existing media item
// Note: Uses map-based updates to support setting fields to zero values
func (r *MediaRepository) Update(ctx context.Context, media *models.Media) error {
	// Build update map to support zero values
	updates := map[string]interface{}{
		"file_path":   media.FilePath,
		"title":       media.Title,
		"show_name":   media.ShowName,
		"season":      media.Season,
		"episode":     media.Episode,
		"duration":    media.Duration,
		"video_codec": media.VideoCodec,
		"audio_codec": media.AudioCodec,
		"resolution":  media.Resolution,
		"file_size":   media.FileSize,
	}

	result := r.db.WithContext(ctx).Model(&models.Media{}).Where("id = ?", media.ID.String()).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update media: %w", MapGormError(result.Error))
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete deletes a media item by its UUID
func (r *MediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).Delete(&models.Media{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete media: %w", MapGormError(result.Error))
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ExistsByIDs checks which media IDs exist in the database
// Returns a map where the key is the media ID and the value is true if it exists
func (r *MediaRepository) ExistsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]bool), nil
	}

	// Convert UUIDs to strings for the query
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	// Query for existing IDs
	var existingMedia []models.Media
	result := r.db.WithContext(ctx).Select("id").Where("id IN ?", idStrings).Find(&existingMedia)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to check media existence: %w", MapGormError(result.Error))
	}

	// Build existence map
	existsMap := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		existsMap[id] = false
	}
	for i := range existingMedia {
		existsMap[existingMedia[i].ID] = true
	}

	return existsMap, nil
}
