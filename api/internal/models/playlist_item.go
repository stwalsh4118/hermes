package models

import (
	"time"

	"github.com/google/uuid"
)

// PlaylistItem represents a channel playlist entry
type PlaylistItem struct {
	ID        uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
	ChannelID uuid.UUID `json:"channel_id" gorm:"type:text;not null;column:channel_id" validate:"required"`
	MediaID   uuid.UUID `json:"media_id" gorm:"type:text;not null;column:media_id" validate:"required"`
	Position  int       `json:"position" gorm:"type:integer;not null;column:position" validate:"gte=0"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`

	// Populated by joins, not stored in database
	Media *Media `json:"media,omitempty" gorm:"-"`
}

// NewPlaylistItem creates a new PlaylistItem with generated UUID and timestamp
func NewPlaylistItem(channelID, mediaID uuid.UUID, position int) *PlaylistItem {
	return &PlaylistItem{
		ID:        uuid.New(),
		ChannelID: channelID,
		MediaID:   mediaID,
		Position:  position,
		CreatedAt: time.Now().UTC(),
	}
}
