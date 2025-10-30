// Package streaming provides HLS playlist generation and management.
package streaming

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/stwalsh4118/hermes/internal/logger"
)

// Playlist type constants
const (
	PlaylistTypeEvent = "EVENT"
	PlaylistTypeVOD   = "VOD"
)

// HLS constants
const (
	hlsVersion           = 3
	defaultSegmentLength = 6.0 // Default segment duration in seconds
)

// Common errors
var (
	ErrEmptyVariants       = errors.New("playlist must have at least one variant")
	ErrInvalidBandwidth    = errors.New("bandwidth must be positive")
	ErrInvalidResolution   = errors.New("invalid resolution format")
	ErrInvalidPlaylistType = errors.New("invalid playlist type (must be EVENT or VOD)")
	ErrMissingRequiredTag  = errors.New("missing required HLS tag")
	ErrInvalidDirectory    = errors.New("invalid directory path")
)

// MasterPlaylist represents an HLS master playlist with quality variants
type MasterPlaylist struct {
	Variants []PlaylistVariant
}

// PlaylistVariant represents a single quality variant in a master playlist
type PlaylistVariant struct {
	Bandwidth  int    // Video + audio bitrate in bits per second
	Resolution string // Format: "1920x1080"
	Path       string // Relative path to media playlist
}

// MediaPlaylist represents an HLS media playlist with segments
type MediaPlaylist struct {
	TargetDuration int       // Maximum segment duration in seconds
	MediaSequence  int       // Starting sequence number
	Segments       []Segment // List of segments
	PlaylistType   string    // "EVENT" or "VOD"
}

// Segment represents a single HLS segment
type Segment struct {
	Duration float64 // Segment duration in seconds
	Path     string  // Filename or relative path to segment
}

// MediaPlaylistConfig contains configuration for media playlist generation
type MediaPlaylistConfig struct {
	TargetDuration int    // Maximum segment duration in seconds
	MediaSequence  int    // Starting sequence number
	PlaylistType   string // "EVENT" or "VOD"
	MaxSegments    int    // Maximum number of segments to keep (sliding window)
}

// GenerateMasterPlaylist generates an HLS master playlist from quality variants
func GenerateMasterPlaylist(variants []PlaylistVariant) (string, error) {
	if len(variants) == 0 {
		return "", ErrEmptyVariants
	}

	// Validate all variants
	for i, variant := range variants {
		if variant.Bandwidth <= 0 {
			return "", fmt.Errorf("variant %d: %w", i, ErrInvalidBandwidth)
		}
		if variant.Path == "" {
			return "", fmt.Errorf("variant %d: path cannot be empty", i)
		}
		if !isValidResolution(variant.Resolution) {
			return "", fmt.Errorf("variant %d: %w: %s", i, ErrInvalidResolution, variant.Resolution)
		}
	}

	var builder strings.Builder

	// Write HLS header
	builder.WriteString("#EXTM3U\n")
	builder.WriteString(fmt.Sprintf("#EXT-X-VERSION:%d\n", hlsVersion))

	// Write each variant
	for _, variant := range variants {
		builder.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s\n",
			variant.Bandwidth, variant.Resolution))
		builder.WriteString(fmt.Sprintf("%s\n", variant.Path))
	}

	return builder.String(), nil
}

// GenerateMediaPlaylist generates an HLS media playlist from segments
func GenerateMediaPlaylist(segments []Segment, config MediaPlaylistConfig) (string, error) {
	// Validate playlist type
	if config.PlaylistType != PlaylistTypeEvent && config.PlaylistType != PlaylistTypeVOD {
		return "", fmt.Errorf("%w: %s", ErrInvalidPlaylistType, config.PlaylistType)
	}

	// Apply sliding window if MaxSegments is set
	mediaSequence := config.MediaSequence
	if config.MaxSegments > 0 && len(segments) > config.MaxSegments {
		// Calculate how many segments will be dropped from the beginning
		droppedSegments := len(segments) - config.MaxSegments
		// Keep only the last MaxSegments
		segments = segments[len(segments)-config.MaxSegments:]
		// Update media sequence to reflect the first kept segment
		mediaSequence += droppedSegments
	}

	// Calculate target duration if not provided
	targetDuration := config.TargetDuration
	if targetDuration == 0 {
		targetDuration = calculateTargetDuration(segments)
	}

	var builder strings.Builder

	// Write HLS header
	builder.WriteString("#EXTM3U\n")
	builder.WriteString(fmt.Sprintf("#EXT-X-VERSION:%d\n", hlsVersion))
	builder.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", targetDuration))
	builder.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", mediaSequence))

	// Write playlist type if specified
	if config.PlaylistType != "" {
		builder.WriteString(fmt.Sprintf("#EXT-X-PLAYLIST-TYPE:%s\n", config.PlaylistType))
	}

	// Write each segment
	for _, segment := range segments {
		builder.WriteString(fmt.Sprintf("#EXTINF:%.1f,\n", segment.Duration))
		builder.WriteString(fmt.Sprintf("%s\n", segment.Path))
	}

	// Add end tag for VOD playlists
	if config.PlaylistType == PlaylistTypeVOD {
		builder.WriteString("#EXT-X-ENDLIST\n")
	}

	return builder.String(), nil
}

