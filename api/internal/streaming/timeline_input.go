package streaming

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
	"github.com/stwalsh4118/hermes/internal/timeline"
)

// Constants for timeline input optimization
const (
	SeekOptimizationThreshold = 10   // Skip seeks < 10 seconds for faster startup
	ConcatThreshold           = 30   // Use concat if < 30s remaining for smooth transitions
	MaxStreamDuration         = 7200 // Max 2 hours of content
	MaxConcatFiles            = 10   // Limit concat list size
)

// Errors
var (
	ErrInvalidOffset = errors.New("seek offset exceeds media duration")
	ErrFileNotFound  = errors.New("media file not found")
	ErrEmptyPlaylist = errors.New("playlist is empty")
)

// TimelineInput represents FFmpeg input configuration from timeline position
type TimelineInput struct {
	PrimaryFile    string       // Main input file path
	SeekSeconds    int64        // Seek position in primary file (0 = start)
	UseConcatFile  bool         // Whether to use concat protocol
	ConcatFilePath string       // Path to generated concat.txt (if used)
	ConcatItems    []ConcatItem // Files to concatenate
	TotalDuration  int64        // Total duration to stream (seconds)
}

// ConcatItem represents a file in FFmpeg concat demuxer format
type ConcatItem struct {
	FilePath string // Absolute path to media file
	InPoint  int64  // Start time within file (seconds, 0 = start)
	OutPoint int64  // End time within file (0 = use all)
}

// BuildTimelineInput converts a channel's timeline position into FFmpeg input parameters.
// It integrates with the Timeline Service to get the current playback position and builds
// the appropriate input configuration, including seeking and file concatenation as needed.
func BuildTimelineInput(
	ctx context.Context,
	channelID uuid.UUID,
	timelineService *timeline.TimelineService,
	repos *db.Repositories,
) (*TimelineInput, error) {
	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Msg("Building timeline input for channel")

	// Get current timeline position
	position, err := timelineService.GetCurrentPosition(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline position: %w", err)
	}

	// Fetch channel information
	channel, err := repos.Channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Fetch full playlist with media details
	playlist, err := repos.PlaylistItems.GetWithMedia(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	if len(playlist) == 0 {
		return nil, ErrEmptyPlaylist
	}

	// Find current item in playlist
	var currentItem *models.PlaylistItem
	var currentPosition int
	for i, item := range playlist {
		if item.MediaID == position.MediaID {
			currentItem = item
			currentPosition = i
			break
		}
	}

	if currentItem == nil || currentItem.Media == nil {
		return nil, fmt.Errorf("current media item not found in playlist")
	}

	// Get current media file path
	currentFilePath := currentItem.Media.FilePath

	// Validate offset is within bounds
	if position.OffsetSeconds > position.Duration {
		return nil, fmt.Errorf("%w: offset %d exceeds duration %d",
			ErrInvalidOffset, position.OffsetSeconds, position.Duration)
	}

	// Calculate remaining duration in current item
	remainingSeconds := position.Duration - position.OffsetSeconds

	// Determine if we should use concat protocol for smooth transitions
	shouldConcat := remainingSeconds < ConcatThreshold

	// Build input based on strategy
	if shouldConcat {
		return buildConcatInput(ctx, currentFilePath, position.OffsetSeconds,
			remainingSeconds, playlist, currentPosition, channel.Loop)
	}

	return buildSimpleInput(currentFilePath, position.OffsetSeconds, remainingSeconds)
}

// buildSimpleInput creates a simple seek-based input for a single file
func buildSimpleInput(filePath string, offsetSeconds, remainingSeconds int64) (*TimelineInput, error) {
	// Validate file exists
	if err := validateFilePath(filePath); err != nil {
		return nil, err
	}

	// Optimization: skip seeking if near the start (faster startup)
	seekSeconds := offsetSeconds
	if offsetSeconds < SeekOptimizationThreshold {
		seekSeconds = 0
		logger.Log.Debug().
			Int64("original_offset", offsetSeconds).
			Msg("Skipping seek optimization for fast startup")
	}

	input := &TimelineInput{
		PrimaryFile:   filePath,
		SeekSeconds:   seekSeconds,
		UseConcatFile: false,
		TotalDuration: remainingSeconds,
	}

	logger.Log.Info().
		Str("file", filePath).
		Int64("seek_seconds", seekSeconds).
		Int64("duration", remainingSeconds).
		Msg("Built simple timeline input")

	return input, nil
}

// buildConcatInput creates a concat-based input for seamless transitions
func buildConcatInput(
	_ context.Context,
	currentFilePath string,
	offsetSeconds int64,
	remainingSeconds int64,
	playlist []*models.PlaylistItem,
	currentPosition int,
	loop bool,
) (*TimelineInput, error) {
	// Build concat items list starting with current file
	concatItems := []ConcatItem{
		{
			FilePath: currentFilePath,
			InPoint:  offsetSeconds,
			OutPoint: 0, // Use all remaining
		},
	}

	// Get next playlist items for smooth transition
	nextItems := GetNextPlaylistItems(playlist, currentPosition, MaxConcatFiles-1, loop)

	// Add next items to concat list
	for _, item := range nextItems {
		if item.Media == nil {
			continue
		}
		concatItems = append(concatItems, ConcatItem{
			FilePath: item.Media.FilePath,
			InPoint:  0,
			OutPoint: 0,
		})
	}

	// Validate all file paths
	if err := ValidateFilePaths(concatItems); err != nil {
		return nil, err
	}

	// Calculate total duration
	totalDuration := CalculateStreamDuration(remainingSeconds, nextItems, MaxStreamDuration)

	// Generate concat file path in temp directory
	concatFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("hermes-concat-%s.txt", uuid.New().String()))

	// Build concat file
	if err := BuildConcatFile(concatItems, concatFilePath); err != nil {
		return nil, fmt.Errorf("failed to build concat file: %w", err)
	}

	input := &TimelineInput{
		PrimaryFile:    concatFilePath,
		SeekSeconds:    0, // Seeking is handled in concat file via inpoint
		UseConcatFile:  true,
		ConcatFilePath: concatFilePath,
		ConcatItems:    concatItems,
		TotalDuration:  totalDuration,
	}

	logger.Log.Info().
		Str("concat_file", concatFilePath).
		Int("file_count", len(concatItems)).
		Int64("total_duration", totalDuration).
		Msg("Built concat timeline input")

	return input, nil
}

