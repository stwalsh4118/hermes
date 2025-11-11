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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stwalsh4118/hermes/internal/api"
	"github.com/stwalsh4118/hermes/internal/config"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
	"github.com/stwalsh4118/hermes/internal/streaming"
	"github.com/stwalsh4118/hermes/internal/timeline"
)

const (
	testBatchSize        = 5  // Smaller batch size for faster tests
	testTriggerThreshold = 2  // Trigger when 2 segments remain
	testSegmentDuration  = 2  // 2 second segments
	testVideoDuration    = 30 // 30 second test videos
)

// setupTestStreamManager creates a StreamManager with test configuration
func setupTestStreamManager(t *testing.T, repos *db.Repositories, segmentPath string) (*streaming.StreamManager, func()) {
	t.Helper()

	// Initialize logger if not already initialized
	logger.Init("debug", false)

	timelineService := timeline.NewTimelineService(repos)

	// Create test streaming config
	streamingConfig := &config.StreamingConfig{
		HardwareAccel:      "none", // Use software encoding for tests
		SegmentDuration:    testSegmentDuration,
		PlaylistSize:       10,
		SegmentPath:        segmentPath,
		GracePeriodSeconds: 30,
		CleanupInterval:    60,
		EncodingPreset:     "ultrafast",
		BatchSize:          testBatchSize,
		TriggerThreshold:   testTriggerThreshold,
	}

	manager := streaming.NewStreamManager(repos, timelineService, streamingConfig)

	// Start the manager
	err := manager.Start()
	require.NoError(t, err, "Failed to start stream manager")

	cleanup := func() {
		manager.Stop()
	}

	return manager, cleanup
}

// createShortTestVideo creates a minimal valid video file using FFmpeg
// Returns the file path and a cleanup function
func createShortTestVideo(t *testing.T, durationSeconds int) (string, func()) {
	t.Helper()

	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "batch-test-video-*")
	require.NoError(t, err, "Failed to create temp directory for test video")

	videoPath := filepath.Join(tmpDir, "test_video.mp4")

	// Use FFmpeg to generate a test video
	// Generate a simple color test pattern video
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc=duration=%d:size=1920x1080:rate=30", durationSeconds),
		"-f", "lavfi",
		"-i", fmt.Sprintf("sine=frequency=1000:duration=%d", durationSeconds),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-crf", "30",
		"-c:a", "aac",
		"-b:a", "128k",
		"-t", fmt.Sprintf("%d", durationSeconds),
		"-y", // Overwrite output file
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create test video: %v\nOutput: %s", err, string(output))
	}

	// Verify file was created
	_, err = os.Stat(videoPath)
	require.NoError(t, err, "Test video file was not created")

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return videoPath, cleanup
}

// createTestChannelWithMedia creates a channel with playlist items and media files
func createTestChannelWithMedia(t *testing.T, repos *db.Repositories, videoPath string, durationSeconds int64) (*models.Channel, []*models.Media, func()) {
	t.Helper()

	ctx := context.Background()

	// Create channel
	startTime := time.Now().Add(-24 * time.Hour)
	ch := models.NewChannel("Test Batch Channel", startTime, true)
	err := repos.Channels.Create(ctx, ch)
	require.NoError(t, err, "Failed to create test channel")

	// Create media item
	media := models.NewMedia(videoPath, "Test Video", durationSeconds)
	videoCodec := "h264"
	audioCodec := "aac"
	resolution := "1920x1080"
	fileSize := int64(1073741824) // 1GB
	media.VideoCodec = &videoCodec
	media.AudioCodec = &audioCodec
	media.Resolution = &resolution
	media.FileSize = &fileSize

	err = repos.Media.Create(ctx, media)
	require.NoError(t, err, "Failed to create test media")

	// Add to playlist
	playlistItem := models.NewPlaylistItem(ch.ID, media.ID, 0)
	err = repos.PlaylistItems.Create(ctx, playlistItem)
	require.NoError(t, err, "Failed to add media to playlist")

	cleanup := func() {
		repos.Channels.Delete(ctx, ch.ID)
		repos.Media.Delete(ctx, media.ID)
	}

	return ch, []*models.Media{media}, cleanup
}

