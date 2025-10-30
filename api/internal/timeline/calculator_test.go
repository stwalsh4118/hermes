package timeline

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/models"
)

// Helper function to create a test media item
func createTestMedia(id uuid.UUID, title string, durationSeconds int64) *models.Media {
	return &models.Media{
		ID:       id,
		Title:    title,
		Duration: durationSeconds,
	}
}

// Helper function to create a test playlist item
func createTestPlaylistItem(position int, media *models.Media) *models.PlaylistItem {
	return &models.PlaylistItem{
		ID:       uuid.New(),
		Position: position,
		Media:    media,
	}
}

func TestCalculatePosition_EmptyPlaylist(t *testing.T) {
	startTime := time.Now().UTC()
	currentTime := startTime.Add(1 * time.Hour)
	playlist := []*models.PlaylistItem{}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true)

	assert.Nil(t, pos)
	assert.ErrorIs(t, err, ErrEmptyPlaylist)
}

func TestCalculatePosition_ChannelNotStarted(t *testing.T) {
	startTime := time.Now().UTC().Add(1 * time.Hour) // Starts in future
	currentTime := time.Now().UTC()

	media := createTestMedia(uuid.New(), "Test Media", 3600)
	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true)

	assert.Nil(t, pos)
	assert.ErrorIs(t, err, ErrChannelNotStarted)
}

func TestCalculatePosition_FirstItem_JustStarted(t *testing.T) {
	startTime := time.Now().UTC()
	currentTime := startTime // Exactly at start

	media := createTestMedia(uuid.New(), "First Video", 3600)
	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true)

	require.NoError(t, err)
	require.NotNil(t, pos)

	assert.Equal(t, media.ID, pos.MediaID)
	assert.Equal(t, "First Video", pos.MediaTitle)
	assert.Equal(t, int64(0), pos.OffsetSeconds)
	assert.Equal(t, int64(3600), pos.Duration)
	assert.WithinDuration(t, startTime, pos.StartedAt, 1*time.Second)
	assert.WithinDuration(t, startTime.Add(3600*time.Second), pos.EndsAt, 1*time.Second)
}

func TestCalculatePosition_MidPlaylist(t *testing.T) {
	startTime := time.Now().UTC()
	// Current time is 1.5 hours after start (should be in middle of item 2)
	currentTime := startTime.Add(90 * time.Minute)

	media1 := createTestMedia(uuid.New(), "Video 1", 3600) // 1 hour
	media2 := createTestMedia(uuid.New(), "Video 2", 3600) // 1 hour
	media3 := createTestMedia(uuid.New(), "Video 3", 3600) // 1 hour

	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media1),
		createTestPlaylistItem(1, media2),
		createTestPlaylistItem(2, media3),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true)

	require.NoError(t, err)
	require.NotNil(t, pos)

	// Should be in the second item (index 1), 30 minutes in
	assert.Equal(t, media2.ID, pos.MediaID)
	assert.Equal(t, "Video 2", pos.MediaTitle)
	assert.Equal(t, int64(1800), pos.OffsetSeconds) // 30 minutes
	assert.Equal(t, int64(3600), pos.Duration)
}

func TestCalculatePosition_WithinItem(t *testing.T) {
	startTime := time.Now().UTC()
	// 30 minutes into first video
	currentTime := startTime.Add(30 * time.Minute)

	media := createTestMedia(uuid.New(), "Video", 3600) // 1 hour
	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true)

	require.NoError(t, err)
	require.NotNil(t, pos)

	assert.Equal(t, media.ID, pos.MediaID)
	assert.Equal(t, int64(1800), pos.OffsetSeconds) // 30 minutes = 1800 seconds
	assert.Equal(t, int64(3600), pos.Duration)
}

func TestCalculatePosition_LastItem(t *testing.T) {
	startTime := time.Now().UTC()
	// 2.5 hours after start (in the 3rd video)
	currentTime := startTime.Add(2*time.Hour + 30*time.Minute)

	media1 := createTestMedia(uuid.New(), "Video 1", 3600) // 1 hour
	media2 := createTestMedia(uuid.New(), "Video 2", 3600) // 1 hour
	media3 := createTestMedia(uuid.New(), "Video 3", 3600) // 1 hour

	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media1),
		createTestPlaylistItem(1, media2),
		createTestPlaylistItem(2, media3),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, false)

	require.NoError(t, err)
	require.NotNil(t, pos)

	// Should be in the third item, 30 minutes in
	assert.Equal(t, media3.ID, pos.MediaID)
	assert.Equal(t, "Video 3", pos.MediaTitle)
	assert.Equal(t, int64(1800), pos.OffsetSeconds) // 30 minutes
}

