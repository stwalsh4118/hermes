package channel

import "errors"

// Custom channel service errors
var (
	// ErrDuplicateChannelName indicates a channel with the same name already exists
	ErrDuplicateChannelName = errors.New("channel name already exists")

	// ErrInvalidStartTime indicates the start time is more than 1 year in the future
	ErrInvalidStartTime = errors.New("start time cannot be more than 1 year in the future")

	// ErrChannelNotFound indicates the requested channel does not exist
	ErrChannelNotFound = errors.New("channel not found")

	// ErrMediaNotFound indicates the requested media does not exist
	ErrMediaNotFound = errors.New("media not found")

	// ErrPlaylistItemNotFound indicates the requested playlist item does not exist
	ErrPlaylistItemNotFound = errors.New("playlist item not found")

	// ErrInvalidPosition indicates the position is negative
	ErrInvalidPosition = errors.New("position must be non-negative")

	// ErrEmptyPlaylist indicates the playlist has no items
	ErrEmptyPlaylist = errors.New("playlist is empty")
)

// IsDuplicateName checks if the error is a duplicate channel name error
func IsDuplicateName(err error) bool {
	return errors.Is(err, ErrDuplicateChannelName)
}

// IsInvalidStartTime checks if the error is an invalid start time error
func IsInvalidStartTime(err error) bool {
	return errors.Is(err, ErrInvalidStartTime)
}

// IsChannelNotFound checks if the error is a channel not found error
func IsChannelNotFound(err error) bool {
	return errors.Is(err, ErrChannelNotFound)
}

// IsMediaNotFound checks if the error is a media not found error
func IsMediaNotFound(err error) bool {
	return errors.Is(err, ErrMediaNotFound)
}

// IsPlaylistItemNotFound checks if the error is a playlist item not found error
func IsPlaylistItemNotFound(err error) bool {
	return errors.Is(err, ErrPlaylistItemNotFound)
}

// IsInvalidPosition checks if the error is an invalid position error
func IsInvalidPosition(err error) bool {
	return errors.Is(err, ErrInvalidPosition)
}

// IsEmptyPlaylist checks if the error is an empty playlist error
func IsEmptyPlaylist(err error) bool {
	return errors.Is(err, ErrEmptyPlaylist)
}
