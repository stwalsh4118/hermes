package streaming

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents the type of streaming error
type ErrorType int

const (
	// ErrorTypeFFmpegCrash indicates FFmpeg process crashed unexpectedly
	ErrorTypeFFmpegCrash ErrorType = iota
	// ErrorTypeFileMissing indicates input file doesn't exist
	ErrorTypeFileMissing
	// ErrorTypeFileCorrupt indicates input file is corrupted or invalid
	ErrorTypeFileCorrupt
	// ErrorTypeHardwareEncoder indicates hardware encoder failed
	ErrorTypeHardwareEncoder
	// ErrorTypeDiskSpace indicates insufficient disk space
	ErrorTypeDiskSpace
	// ErrorTypePlaylistEnd indicates playlist reached end (non-looping)
	ErrorTypePlaylistEnd
	// ErrorTypeTimeout indicates operation timed out
	ErrorTypeTimeout
)

// String returns the string representation of ErrorType
func (e ErrorType) String() string {
	switch e {
	case ErrorTypeFFmpegCrash:
		return "ffmpeg_crash"
	case ErrorTypeFileMissing:
		return "file_missing"
	case ErrorTypeFileCorrupt:
		return "file_corrupt"
	case ErrorTypeHardwareEncoder:
		return "hardware_encoder"
	case ErrorTypeDiskSpace:
		return "disk_space"
	case ErrorTypePlaylistEnd:
		return "playlist_end"
	case ErrorTypeTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// ErrorSeverity represents the severity of a streaming error
type ErrorSeverity int

const (
	// SeverityInfo represents informational events (e.g., playlist end on non-looping)
	SeverityInfo ErrorSeverity = iota
	// SeverityWarning represents recoverable issues that need attention
	SeverityWarning
	// SeverityError represents errors that may be recoverable with retry
	SeverityError
	// SeverityCritical represents critical errors that require immediate action
	SeverityCritical
)

// String returns the string representation of ErrorSeverity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// StreamError represents a structured streaming error with classification
type StreamError struct {
	Type        ErrorType
	Severity    ErrorSeverity
	Message     string
	Cause       error
	Recoverable bool
}

// NewStreamError creates a new StreamError with the given type, message, and cause
func NewStreamError(errorType ErrorType, message string, cause error) *StreamError {
	severity, recoverable := classifyErrorTypeAttributes(errorType)
	return &StreamError{
		Type:        errorType,
		Severity:    severity,
		Message:     message,
		Cause:       cause,
		Recoverable: recoverable,
	}
}

// Error implements the error interface
func (e *StreamError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type.String(), e.Message)
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *StreamError) Unwrap() error {
	return e.Cause
}

// classifyErrorTypeAttributes returns severity and recoverability for an error type
func classifyErrorTypeAttributes(errorType ErrorType) (ErrorSeverity, bool) {
	switch errorType {
	case ErrorTypeFFmpegCrash:
		return SeverityError, true // Recoverable with restart
	case ErrorTypeFileMissing:
		return SeverityWarning, true // Recoverable by skipping to next
	case ErrorTypeFileCorrupt:
		return SeverityWarning, true // Recoverable by skipping to next
	case ErrorTypeHardwareEncoder:
		return SeverityWarning, true // Recoverable with software fallback
	case ErrorTypeDiskSpace:
		return SeverityCritical, false // Not recoverable
	case ErrorTypePlaylistEnd:
		return SeverityInfo, false // Expected behavior
	case ErrorTypeTimeout:
		return SeverityError, true // Recoverable with retry
	default:
		return SeverityError, false
	}
}

// ClassifyError classifies a generic error into a StreamError
func ClassifyError(err error) *StreamError {
	if err == nil {
		return nil
	}

	// Check if it's already a StreamError
	var streamErr *StreamError
	if errors.As(err, &streamErr) {
		return streamErr
	}

	// Try to classify based on error message
	errMsg := err.Error()

	// Check for disk space errors
	if strings.Contains(errMsg, "no space left") || strings.Contains(errMsg, "disk full") {
		return NewStreamError(ErrorTypeDiskSpace, "Insufficient disk space", err)
	}

	// Check for file not found errors
	if strings.Contains(errMsg, "no such file") || strings.Contains(errMsg, "file not found") {
		return NewStreamError(ErrorTypeFileMissing, "Input file not found", err)
	}

	// Check for hardware encoder errors
	if strings.Contains(errMsg, "nvcuda") || strings.Contains(errMsg, "nvenc") ||
		strings.Contains(errMsg, "qsv") || strings.Contains(errMsg, "vaapi") ||
		strings.Contains(errMsg, "videotoolbox") {
		return NewStreamError(ErrorTypeHardwareEncoder, "Hardware encoder failed", err)
	}

	// Check for timeout errors
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "timed out") {
		return NewStreamError(ErrorTypeTimeout, "Operation timed out", err)
	}

	// Default to FFmpeg crash
	return NewStreamError(ErrorTypeFFmpegCrash, "Unknown FFmpeg error", err)
}

