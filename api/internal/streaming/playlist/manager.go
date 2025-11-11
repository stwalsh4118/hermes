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

	"github.com/Eyevinn/hls-m3u8/m3u8"
	"github.com/stwalsh4118/hermes/internal/logger"
)

// SegmentMeta contains metadata for a single HLS segment
type SegmentMeta struct {
	URI             string     // Segment filename (e.g., "seg-20250111T120000.ts")
	Duration        float64    // Segment duration in seconds (typically 4.0)
	ProgramDateTime *time.Time // Optional program date-time
	Discontinuity   bool       // Whether to insert discontinuity before this segment
}

// Manager manages a sliding-window HLS media playlist
type Manager interface {
	AddSegment(seg SegmentMeta) error
	SetDiscontinuityNext()
	Write() error
	Close() error
	GetCurrentSegments() []string
}

// playlistManager implements Manager using hls-m3u8 library
type playlistManager struct {
	mu                sync.RWMutex
	playlist          *m3u8.MediaPlaylist
	outputPath        string
	windowSize        uint
	maxDuration       float64
	discontinuityNext bool
	totalSegments     uint64 // Track total segments added for SeqNo calculation
}

const (
	// defaultCapacity is the initial capacity for the segment list
	// Should be larger than windowSize to avoid frequent reallocations
	defaultCapacity = 100
)

// NewManager creates a new playlist manager instance
func NewManager(windowSize uint, outputPath string, initialTargetDuration float64) (Manager, error) {
	if windowSize == 0 {
		return nil, fmt.Errorf("window size must be greater than 0")
	}
	if outputPath == "" {
		return nil, fmt.Errorf("output path cannot be empty")
	}
	if initialTargetDuration <= 0 {
		return nil, fmt.Errorf("initial target duration must be greater than 0")
	}

	// Create media playlist with window size and capacity
	// winsize: sliding window size (0 = VOD/EVENT, >0 = live sliding window)
	// capacity: initial capacity of segment list
	playlist, err := m3u8.NewMediaPlaylist(windowSize, defaultCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create media playlist: %w", err)
	}

	// Set initial media sequence (starts at 0)
	playlist.SeqNo = 0

	// Set initial target duration (convert float64 to uint as library expects)
	playlist.TargetDuration = uint(math.Ceil(initialTargetDuration))

	return &playlistManager{
		playlist:          playlist,
		outputPath:        outputPath,
		windowSize:        windowSize,
		maxDuration:       initialTargetDuration,
		discontinuityNext: false,
		totalSegments:     0,
	}, nil
}

// AddSegment adds a new segment to the playlist
func (pm *playlistManager) AddSegment(seg SegmentMeta) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if seg.URI == "" {
		return fmt.Errorf("segment URI cannot be empty")
	}
	if seg.Duration <= 0 {
		return fmt.Errorf("segment duration must be greater than 0")
	}

	// Update max duration and target duration if needed
	if seg.Duration > pm.maxDuration {
		pm.maxDuration = seg.Duration
		pm.playlist.TargetDuration = uint(math.Ceil(pm.maxDuration))
	}

	// Increment total segments added (do this before creating segment)
	pm.totalSegments++

	// Create media segment with sequence ID
	mediaSeg := &m3u8.MediaSegment{
		SeqId:         pm.totalSegments - 1, // SeqId is 0-indexed
		URI:           seg.URI,
		Duration:      seg.Duration,
		Title:         "", // Empty title for now
		Discontinuity: pm.discontinuityNext || seg.Discontinuity,
	}

	// Set program date-time if provided
	if seg.ProgramDateTime != nil {
		mediaSeg.ProgramDateTime = *seg.ProgramDateTime
	}

	// Clear discontinuity flag after use
	if pm.discontinuityNext {
		pm.discontinuityNext = false
	}

	// Append segment to playlist
	// The library handles sliding window automatically
	err := pm.playlist.AppendSegment(mediaSeg)
	if err != nil {
		return fmt.Errorf("failed to append segment %s: %w", seg.URI, err)
	}

	// Update media sequence to reflect pruned segments
	// SeqNo represents the sequence number of the first segment in the playlist
	// When window size is exceeded, oldest segments are pruned
	// SeqNo should be the SeqId of the first segment remaining
	currentCount := pm.playlist.Count()
	if pm.totalSegments > uint64(currentCount) {
		// Segments have been pruned, update SeqNo to first remaining segment's SeqId
		pm.playlist.SeqNo = pm.totalSegments - uint64(currentCount)
	} else {
		// No pruning, SeqNo should be 0 (first segment)
		pm.playlist.SeqNo = 0
	}

	logger.Log.Debug().
		Str("uri", seg.URI).
		Float64("duration", seg.Duration).
		Uint64("media_sequence", pm.playlist.SeqNo).
		Uint("target_duration", pm.playlist.TargetDuration).
		Uint("segments", pm.playlist.Count()).
		Msg("Segment added to playlist")

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

// Write writes the playlist to disk atomically
func (pm *playlistManager) Write() error {
	pm.mu.Lock()

	// Ensure SeqNo is up to date before encoding
	// SeqNo represents the sequence number of the first segment in the playlist
	// After pruning, SeqNo = totalSegments - currentCount
	currentCount := pm.playlist.Count()
	if pm.totalSegments > uint64(currentCount) {
		// Segments have been pruned
		pm.playlist.SeqNo = pm.totalSegments - uint64(currentCount)
	} else {
		// No pruning, SeqNo starts at 0
		pm.playlist.SeqNo = 0
	}

	// Encode playlist to m3u8 format (returns *bytes.Buffer)
	buf := pm.playlist.Encode()
	if buf == nil {
		return fmt.Errorf("failed to encode playlist")
	}
	content := buf.Bytes()

	// Store values needed after lock release
	outputPath := pm.outputPath
	seqNo := pm.playlist.SeqNo
	segCount := pm.playlist.Count()

	// Release lock before file I/O operations (file I/O can be slow)
	pm.mu.Unlock()

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

	// Write content
	if _, err := tempFile.Write(content); err != nil {
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

	// Atomic rename (POSIX systems)
	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	// Success - prevent cleanup
	tempFile = nil

	logger.Log.Debug().
		Str("path", outputPath).
		Int("bytes", len(content)).
		Uint64("media_sequence", seqNo).
		Uint("segments", segCount).
		Msg("Playlist written atomically")

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

// GetCurrentSegments returns the list of segment URIs currently in the playlist window
func (pm *playlistManager) GetCurrentSegments() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	count := pm.playlist.Count()
	if count == 0 {
		return []string{}
	}

	// Encode playlist to get current segments (library handles window correctly)
	buf := pm.playlist.Encode()
	if buf == nil {
		return []string{}
	}

	// Parse segment URIs from encoded playlist
	content := buf.String()
	lines := strings.Split(content, "\n")
	segments := make([]string, 0, count)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Segment URIs appear on lines that don't start with # and end with .ts
		if line != "" && !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ".ts") {
			segments = append(segments, line)
		}
	}

	return segments
}
