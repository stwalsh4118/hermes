package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
)

// Supported video file extensions
var supportedVideoFormats = []string{".mp4", ".mkv", ".avi", ".mov"}

// Scan retention and cleanup settings
const (
	scanRetentionPeriod = 1 * time.Hour    // Keep completed scans for 1 hour
	cleanupInterval     = 15 * time.Minute // Run cleanup every 15 minutes
)

// ScanStatus represents the current state of a media scan
type ScanStatus string

// Media scan status constants
const (
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusCancelled ScanStatus = "cancelled"
	ScanStatusFailed    ScanStatus = "failed"
)

// Common scanner errors
var (
	ErrScanNotFound       = errors.New("scan not found")
	ErrScanAlreadyRunning = errors.New("a scan is already running")
	ErrInvalidDirectory   = errors.New("invalid directory path")
)

// ScanProgress tracks the progress of a media scan operation
type ScanProgress struct {
	ScanID         string     `json:"scan_id"`
	Status         ScanStatus `json:"status"`
	TotalFiles     int        `json:"total_files"`
	ProcessedFiles int        `json:"processed_files"`
	SuccessCount   int        `json:"success_count"`
	FailedCount    int        `json:"failed_count"`
	CurrentFile    string     `json:"current_file"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	Errors         []string   `json:"errors,omitempty"`
	mu             sync.RWMutex
	cancelFunc     context.CancelFunc
}

// Scanner manages asynchronous media scanning operations
type Scanner struct {
	repos       *db.Repositories
	activeScans map[string]*ScanProgress
	mu          sync.RWMutex
	stopCleanup chan struct{} // Signal to stop cleanup goroutine
	cleanupDone chan struct{} // Signal when cleanup goroutine has stopped
}

// NewScanner creates a new media scanner instance
func NewScanner(repos *db.Repositories) *Scanner {
	s := &Scanner{
		repos:       repos,
		activeScans: make(map[string]*ScanProgress),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go s.runCleanupLoop()

	return s
}

// StartScan initiates an asynchronous media scan of the specified directory
// Returns the scan ID that can be used to track progress
func (s *Scanner) StartScan(ctx context.Context, dirPath string) (string, error) {
	// Validate directory exists and is readable
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: directory does not exist", ErrInvalidDirectory)
		}
		return "", fmt.Errorf("%w: %w", ErrInvalidDirectory, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: path is not a directory", ErrInvalidDirectory)
	}

	// Check for concurrent scans and insert atomically
	// Use Lock (not RLock) to ensure check and insert are atomic
	s.mu.Lock()
	for _, scan := range s.activeScans {
		scan.mu.RLock()
		if scan.Status == ScanStatusRunning {
			scan.mu.RUnlock()
			s.mu.Unlock()
			return "", ErrScanAlreadyRunning
		}
		scan.mu.RUnlock()
	}

	// Generate scan ID while holding lock
	scanID := uuid.New().String()

	// Create cancellable context for this scan
	scanCtx, cancel := context.WithCancel(ctx)

	// Initialize progress
	progress := &ScanProgress{
		ScanID:     scanID,
		Status:     ScanStatusRunning,
		StartTime:  time.Now().UTC(),
		Errors:     []string{},
		cancelFunc: cancel,
	}

	// Store in active scans (still holding lock from check above)
	s.activeScans[scanID] = progress
	s.mu.Unlock()

	// Start async scan in goroutine
	go s.performScan(scanCtx, scanID, dirPath)

	logger.Log.Info().
		Str("scan_id", scanID).
		Str("directory", dirPath).
		Msg("Media scan started")

	return scanID, nil
}

// GetScanProgress retrieves the current progress of a scan
func (s *Scanner) GetScanProgress(scanID string) (*ScanProgress, error) {
	s.mu.RLock()
	progress, exists := s.activeScans[scanID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrScanNotFound
	}

	// Return a copy of the progress to avoid race conditions
	progress.mu.RLock()
	defer progress.mu.RUnlock()

	progressCopy := &ScanProgress{
		ScanID:         progress.ScanID,
		Status:         progress.Status,
		TotalFiles:     progress.TotalFiles,
		ProcessedFiles: progress.ProcessedFiles,
		SuccessCount:   progress.SuccessCount,
		FailedCount:    progress.FailedCount,
		CurrentFile:    progress.CurrentFile,
		StartTime:      progress.StartTime,
		EndTime:        progress.EndTime,
		Errors:         append([]string{}, progress.Errors...), // Copy slice
	}

	return progressCopy, nil
}

// CancelScan cancels a running scan
func (s *Scanner) CancelScan(scanID string) error {
	s.mu.RLock()
	progress, exists := s.activeScans[scanID]
	s.mu.RUnlock()

	if !exists {
		return ErrScanNotFound
	}

	progress.mu.Lock()
	if progress.Status != ScanStatusRunning {
		progress.mu.Unlock()
		return fmt.Errorf("scan is not running (status: %s)", progress.Status)
	}

	// Call cancel function
	if progress.cancelFunc != nil {
		progress.cancelFunc()
	}
	progress.mu.Unlock()

	logger.Log.Info().
		Str("scan_id", scanID).
		Msg("Media scan cancellation requested")

	return nil
}

// performScan executes the actual scanning logic asynchronously
func (s *Scanner) performScan(ctx context.Context, scanID, dirPath string) {
	s.mu.RLock()
	progress := s.activeScans[scanID]
	s.mu.RUnlock()

	// Count total files first
	videoFiles := s.findVideoFiles(ctx, dirPath, progress)

	// Check if cancelled during counting
	if ctx.Err() != nil {
		s.finalizeScan(progress, ScanStatusCancelled)
		return
	}

	progress.mu.Lock()
	progress.TotalFiles = len(videoFiles)
	progress.mu.Unlock()

	logger.Log.Info().
		Str("scan_id", scanID).
		Int("total_files", len(videoFiles)).
		Msg("Found video files to process")

	// Process each video file
	for _, filePath := range videoFiles {
		// Check for cancellation
		select {
		case <-ctx.Done():
			s.finalizeScan(progress, ScanStatusCancelled)
			return
		default:
		}

		s.processVideoFile(ctx, filePath, progress)
	}

	// Finalize scan
	s.finalizeScan(progress, ScanStatusCompleted)
}

// findVideoFiles walks the directory tree and returns all video file paths
func (s *Scanner) findVideoFiles(ctx context.Context, dirPath string, progress *ScanProgress) []string {
	var videoFiles []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Handle walk errors
		if err != nil {
			errMsg := fmt.Sprintf("error accessing path %s: %v", path, err)
			logger.Log.Warn().
				Str("path", path).
				Err(err).
				Msg("Error during directory walk")
			progress.mu.Lock()
			progress.Errors = append(progress.Errors, errMsg)
			progress.mu.Unlock()
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file has supported video extension
		if isVideoFile(path) {
			videoFiles = append(videoFiles, path)
		}

		return nil
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		errMsg := fmt.Sprintf("directory walk failed: %v", err)
		logger.Log.Error().Err(err).Msg("Directory walk failed")
		progress.mu.Lock()
		progress.Errors = append(progress.Errors, errMsg)
		progress.mu.Unlock()
	}

	return videoFiles
}

// processVideoFile processes a single video file through the pipeline
func (s *Scanner) processVideoFile(ctx context.Context, filePath string, progress *ScanProgress) {
	// Update current file
	progress.mu.Lock()
	progress.CurrentFile = filePath
	progress.mu.Unlock()

	logger.Log.Debug().
		Str("file", filePath).
		Msg("Processing video file")

	// Validate file is readable
	validationResult := ValidateFile(filePath)
	if !validationResult.Readable {
		s.recordFileError(progress, filePath, fmt.Errorf("file not readable: %s", strings.Join(validationResult.Reasons, ", ")))
		return
	}

	// Extract metadata using FFprobe
	metadata, err := ProbeFile(ctx, filePath)
	if err != nil {
		s.recordFileError(progress, filePath, fmt.Errorf("ffprobe failed: %w", err))
		return
	}

	// Parse filename for show/season/episode
	parseResult := ParseFilename(filePath)

	// Validate codec compatibility
	codecValidation := ValidateMedia(metadata)

	// Create or update media model
	media := models.NewMedia(filePath, parseResult.Title, metadata.Duration)
	media.ShowName = parseResult.ShowName
	media.Season = parseResult.Season
	media.Episode = parseResult.Episode
	media.VideoCodec = &metadata.VideoCodec
	media.AudioCodec = &metadata.AudioCodec
	media.Resolution = &metadata.Resolution
	media.FileSize = &metadata.FileSize

	// Log transcoding requirement if needed
	if codecValidation.RequiresTranscode {
		logger.Log.Debug().
			Str("file", filePath).
			Strs("reasons", codecValidation.Reasons).
			Msg("File requires transcoding")
	}

	// Save to database (create or update if exists)
	err = s.upsertMedia(ctx, media)
	if err != nil {
		s.recordFileError(progress, filePath, fmt.Errorf("database operation failed: %w", err))
		return
	}

	// Record success
	progress.mu.Lock()
	progress.SuccessCount++
	progress.ProcessedFiles++
	progress.mu.Unlock()

	logger.Log.Debug().
		Str("file", filePath).
		Str("title", media.Title).
		Msg("Successfully processed video file")
}

// upsertMedia creates or updates a media record in the database
// Uses optimistic insert to avoid TOCTOU race conditions
func (s *Scanner) upsertMedia(ctx context.Context, media *models.Media) error {
	// Attempt to create first (optimistic approach)
	err := s.repos.Media.Create(ctx, media)
	if err == nil {
		// Successfully created new record
		return nil
	}

	// Check if error is due to duplicate/unique constraint
	if !db.IsDuplicate(err) {
		// Some other error occurred
		return err
	}

	// Duplicate detected - fetch existing record and update
	existing, err := s.repos.Media.GetByPath(ctx, media.FilePath)
	if err != nil {
		return fmt.Errorf("failed to fetch existing media after duplicate: %w", err)
	}

	// Preserve existing ID and CreatedAt, then update
	media.ID = existing.ID
	media.CreatedAt = existing.CreatedAt
	return s.repos.Media.Update(ctx, media)
}

// recordFileError logs and records an error for a specific file
func (s *Scanner) recordFileError(progress *ScanProgress, filePath string, err error) {
	errMsg := fmt.Sprintf("%s: %v", filePath, err)
	logger.Log.Warn().
		Str("file", filePath).
		Err(err).
		Msg("Failed to process video file")

	progress.mu.Lock()
	progress.FailedCount++
	progress.ProcessedFiles++
	progress.Errors = append(progress.Errors, errMsg)
	progress.mu.Unlock()
}

// finalizeScan completes the scan and updates final status
func (s *Scanner) finalizeScan(progress *ScanProgress, status ScanStatus) {
	endTime := time.Now().UTC()

	progress.mu.Lock()
	progress.Status = status
	progress.EndTime = &endTime
	progress.CurrentFile = ""
	progress.mu.Unlock()

	logger.Log.Info().
		Str("scan_id", progress.ScanID).
		Str("status", string(status)).
		Int("total_files", progress.TotalFiles).
		Int("success_count", progress.SuccessCount).
		Int("failed_count", progress.FailedCount).
		Int("error_count", len(progress.Errors)).
		Dur("duration", endTime.Sub(progress.StartTime)).
		Msg("Media scan completed")
}

// isVideoFile checks if a file has a supported video extension
func isVideoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, supportedExt := range supportedVideoFormats {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

// Stop gracefully stops the scanner's background cleanup goroutine
// This should be called when the scanner is no longer needed
func (s *Scanner) Stop() {
	close(s.stopCleanup)
	<-s.cleanupDone
	logger.Log.Debug().Msg("Scanner cleanup goroutine stopped")
}

// runCleanupLoop runs periodic cleanup of old completed scans
func (s *Scanner) runCleanupLoop() {
	defer close(s.cleanupDone)

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	logger.Log.Debug().
		Dur("interval", cleanupInterval).
		Dur("retention", scanRetentionPeriod).
		Msg("Started scan cleanup goroutine")

	for {
		select {
		case <-s.stopCleanup:
			return
		case <-ticker.C:
			s.CleanupOldScans(scanRetentionPeriod)
		}
	}
}

// CleanupOldScans removes completed, cancelled, or failed scans older than the specified duration
func (s *Scanner) CleanupOldScans(olderThan time.Duration) {
	cutoff := time.Now().Add(-olderThan)
	removed := 0

	s.mu.Lock()
	defer s.mu.Unlock()

	for scanID, progress := range s.activeScans {
		progress.mu.RLock()
		status := progress.Status
		endTime := progress.EndTime
		progress.mu.RUnlock()

		// Only remove scans that are not running and have ended
		if status == ScanStatusRunning {
			continue
		}

		if endTime == nil {
			continue
		}

		if endTime.Before(cutoff) {
			delete(s.activeScans, scanID)
			removed++
		}
	}

	if removed > 0 {
		logger.Log.Debug().
			Int("removed_count", removed).
			Int("remaining_count", len(s.activeScans)).
			Dur("older_than", olderThan).
			Msg("Cleaned up old scans")
	}
}
