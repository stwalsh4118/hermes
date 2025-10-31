package streaming

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stwalsh4118/hermes/internal/models"
)

// TestGetNextPlaylistItems tests the playlist iteration logic with looping
func TestGetNextPlaylistItems(t *testing.T) {
	// Create test playlist
	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, 1800), // 30 min
		createTestPlaylistItem(1, 2700), // 45 min
		createTestPlaylistItem(2, 1200), // 20 min
		createTestPlaylistItem(3, 3600), // 60 min
	}

	tests := []struct {
		name            string
		currentPosition int
		count           int
		loop            bool
		expectedCount   int
		expectedIndices []int
	}{
		{
			name:            "Get next 2 items from middle with loop",
			currentPosition: 1,
			count:           2,
			loop:            true,
			expectedCount:   2,
			expectedIndices: []int{2, 3},
		},
		{
			name:            "Get next 3 items wrapping around with loop",
			currentPosition: 2,
			count:           3,
			loop:            true,
			expectedCount:   3,
			expectedIndices: []int{3, 0, 1},
		},
		{
			name:            "Get next items at end with loop - wraps to start",
			currentPosition: 3,
			count:           2,
			loop:            true,
			expectedCount:   2,
			expectedIndices: []int{0, 1},
		},
		{
			name:            "Get next items at end without loop - stops",
			currentPosition: 3,
			count:           2,
			loop:            false,
			expectedCount:   0,
			expectedIndices: []int{},
		},
		{
			name:            "Get next items near end without loop - partial",
			currentPosition: 2,
			count:           3,
			loop:            false,
			expectedCount:   1,
			expectedIndices: []int{3},
		},
		{
			name:            "Request zero items",
			currentPosition: 0,
			count:           0,
			loop:            true,
			expectedCount:   0,
			expectedIndices: []int{},
		},
		{
			name:            "Request more than available without loop",
			currentPosition: 0,
			count:           10,
			loop:            false,
			expectedCount:   3,
			expectedIndices: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNextPlaylistItems(playlist, tt.currentPosition, tt.count, tt.loop)

			assert.Equal(t, tt.expectedCount, len(result), "Should return expected number of items")

			// Verify we got the right items
			for i, expectedIdx := range tt.expectedIndices {
				assert.Equal(t, playlist[expectedIdx].Position, result[i].Position,
					"Item %d should be from position %d", i, expectedIdx)
			}
		})
	}
}

// TestGetNextPlaylistItems_EmptyPlaylist tests edge case with empty playlist
func TestGetNextPlaylistItems_EmptyPlaylist(t *testing.T) {
	result := GetNextPlaylistItems([]*models.PlaylistItem{}, 0, 5, true)
	assert.Nil(t, result, "Should return nil for empty playlist")
}

// TestCalculateStreamDuration tests duration calculation with capping
func TestCalculateStreamDuration(t *testing.T) {
	tests := []struct {
		name             string
		remainingCurrent int64
		nextItems        []*models.PlaylistItem
		maxDuration      int64
		expected         int64
	}{
		{
			name:             "Simple sum under max",
			remainingCurrent: 600, // 10 min
			nextItems: []*models.PlaylistItem{
				createTestPlaylistItem(0, 1800), // 30 min
				createTestPlaylistItem(1, 1200), // 20 min
			},
			maxDuration: 7200, // 2 hours
			expected:    3600, // 60 min total
		},
		{
			name:             "Sum exceeds max - capped",
			remainingCurrent: 1800, // 30 min
			nextItems: []*models.PlaylistItem{
				createTestPlaylistItem(0, 3600), // 60 min
				createTestPlaylistItem(1, 3600), // 60 min
				createTestPlaylistItem(2, 3600), // 60 min
			},
			maxDuration: 5400, // 90 min max
			expected:    5400, // Capped at 90 min
		},
		{
			name:             "No next items",
			remainingCurrent: 1200,
			nextItems:        []*models.PlaylistItem{},
			maxDuration:      7200,
			expected:         1200,
		},
		{
			name:             "Next items with nil media ignored",
			remainingCurrent: 600,
			nextItems: []*models.PlaylistItem{
				createTestPlaylistItem(0, 1800),
				{Media: nil}, // This should be skipped
				createTestPlaylistItem(2, 1200),
			},
			maxDuration: 7200,
			expected:    3600, // Only valid items counted
		},
		{
			name:             "Already at max duration",
			remainingCurrent: 7200,
			nextItems: []*models.PlaylistItem{
				createTestPlaylistItem(0, 1800),
			},
			maxDuration: 7200,
			expected:    7200, // Capped immediately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateStreamDuration(tt.remainingCurrent, tt.nextItems, tt.maxDuration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildConcatFile tests concat file generation
func TestBuildConcatFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		items         []ConcatItem
		expectedLines []string
		expectError   bool
	}{
		{
			name: "Simple concat file with one item",
			items: []ConcatItem{
				{FilePath: "/path/to/video1.mp4", InPoint: 0, OutPoint: 0},
			},
			expectedLines: []string{
				"file '/path/to/video1.mp4'",
			},
			expectError: false,
		},
		{
			name: "Multiple items with inpoint",
			items: []ConcatItem{
				{FilePath: "/path/to/video1.mp4", InPoint: 120, OutPoint: 0},
				{FilePath: "/path/to/video2.mp4", InPoint: 0, OutPoint: 0},
			},
			expectedLines: []string{
				"file '/path/to/video1.mp4'",
				"inpoint 120",
				"file '/path/to/video2.mp4'",
			},
			expectError: false,
		},
		{
			name: "Items with inpoint and outpoint",
			items: []ConcatItem{
				{FilePath: "/path/to/video1.mp4", InPoint: 60, OutPoint: 180},
				{FilePath: "/path/to/video2.mp4", InPoint: 0, OutPoint: 300},
			},
			expectedLines: []string{
				"file '/path/to/video1.mp4'",
				"inpoint 60",
				"outpoint 180",
				"file '/path/to/video2.mp4'",
				"outpoint 300",
			},
			expectError: false,
		},
		{
			name:        "Empty items - error",
			items:       []ConcatItem{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tempDir, "concat.txt")

			err := BuildConcatFile(tt.items, outputPath)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Read and verify file content
			content, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			lines := strings.Split(strings.TrimSpace(string(content)), "\n")

			assert.Equal(t, len(tt.expectedLines), len(lines),
				"Should have expected number of lines")

			for i, expectedLine := range tt.expectedLines {
				assert.Equal(t, expectedLine, lines[i],
					"Line %d should match", i)
			}
		})
	}
}

