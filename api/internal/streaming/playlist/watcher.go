// Package playlist provides HLS media playlist management with sliding window support.
package playlist

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stwalsh4118/hermes/internal/logger"
)

const (
	defaultSafetyBuffer    = 2                // segments beyond window
	defaultPruneInterval   = 30 * time.Second // seconds
	defaultPollInterval    = 1 * time.Second  // seconds
	defaultSegmentDuration = 4.0              // seconds
	debounceWindow         = 500 * time.Millisecond
)

// Watcher watches a directory for new TS segments and notifies the playlist manager
type Watcher interface {
	Start() error
	Stop() error
	MarkDiscontinuity() // Signal that encoder has restarted and next segment should have discontinuity tag
}

// segmentWatcher implements Watcher using fsnotify with polling fallback
type segmentWatcher struct {
	segmentDir      string
	playlistManager Manager
	windowSize      uint
	safetyBuffer    uint
	pruneInterval   time.Duration
	segmentDuration float64
	pollInterval    time.Duration

	fsnotifyWatcher *fsnotify.Watcher
	stopChan        chan struct{}
	pruneDone       chan struct{}
	watchDone       chan struct{}

	mu                   sync.RWMutex
	pendingNotifications map[string]time.Time // filename -> first seen time
	stopped              bool
	lastSegmentTime      *time.Time // Track last segment's ProgramDateTime for regression detection
	lastSegmentDetected  *time.Time // Track last segment detection time for cadence calculation
}

// NewWatcher creates a new segment watcher instance
func NewWatcher(
	segmentDir string,
	playlistManager Manager,
	windowSize uint,
	safetyBuffer uint,
	pruneInterval time.Duration,
	segmentDuration float64,
	pollInterval time.Duration,
) (Watcher, error) {
	if segmentDir == "" {
		return nil, fmt.Errorf("segment directory cannot be empty")
	}
	if playlistManager == nil {
		return nil, fmt.Errorf("playlist manager cannot be nil")
	}
	if windowSize == 0 {
		return nil, fmt.Errorf("window size must be greater than 0")
	}
	if segmentDuration <= 0 {
		return nil, fmt.Errorf("segment duration must be greater than 0")
	}
	if pruneInterval <= 0 {
		return nil, fmt.Errorf("prune interval must be greater than 0")
	}
	if pollInterval <= 0 {
		return nil, fmt.Errorf("poll interval must be greater than 0")
	}

	// Ensure segment directory exists
	if err := os.MkdirAll(segmentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create segment directory: %w", err)
	}

	return &segmentWatcher{
		segmentDir:           segmentDir,
		playlistManager:      playlistManager,
		windowSize:           windowSize,
		safetyBuffer:         safetyBuffer,
		pruneInterval:        pruneInterval,
		segmentDuration:      segmentDuration,
		pollInterval:         pollInterval,
		stopChan:             make(chan struct{}),
		pruneDone:            make(chan struct{}),
		watchDone:            make(chan struct{}),
		pendingNotifications: make(map[string]time.Time),
		stopped:              false,
	}, nil
}

// Start begins watching for new segments and starts the pruning goroutine
func (sw *segmentWatcher) Start() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.stopped {
		return fmt.Errorf("watcher has been stopped")
	}

	// Try to create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("segment_dir", sw.segmentDir).
			Msg("Failed to create fsnotify watcher, falling back to polling")
		// Fallback to polling - watcher will be nil
		sw.fsnotifyWatcher = nil
	} else {
		sw.fsnotifyWatcher = watcher
		// Add segment directory to watcher
		if err := watcher.Add(sw.segmentDir); err != nil {
			logger.Log.Warn().
				Err(err).
				Str("segment_dir", sw.segmentDir).
				Msg("Failed to add directory to fsnotify watcher, falling back to polling")
			_ = watcher.Close()
			sw.fsnotifyWatcher = nil
		}
	}

	// Start watching goroutine
	go sw.runWatching()

	// Start pruning goroutine
	go sw.runPruning()

	logger.Log.Info().
		Str("segment_dir", sw.segmentDir).
		Bool("using_fsnotify", sw.fsnotifyWatcher != nil).
		Dur("prune_interval", sw.pruneInterval).
		Msg("Segment watcher started")

	return nil
}

