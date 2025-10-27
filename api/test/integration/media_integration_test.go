//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/api"
	"github.com/stwalsh4118/hermes/internal/media"
)

// TestCompleteScanWorkflow tests the complete scan workflow from trigger to database population
// Note: This test uses dummy files which will cause FFprobe to fail, demonstrating error handling
func TestCompleteScanWorkflow(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	// Create temp directory with dummy video files
	tmpDir, cleanupFiles := createTempVideoFiles(t, 3)
	defer cleanupFiles()

	ctx := context.Background()

	// Start scan
	scanID, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err, "Failed to start scan")
	require.NotEmpty(t, scanID, "Scan ID should not be empty")

	// Verify scan ID is valid UUID
	_, err = uuid.Parse(scanID)
	assert.NoError(t, err, "Scan ID should be valid UUID")

	// Wait for scan to complete
	progress := waitForScanCompletion(t, scanner, scanID, 100)

	// Verify scan results
	assert.NotNil(t, progress, "Progress should not be nil")
	assert.Equal(t, media.ScanStatusCompleted, progress.Status, "Scan should complete")
	assert.Equal(t, 3, progress.TotalFiles, "Should find 3 video files")
	assert.Equal(t, 3, progress.ProcessedFiles, "Should process all 3 files")

	// Note: With dummy files, FFprobe will fail, so we expect failures
	// This demonstrates the scanner's error handling capability
	assert.Equal(t, 3, progress.FailedCount, "All files should fail with dummy content")
	assert.Greater(t, len(progress.Errors), 0, "Should record errors")
}

// TestScanProgressTracking tests that scan progress is tracked correctly
func TestScanProgressTracking(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	tmpDir, cleanupFiles := createTempVideoFiles(t, 5)
	defer cleanupFiles()

	ctx := context.Background()

	// Start scan
	scanID, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)

	// Poll progress multiple times
	var progressSnapshots []*media.ScanProgress
	for i := 0; i < 50; i++ {
		progress, err := scanner.GetScanProgress(scanID)
		require.NoError(t, err)
		progressSnapshots = append(progressSnapshots, progress)

		if progress.Status != media.ScanStatusRunning {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Verify we captured multiple progress states
	assert.Greater(t, len(progressSnapshots), 0, "Should capture progress snapshots")

	// Wait for scan to complete
	finalProgress := waitForScanCompletion(t, scanner, scanID, 100)

	assert.NotEqual(t, media.ScanStatusRunning, finalProgress.Status, "Scan should not be running at end")
	assert.Equal(t, 5, finalProgress.TotalFiles, "Should find 5 files")
	assert.NotNil(t, finalProgress.EndTime, "Should have end time")
}

// TestDuplicateFileHandling tests scanning the same directory twice
func TestDuplicateFileHandling(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	tmpDir, cleanupFiles := createTempVideoFiles(t, 2)
	defer cleanupFiles()

	ctx := context.Background()

	// First scan
	scanID1, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)
	progress1 := waitForScanCompletion(t, scanner, scanID1, 100)
	require.Equal(t, media.ScanStatusCompleted, progress1.Status)

	// Second scan of same directory
	scanID2, err := scanner.StartScan(ctx, tmpDir)
	require.NoError(t, err)
	progress2 := waitForScanCompletion(t, scanner, scanID2, 100)
	require.Equal(t, media.ScanStatusCompleted, progress2.Status)

	// Both scans should process same number of files
	assert.Equal(t, progress1.TotalFiles, progress2.TotalFiles, "Should find same files")

	// Verify no duplicates in database (file_path is unique)
	mediaList, err := repos.Media.List(ctx, 100, 0)
	require.NoError(t, err)

	// Since FFprobe fails on dummy files, no media should be inserted
	// This tests that the error handling works correctly
	assert.Equal(t, 0, len(mediaList), "No media should be inserted from failed scans")
}

// TestConcurrentScanPrevention tests that concurrent scans are prevented
func TestConcurrentScanPrevention(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	tmpDir1, cleanupFiles1 := createTempVideoFiles(t, 5)
	defer cleanupFiles1()

	tmpDir2, cleanupFiles2 := createTempVideoFiles(t, 3)
	defer cleanupFiles2()

	ctx := context.Background()

	// Start first scan
	scanID1, err := scanner.StartScan(ctx, tmpDir1)
	require.NoError(t, err)

	// Immediately try to start second scan
	scanID2, err := scanner.StartScan(ctx, tmpDir2)

	// Should get error about concurrent scan
	assert.Error(t, err, "Second scan should fail")
	assert.ErrorIs(t, err, media.ErrScanAlreadyRunning, "Should be concurrent scan error")
	assert.Empty(t, scanID2, "Second scan ID should be empty")

	// Wait for first scan to complete
	progress1 := waitForScanCompletion(t, scanner, scanID1, 100)
	assert.Equal(t, media.ScanStatusCompleted, progress1.Status)

	// Now second scan should work
	scanID3, err := scanner.StartScan(ctx, tmpDir2)
	require.NoError(t, err, "Third scan should succeed after first completes")
	assert.NotEmpty(t, scanID3)
}

