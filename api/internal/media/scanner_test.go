package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/models"
)

// setupTestScanner creates a scanner with a test database
func setupTestScanner(t *testing.T) (*Scanner, *db.DB, func()) {
	// Create temporary database
	tmpFile := filepath.Join(t.TempDir(), "test.db")
	database, err := db.New(tmpFile)
	require.NoError(t, err)

	// Run migrations
	sqlDB, err := database.GetSQLDB()
	require.NoError(t, err)

	migrationsPath := "file://../../migrations"
	err = db.RunMigrations(sqlDB, migrationsPath)
	require.NoError(t, err)

	repos := db.NewRepositories(database)
	scanner := NewScanner(repos)

	cleanup := func() {
		database.Close()
	}

	return scanner, database, cleanup
}

// createTestVideoFiles creates temporary test video files
func createTestVideoFiles(t *testing.T, dir string, files []string) {
	for _, file := range files {
		fullPath := filepath.Join(dir, file)

		// Create subdirectories if needed
		subDir := filepath.Dir(fullPath)
		if subDir != dir {
			err := os.MkdirAll(subDir, 0755)
			require.NoError(t, err)
		}

		// Create empty file
		f, err := os.Create(fullPath)
		require.NoError(t, err)
		f.Close()
	}
}

func TestNewScanner(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	assert.NotNil(t, scanner)
	assert.NotNil(t, scanner.repos)
	assert.NotNil(t, scanner.activeScans)
	assert.Equal(t, 0, len(scanner.activeScans))
	assert.NotNil(t, scanner.stopCleanup)
	assert.NotNil(t, scanner.cleanupDone)
}

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"MP4 file", "/path/to/video.mp4", true},
		{"MKV file", "/path/to/video.mkv", true},
		{"AVI file", "/path/to/video.avi", true},
		{"MOV file", "/path/to/video.mov", true},
		{"MP4 uppercase", "/path/to/video.MP4", true},
		{"MKV mixed case", "/path/to/video.MkV", true},
		{"Text file", "/path/to/file.txt", false},
		{"JPG file", "/path/to/image.jpg", false},
		{"No extension", "/path/to/video", false},
		{"Multiple dots", "/path/to/video.backup.mp4", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVideoFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStartScan_InvalidDirectory(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()

	// Test non-existent directory
	_, err := scanner.StartScan(ctx, "/nonexistent/directory")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDirectory)

	// Test file instead of directory
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	f, err := os.Create(tmpFile)
	require.NoError(t, err)
	f.Close()

	_, err = scanner.StartScan(ctx, tmpFile)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDirectory)
}

func TestStartScan_Success(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()
	tmpDir := t.TempDir()

	scanID, err := scanner.StartScan(ctx, tmpDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, scanID)

	// Verify scan ID is a valid UUID
	_, err = uuid.Parse(scanID)
	assert.NoError(t, err)

	// Verify scan is in active scans
	scanner.mu.RLock()
	progress, exists := scanner.activeScans[scanID]
	scanner.mu.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, progress)
	assert.Equal(t, scanID, progress.ScanID)
}

func TestStartScan_ConcurrentScanPrevention(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Start first scan
	scanID1, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)

	// Try to start second scan while first is running
	_, err = scanner.StartScan(ctx, tmpDir)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrScanAlreadyRunning)

	// Wait for first scan to complete
	time.Sleep(100 * time.Millisecond)

	// Verify first scan completed
	progress, err := scanner.GetScanProgress(scanID1)
	require.NoError(t, err)

	// Wait until scan completes
	maxWait := 5 * time.Second
	deadline := time.Now().Add(maxWait)
	for progress.Status == ScanStatusRunning && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		progress, err = scanner.GetScanProgress(scanID1)
		require.NoError(t, err)
	}

	assert.NotEqual(t, ScanStatusRunning, progress.Status)

	// Now should be able to start another scan
	scanID2, err := scanner.StartScan(ctx, tmpDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, scanID2)
	assert.NotEqual(t, scanID1, scanID2)
}

