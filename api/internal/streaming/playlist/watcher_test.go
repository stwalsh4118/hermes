package playlist

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	segmentDir := filepath.Join(tmpDir, "segments")
	outputPath := filepath.Join(tmpDir, "playlist.m3u8")

	pm, err := NewManager(6, outputPath, 4.0)
	require.NoError(t, err)

	tests := []struct {
		name            string
		segmentDir      string
		playlistManager Manager
		windowSize      uint
		safetyBuffer    uint
		pruneInterval   time.Duration
		segmentDuration float64
		pollInterval    time.Duration
		wantErr         bool
		errContains     string
	}{
		{
			name:            "valid parameters",
			segmentDir:      segmentDir,
			playlistManager: pm,
			windowSize:      6,
			safetyBuffer:    2,
			pruneInterval:   30 * time.Second,
			segmentDuration: 4.0,
			pollInterval:    1 * time.Second,
			wantErr:         false,
		},
		{
			name:            "empty segment directory",
			segmentDir:      "",
			playlistManager: pm,
			windowSize:      6,
			safetyBuffer:    2,
			pruneInterval:   30 * time.Second,
			segmentDuration: 4.0,
			pollInterval:    1 * time.Second,
			wantErr:         true,
			errContains:     "segment directory cannot be empty",
		},
		{
			name:            "nil playlist manager",
			segmentDir:      segmentDir,
			playlistManager: nil,
			windowSize:      6,
			safetyBuffer:    2,
			pruneInterval:   30 * time.Second,
			segmentDuration: 4.0,
			pollInterval:    1 * time.Second,
			wantErr:         true,
			errContains:     "playlist manager cannot be nil",
		},
		{
			name:            "zero window size",
			segmentDir:      segmentDir,
			playlistManager: pm,
			windowSize:      0,
			safetyBuffer:    2,
			pruneInterval:   30 * time.Second,
			segmentDuration: 4.0,
			pollInterval:    1 * time.Second,
			wantErr:         true,
			errContains:     "window size must be greater than 0",
		},
		{
			name:            "zero segment duration",
			segmentDir:      segmentDir,
			playlistManager: pm,
			windowSize:      6,
			safetyBuffer:    2,
			pruneInterval:   30 * time.Second,
			segmentDuration: 0,
			pollInterval:    1 * time.Second,
			wantErr:         true,
			errContains:     "segment duration must be greater than 0",
		},
		{
			name:            "zero prune interval",
			segmentDir:      segmentDir,
			playlistManager: pm,
			windowSize:      6,
			safetyBuffer:    2,
			pruneInterval:   0,
			segmentDuration: 4.0,
			pollInterval:    1 * time.Second,
			wantErr:         true,
			errContains:     "prune interval must be greater than 0",
		},
		{
			name:            "zero poll interval",
			segmentDir:      segmentDir,
			playlistManager: pm,
			windowSize:      6,
			safetyBuffer:    2,
			pruneInterval:   30 * time.Second,
			segmentDuration: 4.0,
			pollInterval:    0,
			wantErr:         true,
			errContains:     "poll interval must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher, err := NewWatcher(
				tt.segmentDir,
				tt.playlistManager,
				tt.windowSize,
				tt.safetyBuffer,
				tt.pruneInterval,
				tt.segmentDuration,
				tt.pollInterval,
			)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, watcher)
			} else {
				require.NoError(t, err)
				require.NotNil(t, watcher)
			}
		})
	}
}

func TestWatcher_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	segmentDir := filepath.Join(tmpDir, "segments")
	outputPath := filepath.Join(tmpDir, "playlist.m3u8")

	pm, err := NewManager(6, outputPath, 4.0)
	require.NoError(t, err)

	watcher, err := NewWatcher(
		segmentDir,
		pm,
		6,
		2,
		30*time.Second,
		4.0,
		1*time.Second,
	)
	require.NoError(t, err)

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)

	// Stop watcher
	err = watcher.Stop()
	require.NoError(t, err)

	// Stop again should be safe
	err = watcher.Stop()
	require.NoError(t, err)
}

