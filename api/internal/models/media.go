package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Media represents a media file metadata entity
type Media struct {
	ID         uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
	FilePath   string    `json:"file_path" gorm:"type:text;not null;uniqueIndex;column:file_path" validate:"required"`
	Title      string    `json:"title" gorm:"type:text;not null;column:title" validate:"required"`
	ShowName   *string   `json:"show_name,omitempty" gorm:"type:text;column:show_name"`
	Season     *int      `json:"season,omitempty" gorm:"type:integer;column:season"`
	Episode    *int      `json:"episode,omitempty" gorm:"type:integer;column:episode"`
	Duration   int64     `json:"duration" gorm:"type:integer;not null;column:duration" validate:"required,gt=0"` // seconds
	VideoCodec *string   `json:"video_codec,omitempty" gorm:"type:text;column:video_codec"`
	AudioCodec *string   `json:"audio_codec,omitempty" gorm:"type:text;column:audio_codec"`
	Resolution *string   `json:"resolution,omitempty" gorm:"type:text;column:resolution"`
	FileSize   *int64    `json:"file_size,omitempty" gorm:"type:integer;column:file_size"`
	CreatedAt  time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
}

// NewMedia creates a new Media with generated UUID and timestamp
func NewMedia(filePath, title string, duration int64) *Media {
	return &Media{
		ID:        uuid.New(),
		FilePath:  filePath,
		Title:     title,
		Duration:  duration,
		CreatedAt: time.Now().UTC(),
	}
}

// DurationString returns duration in HH:MM:SS format
func (m *Media) DurationString() string {
	hours := m.Duration / 3600
	minutes := (m.Duration % 3600) / 60
	seconds := m.Duration % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
