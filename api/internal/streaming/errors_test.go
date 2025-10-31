package streaming

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		expected string
	}{
		{"FFmpeg Crash", ErrorTypeFFmpegCrash, "ffmpeg_crash"},
		{"File Missing", ErrorTypeFileMissing, "file_missing"},
		{"File Corrupt", ErrorTypeFileCorrupt, "file_corrupt"},
		{"Hardware Encoder", ErrorTypeHardwareEncoder, "hardware_encoder"},
		{"Disk Space", ErrorTypeDiskSpace, "disk_space"},
		{"Playlist End", ErrorTypePlaylistEnd, "playlist_end"},
		{"Timeout", ErrorTypeTimeout, "timeout"},
		{"Unknown", ErrorType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errType.String()
			if result != tt.expected {
				t.Errorf("ErrorType.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		expected string
	}{
		{"Info", SeverityInfo, "info"},
		{"Warning", SeverityWarning, "warning"},
		{"Error", SeverityError, "error"},
		{"Critical", SeverityCritical, "critical"},
		{"Unknown", ErrorSeverity(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.String()
			if result != tt.expected {
				t.Errorf("ErrorSeverity.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewStreamError(t *testing.T) {
	cause := errors.New("underlying error")

	tests := []struct {
		name                string
		errorType           ErrorType
		message             string
		cause               error
		expectedSeverity    ErrorSeverity
		expectedRecoverable bool
	}{
		{
			name:                "FFmpeg Crash",
			errorType:           ErrorTypeFFmpegCrash,
			message:             "process crashed",
			cause:               cause,
			expectedSeverity:    SeverityError,
			expectedRecoverable: true,
		},
		{
			name:                "Disk Space",
			errorType:           ErrorTypeDiskSpace,
			message:             "not enough space",
			cause:               nil,
			expectedSeverity:    SeverityCritical,
			expectedRecoverable: false,
		},
		{
			name:                "File Missing",
			errorType:           ErrorTypeFileMissing,
			message:             "file not found",
			cause:               cause,
			expectedSeverity:    SeverityWarning,
			expectedRecoverable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamErr := NewStreamError(tt.errorType, tt.message, tt.cause)

			if streamErr.Type != tt.errorType {
				t.Errorf("Type = %v, want %v", streamErr.Type, tt.errorType)
			}
			if streamErr.Message != tt.message {
				t.Errorf("Message = %v, want %v", streamErr.Message, tt.message)
			}
			// Check cause - handle nil case separately since errors.Is(nil, nil) is false
			if (streamErr.Cause == nil && tt.cause != nil) || (streamErr.Cause != nil && !errors.Is(streamErr.Cause, tt.cause)) {
				t.Errorf("Cause = %v, want %v", streamErr.Cause, tt.cause)
			}
			if streamErr.Severity != tt.expectedSeverity {
				t.Errorf("Severity = %v, want %v", streamErr.Severity, tt.expectedSeverity)
			}
			if streamErr.Recoverable != tt.expectedRecoverable {
				t.Errorf("Recoverable = %v, want %v", streamErr.Recoverable, tt.expectedRecoverable)
			}
		})
	}
}

func TestStreamError_Error(t *testing.T) {
	cause := errors.New("underlying error")

	tests := []struct {
		name     string
		err      *StreamError
		expected string
	}{
		{
			name:     "Error with cause",
			err:      NewStreamError(ErrorTypeFFmpegCrash, "crash occurred", cause),
			expected: "ffmpeg_crash: crash occurred (caused by: underlying error)",
		},
		{
			name:     "Error without cause",
			err:      NewStreamError(ErrorTypeDiskSpace, "no space", nil),
			expected: "disk_space: no space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStreamError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	streamErr := NewStreamError(ErrorTypeFFmpegCrash, "crash", cause)

	unwrapped := streamErr.Unwrap()
	if !errors.Is(unwrapped, cause) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test errors.Is
	if !errors.Is(streamErr, cause) {
		t.Error("errors.Is should return true for wrapped error")
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedType     ErrorType
		expectedSeverity ErrorSeverity
	}{
		{
			name:             "Disk space error",
			err:              errors.New("no space left on device"),
			expectedType:     ErrorTypeDiskSpace,
			expectedSeverity: SeverityCritical,
		},
		{
			name:             "File not found error",
			err:              errors.New("no such file or directory"),
			expectedType:     ErrorTypeFileMissing,
			expectedSeverity: SeverityWarning,
		},
		{
			name:             "Hardware encoder error",
			err:              errors.New("cannot load nvcuda"),
			expectedType:     ErrorTypeHardwareEncoder,
			expectedSeverity: SeverityWarning,
		},
		{
			name:             "Timeout error",
			err:              errors.New("operation timed out"),
			expectedType:     ErrorTypeTimeout,
			expectedSeverity: SeverityError,
		},
		{
			name:             "Unknown error",
			err:              errors.New("some random error"),
			expectedType:     ErrorTypeFFmpegCrash,
			expectedSeverity: SeverityError,
		},
		{
			name:             "Nil error",
			err:              nil,
			expectedType:     ErrorType(0),
			expectedSeverity: ErrorSeverity(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)

			if tt.err == nil {
				if result != nil {
					t.Errorf("ClassifyError(nil) should return nil, got %v", result)
				}
				return
			}

			if result.Type != tt.expectedType {
				t.Errorf("ClassifyError() Type = %v, want %v", result.Type, tt.expectedType)
			}
			if result.Severity != tt.expectedSeverity {
				t.Errorf("ClassifyError() Severity = %v, want %v", result.Severity, tt.expectedSeverity)
			}
		})
	}
}

func TestClassifyError_AlreadyStreamError(t *testing.T) {
	originalErr := NewStreamError(ErrorTypeDiskSpace, "out of space", nil)
	result := ClassifyError(originalErr)

	if result != originalErr {
		t.Error("ClassifyError should return the same StreamError if already classified")
	}
}

func TestParseFFmpegError(t *testing.T) {
	tests := []struct {
		name         string
		stderr       string
		expectedType ErrorType
	}{
		{
			name:         "File not found",
			stderr:       "Error: No such file or directory",
			expectedType: ErrorTypeFileMissing,
		},
		{
			name:         "Invalid data",
			stderr:       "Invalid data found when processing input",
			expectedType: ErrorTypeFileCorrupt,
		},
		{
			name:         "NVENC not available",
			stderr:       "Cannot load nvcuda.dll",
			expectedType: ErrorTypeHardwareEncoder,
		},
		{
			name:         "QSV not available",
			stderr:       "QSV not available",
			expectedType: ErrorTypeHardwareEncoder,
		},
		{
			name:         "No space left",
			stderr:       "No space left on device",
			expectedType: ErrorTypeDiskSpace,
		},
		{
			name:         "I/O timeout",
			stderr:       "I/O error: timeout exceeded",
			expectedType: ErrorTypeTimeout,
		},
		{
			name:         "Generic error",
			stderr:       "Unknown encoding error occurred",
			expectedType: ErrorTypeFFmpegCrash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFFmpegError(tt.stderr)

			if result.Type != tt.expectedType {
				t.Errorf("ParseFFmpegError() Type = %v, want %v", result.Type, tt.expectedType)
			}

			// Verify the stderr is included in the error
			if result.Cause == nil {
				t.Error("ParseFFmpegError() should always create an error with cause")
			}
		})
	}
}

func TestParseFFmpegError_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
	}{
		{"Lowercase", "no such file or directory"},
		{"Uppercase", "NO SUCH FILE OR DIRECTORY"},
		{"Mixed case", "No Such File Or Directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFFmpegError(tt.stderr)
			if result.Type != ErrorTypeFileMissing {
				t.Errorf("ParseFFmpegError() should be case-insensitive, got Type = %v", result.Type)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// Test ErrInsufficientDiskSpace
	if ErrInsufficientDiskSpace.Type != ErrorTypeDiskSpace {
		t.Errorf("ErrInsufficientDiskSpace Type = %v, want %v", ErrInsufficientDiskSpace.Type, ErrorTypeDiskSpace)
	}
	if ErrInsufficientDiskSpace.Recoverable {
		t.Error("ErrInsufficientDiskSpace should not be recoverable")
	}

	// Test ErrPlaylistEnded
	if ErrPlaylistEnded.Type != ErrorTypePlaylistEnd {
		t.Errorf("ErrPlaylistEnded Type = %v, want %v", ErrPlaylistEnded.Type, ErrorTypePlaylistEnd)
	}
	if ErrPlaylistEnded.Recoverable {
		t.Error("ErrPlaylistEnded should not be recoverable")
	}
}

func TestClassifyErrorTypeAttributes(t *testing.T) {
	tests := []struct {
		name                string
		errorType           ErrorType
		expectedSeverity    ErrorSeverity
		expectedRecoverable bool
	}{
		{"FFmpeg Crash", ErrorTypeFFmpegCrash, SeverityError, true},
		{"File Missing", ErrorTypeFileMissing, SeverityWarning, true},
		{"File Corrupt", ErrorTypeFileCorrupt, SeverityWarning, true},
		{"Hardware Encoder", ErrorTypeHardwareEncoder, SeverityWarning, true},
		{"Disk Space", ErrorTypeDiskSpace, SeverityCritical, false},
		{"Playlist End", ErrorTypePlaylistEnd, SeverityInfo, false},
		{"Timeout", ErrorTypeTimeout, SeverityError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity, recoverable := classifyErrorTypeAttributes(tt.errorType)

			if severity != tt.expectedSeverity {
				t.Errorf("Severity = %v, want %v", severity, tt.expectedSeverity)
			}
			if recoverable != tt.expectedRecoverable {
				t.Errorf("Recoverable = %v, want %v", recoverable, tt.expectedRecoverable)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := fmt.Errorf("base error")
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)
	streamErr := ClassifyError(wrappedErr)

	// Test that we can unwrap through StreamError
	if !errors.Is(streamErr, baseErr) {
		t.Error("Should be able to unwrap to base error through StreamError")
	}
}