// DiscoverSegments scans a directory and discovers all .ts segment files
func DiscoverSegments(directory string) ([]Segment, error) {
	// Validate directory
	info, err := os.Stat(directory)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist yet, return empty list
			return []Segment{}, nil
		}
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: not a directory", ErrInvalidDirectory)
	}

	// Read directory contents
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Pattern to match segment files: *_segment_NNN.ts
	segmentPattern := regexp.MustCompile(`.*_segment_(\d+)\.ts$`)

	type segmentInfo struct {
		sequence int
		filename string
	}

	var segmentInfos []segmentInfo

	// Parse segment files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		matches := segmentPattern.FindStringSubmatch(filename)
		if matches == nil {
			continue
		}

		// Extract sequence number
		sequence, err := strconv.Atoi(matches[1])
		if err != nil {
			logger.Log.Warn().
				Str("filename", filename).
				Msg("Failed to parse segment sequence number")
			continue
		}

		segmentInfos = append(segmentInfos, segmentInfo{
			sequence: sequence,
			filename: filename,
		})
	}

	// Sort by sequence number
	sort.Slice(segmentInfos, func(i, j int) bool {
		return segmentInfos[i].sequence < segmentInfos[j].sequence
	})

	// Convert to Segment slice
	segments := make([]Segment, len(segmentInfos))
	for i, info := range segmentInfos {
		segments[i] = Segment{
			Duration: defaultSegmentLength,
			Path:     info.filename,
		}
	}

	return segments, nil
}

// WritePlaylistAtomic writes a playlist to a file atomically using temp file + rename
func WritePlaylistAtomic(path string, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in the same directory
	tempFile, err := os.CreateTemp(dir, ".playlist-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup on error
	defer func() {
		if tempFile != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}()

	// Write content with LF line endings
	_, err = io.WriteString(tempFile, content)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Sync to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Rename is atomic on POSIX systems
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	// Success - prevent cleanup
	tempFile = nil

	logger.Log.Debug().
		Str("path", path).
		Int("bytes", len(content)).
		Msg("Playlist written atomically")

	return nil
}

// ValidateMasterPlaylist validates an HLS master playlist for RFC 8216 compliance
func ValidateMasterPlaylist(content string) error {
	lines := strings.Split(content, "\n")

	if len(lines) < 2 {
		return fmt.Errorf("%w: playlist too short", ErrMissingRequiredTag)
	}

	// Check for required tags
	if !strings.HasPrefix(lines[0], "#EXTM3U") {
		return fmt.Errorf("%w: #EXTM3U", ErrMissingRequiredTag)
	}

	hasVersion := false
	hasStreamInf := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXT-X-VERSION:") {
			hasVersion = true
		}
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			hasStreamInf = true

			// Validate BANDWIDTH and RESOLUTION are present
			if !strings.Contains(line, "BANDWIDTH=") {
				return fmt.Errorf("missing BANDWIDTH in #EXT-X-STREAM-INF")
			}
			if !strings.Contains(line, "RESOLUTION=") {
				return fmt.Errorf("missing RESOLUTION in #EXT-X-STREAM-INF")
			}
		}
	}

	if !hasVersion {
		return fmt.Errorf("%w: #EXT-X-VERSION", ErrMissingRequiredTag)
	}
	if !hasStreamInf {
		return fmt.Errorf("%w: #EXT-X-STREAM-INF", ErrMissingRequiredTag)
	}

	return nil
}

// ValidateMediaPlaylist validates an HLS media playlist for RFC 8216 compliance
func ValidateMediaPlaylist(content string) error {
	lines := strings.Split(content, "\n")

	if len(lines) < 2 {
		return fmt.Errorf("%w: playlist too short", ErrMissingRequiredTag)
	}

	// Check for required tags
	if !strings.HasPrefix(lines[0], "#EXTM3U") {
		return fmt.Errorf("%w: #EXTM3U", ErrMissingRequiredTag)
	}

	hasVersion := false
	hasTargetDuration := false
	hasMediaSequence := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXT-X-VERSION:") {
			hasVersion = true
		}
		if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
			hasTargetDuration = true
		}
		if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
			hasMediaSequence = true
		}
	}

	if !hasVersion {
		return fmt.Errorf("%w: #EXT-X-VERSION", ErrMissingRequiredTag)
	}
	if !hasTargetDuration {
		return fmt.Errorf("%w: #EXT-X-TARGETDURATION", ErrMissingRequiredTag)
	}
	if !hasMediaSequence {
		return fmt.Errorf("%w: #EXT-X-MEDIA-SEQUENCE", ErrMissingRequiredTag)
	}

	return nil
}

// Helper functions

// calculateTargetDuration calculates the target duration as the ceiling of max segment duration
func calculateTargetDuration(segments []Segment) int {
	if len(segments) == 0 {
		return int(math.Ceil(defaultSegmentLength))
	}

	maxDuration := 0.0
	for _, segment := range segments {
		if segment.Duration > maxDuration {
			maxDuration = segment.Duration
		}
	}

	return int(math.Ceil(maxDuration))
}

// isValidResolution checks if a resolution string is valid (format: WIDTHxHEIGHT)
func isValidResolution(resolution string) bool {
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return false
	}

	// Check both parts are valid positive integers
	width, err1 := strconv.Atoi(parts[0])
	height, err2 := strconv.Atoi(parts[1])

	return err1 == nil && err2 == nil && width > 0 && height > 0
}

// GetBandwidthForQuality returns the bandwidth in bps for a quality level
func GetBandwidthForQuality(quality string) (int, error) {
	switch quality {
	case Quality1080p:
		return (bitrate1080p + audioBitrate) * 1000, nil // Convert kbps to bps
	case Quality720p:
		return (bitrate720p + audioBitrate) * 1000, nil
	case Quality480p:
		return (bitrate480p + audioBitrate) * 1000, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrInvalidQuality, quality)
	}
}

// GetResolutionForQuality returns the resolution string for a quality level
func GetResolutionForQuality(quality string) (string, error) {
	switch quality {
	case Quality1080p:
		return resolution1080p, nil
	case Quality720p:
		return resolution720p, nil
	case Quality480p:
		return resolution480p, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidQuality, quality)
	}
}