// Stop gracefully stops the watcher
func (sw *segmentWatcher) Stop() error {
	sw.mu.Lock()
	if sw.stopped {
		sw.mu.Unlock()
		return nil
	}
	sw.stopped = true
	sw.mu.Unlock()

	// Signal stop
	close(sw.stopChan)

	// Close fsnotify watcher if it exists
	if sw.fsnotifyWatcher != nil {
		if err := sw.fsnotifyWatcher.Close(); err != nil {
			logger.Log.Warn().
				Err(err).
				Msg("Error closing fsnotify watcher")
		}
	}

	// Wait for goroutines to finish
	<-sw.watchDone
	<-sw.pruneDone

	logger.Log.Debug().
		Str("segment_dir", sw.segmentDir).
		Msg("Segment watcher stopped")

	return nil
}

// runWatching runs the file watching loop (fsnotify or polling)
func (sw *segmentWatcher) runWatching() {
	defer close(sw.watchDone)

	if sw.fsnotifyWatcher != nil {
		sw.startWatching()
	} else {
		sw.startPolling()
	}
}

// startWatching uses fsnotify to watch for file events
func (sw *segmentWatcher) startWatching() {
	ticker := time.NewTicker(debounceWindow)
	defer ticker.Stop()

	for {
		select {
		case <-sw.stopChan:
			return
		case event, ok := <-sw.fsnotifyWatcher.Events:
			if !ok {
				return
			}
			// Handle CREATE and WRITE events
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				sw.handleFileEvent(event.Name)
			}
		case err, ok := <-sw.fsnotifyWatcher.Errors:
			if !ok {
				return
			}
			logger.Log.Warn().
				Err(err).
				Msg("fsnotify error, continuing")
		case <-ticker.C:
			// Process pending notifications
			sw.processPendingNotifications()
		}
	}
}

// startPolling polls the directory for new files
func (sw *segmentWatcher) startPolling() {
	ticker := time.NewTicker(sw.pollInterval)
	defer ticker.Stop()

	// Track seen files to detect new ones
	seenFiles := make(map[string]bool)

	for {
		select {
		case <-sw.stopChan:
			return
		case <-ticker.C:
			// Scan directory for new .ts files
			entries, err := os.ReadDir(sw.segmentDir)
			if err != nil {
				logger.Log.Warn().
					Err(err).
					Str("segment_dir", sw.segmentDir).
					Msg("Failed to read segment directory during polling")
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				filename := entry.Name()
				if !strings.HasSuffix(filename, ".ts") {
					continue
				}

				// Check if this is a new file
				if !seenFiles[filename] {
					seenFiles[filename] = true
					fullPath := filepath.Join(sw.segmentDir, filename)
					sw.handleFileEvent(fullPath)
				}
			}
		}
	}
}

// handleFileEvent processes a file event (new segment detected)
func (sw *segmentWatcher) handleFileEvent(filePath string) {
	// Extract filename
	filename := filepath.Base(filePath)

	// Check if it's a .ts file
	if !strings.HasSuffix(filename, ".ts") {
		return
	}

	// Add to pending notifications with debouncing
	sw.mu.Lock()
	if _, exists := sw.pendingNotifications[filename]; !exists {
		sw.pendingNotifications[filename] = time.Now()
	}
	sw.mu.Unlock()
}