// ParseFFmpegError parses FFmpeg stderr output to classify errors
func ParseFFmpegError(stderr string) *StreamError {
	stderrLower := strings.ToLower(stderr)

	// Check for file missing errors
	if strings.Contains(stderrLower, "no such file or directory") {
		return NewStreamError(ErrorTypeFileMissing, "Input file not found", fmt.Errorf("ffmpeg: %s", stderr))
	}

	// Check for corrupt file errors
	if strings.Contains(stderrLower, "invalid data found") ||
		strings.Contains(stderrLower, "could not find codec") ||
		strings.Contains(stderrLower, "invalid argument") {
		return NewStreamError(ErrorTypeFileCorrupt, "Input file is corrupted or invalid", fmt.Errorf("ffmpeg: %s", stderr))
	}

	// Check for hardware encoder errors
	if strings.Contains(stderrLower, "cannot load nvcuda") ||
		strings.Contains(stderrLower, "cannot load nvenc") ||
		strings.Contains(stderrLower, "failed loading nvcuda") ||
		strings.Contains(stderrLower, "driver does not support") ||
		strings.Contains(stderrLower, "qsv not available") ||
		strings.Contains(stderrLower, "vaapi") && strings.Contains(stderrLower, "failed") ||
		strings.Contains(stderrLower, "videotoolbox") && strings.Contains(stderrLower, "failed") {
		return NewStreamError(ErrorTypeHardwareEncoder, "Hardware encoder not available or failed", fmt.Errorf("ffmpeg: %s", stderr))
	}

	// Check for disk space errors
	if strings.Contains(stderrLower, "no space left on device") ||
		strings.Contains(stderrLower, "disk full") {
		return NewStreamError(ErrorTypeDiskSpace, "Insufficient disk space for segment generation", fmt.Errorf("ffmpeg: %s", stderr))
	}

	// Check for timeout errors
	if strings.Contains(stderrLower, "timeout") ||
		strings.Contains(stderrLower, "i/o error") && strings.Contains(stderrLower, "timeout") {
		return NewStreamError(ErrorTypeTimeout, "FFmpeg operation timed out", fmt.Errorf("ffmpeg: %s", stderr))
	}

	// Default to generic crash
	return NewStreamError(ErrorTypeFFmpegCrash, "FFmpeg process failed", fmt.Errorf("ffmpeg: %s", stderr))
}

// Common streaming errors
var (
	// ErrInsufficientDiskSpace indicates there's not enough disk space
	ErrInsufficientDiskSpace = NewStreamError(ErrorTypeDiskSpace, "Insufficient disk space for streaming", nil)
	// ErrPlaylistEnded indicates a non-looping playlist reached its end
	ErrPlaylistEnded = NewStreamError(ErrorTypePlaylistEnd, "Playlist reached end", nil)
)