// TestBuildConcatFile_AtomicWrite tests that file writes are atomic
func TestBuildConcatFile_AtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "concat.txt")

	items := []ConcatItem{
		{FilePath: "/path/to/video.mp4", InPoint: 0, OutPoint: 0},
	}

	err := BuildConcatFile(items, outputPath)
	require.NoError(t, err)

	// Verify no .tmp file remains
	tmpPath := outputPath + ".tmp"
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "Temp file should not exist after successful write")

	// Verify final file exists
	_, err = os.Stat(outputPath)
	assert.NoError(t, err, "Final file should exist")
}

// TestValidateFilePaths tests file path validation
func TestValidateFilePaths(t *testing.T) {
	// Create test files
	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "video.mp4")
	err := os.WriteFile(validFile, []byte("test"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		items       []ConcatItem
		expectError bool
		errorMatch  string
	}{
		{
			name: "All valid paths",
			items: []ConcatItem{
				{FilePath: validFile},
			},
			expectError: false,
		},
		{
			name: "Empty path",
			items: []ConcatItem{
				{FilePath: ""},
			},
			expectError: true,
			errorMatch:  "file path is empty",
		},
		{
			name: "Relative path",
			items: []ConcatItem{
				{FilePath: "relative/path/video.mp4"},
			},
			expectError: true,
			errorMatch:  "must be absolute",
		},
		{
			name: "Non-existent file",
			items: []ConcatItem{
				{FilePath: "/nonexistent/path/video.mp4"},
			},
			expectError: true,
			errorMatch:  "not found",
		},
		{
			name: "Directory instead of file",
			items: []ConcatItem{
				{FilePath: tempDir},
			},
			expectError: true,
			errorMatch:  "is a directory",
		},
		{
			name: "Multiple items - one invalid",
			items: []ConcatItem{
				{FilePath: validFile},
				{FilePath: "/nonexistent/video.mp4"},
			},
			expectError: true,
			errorMatch:  "item 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePaths(tt.items)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMatch != "" {
					assert.Contains(t, err.Error(), tt.errorMatch)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildSimpleInput tests simple seek-based input construction
func TestBuildSimpleInput(t *testing.T) {
	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "video.mp4")
	err := os.WriteFile(validFile, []byte("test"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name             string
		filePath         string
		offsetSeconds    int64
		remainingSeconds int64
		expectError      bool
		expectedSeek     int64
		expectedPrimary  string
		expectedConcat   bool
	}{
		{
			name:             "Normal seek in middle of file",
			filePath:         validFile,
			offsetSeconds:    300,
			remainingSeconds: 1200,
			expectError:      false,
			expectedSeek:     300,
			expectedPrimary:  validFile,
			expectedConcat:   false,
		},
		{
			name:             "Near start - optimization applies",
			filePath:         validFile,
			offsetSeconds:    5, // Less than threshold (10)
			remainingSeconds: 1800,
			expectError:      false,
			expectedSeek:     0, // Optimized to 0
			expectedPrimary:  validFile,
			expectedConcat:   false,
		},
		{
			name:             "At exact threshold - no optimization",
			filePath:         validFile,
			offsetSeconds:    10,
			remainingSeconds: 1800,
			expectError:      false,
			expectedSeek:     10,
			expectedPrimary:  validFile,
			expectedConcat:   false,
		},
		{
			name:             "Non-existent file",
			filePath:         "/nonexistent/video.mp4",
			offsetSeconds:    100,
			remainingSeconds: 1200,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildSimpleInput(tt.filePath, tt.offsetSeconds, tt.remainingSeconds)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedSeek, result.SeekSeconds)
			assert.Equal(t, tt.expectedPrimary, result.PrimaryFile)
			assert.Equal(t, tt.expectedConcat, result.UseConcatFile)
			assert.Equal(t, tt.remainingSeconds, result.TotalDuration)
		})
	}
}

// TestBuildConcatInput tests concat-based input construction
func TestBuildConcatInput(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "video1.mp4")
	file2 := filepath.Join(tempDir, "video2.mp4")
	file3 := filepath.Join(tempDir, "video3.mp4")

	for _, f := range []string{file1, file2, file3} {
		err := os.WriteFile(f, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create test playlist
	playlist := []*models.PlaylistItem{
		createTestPlaylistItemWithPath(0, 1800, file1),
		createTestPlaylistItemWithPath(1, 2700, file2),
		createTestPlaylistItemWithPath(2, 1200, file3),
	}

	tests := []struct {
		name             string
		currentFilePath  string
		offsetSeconds    int64
		remainingSeconds int64
		currentPosition  int
		loop             bool
		expectedConcat   bool
		expectedFiles    int
		minDuration      int64
	}{
		{
			name:             "Near end with loop - includes next files",
			currentFilePath:  file1,
			offsetSeconds:    120,
			remainingSeconds: 20, // Less than threshold (30)
			currentPosition:  0,
			loop:             true,
			expectedConcat:   true,
			expectedFiles:    MaxConcatFiles, // Current + up to MaxConcatFiles-1 next
			minDuration:      20,
		},
		{
			name:             "Near end without loop",
			currentFilePath:  file2,
			offsetSeconds:    100,
			remainingSeconds: 25,
			currentPosition:  1,
			loop:             false,
			expectedConcat:   true,
			expectedFiles:    2, // Current + 1 next (last item)
			minDuration:      25,
		},
		{
			name:             "Last item with loop - wraps around",
			currentFilePath:  file3,
			offsetSeconds:    0,
			remainingSeconds: 20,
			currentPosition:  2,
			loop:             true,
			expectedConcat:   true,
			expectedFiles:    MaxConcatFiles, // Current + wraps around for more
			minDuration:      20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := buildConcatInput(ctx, tt.currentFilePath, tt.offsetSeconds,
				tt.remainingSeconds, playlist, tt.currentPosition, tt.loop)

			require.NoError(t, err)
			assert.True(t, result.UseConcatFile, "Should use concat file")
			assert.Equal(t, tt.expectedFiles, len(result.ConcatItems))
			assert.GreaterOrEqual(t, result.TotalDuration, tt.minDuration)

			// Verify first item has correct inpoint
			assert.Equal(t, tt.offsetSeconds, result.ConcatItems[0].InPoint)

			// Verify concat file was created
			assert.NotEmpty(t, result.ConcatFilePath)
			_, err = os.Stat(result.ConcatFilePath)
			assert.NoError(t, err, "Concat file should exist")

			// Cleanup
			_ = os.Remove(result.ConcatFilePath)
		})
	}
}

// TestBuildConcatInput_MissingFile tests error handling for missing files
func TestBuildConcatInput_MissingFile(t *testing.T) {
	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "video1.mp4")
	err := os.WriteFile(validFile, []byte("test"), 0644)
	require.NoError(t, err)

	playlist := []*models.PlaylistItem{
		createTestPlaylistItemWithPath(0, 1800, validFile),
		createTestPlaylistItemWithPath(1, 2700, "/nonexistent/video2.mp4"),
	}

	ctx := context.Background()
	_, err = buildConcatInput(ctx, validFile, 120, 20, playlist, 0, true)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Helper function to create test playlist item
func createTestPlaylistItem(position int, duration int64) *models.PlaylistItem {
	return &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		MediaID:   uuid.New(),
		Position:  position,
		CreatedAt: time.Now(),
		Media: &models.Media{
			ID:        uuid.New(),
			FilePath:  "/test/path/video.mp4",
			Title:     "Test Video",
			Duration:  duration,
			CreatedAt: time.Now(),
		},
	}
}

// Helper function to create test playlist item with specific file path
func createTestPlaylistItemWithPath(position int, duration int64, filePath string) *models.PlaylistItem {
	return &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		MediaID:   uuid.New(),
		Position:  position,
		CreatedAt: time.Now(),
		Media: &models.Media{
			ID:        uuid.New(),
			FilePath:  filePath,
			Title:     "Test Video",
			Duration:  duration,
			CreatedAt: time.Now(),
		},
	}
}