// processPendingNotifications processes all pending notifications
func (sw *segmentWatcher) processPendingNotifications() {
	sw.mu.Lock()
	pending := make(map[string]time.Time)
	for k, v := range sw.pendingNotifications {
		pending[k] = v
	}
	sw.pendingNotifications = make(map[string]time.Time)
	sw.mu.Unlock()

	// Process each pending notification
	for filename, firstSeen := range pending {
		// Check if file exists and is readable (handles atomic writes)
		fullPath := filepath.Join(sw.segmentDir, filename)
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			// File doesn't exist yet or was deleted, skip
			continue
		}

		// Ensure file is not too new (give it time to finish writing)
		// If file was just created, wait a bit more
		if time.Since(firstSeen) < 100*time.Millisecond {
			// Re-add to pending with updated time
			sw.mu.Lock()
			sw.pendingNotifications[filename] = firstSeen
			sw.mu.Unlock()
			continue
		}

		// Notify playlist manager
		sw.notifyNewSegment(filename, fileInfo.ModTime())
	}
}

// notifyNewSegment notifies the playlist manager about a new segment
func (sw *segmentWatcher) notifyNewSegment(filename string, modTime time.Time) {
	detectionStartTime := time.Now()

	// Create segment metadata
	seg := SegmentMeta{
		URI:      filename,
		Duration: sw.segmentDuration,
	}

	// Optionally set program date-time from modification time
	programDateTime := modTime.UTC()
	seg.ProgramDateTime = &programDateTime

	// Calculate cadence (time since last segment detection)
	var cadenceMs int64
	sw.mu.Lock()
	if sw.lastSegmentDetected != nil {
		cadenceMs = detectionStartTime.Sub(*sw.lastSegmentDetected).Milliseconds()
	}
	// Update last segment detection time
	sw.lastSegmentDetected = &detectionStartTime

	// Check for timestamp regression (PTS regression detection)
	if sw.lastSegmentTime != nil {
		// Compare new segment time with last segment time
		if programDateTime.Before(*sw.lastSegmentTime) {
			// Timestamp regression detected - signal discontinuity
			logger.Log.Info().
				Str("segment_uri", filename).
				Time("previous_time", *sw.lastSegmentTime).
				Time("current_time", programDateTime).
				Msg("Timestamp regression detected, marking discontinuity")
			sw.playlistManager.SetDiscontinuityNext()
		}
	}
	// Update last segment time
	sw.lastSegmentTime = &programDateTime
	sw.mu.Unlock()

	// Add segment to playlist
	if err := sw.playlistManager.AddSegment(seg); err != nil {
		logger.Log.Error().
			Err(err).
			Str("segment_uri", filename).
			Float64("segment_duration", sw.segmentDuration).
			Int64("cadence_ms", cadenceMs).
			Msg("Failed to add segment to playlist")
		return
	}

	// Write playlist and measure latency
	writeStartTime := time.Now()
	if err := sw.playlistManager.Write(); err != nil {
		writeLatencyMs := time.Since(writeStartTime).Milliseconds()
		logger.Log.Error().
			Err(err).
			Str("segment_uri", filename).
			Float64("segment_duration", sw.segmentDuration).
			Int64("cadence_ms", cadenceMs).
			Int64("write_latency_ms", writeLatencyMs).
			Msg("Failed to write playlist after adding segment")
		return
	}
	writeLatencyMs := time.Since(writeStartTime).Milliseconds()
	totalLatencyMs := time.Since(detectionStartTime).Milliseconds()

	// Log segment detection with observability metrics
	logger.Log.Info().
		Str("segment_uri", filename).
		Float64("segment_duration", sw.segmentDuration).
		Int64("cadence_ms", cadenceMs).
		Int64("write_latency_ms", writeLatencyMs).
		Int64("total_latency_ms", totalLatencyMs).
		Msg("Segment detected and playlist updated")

	// Warn if cadence is unusually slow (more than 2x expected segment duration)
	expectedCadenceMs := int64(sw.segmentDuration * 1000)
	if cadenceMs > 0 && cadenceMs > expectedCadenceMs*2 {
		logger.Log.Warn().
			Str("segment_uri", filename).
			Int64("cadence_ms", cadenceMs).
			Int64("expected_cadence_ms", expectedCadenceMs).
			Msg("Segment cadence slower than expected")
	}
}

