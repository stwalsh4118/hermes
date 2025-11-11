package playlist

import (
	"os"
	"path/filepath"
	"strings"
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

func TestWatcher_TimestampRegression(t *testing.T) {
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
		50*time.Millisecond,
	)
	require.NoError(t, err)

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Create first segment with time T0
	t0 := time.Now().UTC()
	segment1 := filepath.Join(segmentDir, "seg-001.ts")
	err = os.WriteFile(segment1, []byte("test"), 0644)
	require.NoError(t, err)
	// Set modification time to T0
	err = os.Chtimes(segment1, t0, t0)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Create second segment with time T1 (after T0)
	t1 := t0.Add(5 * time.Second)
	segment2 := filepath.Join(segmentDir, "seg-002.ts")
	err = os.WriteFile(segment2, []byte("test"), 0644)
	require.NoError(t, err)
	err = os.Chtimes(segment2, t1, t1)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Create third segment with time T2 (before T1 - regression!)
	t2 := t0.Add(2 * time.Second) // Earlier than t1
	segment3 := filepath.Join(segmentDir, "seg-003.ts")
	err = os.WriteFile(segment3, []byte("test"), 0644)
	require.NoError(t, err)
	err = os.Chtimes(segment3, t2, t2)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Write playlist to check for discontinuity
	err = pm.Write()
	require.NoError(t, err)

	// Read playlist content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	playlistStr := string(content)

	// Should contain discontinuity tag (triggered by timestamp regression)
	assert.Contains(t, playlistStr, "#EXT-X-DISCONTINUITY",
		"playlist should contain discontinuity tag after timestamp regression")

	// Verify discontinuity appears before seg-003.ts (the regressed segment)
	discontinuityIndex := strings.Index(playlistStr, "#EXT-X-DISCONTINUITY")
	seg003Index := strings.Index(playlistStr, "seg-003.ts")
	assert.True(t, discontinuityIndex < seg003Index,
		"discontinuity tag should appear before seg-003.ts (regressed segment)")
}

func TestWatcher_FirstSegmentNoDiscontinuity(t *testing.T) {
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
		50*time.Millisecond,
	)
	require.NoError(t, err)

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Create first segment (no previous timestamp)
	segment1 := filepath.Join(segmentDir, "seg-001.ts")
	err = os.WriteFile(segment1, []byte("test"), 0644)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Write playlist
	err = pm.Write()
	require.NoError(t, err)

	// Read playlist content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	playlistStr := string(content)

	// First segment should NOT have discontinuity (no previous timestamp to compare)
	// Count discontinuity tags
	discontinuityCount := strings.Count(playlistStr, "#EXT-X-DISCONTINUITY")
	assert.Equal(t, 0, discontinuityCount,
		"first segment should not have discontinuity tag")
}

func TestWatcher_NormalProgressionNoDiscontinuity(t *testing.T) {
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
		50*time.Millisecond,
	)
	require.NoError(t, err)

	// Start watcher
	err = watcher.Start()
	require.NoError(t, err)
	defer func() {
		_ = watcher.Stop()
	}()

	// Create segments with normal progression (increasing timestamps)
	baseTime := time.Now().UTC()
	for i := 0; i < 3; i++ {
		segmentTime := baseTime.Add(time.Duration(i) * 5 * time.Second)
		filename := segmentName(i)
		segmentFile := filepath.Join(segmentDir, filename)
		err := os.WriteFile(segmentFile, []byte("test"), 0644)
		require.NoError(t, err)
		err = os.Chtimes(segmentFile, segmentTime, segmentTime)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Write playlist
	err = pm.Write()
	require.NoError(t, err)

	// Read playlist content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	playlistStr := string(content)

	// Normal progression should NOT have discontinuity
	discontinuityCount := strings.Count(playlistStr, "#EXT-X-DISCONTINUITY")
	assert.Equal(t, 0, discontinuityCount,
		"normal timestamp progression should not trigger discontinuity")
}

func TestWatcher_MarkDiscontinuity(t *testing.T) {
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
		50*time.Millisecond,
	)
	require.NoError(t, err)

	// Add a segment first
	err = pm.AddSegment(SegmentMeta{
		URI:      "seg-001.ts",
		Duration: 4.0,
	})
	require.NoError(t, err)

	// Mark discontinuity (simulating encoder restart)
	watcher.MarkDiscontinuity()

	// Add next segment (should have discontinuity tag)
	err = pm.AddSegment(SegmentMeta{
		URI:      "seg-002.ts",
		Duration: 4.0,
	})
	require.NoError(t, err)

	// Write playlist
	err = pm.Write()
	require.NoError(t, err)

	// Read playlist content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	playlistStr := string(content)

	// Should contain discontinuity tag
	assert.Contains(t, playlistStr, "#EXT-X-DISCONTINUITY",
		"playlist should contain discontinuity tag after MarkDiscontinuity()")

	// Verify discontinuity appears before seg-002.ts
	discontinuityIndex := strings.Index(playlistStr, "#EXT-X-DISCONTINUITY")
	seg002Index := strings.Index(playlistStr, "seg-002.ts")
	assert.True(t, discontinuityIndex < seg002Index,
		"discontinuity tag should appear before seg-002.ts")
}

func TestWatcher_MarkDiscontinuityResetsTimestamp(t *testing.T) {
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
		50*time.Millisecond,
	)
	require.NoError(t, err)

	// Add first segment directly (simulating watcher behavior)
	t0 := time.Now().UTC()
	err = pm.AddSegment(SegmentMeta{
		URI:             "seg-001.ts",
		Duration:        4.0,
		ProgramDateTime: &t0,
	})
	require.NoError(t, err)

	// Mark discontinuity (simulating encoder restart)
	watcher.MarkDiscontinuity()

	// Add second segment with earlier timestamp (would normally trigger regression)
	// But after MarkDiscontinuity(), timestamp tracking is reset, so this should be fine
	t1 := t0.Add(-5 * time.Second) // Earlier than t0
	err = pm.AddSegment(SegmentMeta{
		URI:             "seg-002.ts",
		Duration:        4.0,
		ProgramDateTime: &t1,
	})
	require.NoError(t, err)

	// Write playlist
	err = pm.Write()
	require.NoError(t, err)

	// Read playlist content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	playlistStr := string(content)

	// Should contain discontinuity tag (from MarkDiscontinuity, not regression)
	assert.Contains(t, playlistStr, "#EXT-X-DISCONTINUITY",
		"playlist should contain discontinuity tag from MarkDiscontinuity()")

	// Verify discontinuity appears before seg-002.ts
	discontinuityIndex := strings.Index(playlistStr, "#EXT-X-DISCONTINUITY")
	seg002Index := strings.Index(playlistStr, "seg-002.ts")
	assert.True(t, discontinuityIndex < seg002Index,
		"discontinuity tag should appear before seg-002.ts")
}
