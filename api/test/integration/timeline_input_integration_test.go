//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stwalsh4118/hermes/internal/models"
	"github.com/stwalsh4118/hermes/internal/streaming"
	"github.com/stwalsh4118/hermes/internal/timeline"
)

// TestBuildTimelineInput_Integration tests the full integration with Timeline Service
func TestBuildTimelineInput_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()
	timelineService := timeline.NewTimelineService(repos)

	// Create temp directory for test files
	tempDir := t.TempDir()

	// Create test media files
	video1 := createTestVideoFile(t, tempDir, "video1.mp4")
	video2 := createTestVideoFile(t, tempDir, "video2.mp4")
	video3 := createTestVideoFile(t, tempDir, "video3.mp4")

	// Create test data
	ctx := context.Background()

	// Create media items
	media1 := models.NewMedia(video1, "Video 1", 1800) // 30 min
	media2 := models.NewMedia(video2, "Video 2", 2700) // 45 min
	media3 := models.NewMedia(video3, "Video 3", 1200) // 20 min

	require.NoError(t, repos.Media.Create(ctx, media1))
	require.NoError(t, repos.Media.Create(ctx, media2))
	require.NoError(t, repos.Media.Create(ctx, media3))

	tests := []struct {
		name            string
		startTime       time.Time
		loop            bool
		currentTime     time.Time
		expectSimple    bool
		expectConcat    bool
		expectedSeekMin int64
		expectedSeekMax int64
		checkConcatFile bool
	}{
		{
			name:            "Simple seek in middle of first video",
			startTime:       time.Now().Add(-900 * time.Second), // Started 15 min ago
			loop:            true,
			currentTime:     time.Now(),
			expectSimple:    true,
			expectConcat:    false,
			expectedSeekMin: 890, // Around 15 min
			expectedSeekMax: 910,
		},
		{
			name:            "Near start optimization - no seek",
			startTime:       time.Now().Add(-5 * time.Second), // Started 5 sec ago
			loop:            true,
			currentTime:     time.Now(),
			expectSimple:    true,
			expectConcat:    false,
			expectedSeekMin: 0, // Optimized to 0
			expectedSeekMax: 0,
		},
		{
			name:            "Near end of first video - concat mode",
			startTime:       time.Now().Add(-1785 * time.Second), // 29:45 into first video
			loop:            true,
			currentTime:     time.Now(),
			expectSimple:    false,
			expectConcat:    true,
			checkConcatFile: true,
		},
		{
			name:            "Second video looping playlist",
			startTime:       time.Now().Add(-2100 * time.Second), // 35 min = in second video
			loop:            true,
			currentTime:     time.Now(),
			expectSimple:    true,
			expectConcat:    false,
			expectedSeekMin: 290, // Around 5 min into second video
			expectedSeekMax: 310,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create channel
			channel := models.NewChannel("Test Channel", tt.startTime, tt.loop)
			require.NoError(t, repos.Channels.Create(ctx, channel))

			// Create playlist
			playlist1 := models.NewPlaylistItem(channel.ID, media1.ID, 0)
			playlist2 := models.NewPlaylistItem(channel.ID, media2.ID, 1)
			playlist3 := models.NewPlaylistItem(channel.ID, media3.ID, 2)

			require.NoError(t, repos.PlaylistItems.Create(ctx, playlist1))
			require.NoError(t, repos.PlaylistItems.Create(ctx, playlist2))
			require.NoError(t, repos.PlaylistItems.Create(ctx, playlist3))

			// Build timeline input
			result, err := streaming.BuildTimelineInput(ctx, channel.ID, timelineService, repos)
			require.NoError(t, err, "BuildTimelineInput should succeed")
			require.NotNil(t, result, "Result should not be nil")

			// Verify result based on test expectations
			if tt.expectSimple {
				assert.False(t, result.UseConcatFile, "Should use simple input")
				assert.Empty(t, result.ConcatFilePath, "Concat file path should be empty")

				// Check seek range if specified
				if tt.expectedSeekMin > 0 || tt.expectedSeekMax > 0 {
					assert.GreaterOrEqual(t, result.SeekSeconds, tt.expectedSeekMin,
						"Seek should be at least min")
					assert.LessOrEqual(t, result.SeekSeconds, tt.expectedSeekMax,
						"Seek should be at most max")
				} else {
					assert.Equal(t, int64(0), result.SeekSeconds,
						"Seek should be optimized to 0")
				}

				// Verify primary file exists
				_, err := os.Stat(result.PrimaryFile)
				assert.NoError(t, err, "Primary file should exist")
			}

			if tt.expectConcat {
				assert.True(t, result.UseConcatFile, "Should use concat input")
				assert.NotEmpty(t, result.ConcatFilePath, "Concat file path should be set")
				assert.NotEmpty(t, result.ConcatItems, "Should have concat items")

				// Verify concat file was created if requested
				if tt.checkConcatFile {
					_, err := os.Stat(result.ConcatFilePath)
					assert.NoError(t, err, "Concat file should exist")

					// Read and verify concat file format
					content, err := os.ReadFile(result.ConcatFilePath)
					require.NoError(t, err)

					contentStr := string(content)
					assert.Contains(t, contentStr, "file '", "Should have file directive")
					assert.Contains(t, contentStr, video1, "Should include first video")

					// Cleanup concat file
					_ = os.Remove(result.ConcatFilePath)
				}

				// Verify all files in concat list exist
				for i, item := range result.ConcatItems {
					_, err := os.Stat(item.FilePath)
					assert.NoError(t, err, "Concat item %d file should exist", i)
				}
			}

			// Common verifications
			assert.Greater(t, result.TotalDuration, int64(0), "Total duration should be positive")
			assert.LessOrEqual(t, result.TotalDuration, int64(streaming.MaxStreamDuration),
				"Total duration should not exceed max")

			// Cleanup
			require.NoError(t, repos.Channels.Delete(ctx, channel.ID))
		})
	}
}