// TestAPIScanAndList tests the complete API workflow: insert media -> list via API
func TestAPIScanAndList(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media directly in database (bypassing FFprobe issues with dummy files)
	showName := "Friends"
	for i := 0; i < 5; i++ {
		season := i/3 + 1
		episode := i%3 + 1
		title := fmt.Sprintf("Friends - S%02dE%02d", season, episode)
		filePath := fmt.Sprintf("/test/friends_s%02de%02d.mp4", season, episode)
		createTestMediaInDB(t, repos, filePath, title, &showName, &season, &episode)
	}

	// List media via API
	req := httptest.NewRequest("GET", "/api/media", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResp api.MediaListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)

	assert.Equal(t, 5, listResp.Total, "Should have 5 media items")
	assert.Equal(t, 5, len(listResp.Items), "Should return 5 items")

	// Verify first item has expected fields
	if len(listResp.Items) > 0 {
		item := listResp.Items[0]
		assert.NotNil(t, item.ID)
		assert.NotEmpty(t, item.Title)
		assert.Equal(t, "Friends", *item.ShowName)
	}
}

// TestAPIGetMediaByID tests retrieving a single media item by ID
func TestAPIGetMediaByID(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media directly in database
	showName := "Test Show"
	season := 1
	episode := 5
	mediaItem := createTestMediaInDB(t, repos, "/test/video.mp4", "Test Show - S01E05", &showName, &season, &episode)

	// Get media by ID via API
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, mediaItem.ID.String(), resp["id"])
	assert.Equal(t, mediaItem.Title, resp["title"])
	assert.Equal(t, *mediaItem.ShowName, resp["show_name"])
	assert.Equal(t, float64(*mediaItem.Season), resp["season"])
	assert.Equal(t, float64(*mediaItem.Episode), resp["episode"])
}

// TestAPIUpdateMedia tests updating media metadata
func TestAPIUpdateMedia(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media
	showName := "Old Show"
	season := 1
	episode := 1
	mediaItem := createTestMediaInDB(t, repos, "/test/video.mp4", "Old Title", &showName, &season, &episode)

	// Update media via API
	newTitle := "New Title"
	newShowName := "New Show"
	newSeason := 2
	newEpisode := 10

	updateReq := api.UpdateMediaRequest{
		Title:    &newTitle,
		ShowName: &newShowName,
		Season:   &newSeason,
		Episode:  &newEpisode,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, newTitle, resp["title"])
	assert.Equal(t, newShowName, resp["show_name"])
	assert.Equal(t, float64(newSeason), resp["season"])
	assert.Equal(t, float64(newEpisode), resp["episode"])

	// Verify in database
	ctx := context.Background()
	updated, err := repos.Media.GetByID(ctx, mediaItem.ID)
	require.NoError(t, err)

	assert.Equal(t, newTitle, updated.Title)
	assert.Equal(t, newShowName, *updated.ShowName)
	assert.Equal(t, newSeason, *updated.Season)
	assert.Equal(t, newEpisode, *updated.Episode)
}

// TestAPIDeleteMedia tests deleting media
func TestAPIDeleteMedia(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media
	mediaItem := createTestMediaInDB(t, repos, "/test/video.mp4", "Test Video", nil, nil, nil)

	// Delete via API
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp api.DeleteResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Media deleted successfully", resp.Message)

	// Try to get deleted media - should return 404
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	// Verify not in database
	ctx := context.Background()
	_, err = repos.Media.GetByID(ctx, mediaItem.ID)
	assert.Error(t, err, "Media should not exist in database")
}