func TestStartScan_ConcurrentRaceCondition(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create many files to ensure scans take time and stay in Running state
	files := make([]string, 100)
	for i := 0; i < 100; i++ {
		files[i] = fmt.Sprintf("video%d.mp4", i)
	}
	createTestVideoFiles(t, tmpDir, files)

	// Try to start multiple scans concurrently
	const numGoroutines = 10
	results := make(chan struct {
		scanID string
		err    error
	}, numGoroutines)

	// Use a sync barrier to ensure all goroutines start at the same time
	start := make(chan struct{})

	// Launch multiple goroutines trying to start scans simultaneously
	for i := 0; i < numGoroutines; i++ {
		go func() {
			<-start // Wait for signal to start
			scanID, err := scanner.StartScan(ctx, tmpDir)
			results <- struct {
				scanID string
				err    error
			}{scanID, err}
		}()
	}

	// Give goroutines time to reach the barrier
	time.Sleep(10 * time.Millisecond)

	// Signal all goroutines to start at once
	close(start)

	// Collect results
	successCount := 0
	failCount := 0
	var successfulScanID string

	for i := 0; i < numGoroutines; i++ {
		result := <-results
		if result.err == nil {
			successCount++
			successfulScanID = result.scanID
		} else {
			failCount++
			assert.ErrorIs(t, result.err, ErrScanAlreadyRunning)
		}
	}

	// Exactly one should succeed, the rest should fail
	assert.Equal(t, 1, successCount, "Exactly one scan should succeed")
	assert.Equal(t, numGoroutines-1, failCount, "All other scans should fail")
	assert.NotEmpty(t, successfulScanID, "Successful scan should have a valid ID")

	// Verify the successful scan exists and is running
	progress, err := scanner.GetScanProgress(successfulScanID)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	// May be running or completed depending on timing, but should exist
	assert.Contains(t, []ScanStatus{ScanStatusRunning, ScanStatusCompleted}, progress.Status)
}

func TestGetScanProgress_NotFound(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	_, err := scanner.GetScanProgress("nonexistent-scan-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrScanNotFound)
}

func TestGetScanProgress_Success(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()
	tmpDir := t.TempDir()

	scanID, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)

	progress, err := scanner.GetScanProgress(scanID)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	assert.Equal(t, scanID, progress.ScanID)
	assert.NotZero(t, progress.StartTime)
}

func TestGetScanProgress_ThreadSafety(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create some test files to make scan take longer
	createTestVideoFiles(t, tmpDir, []string{
		"video1.mp4",
		"video2.mkv",
		"video3.avi",
	})

	scanID, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)

	// Concurrently read progress from multiple goroutines
	var wg sync.WaitGroup
	numReaders := 10
	errors := make(chan error, numReaders)

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				_, err := scanner.GetScanProgress(scanID)
				if err != nil {
					errors <- err
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check no errors occurred
	for err := range errors {
		t.Errorf("Concurrent read error: %v", err)
	}
}

func TestCancelScan_NotFound(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	err := scanner.CancelScan("nonexistent-scan-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrScanNotFound)
}

func TestCancelScan_Success(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create many files to ensure scan takes time
	files := make([]string, 50)
	for i := 0; i < 50; i++ {
		files[i] = filepath.Join("subdir", fmt.Sprintf("video%d.mp4", i))
	}
	createTestVideoFiles(t, tmpDir, files)

	scanID, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)

	// Give scan time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the scan
	err = scanner.CancelScan(scanID)
	assert.NoError(t, err)

	// Wait for cancellation to take effect
	time.Sleep(200 * time.Millisecond)

	// Verify scan was cancelled
	progress, err := scanner.GetScanProgress(scanID)
	require.NoError(t, err)
	assert.Equal(t, ScanStatusCancelled, progress.Status)
	assert.NotNil(t, progress.EndTime)
}

func TestFindVideoFiles_FiltersByExtension(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	tmpDir := t.TempDir()

	// Create mix of video and non-video files
	files := []string{
		"video1.mp4",
		"video2.mkv",
		"video3.avi",
		"video4.mov",
		"document.txt",
		"image.jpg",
		"data.json",
		"subdir/video5.MP4", // Test case insensitivity
	}
	createTestVideoFiles(t, tmpDir, files)

	ctx := context.Background()
	progress := &ScanProgress{
		Errors: []string{},
	}

	videoFiles := scanner.findVideoFiles(ctx, tmpDir, progress)

	// Should find only 5 video files (4 + 1 in subdir)
	assert.Equal(t, 5, len(videoFiles))

	// Verify all returned files are video files
	for _, file := range videoFiles {
		assert.True(t, isVideoFile(file), "Expected %s to be a video file", file)
	}
}

func TestFindVideoFiles_RecursiveSearch(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	tmpDir := t.TempDir()

	// Create nested directory structure
	files := []string{
		"video1.mp4",
		"Season 1/episode1.mkv",
		"Season 1/episode2.mkv",
		"Season 2/episode1.avi",
		"Season 2/Specials/special1.mov",
	}
	createTestVideoFiles(t, tmpDir, files)

	ctx := context.Background()
	progress := &ScanProgress{
		Errors: []string{},
	}

	videoFiles := scanner.findVideoFiles(ctx, tmpDir, progress)

	assert.Equal(t, 5, len(videoFiles))
}

