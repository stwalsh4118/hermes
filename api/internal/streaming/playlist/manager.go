// Package playlist provides HLS media playlist management with sliding window support.
package playlist

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/stwalsh4118/hermes/internal/logger"
)

// SegmentMeta contains metadata for a single HLS segment
type SegmentMeta struct {
	URI             string     // Segment filename (e.g., "seg-20250111T120000.ts")
	Duration        float64    // Segment duration in seconds (typically 4.0)
	ProgramDateTime *time.Time // Optional program date-time
	Discontinuity   bool       // Whether to insert discontinuity before this segment
}

// HealthStatus represents the health status of a playlist manager
type HealthStatus struct {
	Healthy            bool          // Whether the playlist is healthy
	LastWriteTime      *time.Time    // Last successful write timestamp
	TimeSinceLastWrite time.Duration // Time elapsed since last write
	WindowSize         uint          // Current window size
	MaxDuration        float64       // Maximum observed segment duration
	StaleThreshold     time.Duration // Threshold for considering playlist stale
}

// Manager manages a sliding-window HLS media playlist.
// Provides full control over playlist generation with manual segment tracking
// and sliding window management.
type Manager interface {
	AddSegment(seg SegmentMeta) error
	SetDiscontinuityNext()
	Write() error
	Close() error
	GetCurrentSegments() []string
	GetLastSuccessfulWrite() *time.Time
	GetWindowSize() uint
	GetMaxDuration() float64
	HealthCheck(staleThreshold time.Duration) HealthStatus
	// GetMediaSequence returns the current media sequence number.
	// Media sequence starts at 0 and increments by 1 for each segment pruned
	// from the front of the playlist (when windowSize > 0).
	// For VOD/EVENT mode (windowSize == 0), media sequence stays at 0.
	GetMediaSequence() uint64
	// GetSegmentCount returns the total number of segments currently in the playlist.
	// This is the length of the segments slice, which may be less than totalSegments
	// if segments have been pruned in sliding window mode.
	GetSegmentCount() uint
}

// playlistManager implements Manager using simple Go data structures.
// This replaces the buggy hls-m3u8 library with a custom implementation
// that provides full control over playlist generation.
//
// Thread-Safety:
//   - All field access is protected by sync.RWMutex
//   - Read operations use RLock()/RUnlock()
//   - Write operations use Lock()/Unlock()
//   - File I/O operations release lock before performing I/O to avoid blocking
//
// Modes:
//   - VOD/EVENT mode (windowSize == 0): Keeps all segments, no pruning, media sequence stays at 0
//   - Live mode (windowSize > 0): Sliding window that prunes old segments, media sequence increments on prune
//
// Media Sequence Tracking:
//   - Starts at 0
//   - When windowSize > 0 and segments are pruned (removed from front), mediaSequence increments
//     by the number of segments removed
//   - Formula: mediaSequence = totalSegments - len(segments) when pruning occurs
//   - For VOD/EVENT mode, media sequence stays at 0 (no pruning)
//
// Sliding Window Behavior:
//   - When len(segments) >= windowSize, oldest segments are removed from front
//   - Pruning happens automatically in AddSegment when window size is exceeded
//   - Media sequence is updated to reflect the first remaining segment's sequence number
type playlistManager struct {
	mu sync.RWMutex // Thread-safety mutex for all field access

	// segments is a simple slice of SegmentMeta that we control manually.
	// For VOD/EVENT mode (windowSize == 0), all segments are kept.
	// For live mode (windowSize > 0), segments beyond windowSize are pruned from front.
	segments []SegmentMeta

	outputPath string // Full path to output .m3u8 file
	segmentDir string // Directory path where segments are stored (for file cleanup operations)

	windowSize  uint    // Number of segments to keep in sliding window (0 = VOD/EVENT, >0 = live)
	maxDuration float64 // Maximum observed segment duration (for TARGETDURATION calculation)

	// mediaSequence tracks the media sequence number manually.
	// Starts at 0, increments by 1 for each segment pruned from the front.
	// For VOD/EVENT mode, stays at 0 since no pruning occurs.
	mediaSequence uint64

	// totalSegments tracks the total count of segments ever added.
	// Used for sequence number calculation: mediaSequence = totalSegments - len(segments)
	totalSegments uint64

	discontinuityNext   bool       // Flag to insert discontinuity tag before next segment
	lastSuccessfulWrite *time.Time // Timestamp of last successful playlist write (for health checks)
}

// NewManager creates a new playlist manager instance.
//
// Parameters:
//   - windowSize: 0 = VOD/EVENT mode (no sliding window, keep all segments), >0 = live sliding window mode
//   - outputPath: Full path to output .m3u8 file
//   - initialTargetDuration: Initial target duration in seconds (must be > 0)
//
// The segmentDir is derived from outputPath by taking the directory portion.
// This maintains backward compatibility with existing callers.
// Future tasks may add an explicit segmentDir parameter if needed.
func NewManager(windowSize uint, outputPath string, initialTargetDuration float64) (Manager, error) {
	if outputPath == "" {
		return nil, fmt.Errorf("output path cannot be empty")
	}
	if initialTargetDuration <= 0 {
		return nil, fmt.Errorf("initial target duration must be greater than 0")
	}

	// Derive segmentDir from outputPath for backward compatibility
	// This is the directory where segment files are stored (for cleanup operations)
	segmentDir := filepath.Dir(outputPath)

	return &playlistManager{
		segments:            make([]SegmentMeta, 0),
		outputPath:          outputPath,
		segmentDir:          segmentDir,
		windowSize:          windowSize,
		maxDuration:         initialTargetDuration,
		mediaSequence:       0, // Starts at 0
		totalSegments:       0, // No segments added yet
		discontinuityNext:   false,
		lastSuccessfulWrite: nil, // No write has occurred yet
	}, nil
}