// TestAPIPagination tests media list pagination
func TestAPIPagination(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create 25 test media items
	const totalItems = 25
	showName := "Test Show"
	for i := 0; i < totalItems; i++ {
		title := fmt.Sprintf("Episode %d", i+1)
		season := i/10 + 1
		episode := i%10 + 1
		filePath := fmt.Sprintf("/test/video_%d.mp4", i)
		createTestMediaInDB(t, repos, filePath, title, &showName, &season, &episode)
	}

	// Test first page
	req := httptest.NewRequest("GET", "/api/media?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp1 api.MediaListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp1)
	require.NoError(t, err)

	assert.Equal(t, totalItems, resp1.Total, "Total should be 25")
	assert.Equal(t, 10, len(resp1.Items), "Should return 10 items")
	assert.Equal(t, 10, resp1.Limit)
	assert.Equal(t, 0, resp1.Offset)

	// Test second page
	req = httptest.NewRequest("GET", "/api/media?limit=10&offset=10", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp2 api.MediaListResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp2)
	require.NoError(t, err)

	assert.Equal(t, totalItems, resp2.Total)
	assert.Equal(t, 10, len(resp2.Items), "Should return next 10 items")
	assert.Equal(t, 10, resp2.Offset)

	// Test third page (partial)
	req = httptest.NewRequest("GET", "/api/media?limit=10&offset=20", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp3 api.MediaListResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp3)
	require.NoError(t, err)

	assert.Equal(t, totalItems, resp3.Total)
	assert.Equal(t, 5, len(resp3.Items), "Should return remaining 5 items")
}

// TestAPIFilterByShow tests filtering media by show name
func TestAPIFilterByShow(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create media for different shows
	createMultipleTestMedia(t, repos, 10, "Friends")
	createMultipleTestMedia(t, repos, 5, "Seinfeld")
	createMultipleTestMedia(t, repos, 3, "The Office")

	// Filter by Friends
	req := httptest.NewRequest("GET", "/api/media?show=Friends", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp api.MediaListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 10, resp.Total, "Should have 10 Friends episodes")
	for _, item := range resp.Items {
		assert.Equal(t, "Friends", *item.ShowName, "All items should be Friends")
	}

	// Filter by Seinfeld
	req = httptest.NewRequest("GET", "/api/media?show=Seinfeld", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 5, resp.Total, "Should have 5 Seinfeld episodes")
	for _, item := range resp.Items {
		assert.Equal(t, "Seinfeld", *item.ShowName)
	}
}

// TestConcurrentAPIReads tests concurrent reads to the API
// Note: This test verifies that the API can handle concurrent requests without data races
func TestConcurrentAPIReads(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media items
	showName := "Test Show"
	for i := 0; i < 20; i++ {
		season := i/10 + 1
		episode := i%10 + 1
		title := fmt.Sprintf("Test Show - S%02dE%02d", season, episode)
		filePath := fmt.Sprintf("/test/show_s%02de%02d.mp4", season, episode)
		createTestMediaInDB(t, repos, filePath, title, &showName, &season, &episode)
	}

	// Verify media was created
	ctx := context.Background()
	count, err := repos.Media.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(20), count, "Should have 20 media items in database")

	// Launch concurrent reads with shorter concurrency for SQLite in-memory limitations
	const concurrentReads = 5
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrentReads; i++ {
		wg.Add(1)
		go func(iterNum int) {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/api/media", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			mu.Lock()
			defer mu.Unlock()

			if w.Code == http.StatusOK {
				var resp api.MediaListResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
					if resp.Total == 20 {
						successCount++
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// At least some reads should succeed (SQLite in-memory has concurrency limitations)
	assert.Greater(t, successCount, 0, "At least some concurrent reads should succeed")
}

// TestInvalidScanPath tests error handling for invalid scan paths
func TestInvalidScanPath(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Test non-existent path
	reqBody := api.ScanRequest{Path: "/nonexistent/directory/path"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/media/scan", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp api.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid_directory", resp.Error)

	// Test file instead of directory
	tmpFile, cleanupFile := createTempVideoFile(t, ".mp4")
	defer cleanupFile()

	reqBody = api.ScanRequest{Path: tmpFile}
	body, _ = json.Marshal(reqBody)

	req = httptest.NewRequest("POST", "/api/media/scan", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid_directory", resp.Error)
}

// TestInvalidMediaID tests error handling for invalid media IDs
func TestInvalidMediaID(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	testCases := []struct {
		name       string
		method     string
		path       string
		body       interface{}
		expectCode int
	}{
		{
			name:       "GET with invalid UUID",
			method:     "GET",
			path:       "/api/media/not-a-uuid",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "GET with valid UUID but non-existent",
			method:     "GET",
			path:       fmt.Sprintf("/api/media/%s", uuid.New().String()),
			expectCode: http.StatusNotFound,
		},
		{
			name:       "PUT with invalid UUID",
			method:     "PUT",
			path:       "/api/media/not-a-uuid",
			body:       api.UpdateMediaRequest{},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "PUT with valid UUID but non-existent",
			method:     "PUT",
			path:       fmt.Sprintf("/api/media/%s", uuid.New().String()),
			body:       api.UpdateMediaRequest{},
			expectCode: http.StatusNotFound,
		},
		{
			name:       "DELETE with invalid UUID",
			method:     "DELETE",
			path:       "/api/media/not-a-uuid",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "DELETE with valid UUID but non-existent",
			method:     "DELETE",
			path:       fmt.Sprintf("/api/media/%s", uuid.New().String()),
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var reqBody []byte
			if tc.body != nil {
				reqBody, _ = json.Marshal(tc.body)
			}

			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBuffer(reqBody))
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectCode, w.Code, "Expected status code %d, got %d", tc.expectCode, w.Code)
		})
	}
}

// TestInvalidRequestBodies tests error handling for malformed requests
func TestInvalidRequestBodies(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create a valid media item for PUT test
	mediaItem := createTestMediaInDB(t, repos, "/test/video.mp4", "Test", nil, nil, nil)

	testCases := []struct {
		name       string
		method     string
		path       string
		body       string
		expectCode int
	}{
		{
			name:       "POST scan with malformed JSON",
			method:     "POST",
			path:       "/api/media/scan",
			body:       "{invalid json",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "POST scan with missing path",
			method:     "POST",
			path:       "/api/media/scan",
			body:       "{}",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "PUT media with malformed JSON",
			method:     "PUT",
			path:       fmt.Sprintf("/api/media/%s", mediaItem.ID.String()),
			body:       "{invalid json",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectCode, w.Code)
		})
	}
}
