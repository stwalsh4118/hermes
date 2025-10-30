// Package timeline provides calculations for determining what should be playing
// on a channel at any given moment, creating the illusion of a continuously
// broadcasting television channel.
package timeline

import (
	"time"

	"github.com/stwalsh4118/hermes/internal/models"
)

// CalculatePosition calculates the current timeline position for a channel.
// This is a pure function with no I/O - it takes all required data as parameters
// and returns the current playback position or an error for edge cases.
//
// Parameters:
//   - startTime: When the channel started broadcasting (UTC)
//   - currentTime: The time to calculate position for (UTC)
//   - playlist: Ordered list of playlist items with populated Media field
//   - loop: Whether the channel loops the playlist
//
// Returns:
//   - TimelinePosition: Current playback position with all fields populated
//   - error: ErrChannelNotStarted, ErrEmptyPlaylist, ErrPlaylistFinished, or nil
//
// Performance: O(n) where n is playlist length, optimized for < 100ms on 1000 items
func CalculatePosition(startTime, currentTime time.Time, playlist []*models.PlaylistItem, loop bool) (*TimelinePosition, error) {
	// Edge case: Empty playlist
	if len(playlist) == 0 {
		return nil, ErrEmptyPlaylist
	}

	// Calculate elapsed time in seconds since channel start
	elapsed := int64(currentTime.Sub(startTime).Seconds())

	// Edge case: Channel hasn't started yet
	if elapsed < 0 {
		return nil, ErrChannelNotStarted
	}

	// Calculate total playlist duration by summing all media durations
	var totalDuration int64
	for _, item := range playlist {
		// Defensive check - media should always be populated but be safe
		if item.Media == nil {
			continue
		}
		totalDuration += item.Media.Duration
	}

	// Edge case: Playlist has no valid media (all items have nil Media or zero duration)
	if totalDuration == 0 {
		return nil, ErrEmptyPlaylist
	}

	// Calculate position within the playlist timeline
	var position int64
	if loop {
		// For looping channels, wrap around using modulo
		position = elapsed % totalDuration
	} else {
		// For non-looping channels, check if we're past the end
		if elapsed >= totalDuration {
			return nil, ErrPlaylistFinished
		}
		position = elapsed
	}

	// Find which playlist item contains the current position
	// Single-pass O(n) linear search through the playlist
	var accumulated int64
	for _, item := range playlist {
		if item.Media == nil {
			continue
		}

		itemDuration := item.Media.Duration

		// Check if current position falls within this item
		if position < accumulated+itemDuration {
			// Calculate offset within this specific item
			offsetInItem := position - accumulated

			// Calculate when this item started playing
			// itemStartedAt = currentTime - offsetInItem
			itemStartedAt := currentTime.Add(-time.Duration(offsetInItem) * time.Second)

			// Calculate when this item will end
			// itemEndsAt = itemStartedAt + itemDuration
			itemEndsAt := itemStartedAt.Add(time.Duration(itemDuration) * time.Second)

			// Build and return the complete TimelinePosition
			return &TimelinePosition{
				MediaID:       item.Media.ID,
				MediaTitle:    item.Media.Title,
				OffsetSeconds: offsetInItem,
				StartedAt:     itemStartedAt,
				EndsAt:        itemEndsAt,
				Duration:      itemDuration,
			}, nil
		}

		accumulated += itemDuration
	}

	// This should never happen if our logic is correct, but be defensive
	// If we get here, something went wrong with the calculation
	// Return the last item in the playlist as a fallback
	lastItem := playlist[len(playlist)-1]
	if lastItem.Media != nil {
		return &TimelinePosition{
			MediaID:       lastItem.Media.ID,
			MediaTitle:    lastItem.Media.Title,
			OffsetSeconds: lastItem.Media.Duration - 1, // Near end of last item
			StartedAt:     currentTime.Add(-time.Duration(lastItem.Media.Duration-1) * time.Second),
			EndsAt:        currentTime.Add(time.Second),
			Duration:      lastItem.Media.Duration,
		}, nil
	}

	return nil, ErrEmptyPlaylist
}
