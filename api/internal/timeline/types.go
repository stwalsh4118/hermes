package timeline

import (
	"time"

	"github.com/google/uuid"
)

// TimelinePosition represents the current playback state of a channel at a given moment.
// It describes which media item should be playing and at what position within that item.
//
//nolint:revive // Timeline prefix is intentional and matches PRD specification
type TimelinePosition struct {
	// MediaID is the UUID of the currently playing media item
	MediaID uuid.UUID `json:"media_id"`

	// MediaTitle is the title of the currently playing media for display purposes
	MediaTitle string `json:"media_title"`

	// OffsetSeconds is the playback position within the current media item (in seconds)
	OffsetSeconds int64 `json:"offset_seconds"`

	// StartedAt is the timestamp when the current media item started playing
	StartedAt time.Time `json:"started_at"`

	// EndsAt is the timestamp when the current media item will finish playing
	EndsAt time.Time `json:"ends_at"`

	// Duration is the total duration of the current media item (in seconds)
	Duration int64 `json:"duration"`
}

// TimelineState represents the various states a channel's timeline can be in
//
//nolint:revive // Timeline prefix is intentional and matches PRD specification
type TimelineState string

const (
	// TimelineStateNotStarted indicates the channel hasn't started broadcasting yet
	// (current time is before the channel's start time)
	TimelineStateNotStarted TimelineState = "not_started"

	// TimelineStatePlaying indicates the channel is actively playing content
	TimelineStatePlaying TimelineState = "playing"

	// TimelineStateFinished indicates a non-looping channel has completed its playlist
	TimelineStateFinished TimelineState = "finished"

	// TimelineStateEmpty indicates the channel has no playlist items
	TimelineStateEmpty TimelineState = "empty"
)
