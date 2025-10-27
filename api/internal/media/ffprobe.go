package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/stwalsh4118/hermes/internal/logger"
)

// Timeout for FFprobe execution
const ffprobeTimeout = 30 * time.Second

// Common errors
var (
	ErrFFprobeNotFound = errors.New("ffprobe not found in PATH")
	ErrFileNotFound    = errors.New("file not found or not readable")
	ErrInvalidFile     = errors.New("invalid or corrupted video file")
	ErrTimeout         = errors.New("ffprobe execution timed out")
)

// FFprobeResult represents the top-level JSON output from FFprobe
type FFprobeResult struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

// Stream represents a video or audio stream
type Stream struct {
	Index         int    `json:"index"`
	CodecName     string `json:"codec_name"`
	CodecLongName string `json:"codec_long_name"`
	CodecType     string `json:"codec_type"` // "video" or "audio"
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Duration      string `json:"duration,omitempty"`
	BitRate       string `json:"bit_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
}

// Format represents the file format information
type Format struct {
	Filename       string `json:"filename"`
	NbStreams      int    `json:"nb_streams"`
	NbPrograms     int    `json:"nb_programs"`
	FormatName     string `json:"format_name"`
	FormatLongName string `json:"format_long_name"`
	Duration       string `json:"duration"`
	Size           string `json:"size"`
	BitRate        string `json:"bit_rate"`
}

// VideoMetadata represents simplified metadata for application use
type VideoMetadata struct {
	Duration   int64  // Duration in seconds
	VideoCodec string // e.g., "h264", "hevc"
	AudioCodec string // e.g., "aac", "mp3"
	Resolution string // e.g., "1920x1080"
	FileSize   int64  // File size in bytes
	Width      int
	Height     int
}

// CheckFFprobeInstalled checks if FFprobe is available in PATH
func CheckFFprobeInstalled() error {
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		return ErrFFprobeNotFound
	}
	return nil
}

// ProbeFile executes FFprobe on the given file and returns metadata
func ProbeFile(ctx context.Context, filePath string) (*VideoMetadata, error) {
	// Check FFprobe is available
	if err := CheckFFprobeInstalled(); err != nil {
		return nil, err
	}

	logger.Log.Debug().
		Str("file_path", filePath).
		Msg("Probing video file with FFprobe")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, ffprobeTimeout)
	defer cancel()

	// Build FFprobe command
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	// Execute command
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Log.Error().
				Str("file_path", filePath).
				Msg("FFprobe execution timed out")
			return nil, ErrTimeout
		}

		// Check if it's a file not found error
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			logger.Log.Error().
				Str("file_path", filePath).
				Str("stderr", stderr).
				Msg("FFprobe execution failed")

			// Check for common error patterns
			if len(stderr) > 0 {
				return nil, fmt.Errorf("%w: %s", ErrInvalidFile, stderr)
			}
		}

		logger.Log.Error().
			Err(err).
			Str("file_path", filePath).
			Msg("FFprobe command failed")
		return nil, fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}

	// Parse JSON output
	var result FFprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		logger.Log.Error().
			Err(err).
			Str("file_path", filePath).
			Msg("Failed to parse FFprobe JSON output")
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Extract metadata
	metadata, err := extractMetadata(&result)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("file_path", filePath).
			Msg("Failed to extract metadata from FFprobe result")
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	logger.Log.Info().
		Str("file_path", filePath).
		Int64("duration", metadata.Duration).
		Str("video_codec", metadata.VideoCodec).
		Str("audio_codec", metadata.AudioCodec).
		Str("resolution", metadata.Resolution).
		Int64("file_size", metadata.FileSize).
		Msg("Successfully probed video file")

	return metadata, nil
}

// extractMetadata converts FFprobeResult to VideoMetadata
func extractMetadata(result *FFprobeResult) (*VideoMetadata, error) {
	metadata := &VideoMetadata{}

	// Find video and audio streams
	var videoStream *Stream
	var audioStream *Stream

	for i := range result.Streams {
		stream := &result.Streams[i]
		if stream.CodecType == "video" && videoStream == nil {
			videoStream = stream
		}
		if stream.CodecType == "audio" && audioStream == nil {
			audioStream = stream
		}
	}

	// Extract video metadata
	if videoStream != nil {
		metadata.VideoCodec = videoStream.CodecName
		metadata.Width = videoStream.Width
		metadata.Height = videoStream.Height
		if videoStream.Width > 0 && videoStream.Height > 0 {
			metadata.Resolution = fmt.Sprintf("%dx%d", videoStream.Width, videoStream.Height)
		}
	}

	// Extract audio metadata
	if audioStream != nil {
		metadata.AudioCodec = audioStream.CodecName
	}

	// Extract duration (try stream first, then format)
	var durationFloat float64
	var err error

	if videoStream != nil && videoStream.Duration != "" {
		durationFloat, err = strconv.ParseFloat(videoStream.Duration, 64)
		if err == nil {
			metadata.Duration = int64(durationFloat)
		}
	}

	// Fall back to format duration if stream duration not available
	if metadata.Duration == 0 && result.Format.Duration != "" {
		durationFloat, err = strconv.ParseFloat(result.Format.Duration, 64)
		if err == nil {
			metadata.Duration = int64(durationFloat)
		}
	}

	// Extract file size
	if result.Format.Size != "" {
		fileSize, err := strconv.ParseInt(result.Format.Size, 10, 64)
		if err == nil {
			metadata.FileSize = fileSize
		}
	}

	// Validate we got at least duration
	if metadata.Duration == 0 {
		return nil, fmt.Errorf("%w: could not determine video duration", ErrInvalidFile)
	}

	return metadata, nil
}
