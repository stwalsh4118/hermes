package timeline

import "errors"

var (
	// ErrChannelNotStarted is returned when attempting to calculate timeline for a channel
	// that hasn't started broadcasting yet (current time < start time)
	ErrChannelNotStarted = errors.New("channel has not started broadcasting yet")

	// ErrEmptyPlaylist is returned when a channel has no playlist items
	ErrEmptyPlaylist = errors.New("channel playlist is empty")

	// ErrPlaylistFinished is returned when a non-looping channel has completed its playlist
	ErrPlaylistFinished = errors.New("channel playlist has finished (non-looping)")
)
