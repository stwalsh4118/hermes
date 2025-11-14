// Package streaming provides FFmpeg command building for HLS stream generation.
package streaming

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	ErrInvalidQuality              = errors.New("invalid quality level")
	ErrInvalidHardwareAccel        = errors.New("invalid hardware acceleration method")
	ErrEmptyInputFile              = errors.New("input file cannot be empty")
	ErrEmptyOutputPath             = errors.New("output path cannot be empty")
	ErrInvalidSegmentDuration      = errors.New("segment duration must be positive")
	ErrInvalidPlaylistSize         = errors.New("playlist size must be positive")
	ErrInvalidBatchSize            = errors.New("batch size must be positive when batch mode is enabled")
	ErrEmptySegmentOutputDir       = errors.New("segment output directory cannot be empty when stream segment mode is enabled")
	ErrEmptySegmentFilenamePattern = errors.New("segment filename pattern cannot be empty when stream segment mode is enabled")
	ErrInvalidFPS                  = errors.New("FPS must be positive")
)

// StreamParams contains all parameters needed to build an FFmpeg HLS command
type StreamParams struct {
	InputFile              string        // Path to input video file
	OutputPath             string        // Full path to output .m3u8 playlist (HLS mode) or segment directory (stream_segment mode)
	Quality                string        // Quality level (1080p, 720p, 480p)
	HardwareAccel          HardwareAccel // Hardware acceleration method
	SeekSeconds            int64         // Starting position in seconds (0 = beginning) - position within current video file
	StreamPositionSeconds  int64         // Cumulative stream position in seconds (segmentNumber * segmentDuration) - for PTS timestamps
	SegmentDuration        int           // HLS segment duration in seconds
	PlaylistSize           int           // Number of segments to keep in playlist
	EncodingPreset         string        // FFmpeg encoding preset (ultrafast, veryfast, medium, slow)
	BatchMode              bool          // Enable batch generation mode (generates N segments then exits)
	BatchSize              int           // Number of segments to generate per batch (required when BatchMode is true)
	StreamSegmentMode      bool          // Enable stream_segment muxer mode (generates TS segments without playlist)
	SegmentOutputDir       string        // Directory for segment output (required when StreamSegmentMode is true)
	SegmentFilenamePattern string        // Filename pattern for segments with strftime (e.g., seg-%Y%m%dT%H%M%S.ts)
	FPS                    int           // Frames per second for GOP calculations (default: 30 if not provided)
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
// When StreamSegmentMode is true, it builds a stream_segment command instead
func BuildHLSCommand(params StreamParams) (*FFmpegCommand, error) {
	// Validate parameters
	if err := validateStreamParams(params); err != nil {
		return nil, err
	}

	// Build command arguments in correct order
	args := make([]string, 0, 40)

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

	// 5. Stream segment mode specific args
	if params.StreamSegmentMode {
		// GOP alignment for deterministic segment boundaries
		gopArgs := buildGOPArgs(params.FPS, params.SegmentDuration)
		args = append(args, gopArgs...)

		// Force keyframes at segment boundaries
		keyframeArgs := buildKeyframeArgs(params.SegmentDuration)
		args = append(args, keyframeArgs...)

		// Explicit stream mapping
		mappingArgs := buildStreamMappingArgs()
		args = append(args, mappingArgs...)

		// Stream segment output args (includes output path)
		segmentArgs := buildStreamSegmentArgs(params)
		args = append(args, segmentArgs...)
	} else {
		// HLS output args
		hlsArgs := buildHLSArgs(params)
		args = append(args, hlsArgs...)

		// Output file (playlist path)
		args = append(args, params.OutputPath)
	}

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

	// Validate output path (only required when NOT in stream_segment mode)
	if !params.StreamSegmentMode {
		if params.OutputPath == "" {
			return ErrEmptyOutputPath
		}
	}

	// Validate segment duration
	if params.SegmentDuration <= 0 {
		return ErrInvalidSegmentDuration
	}

	// Validate playlist size (only required when NOT in stream_segment mode)
	if !params.StreamSegmentMode {
		if params.PlaylistSize <= 0 {
			return ErrInvalidPlaylistSize
		}
	}

	// Validate seek seconds
	if params.SeekSeconds < 0 {
		return fmt.Errorf("seek seconds must be non-negative, got: %d", params.SeekSeconds)
	}

	// Validate batch size when batch mode is enabled
	if params.BatchMode && params.BatchSize <= 0 {
		return ErrInvalidBatchSize
	}

	// Validate stream segment mode parameters
	if params.StreamSegmentMode {
		if params.SegmentOutputDir == "" {
			return ErrEmptySegmentOutputDir
		}
		if params.SegmentFilenamePattern == "" {
			return ErrEmptySegmentFilenamePattern
		}
		if params.FPS <= 0 {
			return ErrInvalidFPS
		}
	}

	return nil
}

// buildInputArgs builds input-related FFmpeg arguments
func buildInputArgs(params StreamParams) []string {
	args := make([]string, 0, 10)

	// Add seeking if specified (must come BEFORE input for fast seeking)
	if params.SeekSeconds > 0 {
		args = append(args, "-ss", strconv.FormatInt(params.SeekSeconds, 10))
		// When seeking, ensure PTS timestamps are regenerated from 0
		// This is critical for proper HLS playback when starting from middle of video
		args = append(args, "-fflags", "+genpts")
	}

	// Add infinite loop only for continuous streaming (not batch mode, not single segment mode)
	// When StreamSegmentMode is true and BatchMode is false, we're generating a single segment
	// so we don't want infinite looping - the -t duration limit will handle stopping
	if !params.BatchMode && !params.StreamSegmentMode {
		// Only add loop for non-segment mode continuous streaming
		args = append(args, "-stream_loop", "-1")
	}

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

	args := []string{
		"-f", "hls",
		"-hls_time", strconv.Itoa(params.SegmentDuration),
		"-hls_list_size", strconv.Itoa(params.PlaylistSize),
		"-hls_flags", hlsFlags,
		"-hls_segment_filename", segmentPattern,
		// No hls_playlist_type - allows sliding window of recent segments
	}

	return args
}

// buildStreamSegmentArgs builds output arguments for single segment generation
// Uses -f mpegts to output a single TS file
func buildStreamSegmentArgs(params StreamParams) []string {
	// Single segment mode: use simple mpegts muxer for one file
	// Generate filename with current timestamp including milliseconds for uniqueness
	// Format: seg-YYYYMMDDTHHMMSS.mmm.ts (e.g., seg-20251111T184324.123.ts)
	now := time.Now()
	filename := fmt.Sprintf("seg-%s.%03d.ts",
		now.Format("20060102T150405"),
		now.Nanosecond()/1000000) // Convert nanoseconds to milliseconds
	outputPath := filepath.Join(params.SegmentOutputDir, filename)
	// Build args list
	// Use -output_ts_offset to set proper PTS timestamps for sequential buffering
	// When seeking with -ss, FFmpeg resets timestamps to 0, but we need sequential PTS
	// so HLS.js can buffer segments sequentially instead of overwriting them
	// The offset should be the segment's position in the timeline (offsetSeconds)
	args := []string{
		"-t", strconv.Itoa(params.SegmentDuration),
		"-f", "mpegts",
	}

	// Add timestamp offset based on cumulative stream position
	// This ensures each segment has sequential PTS values (0-4s, 4-8s, 8-12s, etc.)
	// which allows HLS.js to buffer them sequentially instead of overwriting
	// Use StreamPositionSeconds for cumulative stream timeline (segmentNumber * segmentDuration)
	// This ensures consistent timestamps even when switching between video files
	// -output_ts_offset expects seconds (not 90kHz PTS units)
	// IMPORTANT: Always use StreamPositionSeconds (0 for segment 0, 4 for segment 1, etc.)
	// Do NOT use SeekSeconds as fallback - that would break sequential PTS timestamps
	tsOffset := params.StreamPositionSeconds
	args = append(args, "-output_ts_offset", strconv.FormatInt(tsOffset, 10))

	// When seeking, also add -vsync 0 to drop frames and regenerate timestamps properly
	// This ensures PTS timestamps are sequential even when starting from middle of video
	if params.SeekSeconds > 0 {
		args = append(args, "-vsync", "0")
	}

	// Output path must be last
	args = append(args, outputPath)

	return args
}

// buildGOPArgs builds GOP alignment arguments for deterministic segment boundaries
func buildGOPArgs(fps int, segmentDuration int) []string {
	if fps <= 0 {
		fps = 30 // Default FPS if not provided
	}
	gopSize := fps * segmentDuration

	return []string{
		"-g", strconv.Itoa(gopSize),
		"-keyint_min", strconv.Itoa(gopSize),
		"-sc_threshold", "0", // Disable scene change detection, rely on forced keyframes
	}
}

// buildKeyframeArgs builds keyframe forcing arguments
func buildKeyframeArgs(segmentDuration int) []string {
	// Force keyframe every segmentDuration seconds
	// Expression: gte(t,n_forced*segmentDuration) means "greater than or equal to time t, where t >= n_forced * segmentDuration"
	// This forces a keyframe at 0, segmentDuration, 2*segmentDuration, etc.
	keyframeExpr := fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentDuration)
	return []string{"-force_key_frames", keyframeExpr}
}

// buildStreamMappingArgs builds explicit stream mapping arguments
func buildStreamMappingArgs() []string {
	// Map first video stream and first audio stream explicitly
	return []string{
		"-map", "0:v:0", // Map first video stream from first input
		"-map", "0:a:0", // Map first audio stream from first input (if available)
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
