// Package streaming provides FFmpeg command building for HLS stream generation.
package streaming

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Quality level constants
const (
	Quality1080p = "1080p"
	Quality720p  = "720p"
	Quality480p  = "480p"
)

// Bitrate constants (in kbps)
const (
	bitrate1080p  = 5000
	bitrate720p   = 3000
	bitrate480p   = 1500
	audioBitrate  = 192
	audioChannels = 2
)

// Resolution constants
const (
	resolution1080p = "1920x1080"
	resolution720p  = "1280x720"
	resolution480p  = "854x480"
)

// HLS parameter defaults
const (
	defaultSegmentDuration = 6
	defaultPlaylistSize    = 10
	hlsFlags               = "delete_segments"
)

// Common errors
var (
	ErrInvalidQuality         = errors.New("invalid quality level")
	ErrInvalidHardwareAccel   = errors.New("invalid hardware acceleration method")
	ErrEmptyInputFile         = errors.New("input file cannot be empty")
	ErrEmptyOutputPath        = errors.New("output path cannot be empty")
	ErrInvalidSegmentDuration = errors.New("segment duration must be positive")
	ErrInvalidPlaylistSize    = errors.New("playlist size must be positive")
)

// StreamParams contains all parameters needed to build an FFmpeg HLS command
type StreamParams struct {
	InputFile       string        // Path to input video file
	OutputPath      string        // Full path to output .m3u8 playlist
	Quality         string        // Quality level (1080p, 720p, 480p)
	HardwareAccel   HardwareAccel // Hardware acceleration method
	SeekSeconds     int64         // Starting position in seconds (0 = beginning)
	SegmentDuration int           // HLS segment duration in seconds
	PlaylistSize    int           // Number of segments to keep in playlist
	RealtimePacing  bool          // Enable -re flag for 1x speed encoding
	EncodingPreset  string        // FFmpeg encoding preset (ultrafast, veryfast, medium, slow)
}

// FFmpegCommand represents a built FFmpeg command
type FFmpegCommand struct {
	Args []string // Command arguments (without "ffmpeg" itself)
}

// qualitySpec contains bitrate and resolution for a quality level
type qualitySpec struct {
	videoBitrate int
	maxrate      int
	bufsize      int
	resolution   string
}

// getQualitySpec returns the quality specifications for a given quality level
func getQualitySpec(quality string) (*qualitySpec, error) {
	switch quality {
	case Quality1080p:
		return &qualitySpec{
			videoBitrate: bitrate1080p,
			maxrate:      bitrate1080p,
			bufsize:      bitrate1080p * 2,
			resolution:   resolution1080p,
		}, nil
	case Quality720p:
		return &qualitySpec{
			videoBitrate: bitrate720p,
			maxrate:      bitrate720p,
			bufsize:      bitrate720p * 2,
			resolution:   resolution720p,
		}, nil
	case Quality480p:
		return &qualitySpec{
			videoBitrate: bitrate480p,
			maxrate:      bitrate480p,
			bufsize:      bitrate480p * 2,
			resolution:   resolution480p,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s (must be one of: %s, %s, %s)",
			ErrInvalidQuality, quality, Quality1080p, Quality720p, Quality480p)
	}
}

// BuildHLSCommand builds a complete FFmpeg command for HLS stream generation
func BuildHLSCommand(params StreamParams) (*FFmpegCommand, error) {
	// Validate parameters
	if err := validateStreamParams(params); err != nil {
		return nil, err
	}

	// Build command arguments in correct order
	args := make([]string, 0, 30)

	// 1. Input args (with seeking if specified)
	inputArgs := buildInputArgs(params)
	args = append(args, inputArgs...)

	// 2. Video encoding args (with hardware acceleration and preset)
	videoArgs := buildVideoEncodeArgs(params.HardwareAccel, params.EncodingPreset)
	args = append(args, videoArgs...)

	// 3. Audio encoding args
	audioArgs := buildAudioEncodeArgs()
	args = append(args, audioArgs...)

	// 4. Quality/bitrate args
	qualityArgs, err := buildQualityArgs(params.Quality)
	if err != nil {
		return nil, err
	}
	args = append(args, qualityArgs...)

	// 5. HLS output args
	hlsArgs := buildHLSArgs(params)
	args = append(args, hlsArgs...)

	// 6. Output file
	args = append(args, params.OutputPath)

	return &FFmpegCommand{Args: args}, nil
}

// validateStreamParams validates all stream parameters
func validateStreamParams(params StreamParams) error {
	// Validate quality
	if _, err := getQualitySpec(params.Quality); err != nil {
		return err
	}

	// Validate hardware acceleration
	if !params.HardwareAccel.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidHardwareAccel, params.HardwareAccel)
	}

	// Validate input file
	if params.InputFile == "" {
		return ErrEmptyInputFile
	}

	// Validate output path
	if params.OutputPath == "" {
		return ErrEmptyOutputPath
	}

	// Validate segment duration
	if params.SegmentDuration <= 0 {
		return ErrInvalidSegmentDuration
	}

	// Validate playlist size
	if params.PlaylistSize <= 0 {
		return ErrInvalidPlaylistSize
	}

	return nil
}