// runPruning runs the periodic pruning goroutine
func (sw *segmentWatcher) runPruning() {
	defer close(sw.pruneDone)

	ticker := time.NewTicker(sw.pruneInterval)
	defer ticker.Stop()

	// Run initial prune after first interval
	select {
	case <-sw.stopChan:
		return
	case <-ticker.C:
		sw.pruneOldSegments()
	}

	// Continue pruning periodically
	for {
		select {
		case <-sw.stopChan:
			return
		case <-ticker.C:
			sw.pruneOldSegments()
		}
	}
}

// pruneOldSegments removes segments older than (windowSize + safetyBuffer)
func (sw *segmentWatcher) pruneOldSegments() {
	// Get current segments from playlist manager
	currentSegments := sw.playlistManager.GetCurrentSegments()
	currentSegmentsMap := make(map[string]bool)
	for _, seg := range currentSegments {
		currentSegmentsMap[seg] = true
	}

	// Calculate prune threshold
	pruneThreshold := int(sw.windowSize + sw.safetyBuffer)

	// Read directory
	entries, err := os.ReadDir(sw.segmentDir)
	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("segment_dir", sw.segmentDir).
			Msg("Failed to read segment directory for pruning")
		return
	}

	// Collect .ts files with modification times
	type segmentFile struct {
		name     string
		fullPath string
		modTime  time.Time
	}

	var segmentFiles []segmentFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".ts") {
			continue
		}

		// Skip if file is in current playlist
		if currentSegmentsMap[filename] {
			continue
		}

		// Get file info
		fullPath := filepath.Join(sw.segmentDir, filename)
		info, err := entry.Info()
		if err != nil {
			logger.Log.Warn().
				Err(err).
				Str("filename", filename).
				Msg("Failed to get file info for pruning")
			continue
		}

		segmentFiles = append(segmentFiles, segmentFile{
			name:     filename,
			fullPath: fullPath,
			modTime:  info.ModTime(),
		})
	}

	// Sort by modification time (oldest first)
	// Simple insertion sort for small lists
	for i := 1; i < len(segmentFiles); i++ {
		j := i
		for j > 0 && segmentFiles[j].modTime.Before(segmentFiles[j-1].modTime) {
			segmentFiles[j], segmentFiles[j-1] = segmentFiles[j-1], segmentFiles[j]
			j--
		}
	}

	// Delete files beyond threshold (keep only the most recent pruneThreshold files)
	deletedCount := 0
	if len(segmentFiles) > pruneThreshold {
		filesToDelete := segmentFiles[:len(segmentFiles)-pruneThreshold]
		for _, file := range filesToDelete {
			// Double-check file is not in playlist (safety check)
			if currentSegmentsMap[file.name] {
				logger.Log.Warn().
					Str("filename", file.name).
					Msg("Skipping deletion of segment still in playlist")
				continue
			}

			if err := os.Remove(file.fullPath); err != nil {
				if !os.IsNotExist(err) {
					logger.Log.Warn().
						Err(err).
						Str("filename", file.name).
						Msg("Failed to delete old segment")
				}
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		logger.Log.Info().
			Int("deleted", deletedCount).
			Int("threshold", pruneThreshold).
			Uint("window_size", sw.windowSize).
			Uint("safety_buffer", sw.safetyBuffer).
			Str("segment_dir", sw.segmentDir).
			Msg("Pruned old segments")
	}
}

// MarkDiscontinuity signals that the encoder has restarted and the next segment should have a discontinuity tag
func (sw *segmentWatcher) MarkDiscontinuity() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Signal discontinuity to playlist manager
	sw.playlistManager.SetDiscontinuityNext()

	// Reset last segment time to allow fresh start after restart
	sw.lastSegmentTime = nil
	// Reset cadence tracking since encoder restarted
	sw.lastSegmentDetected = nil

	logger.Log.Info().
		Str("segment_dir", sw.segmentDir).
		Msg("Discontinuity marked for next segment (encoder restart)")
}
