package models

import (
	"time"

	"github.com/google/uuid"
)

// Channel represents a TV channel entity
type Channel struct {
	ID        uuid.UUID `json:"id" gorm:"type:text;primaryKey;column:id"`
	Name      string    `json:"name" gorm:"type:text;not null;column:name" validate:"required,min=1,max=255"`
	Icon      *string   `json:"icon,omitempty" gorm:"type:text;column:icon"`
	StartTime time.Time `json:"start_time" gorm:"type:datetime;not null;column:start_time" validate:"required"`
	Loop      bool      `json:"loop" gorm:"type:integer;not null;default:0;column:loop"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP;column:updated_at"`
}

// NewChannel creates a new Channel with generated UUID and timestamps
func NewChannel(name string, startTime time.Time, loop bool) *Channel {
	now := time.Now().UTC()
	return &Channel{
		ID:        uuid.New(),
		Name:      name,
		StartTime: startTime,
		Loop:      loop,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