// GetNextPlaylistItems returns the next N items from the playlist, handling looping
func GetNextPlaylistItems(
	playlist []*models.PlaylistItem,
	currentPosition int,
	count int,
	loop bool,
) []*models.PlaylistItem {
	if len(playlist) == 0 || count <= 0 {
		return nil
	}

	result := make([]*models.PlaylistItem, 0, count)
	playlistLen := len(playlist)

	for i := 0; i < count; i++ {
		nextPos := currentPosition + i + 1

		// Handle looping
		if nextPos >= playlistLen {
			if !loop {
				// Non-looping playlist reached the end
				break
			}
			nextPos %= playlistLen
		}

		result = append(result, playlist[nextPos])
	}

	return result
}

// CalculateStreamDuration calculates the total duration to stream, capped at maxDuration
func CalculateStreamDuration(
	remainingCurrent int64,
	nextItems []*models.PlaylistItem,
	maxDuration int64,
) int64 {
	total := remainingCurrent

	for _, item := range nextItems {
		if item.Media == nil {
			continue
		}

		// Stop if we've reached the max duration
		if total+item.Media.Duration > maxDuration {
			return maxDuration
		}

		total += item.Media.Duration
	}

	return total
}

// BuildConcatFile generates an FFmpeg concat demuxer format file
func BuildConcatFile(items []ConcatItem, outputPath string) error {
	if len(items) == 0 {
		return errors.New("cannot create concat file with zero items")
	}

	var builder strings.Builder

	for _, item := range items {
		// Write file directive with absolute path
		// FFmpeg concat format requires single quotes around paths
		builder.WriteString(fmt.Sprintf("file '%s'\n", item.FilePath))

		// Write inpoint if non-zero (in seconds)
		if item.InPoint > 0 {
			builder.WriteString(fmt.Sprintf("inpoint %d\n", item.InPoint))
		}

		// Write outpoint if specified (0 means use all)
		if item.OutPoint > 0 {
			builder.WriteString(fmt.Sprintf("outpoint %d\n", item.OutPoint))
		}
	}

	content := builder.String()

	// Write atomically using temp file + rename
	tempPath := outputPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write concat file: %w", err)
	}

	if err := os.Rename(tempPath, outputPath); err != nil {
		_ = os.Remove(tempPath) // Cleanup on error (best effort)
		return fmt.Errorf("failed to rename concat file: %w", err)
	}

	logger.Log.Debug().
		Str("path", outputPath).
		Int("items", len(items)).
		Msg("Created concat file")

	return nil
}

// ValidateFilePaths checks that all file paths exist and are absolute
func ValidateFilePaths(items []ConcatItem) error {
	for i, item := range items {
		if err := validateFilePath(item.FilePath); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
	}
	return nil
}

// validateFilePath checks a single file path
func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path is empty")
	}

	if !filepath.IsAbs(path) {
		return fmt.Errorf("file path must be absolute: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	return nil
}
