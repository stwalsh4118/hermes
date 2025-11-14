// Package playlist provides HLS media playlist management with sliding window support.
package playlist

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
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
	AddSegment(seg SegmentMeta) ([]string, error)
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
// Returns a list of segment URIs that were pruned (for file deletion).
// When windowSize > 0 and len(segments) >= windowSize, oldest segments are pruned from front.
func (pm *playlistManager) AddSegment(seg SegmentMeta) ([]string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Input validation
	if seg.URI == "" {
		return nil, fmt.Errorf("segment URI cannot be empty")
	}
	if seg.Duration <= 0 {
		return nil, fmt.Errorf("segment duration must be greater than 0")
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

	// Initialize pruned URIs slice
	prunedURIs := []string{}

	// Implement sliding window pruning logic
	if pm.windowSize > 0 && len(pm.segments) >= int(pm.windowSize) {
		// Calculate how many segments to prune to make room for the new one
		// If we have windowSize segments, we need to prune 1 to add the new one
		segmentsToPrune := len(pm.segments) - int(pm.windowSize) + 1

		// Collect URIs of segments to be pruned (from front of slice)
		for i := 0; i < segmentsToPrune; i++ {
			prunedURIs = append(prunedURIs, pm.segments[i].URI)
		}

		// Prune segments from front
		pm.segments = pm.segments[segmentsToPrune:]

		// Increment mediaSequence by number of segments pruned
		pm.mediaSequence += uint64(segmentsToPrune)
	}
	// For VOD/EVENT mode (windowSize == 0), no pruning occurs and mediaSequence stays at 0

	// Add new segment to slice
	pm.segments = append(pm.segments, seg)

	// Log segment addition with observability metrics
	logger.Log.Debug().
		Str("segment_uri", seg.URI).
		Float64("duration", seg.Duration).
		Uint("segment_count", uint(len(pm.segments))).
		Uint64("media_sequence", pm.mediaSequence).
		Uint("window_size", pm.windowSize).
		Int("pruned_count", len(prunedURIs)).
		Msg("Segment added to playlist")

	return prunedURIs, nil
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
// Generates RFC 8216 compliant m3u8 format directly as text using strings.Builder.
func (pm *playlistManager) Write() error {
	startTime := time.Now()

	// Acquire read lock to read current state
	pm.mu.RLock()
	outputPath := pm.outputPath
	segments := make([]SegmentMeta, len(pm.segments))
	copy(segments, pm.segments)
	mediaSequence := pm.mediaSequence
	maxDuration := pm.maxDuration
	windowSize := pm.windowSize
	pm.mu.RUnlock()

	// Generate playlist content using strings.Builder
	var builder strings.Builder

	// Write header
	builder.WriteString("#EXTM3U\n")
	builder.WriteString("#EXT-X-VERSION:3\n")

	// Write media sequence
	builder.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", mediaSequence))

	// Write target duration (ceiling of max duration)
	targetDuration := uint(math.Ceil(maxDuration))
	builder.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", targetDuration))

	// Write each segment
	for _, seg := range segments {
		// Write discontinuity tag if set
		if seg.Discontinuity {
			builder.WriteString("#EXT-X-DISCONTINUITY\n")
		}

		// Write program date-time if present
		// Format as ISO-8601: YYYY-MM-DDTHH:MM:SSZ (e.g., 2025-11-14T01:05:12Z)
		if seg.ProgramDateTime != nil {
			builder.WriteString(fmt.Sprintf("#EXT-X-PROGRAM-DATE-TIME:%s\n", seg.ProgramDateTime.UTC().Format("2006-01-02T15:04:05Z")))
		}

		// Write EXTINF tag with duration (3 decimal places)
		builder.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", seg.Duration))

		// Write segment URI
		builder.WriteString(fmt.Sprintf("%s\n", seg.URI))
	}

	// Write ENDLIST tag only in VOD/EVENT mode (windowSize == 0)
	if windowSize == 0 {
		builder.WriteString("#EXT-X-ENDLIST\n")
	}

	playlistContent := builder.String()
	contentBytes := len(playlistContent)

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in same directory
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

	// Write content to temp file
	if _, err := tempFile.WriteString(playlistContent); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Sync temp file to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomically rename temp file to final path
	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	// Success - prevent cleanup
	tempFile = nil

	// Calculate latency
	latency := time.Since(startTime)

	// Update last successful write timestamp (acquire write lock)
	pm.mu.Lock()
	now := time.Now()
	pm.lastSuccessfulWrite = &now
	pm.mu.Unlock()

	// Log write operation with observability metrics
	logger.Log.Debug().
		Str("path", outputPath).
		Int("bytes", contentBytes).
		Uint("segment_count", uint(len(segments))).
		Uint64("media_sequence", mediaSequence).
		Uint("window_size", windowSize).
		Dur("latency_ms", latency).
		Msg("Playlist written successfully")

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
// Returns a copy of segment URIs to avoid race conditions.
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

// GetWindowSize returns the configured window size (maximum number of segments to keep)
func (pm *playlistManager) GetWindowSize() uint {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.windowSize
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
