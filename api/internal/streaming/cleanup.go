package streaming

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
)

// Cleanup errors
var (
	ErrDirectoryCreation = fmt.Errorf("failed to create directory")
)

// createSegmentDirectories creates the necessary directories for stream segments
func createSegmentDirectories(baseDir, channelID string) error {
	// Create base directory for channel (baseDir already includes channel ID)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("%w: %w", ErrDirectoryCreation, err)
	}

	// Create quality-specific directories
	qualities := []string{Quality1080p, Quality720p, Quality480p}
	for _, quality := range qualities {
		qualityDir := filepath.Join(baseDir, quality)
		if err := os.MkdirAll(qualityDir, 0755); err != nil {
			return fmt.Errorf("%w for quality %s: %w", ErrDirectoryCreation, quality, err)
		}
	}

	logger.Log.Debug().
		Str("channel_id", channelID).
		Str("base_dir", baseDir).
		Msg("Segment directories created")

	return nil
}

// cleanupSegments removes all segment files and directories for a channel
func cleanupSegments(outputDir string) error {
	// Check if directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to clean up
		logger.Log.Debug().
			Str("output_dir", outputDir).
			Msg("Output directory does not exist, nothing to cleanup")
		return nil
	}

	// Remove entire directory tree
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to remove output directory: %w", err)
	}

	logger.Log.Info().
		Str("output_dir", outputDir).
		Msg("Segments cleaned up successfully")

	return nil
}

// cleanupOrphanedDirectories removes segment directories for channels that no longer have active streams
func cleanupOrphanedDirectories(baseDir string, activeSessions []*models.StreamSession) error {
	// Check if base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to clean up
		return nil
	}

	// Build set of active channel IDs
	activeChannels := make(map[string]bool)
	for _, session := range activeSessions {
		activeChannels[session.ChannelID.String()] = true
	}

	// Read all directories in base directory
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("failed to read base directory: %w", err)
	}

	orphanedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		channelID := entry.Name()

		// Skip if this is an active channel
		if activeChannels[channelID] {
			continue
		}

		// This is an orphaned directory - check if it looks like a UUID
		if !isLikelyChannelID(channelID) {
			logger.Log.Warn().
				Str("directory", channelID).
				Msg("Skipping directory that doesn't look like a channel ID")
			continue
		}

		// Remove orphaned directory
		dirPath := filepath.Join(baseDir, channelID)
		if err := os.RemoveAll(dirPath); err != nil {
			logger.Log.Warn().
				Err(err).
				Str("directory", dirPath).
				Msg("Failed to remove orphaned directory")
		} else {
			logger.Log.Info().
				Str("directory", dirPath).
				Msg("Removed orphaned segment directory")
			orphanedCount++
		}
	}

	if orphanedCount > 0 {
		logger.Log.Info().
			Int("orphaned_count", orphanedCount).
			Str("base_dir", baseDir).
			Msg("Orphaned directories cleaned up")
	}

	return nil
}

// isLikelyChannelID checks if a string looks like a UUID
// This is a simple heuristic to avoid deleting non-channel directories
func isLikelyChannelID(name string) bool {
	// UUIDs are 36 characters with hyphens in specific positions
	// Example: 550e8400-e29b-41d4-a716-446655440000
	if len(name) != 36 {
		return false
	}

	// Check for hyphens in the right positions
	if name[8] != '-' || name[13] != '-' || name[18] != '-' || name[23] != '-' {
		return false
	}

	// Check that other characters are hex digits
	hexChars := "0123456789abcdefABCDEF"
	for i, c := range name {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !strings.ContainsRune(hexChars, c) {
			return false
		}
	}

	return true
}

// getDirectorySize calculates the total size of a directory in bytes
// This is currently unused but may be useful for monitoring disk usage
func getDirectorySize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// Suppress unused warning - this function may be used for future monitoring features
var _ = getDirectorySize

// cleanupOldSegments removes segment files older than a certain age
// This can be used for additional cleanup beyond what FFmpeg's delete_segments does
// Currently unused but reserved for future use
func cleanupOldSegments(_ string, _ int64) error {
	// This is a helper function for future use
	// Currently, FFmpeg's hls_flags delete_segments handles most cleanup
	// But this provides additional safety net
	return nil
}

// Suppress unused warning - reserved for future monitoring
var _ = cleanupOldSegments

// cleanupOldBatches removes segments from N-2 batch (two batches ago) after N batch completes successfully.
// This keeps N-1 batch available during N batch generation to prevent gaps if generation fails.
// Only cleans up when batch number >= 2 (need at least 2 batches before cleanup).
func cleanupOldBatches(session *models.StreamSession, batchSize int, outputDir string, quality string) {
	currentBatch := session.GetCurrentBatch()
	if currentBatch == nil || currentBatch.BatchNumber < 2 {
		// Need at least 2 batches before cleanup
		return
	}

	// Calculate batch to delete (N-2)
	batchToDelete := currentBatch.BatchNumber - 2

	// Calculate segment range for that batch
	startSegment := batchToDelete * batchSize
	endSegment := startSegment + batchSize - 1

	channelIDStr := session.ChannelID.String()

	logger.Log.Debug().
		Str("channel_id", channelIDStr).
		Int("current_batch", currentBatch.BatchNumber).
		Int("batch_to_delete", batchToDelete).
		Int("start_segment", startSegment).
		Int("end_segment", endSegment).
		Msg("Cleaning up old batch segments")

	// Build quality directory path
	qualityDir := filepath.Join(outputDir, quality)

	// Iterate through segment range and delete segment files
	deletedCount := 0
	for logicalSegment := startSegment; logicalSegment <= endSegment; logicalSegment++ {
		// Handle segment filename wrapping (%03d pattern wraps at 1000)
		filenameSegment := logicalSegment % 1000

		// Build segment filename: {quality}_segment_{NNN}.ts
		segmentFilename := fmt.Sprintf("%s_segment_%03d.ts", quality, filenameSegment)
		segmentPath := filepath.Join(qualityDir, segmentFilename)

		// Delete segment file
		if err := os.Remove(segmentPath); err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist - may have been deleted already, ignore
				continue
			}
			// Log other errors but don't fail (best-effort cleanup)
			logger.Log.Warn().
				Err(err).
				Str("channel_id", channelIDStr).
				Str("segment_path", segmentPath).
				Int("logical_segment", logicalSegment).
				Int("filename_segment", filenameSegment).
				Msg("Failed to delete old segment")
		} else {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		logger.Log.Info().
			Str("channel_id", channelIDStr).
			Int("batch_to_delete", batchToDelete).
			Int("deleted_count", deletedCount).
			Int("total_segments", endSegment-startSegment+1).
			Msg("Cleaned up old batch segments")
	}
}