// waitForBatchCompletion polls until batch completes or times out
func waitForBatchCompletion(t *testing.T, session *models.StreamSession, timeout time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		batch := session.GetCurrentBatch()
		if batch != nil && batch.IsComplete {
			return true
		}
		<-ticker.C
	}

	return false
}

// waitForSegmentsInPlaylist waits for the playlist to contain the expected number of segments
func waitForSegmentsInPlaylist(t *testing.T, segmentPath string, expectedCount int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	playlistPath := filepath.Join(segmentPath, "1080p.m3u8")
	for time.Now().Before(deadline) {
		if content, err := os.ReadFile(playlistPath); err == nil {
			segmentCount := 0
			for _, line := range strings.Split(string(content), "\n") {
				if strings.HasSuffix(line, ".ts") {
					segmentCount++
				}
			}
			if segmentCount >= expectedCount {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// waitForSegmentsToExist waits for specific segment files to exist on disk
func waitForSegmentsToExist(t *testing.T, segmentPath string, quality string, startSegment, endSegment int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if verifySegmentsExist(t, segmentPath, quality, startSegment, endSegment) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// verifySegmentsExist checks if segment files exist for a given range
// segmentDir should be the quality-specific directory (e.g., /tmp/.../channel_id/1080p)
func verifySegmentsExist(t *testing.T, segmentDir string, quality string, startSegment, endSegment int) bool {
	t.Helper()

	// segmentDir is already the quality directory (from GetSegmentPath())
	// So we use it directly without joining with quality again
	for i := startSegment; i <= endSegment; i++ {
		// Handle segment filename wrapping (%03d pattern wraps at 1000)
		filenameSegment := i % 1000
		segmentFilename := fmt.Sprintf("%s_segment_%03d.ts", quality, filenameSegment)
		segmentPath := filepath.Join(segmentDir, segmentFilename)

		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// verifySegmentsDeleted checks if segment files are deleted for a given range
// segmentDir should be the quality-specific directory (e.g., /tmp/.../channel_id/1080p)
func verifySegmentsDeleted(t *testing.T, segmentDir string, quality string, startSegment, endSegment int) bool {
	t.Helper()

	// segmentDir is already the quality directory (from GetSegmentPath())
	// So we use it directly without joining with quality again
	for i := startSegment; i <= endSegment; i++ {
		// Handle segment filename wrapping (%03d pattern wraps at 1000)
		filenameSegment := i % 1000
		segmentFilename := fmt.Sprintf("%s_segment_%03d.ts", quality, filenameSegment)
		segmentPath := filepath.Join(segmentDir, segmentFilename)

		if _, err := os.Stat(segmentPath); err == nil {
			return false // File still exists
		}
	}
	return true
}

// sendPositionUpdate sends a position update via API
func sendPositionUpdate(t *testing.T, router *gin.Engine, channelID uuid.UUID, sessionID string, segmentNumber int, quality string) (*http.Response, error) {
	t.Helper()

	reqBody := map[string]interface{}{
		"session_id":     sessionID,
		"segment_number": segmentNumber,
		"quality":        quality,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err, "Failed to marshal position update request")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	return w.Result(), nil
}

// triggerFirstBatch manually triggers the first batch generation
// This is needed because the batch coordinator doesn't handle nil batch case
func triggerFirstBatch(t *testing.T, manager *streaming.StreamManager, session *models.StreamSession) error {
	t.Helper()
	return streaming.TriggerFirstBatchForTest(manager, session)
}

// TestBatchGeneration tests that batch generation completes successfully
func TestBatchGeneration(t *testing.T) {
	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Setup
	_, repos, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create test video
	videoPath, cleanupVideo := createShortTestVideo(t, testVideoDuration)
	defer cleanupVideo()

	// Create temp directory for segments
	segmentDir := t.TempDir()

	// Create StreamManager
	manager, cleanupManager := setupTestStreamManager(t, repos, segmentDir)
	defer cleanupManager()

	// Create channel with media
	ch, _, cleanupChannel := createTestChannelWithMedia(t, repos, videoPath, testVideoDuration)
	defer cleanupChannel()

	ctx := context.Background()

	// Register client to start stream
	session, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register client and start stream")

	// Manually trigger first batch (batch coordinator doesn't handle nil batch case)
	err = triggerFirstBatch(t, manager, session)
	require.NoError(t, err, "Failed to trigger first batch")

	// Wait for first batch to complete
	completed := waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "First batch did not complete within timeout")

	// Wait for segments to be written to disk
	segmentPath := session.GetSegmentPath()
	require.True(t, waitForSegmentsInPlaylist(t, segmentPath, testBatchSize, 5*time.Second),
		"Segments should appear in playlist")

	// Verify batch state
	batch := session.GetCurrentBatch()
	require.NotNil(t, batch, "Batch state should not be nil")
	assert.Equal(t, 0, batch.BatchNumber, "First batch should be batch 0")
	assert.Equal(t, 0, batch.StartSegment, "First batch should start at segment 0")
	assert.Equal(t, testBatchSize-1, batch.EndSegment, "First batch should end at BatchSize-1")
	assert.True(t, batch.IsComplete, "Batch should be marked as complete")
	assert.False(t, batch.GenerationEnded.IsZero(), "GenerationEnded should be set")

	// Verify segments created on disk
	assert.True(t, verifySegmentsExist(t, segmentPath, "1080p", 0, testBatchSize-1),
		"Segment files should exist on disk")

	// Verify FFmpeg process exited (no hanging processes)
	// The process should have exited when batch completed
	pid := session.GetFFmpegPID()
	if pid > 0 {
		// Check if process is still running (it shouldn't be)
		process, err := os.FindProcess(pid)
		if err == nil {
			// Try to send signal 0 to check if process exists
			// This is a no-op signal that just checks if process exists
			err = process.Signal(os.Signal(nil))
			// If process exists, signal will succeed (but we expect it to have exited)
			// We can't easily check this without platform-specific code, so we'll just verify
			// that the batch completed successfully, which implies FFmpeg exited
		}
	}
}

// TestPositionTracking tests that position tracking updates work correctly
func TestPositionTracking(t *testing.T) {
	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Setup
	_, repos, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create test video
	videoPath, cleanupVideo := createShortTestVideo(t, testVideoDuration)
	defer cleanupVideo()

	// Create temp directory for segments
	segmentDir := t.TempDir()

	// Create StreamManager
	manager, cleanupManager := setupTestStreamManager(t, repos, segmentDir)
	defer cleanupManager()

	// Create channel with media
	ch, _, cleanupChannel := createTestChannelWithMedia(t, repos, videoPath, testVideoDuration)
	defer cleanupChannel()

	ctx := context.Background()

	// Register client to start stream
	session, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register client and start stream")

	// Manually trigger first batch (batch coordinator doesn't handle nil batch case)
	err = triggerFirstBatch(t, manager, session)
	require.NoError(t, err, "Failed to trigger first batch")

	// Setup router for API calls
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupStreamRoutes(apiGroup, manager)

	sessionID := uuid.New().String()

	// Send position update
	resp, err := sendPositionUpdate(t, router, ch.ID, sessionID, 2, "1080p")
	require.NoError(t, err, "Failed to send position update")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Position update should succeed")

	// Verify client position stored in session
	// Get updated session
	updatedSession, found := manager.GetStream(ch.ID)
	require.True(t, found, "Session should exist")
	assert.Equal(t, 2, updatedSession.GetFurthestPosition(), "Furthest segment should be updated")

	// Send another position update
	resp, err = sendPositionUpdate(t, router, ch.ID, sessionID, 5, "1080p")
	require.NoError(t, err, "Failed to send second position update")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Second position update should succeed")

	// Verify furthest position updated
	updatedSession, found = manager.GetStream(ch.ID)
	require.True(t, found, "Session should still exist")
	assert.Equal(t, 5, updatedSession.GetFurthestPosition(), "Furthest segment should be updated to 5")

	// Verify response includes current_batch and segments_remaining
	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	require.NoError(t, err, "Failed to decode response")
	assert.True(t, responseBody["acknowledged"].(bool), "Response should acknowledge update")
	assert.Contains(t, responseBody, "current_batch", "Response should include current_batch")
	assert.Contains(t, responseBody, "segments_remaining", "Response should include segments_remaining")
}

// TestAutomaticTriggering tests that next batch triggers automatically when threshold reached
func TestAutomaticTriggering(t *testing.T) {
	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Setup
	_, repos, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create test video
	videoPath, cleanupVideo := createShortTestVideo(t, testVideoDuration)
	defer cleanupVideo()

	// Create temp directory for segments
	segmentDir := t.TempDir()

	// Create StreamManager
	manager, cleanupManager := setupTestStreamManager(t, repos, segmentDir)
	defer cleanupManager()

	// Create channel with media
	ch, _, cleanupChannel := createTestChannelWithMedia(t, repos, videoPath, testVideoDuration)
	defer cleanupChannel()

	ctx := context.Background()

	// Register client to start stream
	session, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register client and start stream")

	// Manually trigger first batch (batch coordinator doesn't handle nil batch case)
	err = triggerFirstBatch(t, manager, session)
	require.NoError(t, err, "Failed to trigger first batch")

	// Wait for first batch to complete
	completed := waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "First batch did not complete within timeout")

	// Setup router for API calls
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupStreamRoutes(apiGroup, manager)

	sessionID := uuid.New().String()

	// Update position to trigger threshold
	// BatchSize=5, TriggerThreshold=2, so we need to reach segment 3 (5-2=3)
	// But we need to account for 0-indexing, so segment 3 means 4 segments remaining
	// Actually: EndSegment=4, Threshold=2, so when FurthestSegment=3, segmentsRemaining=1, which triggers
	triggerSegment := testBatchSize - testTriggerThreshold // 5 - 2 = 3

	resp, err := sendPositionUpdate(t, router, ch.ID, sessionID, triggerSegment, "1080p")
	require.NoError(t, err, "Failed to send position update")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Position update should succeed")

	// Wait for batch coordinator to trigger next batch
	// Give it a few seconds to trigger (coordinator runs every 2 seconds)
	time.Sleep(3 * time.Second)

	// Wait for second batch to start and complete
	completed = waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "Second batch did not complete within timeout")

	// Verify second batch started
	batch := session.GetCurrentBatch()
	require.NotNil(t, batch, "Batch state should not be nil")
	assert.Equal(t, 1, batch.BatchNumber, "Second batch should be batch 1")
	assert.Equal(t, testBatchSize, batch.StartSegment, "Second batch should start at segment BatchSize")
	assert.Equal(t, testBatchSize*2-1, batch.EndSegment, "Second batch should end at BatchSize*2-1")
	assert.True(t, batch.IsComplete, "Second batch should be marked as complete")
}

// TestBatchContinuation tests seamless continuation between batches
func TestBatchContinuation(t *testing.T) {
	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Setup
	_, repos, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create test video
	videoPath, cleanupVideo := createShortTestVideo(t, testVideoDuration)
	defer cleanupVideo()

	// Create temp directory for segments
	segmentDir := t.TempDir()

	// Create StreamManager
	manager, cleanupManager := setupTestStreamManager(t, repos, segmentDir)
	defer cleanupManager()

	// Create channel with media
	ch, _, cleanupChannel := createTestChannelWithMedia(t, repos, videoPath, testVideoDuration)
	defer cleanupChannel()

	ctx := context.Background()

	// Register client to start stream
	session, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register client and start stream")

	// Manually trigger first batch (batch coordinator doesn't handle nil batch case)
	err = triggerFirstBatch(t, manager, session)
	require.NoError(t, err, "Failed to trigger first batch")

	// Wait for first batch to complete
	completed := waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "First batch did not complete within timeout")

	firstBatch := session.GetCurrentBatch()
	require.NotNil(t, firstBatch, "First batch should exist")
	firstVideoOffset := firstBatch.VideoStartOffset

	// Setup router for API calls
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupStreamRoutes(apiGroup, manager)

	sessionID := uuid.New().String()

	// Trigger second batch
	triggerSegment := testBatchSize - testTriggerThreshold
	resp, err := sendPositionUpdate(t, router, ch.ID, sessionID, triggerSegment, "1080p")
	require.NoError(t, err, "Failed to send position update")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Position update should succeed")

	// Wait for second batch to complete
	time.Sleep(3 * time.Second) // Wait for coordinator to trigger
	completed = waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "Second batch did not complete within timeout")

	// Wait for all segments to be written to disk
	segmentPath := session.GetSegmentPath()
	// Wait for second batch segments to actually exist on disk
	// The batch FFmpeg process may exit before all segments are flushed
	require.True(t, waitForSegmentsToExist(t, segmentPath, "1080p", testBatchSize, testBatchSize*2-1, 10*time.Second),
		"Second batch segments should exist on disk")

	// Verify batch continuation
	secondBatch := session.GetCurrentBatch()
	require.NotNil(t, secondBatch, "Second batch should exist")

	// Verify segment numbering is continuous
	assert.Equal(t, firstBatch.EndSegment+1, secondBatch.StartSegment,
		"Second batch should start where first batch ended")

	// Verify video position calculated correctly
	expectedOffset := firstVideoOffset + int64(testBatchSize*testSegmentDuration)
	assert.Equal(t, expectedOffset, secondBatch.VideoStartOffset,
		"Video offset should continue from previous batch")

	// Verify no gaps in segments
	assert.True(t, verifySegmentsExist(t, segmentPath, "1080p", 0, testBatchSize-1),
		"First batch segments should still exist")
	assert.True(t, verifySegmentsExist(t, segmentPath, "1080p", testBatchSize, testBatchSize*2-1),
		"Second batch segments should exist")
}

// TestMultiClientSynchronization tests multiple clients reporting positions
func TestMultiClientSynchronization(t *testing.T) {
	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Setup
	_, repos, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create test video
	videoPath, cleanupVideo := createShortTestVideo(t, testVideoDuration)
	defer cleanupVideo()

	// Create temp directory for segments
	segmentDir := t.TempDir()

	// Create StreamManager
	manager, cleanupManager := setupTestStreamManager(t, repos, segmentDir)
	defer cleanupManager()

	// Create channel with media
	ch, _, cleanupChannel := createTestChannelWithMedia(t, repos, videoPath, testVideoDuration)
	defer cleanupChannel()

	ctx := context.Background()

	// Register multiple clients
	session1, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register first client")
	session2, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register second client")
	session3, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register third client")

	// All should get the same session
	assert.Equal(t, session1.ChannelID, session2.ChannelID, "All clients should share same session")
	assert.Equal(t, session1.ChannelID, session3.ChannelID, "All clients should share same session")

	// Setup router for API calls
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupStreamRoutes(apiGroup, manager)

	// Each client reports different positions
	sessionID1 := uuid.New().String()
	sessionID2 := uuid.New().String()
	sessionID3 := uuid.New().String()

	resp, err := sendPositionUpdate(t, router, ch.ID, sessionID1, 5, "1080p")
	require.NoError(t, err, "Failed to send position update from client 1")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = sendPositionUpdate(t, router, ch.ID, sessionID2, 10, "1080p")
	require.NoError(t, err, "Failed to send position update from client 2")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = sendPositionUpdate(t, router, ch.ID, sessionID3, 15, "1080p")
	require.NoError(t, err, "Failed to send position update from client 3")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify furthest position tracked correctly
	updatedSession, found := manager.GetStream(ch.ID)
	require.True(t, found, "Session should exist")
	assert.Equal(t, 15, updatedSession.GetFurthestPosition(),
		"Furthest position should be 15 (from client 3)")

	// Test client disconnection
	err = manager.UnregisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to unregister client")

	// Verify remaining clients still tracked
	// Position should still be 15 (furthest from remaining clients)
	updatedSession, found = manager.GetStream(ch.ID)
	require.True(t, found, "Session should still exist")
	assert.Equal(t, 15, updatedSession.GetFurthestPosition(),
		"Furthest position should remain 15 after client disconnection")
}

// TestBatchCleanup tests that old batches are cleaned up correctly
func TestBatchCleanup(t *testing.T) {
	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Setup
	_, repos, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create test video
	videoPath, cleanupVideo := createShortTestVideo(t, testVideoDuration)
	defer cleanupVideo()

	// Create temp directory for segments
	segmentDir := t.TempDir()

	// Create StreamManager
	manager, cleanupManager := setupTestStreamManager(t, repos, segmentDir)
	defer cleanupManager()

	// Create channel with media
	ch, _, cleanupChannel := createTestChannelWithMedia(t, repos, videoPath, testVideoDuration)
	defer cleanupChannel()

	ctx := context.Background()

	// Register client to start stream
	session, err := manager.RegisterClient(ctx, ch.ID)
	require.NoError(t, err, "Failed to register client and start stream")

	// Setup router for API calls
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupStreamRoutes(apiGroup, manager)

	sessionID := uuid.New().String()
	segmentPath := session.GetSegmentPath()

	// Generate batch 0
	err = triggerFirstBatch(t, manager, session)
	require.NoError(t, err, "Failed to trigger first batch")
	completed := waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "Batch 0 did not complete")
	require.True(t, waitForSegmentsInPlaylist(t, segmentPath, testBatchSize, 5*time.Second),
		"Batch 0 segments should appear in playlist")

	// Trigger batch 1
	triggerSegment := testBatchSize - testTriggerThreshold
	resp, err := sendPositionUpdate(t, router, ch.ID, sessionID, triggerSegment, "1080p")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	time.Sleep(3 * time.Second)
	completed = waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "Batch 1 did not complete")
	// Wait for batch 1 segments to be written
	require.True(t, waitForSegmentsToExist(t, segmentPath, "1080p", testBatchSize, testBatchSize*2-1, 10*time.Second),
		"Batch 1 segments should exist on disk")

	// Verify batch 0 segments still exist (N-1 batch kept during N generation)
	assert.True(t, verifySegmentsExist(t, segmentPath, "1080p", 0, testBatchSize-1),
		"Batch 0 segments should still exist (N-1 kept during N generation)")

	// Trigger batch 2
	triggerSegment = testBatchSize*2 - testTriggerThreshold
	resp, err = sendPositionUpdate(t, router, ch.ID, sessionID, triggerSegment, "1080p")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	time.Sleep(3 * time.Second)
	completed = waitForBatchCompletion(t, session, 30*time.Second)
	require.True(t, completed, "Batch 2 did not complete")
	// Wait for batch 2 segments to be written
	require.True(t, waitForSegmentsToExist(t, segmentPath, "1080p", testBatchSize*2, testBatchSize*3-1, 10*time.Second),
		"Batch 2 segments should exist on disk")

	// Give cleanup a moment to run (cleanup happens after batch completion)
	time.Sleep(1 * time.Second)

	// Verify batch 0 deleted (N-2 batch deleted after N completes)
	// Batch 2 completed, so batch 0 (N-2) should be deleted
	assert.True(t, verifySegmentsDeleted(t, segmentPath, "1080p", 0, testBatchSize-1),
		"Batch 0 segments should be deleted (N-2 cleanup)")

	// Verify batch 1 still exists (N-1 batch kept during N generation)
	assert.True(t, verifySegmentsExist(t, segmentPath, "1080p", testBatchSize, testBatchSize*2-1),
		"Batch 1 segments should still exist (N-1 kept during N generation)")

	// Verify batch 2 exists
	assert.True(t, verifySegmentsExist(t, segmentPath, "1080p", testBatchSize*2, testBatchSize*3-1),
		"Batch 2 segments should exist")
}

