package models

import (
	"time"
)

// Settings represents system configuration
type Settings struct {
	ID               int       `json:"id" gorm:"type:integer;primaryKey;default:1;column:id"`
	MediaLibraryPath string    `json:"media_library_path" gorm:"type:text;not null;column:media_library_path" validate:"required"`
	TranscodeQuality string    `json:"transcode_quality" gorm:"type:text;default:medium;column:transcode_quality" validate:"oneof=high medium low"`
	HardwareAccel    string    `json:"hardware_accel" gorm:"type:text;default:none;column:hardware_accel" validate:"oneof=none nvenc qsv vaapi videotoolbox"`
	ServerPort       int       `json:"server_port" gorm:"type:integer;default:8080;column:server_port" validate:"gte=1,lte=65535"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:updated_at"`
}

// DefaultSettings returns settings with default values
func DefaultSettings() *Settings {
	return &Settings{
		ID:               1,
		MediaLibraryPath: "./media",
		TranscodeQuality: QualityMedium,
		HardwareAccel:    HardwareAccelNone,
		ServerPort:       8080,
		UpdatedAt:        time.Now().UTC(),
	}
}