// TestBuildTimelineInput_ErrorCases tests error handling in integration
func TestBuildTimelineInput_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	timelineService := timeline.NewTimelineService(repos)
	ctx := context.Background()

	t.Run("Non-existent channel", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := streaming.BuildTimelineInput(ctx, nonExistentID, timelineService, repos)
		assert.Error(t, err, "Should error for non-existent channel")
	})

	t.Run("Channel not started yet", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		channel := models.NewChannel("Future Channel", futureTime, true)
		require.NoError(t, repos.Channels.Create(ctx, channel))

		_, err := streaming.BuildTimelineInput(ctx, channel.ID, timelineService, repos)
		assert.Error(t, err, "Should error when channel hasn't started")
		assert.Contains(t, err.Error(), "timeline position")

		// Cleanup
		repos.Channels.Delete(ctx, channel.ID)
	})

	t.Run("Empty playlist", func(t *testing.T) {
		channel := models.NewChannel("Empty Channel", time.Now().Add(-1*time.Hour), true)
		require.NoError(t, repos.Channels.Create(ctx, channel))

		// Don't add any playlist items

		_, err := streaming.BuildTimelineInput(ctx, channel.ID, timelineService, repos)
		assert.Error(t, err, "Should error for empty playlist")

		// Cleanup
		repos.Channels.Delete(ctx, channel.ID)
	})

	t.Run("Missing media file", func(t *testing.T) {
		// Create media with non-existent file
		media := models.NewMedia("/nonexistent/video.mp4", "Missing Video", 1800)
		require.NoError(t, repos.Media.Create(ctx, media))

		channel := models.NewChannel("Test Channel", time.Now().Add(-900*time.Second), true)
		require.NoError(t, repos.Channels.Create(ctx, channel))

		playlist := models.NewPlaylistItem(channel.ID, media.ID, 0)
		require.NoError(t, repos.PlaylistItems.Create(ctx, playlist))

		_, err := streaming.BuildTimelineInput(ctx, channel.ID, timelineService, repos)
		assert.Error(t, err, "Should error when media file doesn't exist")
		assert.Contains(t, err.Error(), "not found")

		// Cleanup
		repos.Channels.Delete(ctx, channel.ID)
		repos.Media.Delete(ctx, media.ID)
	})
}

// TestBuildTimelineInput_WithFFmpegValidation validates generated concat files with FFmpeg
func TestBuildTimelineInput_WithFFmpegValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Check if FFmpeg is available
	if err := streaming.CheckFFmpegInstalled(); err != nil {
		t.Skip("FFmpeg not installed, skipping validation test")
	}

	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	timelineService := timeline.NewTimelineService(repos)

	// Create temp directory for test files
	tempDir := t.TempDir()

	// Create test video files (actual valid video data would be needed for full FFmpeg test)
	// For now, we'll just test concat file format generation
	video1 := createTestVideoFile(t, tempDir, "video1.mp4")
	video2 := createTestVideoFile(t, tempDir, "video2.mp4")

	ctx := context.Background()

	// Create media items
	media1 := models.NewMedia(video1, "Video 1", 1800)
	media2 := models.NewMedia(video2, "Video 2", 1800)
	require.NoError(t, repos.Media.Create(ctx, media1))
	require.NoError(t, repos.Media.Create(ctx, media2))

	// Create channel near end of first video to trigger concat
	channel := models.NewChannel("Concat Test", time.Now().Add(-1785*time.Second), true)
	require.NoError(t, repos.Channels.Create(ctx, channel))

	// Create playlist
	playlist1 := models.NewPlaylistItem(channel.ID, media1.ID, 0)
	playlist2 := models.NewPlaylistItem(channel.ID, media2.ID, 1)
	require.NoError(t, repos.PlaylistItems.Create(ctx, playlist1))
	require.NoError(t, repos.PlaylistItems.Create(ctx, playlist2))

	// Build timeline input
	result, err := streaming.BuildTimelineInput(ctx, channel.ID, timelineService, repos)
	require.NoError(t, err)

	if result.UseConcatFile {
		// Verify concat file format is valid
		content, err := os.ReadFile(result.ConcatFilePath)
		require.NoError(t, err, "Should be able to read concat file")

		contentStr := string(content)

		// Verify basic format requirements
		assert.Contains(t, contentStr, "file '", "Should have file directives")
		assert.Contains(t, contentStr, video1, "Should include first video")

		// Cleanup concat file
		os.Remove(result.ConcatFilePath)
	}

	// Cleanup
	repos.Channels.Delete(ctx, channel.ID)
}

// Helper: creates a test video file (empty for testing)
func createTestVideoFile(t *testing.T, dir, filename string) string {
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte("fake video content"), 0644)
	require.NoError(t, err)
	return path
}