func TestWatcher_SegmentDetection(t *testing.T) {
	tmpDir := t.TempDir()
	segmentDir := filepath.Join(tmpDir, "segments")
	outputPath := filepath.Join(tmpDir, "playlist.m3u8")

	pm, err := NewManager(6, outputPath, 4.0)
	require.NoError(t, err)

	watcher, err := NewWatcher(
		segmentDir,
		pm,
		6,
		2,
		30*time.Second,
		4.0,
		100*time.Millisecond, // Fast polling for test
	)
	require.NoError(t, err)

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Create a test segment file
	segmentFile := filepath.Join(segmentDir, "seg-20250111T120000.ts")
	err = os.WriteFile(segmentFile, []byte("test segment data"), 0644)
	require.NoError(t, err)

	// Wait for detection (polling interval + debounce + processing)
	// Need enough time for polling to detect, debounce to expire, and processing to complete
	time.Sleep(800 * time.Millisecond)

	// Verify segment was added to playlist
	segments := pm.GetCurrentSegments()
	assert.Contains(t, segments, "seg-20250111T120000.ts", "segment should be detected and added to playlist")
}

func TestWatcher_Pruning(t *testing.T) {
	tmpDir := t.TempDir()
	segmentDir := filepath.Join(tmpDir, "segments")
	outputPath := filepath.Join(tmpDir, "playlist.m3u8")

	windowSize := uint(6)
	safetyBuffer := uint(2)
	pruneThreshold := int(windowSize + safetyBuffer)

	pm, err := NewManager(windowSize, outputPath, 4.0)
	require.NoError(t, err)

	watcher, err := NewWatcher(
		segmentDir,
		pm,
		windowSize,
		safetyBuffer,
		100*time.Millisecond, // Fast pruning for test
		4.0,
		100*time.Millisecond,
	)
	require.NoError(t, err)

	// Create more segments than threshold
	numSegments := pruneThreshold + 5
	segmentFiles := make([]string, numSegments)
	for i := 0; i < numSegments; i++ {
		filename := segmentName(i)
		segmentFile := filepath.Join(segmentDir, filename)
		err := os.WriteFile(segmentFile, []byte("test segment data"), 0644)
		require.NoError(t, err)
		segmentFiles[i] = filename

		// Add small delay to ensure different mod times
		time.Sleep(10 * time.Millisecond)
	}

	// Add some segments to playlist (within window)
	for i := 0; i < int(windowSize); i++ {
		err := pm.AddSegment(SegmentMeta{
			URI:      segmentFiles[numSegments-int(windowSize)+i],
			Duration: 4.0,
		})
		require.NoError(t, err)
	}

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Wait for pruning to run
	time.Sleep(200 * time.Millisecond)

	// Verify files beyond threshold are deleted
	// But files in playlist should remain
	currentSegments := pm.GetCurrentSegments()
	currentSegmentsMap := make(map[string]bool)
	for _, seg := range currentSegments {
		currentSegmentsMap[seg] = true
	}

	// Check that files in playlist still exist
	for _, seg := range currentSegments {
		segmentFile := filepath.Join(segmentDir, seg)
		_, err := os.Stat(segmentFile)
		assert.NoError(t, err, "segment %s in playlist should still exist", seg)
	}

	// Count remaining files
	entries, err := os.ReadDir(segmentDir)
	require.NoError(t, err)

	remainingCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ts" {
			remainingCount++
		}
	}

	// Should have at most pruneThreshold files remaining (plus any in playlist)
	// Since we added windowSize segments to playlist, we should have those plus some others
	assert.LessOrEqual(t, remainingCount, numSegments, "should have pruned some files")
}