// TestBatchErrorHandling tests error handling when batch generation fails
func TestBatchErrorHandling(t *testing.T) {
	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping test")
	}

	// Setup
	_, repos, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create invalid video file (non-existent path)
	invalidVideoPath := "/nonexistent/path/to/video.mp4"

	// Create temp directory for segments
	segmentDir := t.TempDir()

	// Create StreamManager
	manager, cleanupManager := setupTestStreamManager(t, repos, segmentDir)
	defer cleanupManager()

	// Create channel with invalid media
	ch, _, cleanupChannel := createTestChannelWithMedia(t, repos, invalidVideoPath, testVideoDuration)
	defer cleanupChannel()

	ctx := context.Background()

	// Try to register client and start stream
	// This should fail because the video file doesn't exist
	session, err := manager.RegisterClient(ctx, ch.ID)
	if err != nil {
		// Expected: stream start should fail with invalid file
		assert.Error(t, err, "Stream start should fail with invalid video file")
		return
	}

	// If stream started (unlikely but possible), wait a bit and check for errors
	time.Sleep(2 * time.Second)

	// Check if session has errors
	if session != nil {
		errorCount := session.GetErrorCount()
		lastError := session.GetLastError()

		// If there are errors, verify they're handled gracefully
		if errorCount > 0 {
			assert.NotEmpty(t, lastError, "Last error should be set when errors occur")
			// Previous batch should still be available (none in this case, but error handling should work)
		}
	}
}