// buildInputArgs builds input-related FFmpeg arguments
func buildInputArgs(params StreamParams) []string {
	args := make([]string, 0, 7)

	// Add -re flag for real-time pacing (must come BEFORE -ss and -i)
	if params.RealtimePacing {
		args = append(args, "-re")
	}

	// Add seeking if specified (must come BEFORE input for fast seeking)
	if params.SeekSeconds > 0 {
		args = append(args, "-ss", strconv.FormatInt(params.SeekSeconds, 10))
	}

	// Add infinite loop for 24/7 channel behavior (must come BEFORE -i)
	args = append(args, "-stream_loop", "-1")

	// Add input file
	args = append(args, "-i", params.InputFile)

	return args
}

// mapToNVENCPreset maps software encoding presets to NVENC preset values
func mapToNVENCPreset(softwarePreset string) string {
	switch softwarePreset {
	case "ultrafast":
		return "p1"
	case "veryfast":
		return "p2"
	case "fast":
		return "p3"
	case "medium":
		return "p4"
	case "slow":
		return "p5"
	default:
		return "p1" // Default to fastest
	}
}

// buildVideoEncodeArgs builds video encoding arguments based on hardware acceleration
func buildVideoEncodeArgs(hwaccel HardwareAccel, preset string) []string {
	switch hwaccel {
	case HardwareAccelNVENC:
		// NVENC uses p1-p7 presets
		nvencPreset := mapToNVENCPreset(preset)
		return []string{"-c:v", "h264_nvenc", "-preset", nvencPreset}
	case HardwareAccelQSV:
		// QSV uses same preset names as software
		return []string{"-c:v", "h264_qsv", "-preset", preset}
	case HardwareAccelVAAPI:
		// VAAPI doesn't support presets in the same way
		return []string{"-c:v", "h264_vaapi"}
	case HardwareAccelVideoToolbox:
		// VideoToolbox doesn't support presets in the same way
		return []string{"-c:v", "h264_videotoolbox"}
	case HardwareAccelNone, HardwareAccelAuto:
		// Software encoding (libx264) with preset
		return []string{"-c:v", "libx264", "-preset", preset}
	default:
		// Fallback to software encoding with preset
		return []string{"-c:v", "libx264", "-preset", preset}
	}
}

// buildAudioEncodeArgs builds audio encoding arguments (constant across all qualities)
func buildAudioEncodeArgs() []string {
	return []string{
		"-c:a", "aac",
		"-b:a", strconv.Itoa(audioBitrate) + "k",
		"-ac", strconv.Itoa(audioChannels),
	}
}

// buildQualityArgs builds quality-specific arguments (bitrate, resolution)
func buildQualityArgs(quality string) ([]string, error) {
	spec, err := getQualitySpec(quality)
	if err != nil {
		return nil, err
	}

	return []string{
		"-b:v", strconv.Itoa(spec.videoBitrate) + "k",
		"-maxrate", strconv.Itoa(spec.maxrate) + "k",
		"-bufsize", strconv.Itoa(spec.bufsize) + "k",
		"-s", spec.resolution,
	}, nil
}

// buildHLSArgs builds HLS-specific output arguments
func buildHLSArgs(params StreamParams) []string {
	// Get segment filename pattern from output path
	segmentPattern := getSegmentFilenamePattern(params.OutputPath)

	return []string{
		"-f", "hls",
		"-hls_time", strconv.Itoa(params.SegmentDuration),
		"-hls_list_size", strconv.Itoa(params.PlaylistSize),
		"-hls_flags", hlsFlags,
		"-hls_segment_filename", segmentPattern,
		// No hls_playlist_type - allows sliding window of recent segments
	}
}

// getSegmentFilenamePattern generates the segment filename pattern based on output path
func getSegmentFilenamePattern(outputPath string) string {
	// Extract directory and base name
	dir := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)

	// Remove .m3u8 extension if present
	base = strings.TrimSuffix(base, ".m3u8")

	// Create pattern: dir/base_segment_%03d.ts
	return filepath.Join(dir, base+"_segment_%03d.ts")
}

// GetOutputPath generates a consistent output path for a channel and quality
func GetOutputPath(baseDir, quality string) string {
	return filepath.Join(baseDir, quality, quality+".m3u8")
}

// GetSegmentPattern generates a consistent segment naming pattern
func GetSegmentPattern(channelID, quality string) string {
	return channelID + "_" + quality + "_segment_%03d.ts"
}

// GetPlaylistFilename returns the playlist filename for a quality level
func GetPlaylistFilename(quality string) string {
	return quality + ".m3u8"
}