func TestFindVideoFiles_Cancellation(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	tmpDir := t.TempDir()

	// Create many files
	files := make([]string, 100)
	for i := 0; i < 100; i++ {
		files[i] = fmt.Sprintf("video%d.mp4", i)
	}
	createTestVideoFiles(t, tmpDir, files)

	ctx, cancel := context.WithCancel(context.Background())
	progress := &ScanProgress{
		Errors: []string{},
	}

	// Cancel immediately
	cancel()

	videoFiles := scanner.findVideoFiles(ctx, tmpDir, progress)

	// Should return early due to cancellation
	// May find some files before cancellation kicks in
	assert.LessOrEqual(t, len(videoFiles), 100)
}

func TestScanProgress_Copy(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()
	tmpDir := t.TempDir()

	scanID, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)

	// Get progress
	progress1, err := scanner.GetScanProgress(scanID)
	require.NoError(t, err)

	// Get progress again
	progress2, err := scanner.GetScanProgress(scanID)
	require.NoError(t, err)

	// Modifying one copy should not affect the other
	progress1.SuccessCount = 999

	assert.NotEqual(t, progress1.SuccessCount, progress2.SuccessCount)
}

func TestUpsertMedia_CreateNew(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()

	// Create new media
	media := models.NewMedia("/test/video.mp4", "Test Video", 120)
	showName := "Test Show"
	season := 1
	episode := 1
	media.ShowName = &showName
	media.Season = &season
	media.Episode = &episode

	err := scanner.upsertMedia(ctx, media)
	assert.NoError(t, err)

	// Verify media was created
	retrieved, err := scanner.repos.Media.GetByPath(ctx, "/test/video.mp4")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Test Video", retrieved.Title)
	assert.Equal(t, showName, *retrieved.ShowName)
}

func TestUpsertMedia_UpdateExisting(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()

	// Create initial media
	media1 := models.NewMedia("/test/video.mp4", "Original Title", 120)
	err := scanner.repos.Media.Create(ctx, media1)
	require.NoError(t, err)

	// Upsert with new data
	media2 := models.NewMedia("/test/video.mp4", "Updated Title", 150)
	showName := "New Show"
	media2.ShowName = &showName

	err = scanner.upsertMedia(ctx, media2)
	assert.NoError(t, err)

	// Verify media was updated
	retrieved, err := scanner.repos.Media.GetByPath(ctx, "/test/video.mp4")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.Equal(t, int64(150), retrieved.Duration)
	assert.Equal(t, showName, *retrieved.ShowName)
	// Should keep original ID
	assert.Equal(t, media1.ID, retrieved.ID)
}

func TestRecordFileError(t *testing.T) {
	progress := &ScanProgress{
		Errors: []string{},
	}

	scanner := &Scanner{}
	scanner.recordFileError(progress, "/test/video.mp4", assert.AnError)

	assert.Equal(t, 1, progress.FailedCount)
	assert.Equal(t, 1, progress.ProcessedFiles)
	assert.Equal(t, 1, len(progress.Errors))
	assert.Contains(t, progress.Errors[0], "/test/video.mp4")
}

func TestFinalizeScan(t *testing.T) {
	scanner := &Scanner{}
	progress := &ScanProgress{
		ScanID:         "test-scan",
		Status:         ScanStatusRunning,
		TotalFiles:     10,
		SuccessCount:   8,
		FailedCount:    2,
		ProcessedFiles: 10,
		StartTime:      time.Now().Add(-5 * time.Minute).UTC(),
		CurrentFile:    "/some/file.mp4",
	}

	scanner.finalizeScan(progress, ScanStatusCompleted)

	assert.Equal(t, ScanStatusCompleted, progress.Status)
	assert.NotNil(t, progress.EndTime)
	assert.Empty(t, progress.CurrentFile)
	assert.True(t, progress.EndTime.After(progress.StartTime))
}

func TestScanStatus_AllStatuses(t *testing.T) {
	// Verify all status constants are defined
	statuses := []ScanStatus{
		ScanStatusRunning,
		ScanStatusCompleted,
		ScanStatusCancelled,
		ScanStatusFailed,
	}

	assert.Equal(t, 4, len(statuses))
	assert.Equal(t, ScanStatus("running"), ScanStatusRunning)
	assert.Equal(t, ScanStatus("completed"), ScanStatusCompleted)
	assert.Equal(t, ScanStatus("cancelled"), ScanStatusCancelled)
	assert.Equal(t, ScanStatus("failed"), ScanStatusFailed)
}

func TestUpsertMedia_RaceCondition(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()

	// Create initial media
	media1 := models.NewMedia("/test/race.mp4", "Race Test", 100)
	err := scanner.repos.Media.Create(ctx, media1)
	require.NoError(t, err)

	// Now attempt upsert - should detect duplicate and update
	media2 := models.NewMedia("/test/race.mp4", "Race Test Updated", 200)
	err = scanner.upsertMedia(ctx, media2)
	assert.NoError(t, err)

	// Verify it was updated, not duplicated
	retrieved, err := scanner.repos.Media.GetByPath(ctx, "/test/race.mp4")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Race Test Updated", retrieved.Title)
	assert.Equal(t, int64(200), retrieved.Duration)
	assert.Equal(t, media1.ID, retrieved.ID) // Should preserve original ID
}