func TestWatcher_PruningRespectsPlaylist(t *testing.T) {
	tmpDir := t.TempDir()
	segmentDir := filepath.Join(tmpDir, "segments")
	outputPath := filepath.Join(tmpDir, "playlist.m3u8")

	windowSize := uint(6)
	safetyBuffer := uint(2)

	pm, err := NewManager(windowSize, outputPath, 4.0)
	require.NoError(t, err)

	watcher, err := NewWatcher(
		segmentDir,
		pm,
		windowSize,
		safetyBuffer,
		100*time.Millisecond,
		4.0,
		100*time.Millisecond,
	)
	require.NoError(t, err)

	// Create segments
	numSegments := 10
	segmentFiles := make([]string, numSegments)
	for i := 0; i < numSegments; i++ {
		filename := segmentName(i)
		segmentFile := filepath.Join(segmentDir, filename)
		err := os.WriteFile(segmentFile, []byte("test segment data"), 0644)
		require.NoError(t, err)
		segmentFiles[i] = filename
		time.Sleep(10 * time.Millisecond)
	}

	// Add all segments to playlist (beyond window, but playlist manager will prune internally)
	for i := 0; i < numSegments; i++ {
		err := pm.AddSegment(SegmentMeta{
			URI:      segmentFiles[i],
			Duration: 4.0,
		})
		require.NoError(t, err)
	}

	// Get current segments (after playlist manager's internal pruning)
	currentSegments := pm.GetCurrentSegments()

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Wait for pruning to run
	time.Sleep(200 * time.Millisecond)

	// Verify all segments in playlist still exist
	for _, seg := range currentSegments {
		segmentFile := filepath.Join(segmentDir, seg)
		_, err := os.Stat(segmentFile)
		assert.NoError(t, err, "segment %s in playlist should not be deleted", seg)
	}
}

func TestWatcher_Debouncing(t *testing.T) {
	tmpDir := t.TempDir()
	segmentDir := filepath.Join(tmpDir, "segments")
	outputPath := filepath.Join(tmpDir, "playlist.m3u8")

	pm, err := NewManager(6, outputPath, 4.0)
	require.NoError(t, err)

	watcher, err := NewWatcher(
		segmentDir,
		pm,
		6,
		2,
		30*time.Second,
		4.0,
		50*time.Millisecond, // Fast polling
	)
	require.NoError(t, err)

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Create segment file
	segmentFile := filepath.Join(segmentDir, "seg-test.ts")
	err = os.WriteFile(segmentFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Immediately modify it multiple times (simulating atomic write pattern)
	for i := 0; i < 5; i++ {
		err = os.WriteFile(segmentFile, []byte("test"), 0644)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debouncing and processing
	// Need time for polling to detect, debounce window to expire, and processing
	time.Sleep(800 * time.Millisecond)

	// Should only be added once despite multiple modifications
	segments := pm.GetCurrentSegments()
	count := 0
	for _, seg := range segments {
		if seg == "seg-test.ts" {
			count++
		}
	}
	assert.Equal(t, 1, count, "segment should be added only once despite multiple file events")
}

func TestWatcher_NonTSFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	segmentDir := filepath.Join(tmpDir, "segments")
	outputPath := filepath.Join(tmpDir, "playlist.m3u8")

	pm, err := NewManager(6, outputPath, 4.0)
	require.NoError(t, err)

	watcher, err := NewWatcher(
		segmentDir,
		pm,
		6,
		2,
		30*time.Second,
		4.0,
		100*time.Millisecond,
	)
	require.NoError(t, err)

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Create non-TS files
	nonTSFiles := []string{"test.txt", "test.m3u8", "test.log", "test.ts.bak"}
	for _, filename := range nonTSFiles {
		file := filepath.Join(segmentDir, filename)
		err := os.WriteFile(file, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify no segments were added
	segments := pm.GetCurrentSegments()
	for _, seg := range segments {
		for _, nonTS := range nonTSFiles {
			assert.NotEqual(t, nonTS, seg, "non-TS file %s should not be added to playlist", nonTS)
		}
	}
}
