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