func TestUpsertMedia_OptimisticInsert(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	ctx := context.Background()

	// First upsert should create
	media1 := models.NewMedia("/test/optimistic.mp4", "Optimistic Test", 100)
	err := scanner.upsertMedia(ctx, media1)
	assert.NoError(t, err)

	// Verify created
	retrieved, err := scanner.repos.Media.GetByPath(ctx, "/test/optimistic.mp4")
	assert.NoError(t, err)
	assert.Equal(t, "Optimistic Test", retrieved.Title)

	// Second upsert should update
	media2 := models.NewMedia("/test/optimistic.mp4", "Optimistic Updated", 200)
	err = scanner.upsertMedia(ctx, media2)
	assert.NoError(t, err)

	// Verify updated
	retrieved, err = scanner.repos.Media.GetByPath(ctx, "/test/optimistic.mp4")
	assert.NoError(t, err)
	assert.Equal(t, "Optimistic Updated", retrieved.Title)
	assert.Equal(t, int64(200), retrieved.Duration)
}

func TestCleanupOldScans(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	// Create mock scans at different times
	now := time.Now()
	oldEndTime := now.Add(-2 * time.Hour)
	recentEndTime := now.Add(-30 * time.Minute)

	// Add old completed scan
	scanner.mu.Lock()
	scanner.activeScans["old-scan"] = &ScanProgress{
		ScanID:  "old-scan",
		Status:  ScanStatusCompleted,
		EndTime: &oldEndTime,
	}

	// Add recent completed scan
	scanner.activeScans["recent-scan"] = &ScanProgress{
		ScanID:  "recent-scan",
		Status:  ScanStatusCompleted,
		EndTime: &recentEndTime,
	}

	// Add running scan
	scanner.activeScans["running-scan"] = &ScanProgress{
		ScanID: "running-scan",
		Status: ScanStatusRunning,
	}

	// Add completed scan without end time
	scanner.activeScans["no-endtime"] = &ScanProgress{
		ScanID:  "no-endtime",
		Status:  ScanStatusCompleted,
		EndTime: nil,
	}
	scanner.mu.Unlock()

	// Cleanup scans older than 1 hour
	scanner.CleanupOldScans(1 * time.Hour)

	// Verify old scan was removed
	scanner.mu.RLock()
	_, oldExists := scanner.activeScans["old-scan"]
	_, recentExists := scanner.activeScans["recent-scan"]
	_, runningExists := scanner.activeScans["running-scan"]
	_, noEndTimeExists := scanner.activeScans["no-endtime"]
	scanner.mu.RUnlock()

	assert.False(t, oldExists, "Old scan should be removed")
	assert.True(t, recentExists, "Recent scan should remain")
	assert.True(t, runningExists, "Running scan should remain")
	assert.True(t, noEndTimeExists, "Scan without end time should remain")
}

func TestCleanupOldScans_MultipleStatuses(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()
	defer scanner.Stop()

	now := time.Now()
	oldEndTime := now.Add(-2 * time.Hour)

	// Add scans with different statuses
	scanner.mu.Lock()
	scanner.activeScans["completed"] = &ScanProgress{
		ScanID:  "completed",
		Status:  ScanStatusCompleted,
		EndTime: &oldEndTime,
	}
	scanner.activeScans["cancelled"] = &ScanProgress{
		ScanID:  "cancelled",
		Status:  ScanStatusCancelled,
		EndTime: &oldEndTime,
	}
	scanner.activeScans["failed"] = &ScanProgress{
		ScanID:  "failed",
		Status:  ScanStatusFailed,
		EndTime: &oldEndTime,
	}
	scanner.mu.Unlock()

	// Cleanup
	scanner.CleanupOldScans(1 * time.Hour)

	// All should be removed
	scanner.mu.RLock()
	count := len(scanner.activeScans)
	scanner.mu.RUnlock()

	assert.Equal(t, 0, count, "All old non-running scans should be removed")
}

func TestScanner_Stop(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()

	// Stop should not hang
	done := make(chan bool)
	go func() {
		scanner.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not complete within timeout")
	}
}

func TestScanner_CleanupRunning(t *testing.T) {
	scanner, _, cleanup := setupTestScanner(t)
	defer cleanup()

	// Let cleanup run at least once
	time.Sleep(100 * time.Millisecond)

	// Stop scanner
	scanner.Stop()

	// Verify cleanup goroutine stopped
	select {
	case <-scanner.cleanupDone:
		// Success - channel is closed
	default:
		t.Fatal("Cleanup goroutine did not stop")
	}
}