func TestCalculatePosition_LoopBoundary(t *testing.T) {
	startTime := time.Now().UTC()
	// 3.5 hours after start with 3 hours total playlist -> should wrap to 30 minutes into first video
	currentTime := startTime.Add(3*time.Hour + 30*time.Minute)

	media1 := createTestMedia(uuid.New(), "Video 1", 3600) // 1 hour
	media2 := createTestMedia(uuid.New(), "Video 2", 3600) // 1 hour
	media3 := createTestMedia(uuid.New(), "Video 3", 3600) // 1 hour

	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media1),
		createTestPlaylistItem(1, media2),
		createTestPlaylistItem(2, media3),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true) // loop=true

	require.NoError(t, err)
	require.NotNil(t, pos)

	// Should wrap back to first item, 30 minutes in
	assert.Equal(t, media1.ID, pos.MediaID)
	assert.Equal(t, "Video 1", pos.MediaTitle)
	assert.Equal(t, int64(1800), pos.OffsetSeconds) // 30 minutes
}

func TestCalculatePosition_NonLoopPastEnd(t *testing.T) {
	startTime := time.Now().UTC()
	// 5 hours after start with 3 hours total playlist and no loop
	currentTime := startTime.Add(5 * time.Hour)

	media1 := createTestMedia(uuid.New(), "Video 1", 3600)
	media2 := createTestMedia(uuid.New(), "Video 2", 3600)
	media3 := createTestMedia(uuid.New(), "Video 3", 3600)

	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media1),
		createTestPlaylistItem(1, media2),
		createTestPlaylistItem(2, media3),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, false) // loop=false

	assert.Nil(t, pos)
	assert.ErrorIs(t, err, ErrPlaylistFinished)
}

func TestCalculatePosition_SingleItemLoop(t *testing.T) {
	startTime := time.Now().UTC()
	// 2.5 hours after start with 1 hour video and loop enabled
	currentTime := startTime.Add(2*time.Hour + 30*time.Minute)

	media := createTestMedia(uuid.New(), "Video", 3600) // 1 hour
	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true) // loop=true

	require.NoError(t, err)
	require.NotNil(t, pos)

	// Should wrap: 2.5 hours % 1 hour = 30 minutes
	assert.Equal(t, media.ID, pos.MediaID)
	assert.Equal(t, int64(1800), pos.OffsetSeconds) // 30 minutes
}

func TestCalculatePosition_SingleItemNoLoop(t *testing.T) {
	startTime := time.Now().UTC()
	// 2 hours after start with 1 hour video and no loop
	currentTime := startTime.Add(2 * time.Hour)

	media := createTestMedia(uuid.New(), "Video", 3600) // 1 hour
	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, false) // loop=false

	assert.Nil(t, pos)
	assert.ErrorIs(t, err, ErrPlaylistFinished)
}

func TestCalculatePosition_MultipleLoops(t *testing.T) {
	startTime := time.Now().UTC()
	// 10 hours after start with 3 hours total playlist
	// Should have looped 3 times and be 1 hour into 4th loop
	currentTime := startTime.Add(10 * time.Hour)

	media1 := createTestMedia(uuid.New(), "Video 1", 3600)
	media2 := createTestMedia(uuid.New(), "Video 2", 3600)
	media3 := createTestMedia(uuid.New(), "Video 3", 3600)

	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media1),
		createTestPlaylistItem(1, media2),
		createTestPlaylistItem(2, media3),
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true)

	require.NoError(t, err)
	require.NotNil(t, pos)

	// 10 hours % 3 hours = 1 hour -> should be in second video at the start
	assert.Equal(t, media2.ID, pos.MediaID)
	assert.Equal(t, int64(0), pos.OffsetSeconds)
}

func TestCalculatePosition_NilMedia(t *testing.T) {
	startTime := time.Now().UTC()
	currentTime := startTime.Add(1 * time.Hour)

	// Playlist item with nil Media (should be handled gracefully)
	playlist := []*models.PlaylistItem{
		{
			ID:       uuid.New(),
			Position: 0,
			Media:    nil, // Nil media
		},
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, true)

	assert.Nil(t, pos)
	assert.ErrorIs(t, err, ErrEmptyPlaylist) // Should treat as empty
}