// AddSegment adds a new segment to the playlist.
// TODO: Full implementation with sliding window logic and file cleanup will be completed in task 14-2.
// This is a minimal implementation for the design phase.
func (pm *playlistManager) AddSegment(seg SegmentMeta) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if seg.URI == "" {
		return fmt.Errorf("segment URI cannot be empty")
	}
	if seg.Duration <= 0 {
		return fmt.Errorf("segment duration must be greater than 0")
	}

	// Update max duration if needed
	if seg.Duration > pm.maxDuration {
		pm.maxDuration = seg.Duration
	}

	// Increment total segments added
	pm.totalSegments++

	// Apply discontinuity flag if set
	seg.Discontinuity = pm.discontinuityNext || seg.Discontinuity
	if pm.discontinuityNext {
		pm.discontinuityNext = false
	}

	// Add segment to slice
	pm.segments = append(pm.segments, seg)

	// TODO: Implement sliding window pruning logic in task 14-2
	// When windowSize > 0 and len(segments) >= windowSize, prune from front
	// and update mediaSequence accordingly

	return nil
}

// SetDiscontinuityNext flags the next segment to have a discontinuity tag
func (pm *playlistManager) SetDiscontinuityNext() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.discontinuityNext = true

	logger.Log.Debug().
		Msg("Discontinuity flag set for next segment")
}

// Write writes the playlist to disk atomically using temp file + rename pattern.
// TODO: Full implementation with direct m3u8 text generation will be completed in task 14-3.
// This is a minimal implementation for the design phase.
func (pm *playlistManager) Write() error {
	pm.mu.RLock()
	// Store values needed after lock release
	outputPath := pm.outputPath
	segmentCount := len(pm.segments)
	pm.mu.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// TODO: Generate m3u8 format directly as text in task 14-3
	// For now, create empty file as placeholder
	tempFile, err := os.CreateTemp(dir, ".playlist-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	defer func() {
		if tempFile != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}()

	// Write placeholder content
	placeholder := fmt.Sprintf("#EXTM3U\n#EXT-X-VERSION:3\n# TODO: Full playlist generation in task 14-3 (segments: %d)\n", segmentCount)
	if _, err := tempFile.WriteString(placeholder); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	tempFile = nil

	// Update last successful write timestamp
	pm.mu.Lock()
	now := time.Now()
	pm.lastSuccessfulWrite = &now
	pm.mu.Unlock()

	return nil
}

// Close performs final write and cleanup
func (pm *playlistManager) Close() error {
	// Final write before closing
	if err := pm.Write(); err != nil {
		logger.Log.Warn().
			Err(err).
			Msg("Failed to write playlist during close")
		return fmt.Errorf("failed to write playlist during close: %w", err)
	}

	logger.Log.Debug().
		Str("path", pm.outputPath).
		Msg("Playlist manager closed")

	return nil
}

// GetCurrentSegments returns the list of segment URIs currently in the playlist window.
// TODO: Full implementation will be completed in task 14-4.
// For now, returns segment URIs from the segments slice.
func (pm *playlistManager) GetCurrentSegments() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.segments) == 0 {
		return []string{}
	}

	segments := make([]string, 0, len(pm.segments))
	for _, seg := range pm.segments {
		segments = append(segments, seg.URI)
	}

	return segments
}

// GetLastSuccessfulWrite returns the timestamp of the last successful playlist write
func (pm *playlistManager) GetLastSuccessfulWrite() *time.Time {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.lastSuccessfulWrite
}

// GetWindowSize returns the current window size (number of segments in playlist)
func (pm *playlistManager) GetWindowSize() uint {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return uint(len(pm.segments))
}

// GetMaxDuration returns the maximum observed segment duration
func (pm *playlistManager) GetMaxDuration() float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.maxDuration
}

// HealthCheck checks the health of the playlist manager based on last successful write
// A playlist is considered unhealthy if no write has occurred within staleThreshold duration
func (pm *playlistManager) HealthCheck(staleThreshold time.Duration) HealthStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status := HealthStatus{
		LastWriteTime:  pm.lastSuccessfulWrite,
		WindowSize:     uint(len(pm.segments)),
		MaxDuration:    pm.maxDuration,
		StaleThreshold: staleThreshold,
	}

	if pm.lastSuccessfulWrite == nil {
		// No write has occurred yet
		status.Healthy = false
		status.TimeSinceLastWrite = 0
		return status
	}

	status.TimeSinceLastWrite = time.Since(*pm.lastSuccessfulWrite)
	status.Healthy = status.TimeSinceLastWrite < staleThreshold

	return status
}

// GetMediaSequence returns the current media sequence number.
// Media sequence starts at 0 and increments by 1 for each segment pruned
// from the front of the playlist (when windowSize > 0).
// For VOD/EVENT mode (windowSize == 0), media sequence stays at 0.
func (pm *playlistManager) GetMediaSequence() uint64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.mediaSequence
}

// GetSegmentCount returns the total number of segments currently in the playlist.
// This is the length of the segments slice, which may be less than totalSegments
// if segments have been pruned in sliding window mode.
func (pm *playlistManager) GetSegmentCount() uint {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return uint(len(pm.segments))
}
