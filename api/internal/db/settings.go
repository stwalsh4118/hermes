package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/stwalsh4118/hermes/internal/models"
)

// SettingsRepository handles database operations for settings
// Settings is a singleton table with only one row
type SettingsRepository struct {
	db *DB
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves the settings (creates with defaults if not exists)
func (r *SettingsRepository) Get(ctx context.Context) (*models.Settings, error) {
	var settings models.Settings
	result := r.db.WithContext(ctx).Where("id = ?", 1).First(&settings)

	// If not found, create with defaults
	if result.Error != nil {
		if errors.Is(MapGormError(result.Error), ErrNotFound) {
			defaultSettings := models.DefaultSettings()
			if err := r.db.WithContext(ctx).Create(defaultSettings).Error; err != nil {
				return nil, fmt.Errorf("failed to create default settings: %w", MapGormError(err))
			}
			return defaultSettings, nil
		}
		return nil, MapGormError(result.Error)
	}

	return &settings, nil
}

// Update updates the settings (singleton row)
func (r *SettingsRepository) Update(ctx context.Context, settings *models.Settings) error {
	// Ensure we're always updating the singleton row
	settings.ID = 1
	settings.UpdatedAt = time.Now().UTC()

	result := r.db.WithContext(ctx).Where("id = ?", 1).Updates(settings)
	if result.Error != nil {
		return fmt.Errorf("failed to update settings: %w", MapGormError(result.Error))
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