func TestCalculatePosition_AccuracyTest(t *testing.T) {
	// Test that calculations are accurate within ±1 second

	startTime := time.Date(2025, 10, 30, 12, 0, 0, 0, time.UTC)

	media1 := createTestMedia(uuid.New(), "Video 1", 1234) // Odd duration
	media2 := createTestMedia(uuid.New(), "Video 2", 5678)

	playlist := []*models.PlaylistItem{
		createTestPlaylistItem(0, media1),
		createTestPlaylistItem(1, media2),
	}

	// Test at various points
	testCases := []struct {
		name            string
		elapsed         time.Duration
		expectedMediaID uuid.UUID
		expectedOffset  int64
	}{
		{"Start", 0 * time.Second, media1.ID, 0},
		{"Mid first", 500 * time.Second, media1.ID, 500},
		{"End first", 1234 * time.Second, media2.ID, 0},
		{"Mid second", 2000 * time.Second, media2.ID, 766}, // 2000 - 1234 = 766
		{"Near end", 6900 * time.Second, media2.ID, 5666},  // 6900 - 1234 = 5666
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			currentTime := startTime.Add(tc.elapsed)
			pos, err := CalculatePosition(startTime, currentTime, playlist, false)

			require.NoError(t, err)
			require.NotNil(t, pos)

			assert.Equal(t, tc.expectedMediaID, pos.MediaID)
			// Allow ±1 second difference for floating point arithmetic
			assert.InDelta(t, tc.expectedOffset, pos.OffsetSeconds, 1)
		})
	}
}

func TestCalculatePosition_VeryLongPlaylist(t *testing.T) {
	// Test with 100 items to ensure algorithm scales
	startTime := time.Now().UTC()
	// Position ourselves in the middle (item 50)
	currentTime := startTime.Add(50 * time.Hour)

	playlist := make([]*models.PlaylistItem, 100)
	expectedMedia := createTestMedia(uuid.New(), "Video 50", 3600)

	for i := 0; i < 100; i++ {
		var media *models.Media
		if i == 50 {
			media = expectedMedia
		} else {
			media = createTestMedia(uuid.New(), "Video "+string(rune(i)), 3600)
		}
		playlist[i] = createTestPlaylistItem(i, media)
	}

	pos, err := CalculatePosition(startTime, currentTime, playlist, false)

	require.NoError(t, err)
	require.NotNil(t, pos)

	assert.Equal(t, expectedMedia.ID, pos.MediaID)
	assert.Equal(t, int64(0), pos.OffsetSeconds)
}

// Benchmark tests to verify performance requirements
func BenchmarkCalculatePosition_SmallPlaylist(b *testing.B) {
	startTime := time.Now().UTC()
	currentTime := startTime.Add(2 * time.Hour)

	playlist := make([]*models.PlaylistItem, 10)
	for i := 0; i < 10; i++ {
		media := createTestMedia(uuid.New(), "Video", 3600)
		playlist[i] = createTestPlaylistItem(i, media)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculatePosition(startTime, currentTime, playlist, true)
	}
}

func BenchmarkCalculatePosition_1000Items(b *testing.B) {
	// Critical benchmark: must complete in < 100ms
	startTime := time.Now().UTC()
	// Position in the middle to test linear search performance
	currentTime := startTime.Add(500 * time.Hour)

	playlist := make([]*models.PlaylistItem, 1000)
	for i := 0; i < 1000; i++ {
		media := createTestMedia(uuid.New(), "Video", 3600)
		playlist[i] = createTestPlaylistItem(i, media)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculatePosition(startTime, currentTime, playlist, true)
	}
}

func BenchmarkCalculatePosition_FirstItem(b *testing.B) {
	// Best case: item is found immediately
	startTime := time.Now().UTC()
	currentTime := startTime.Add(10 * time.Minute)

	playlist := make([]*models.PlaylistItem, 1000)
	for i := 0; i < 1000; i++ {
		media := createTestMedia(uuid.New(), "Video", 3600)
		playlist[i] = createTestPlaylistItem(i, media)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculatePosition(startTime, currentTime, playlist, true)
	}
}

func BenchmarkCalculatePosition_LastItem(b *testing.B) {
	// Worst case: must traverse entire playlist
	startTime := time.Now().UTC()
	currentTime := startTime.Add(999 * time.Hour) // Near the last item

	playlist := make([]*models.PlaylistItem, 1000)
	for i := 0; i < 1000; i++ {
		media := createTestMedia(uuid.New(), "Video", 3600)
		playlist[i] = createTestPlaylistItem(i, media)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculatePosition(startTime, currentTime, playlist, false)
	}
}
