package timeline

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/channel"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/models"
)

// setupTestService creates a service with a test database
func setupTestService(t *testing.T) (*TimelineService, *db.DB, func()) {
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
	service := NewTimelineService(repos)

	cleanup := func() {
		_ = database.Close()
	}

	return service, database, cleanup
}

func TestNewTimelineService(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	assert.NotNil(t, service)
	assert.NotNil(t, service.repos)
}

func TestGetCurrentPosition_Success(t *testing.T) {
	service, database, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	repos := db.NewRepositories(database)

	// Create a channel that started 1 hour ago
	startTime := time.Now().UTC().Add(-1 * time.Hour)
	ch := &models.Channel{
		ID:        uuid.New(),
		Name:      "Test Channel",
		StartTime: startTime,
		Loop:      true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := repos.Channels.Create(ctx, ch)
	require.NoError(t, err)

	// Create a media item (30 minutes)
	const mediaDuration = 1800
	media := &models.Media{
		ID:        uuid.New(),
		FilePath:  "/test/video.mp4",
		Title:     "Test Video",
		Duration:  mediaDuration,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.Media.Create(ctx, media)
	require.NoError(t, err)

	// Add to playlist
	playlistItem := &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: ch.ID,
		MediaID:   media.ID,
		Position:  0,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.PlaylistItems.Create(ctx, playlistItem)
	require.NoError(t, err)

	// Get current position
	position, err := service.GetCurrentPosition(ctx, ch.ID)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, position)
	assert.Equal(t, media.ID, position.MediaID)
	assert.Equal(t, "Test Video", position.MediaTitle)
	assert.Equal(t, int64(mediaDuration), position.Duration)
	// Offset should be somewhere in the video (exact value depends on timing)
	assert.GreaterOrEqual(t, position.OffsetSeconds, int64(0))
	assert.Less(t, position.OffsetSeconds, int64(mediaDuration))
}

func TestGetCurrentPosition_ChannelNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	nonExistentID := uuid.New()

	// Try to get position for non-existent channel
	position, err := service.GetCurrentPosition(ctx, nonExistentID)

	// Verify
	assert.Nil(t, position)
	require.Error(t, err)
	assert.ErrorIs(t, err, channel.ErrChannelNotFound)
}

func TestGetCurrentPosition_EmptyPlaylist(t *testing.T) {
	service, database, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	repos := db.NewRepositories(database)

	// Create a channel with no playlist items
	ch := &models.Channel{
		ID:        uuid.New(),
		Name:      "Empty Channel",
		StartTime: time.Now().UTC().Add(-1 * time.Hour),
		Loop:      true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := repos.Channels.Create(ctx, ch)
	require.NoError(t, err)

	// Get current position
	position, err := service.GetCurrentPosition(ctx, ch.ID)

	// Verify
	assert.Nil(t, position)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyPlaylist)
}

func TestGetCurrentPosition_ChannelNotStarted(t *testing.T) {
	service, database, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	repos := db.NewRepositories(database)

	// Create a channel that starts 1 hour in the future
	futureStartTime := time.Now().UTC().Add(1 * time.Hour)
	ch := &models.Channel{
		ID:        uuid.New(),
		Name:      "Future Channel",
		StartTime: futureStartTime,
		Loop:      true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := repos.Channels.Create(ctx, ch)
	require.NoError(t, err)

	// Create media and playlist item
	media := &models.Media{
		ID:        uuid.New(),
		FilePath:  "/test/video.mp4",
		Title:     "Test Video",
		Duration:  1800,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.Media.Create(ctx, media)
	require.NoError(t, err)

	playlistItem := &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: ch.ID,
		MediaID:   media.ID,
		Position:  0,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.PlaylistItems.Create(ctx, playlistItem)
	require.NoError(t, err)

	// Get current position
	position, err := service.GetCurrentPosition(ctx, ch.ID)

	// Verify
	assert.Nil(t, position)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrChannelNotStarted)
}

func TestGetCurrentPosition_PlaylistFinished(t *testing.T) {
	service, database, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	repos := db.NewRepositories(database)

	// Create a non-looping channel that started 2 hours ago
	startTime := time.Now().UTC().Add(-2 * time.Hour)
	ch := &models.Channel{
		ID:        uuid.New(),
		Name:      "Non-Looping Channel",
		StartTime: startTime,
		Loop:      false, // Non-looping
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := repos.Channels.Create(ctx, ch)
	require.NoError(t, err)

	// Create media with only 30 minutes of content
	const mediaDuration = 1800 // 30 minutes
	media := &models.Media{
		ID:        uuid.New(),
		FilePath:  "/test/video.mp4",
		Title:     "Test Video",
		Duration:  mediaDuration,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.Media.Create(ctx, media)
	require.NoError(t, err)

	playlistItem := &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: ch.ID,
		MediaID:   media.ID,
		Position:  0,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.PlaylistItems.Create(ctx, playlistItem)
	require.NoError(t, err)

	// Get current position (should fail because playlist finished)
	position, err := service.GetCurrentPosition(ctx, ch.ID)

	// Verify
	assert.Nil(t, position)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPlaylistFinished)
}

func TestGetCurrentPosition_MultipleItems(t *testing.T) {
	service, database, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	repos := db.NewRepositories(database)

	// Create a channel that started 45 minutes ago
	startTime := time.Now().UTC().Add(-45 * time.Minute)
	ch := &models.Channel{
		ID:        uuid.New(),
		Name:      "Multi-Item Channel",
		StartTime: startTime,
		Loop:      true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := repos.Channels.Create(ctx, ch)
	require.NoError(t, err)

	// Create first media item (30 minutes)
	media1 := &models.Media{
		ID:        uuid.New(),
		FilePath:  "/test/video1.mp4",
		Title:     "Video 1",
		Duration:  1800, // 30 minutes
		CreatedAt: time.Now().UTC(),
	}
	err = repos.Media.Create(ctx, media1)
	require.NoError(t, err)

	// Create second media item (30 minutes)
	media2 := &models.Media{
		ID:        uuid.New(),
		FilePath:  "/test/video2.mp4",
		Title:     "Video 2",
		Duration:  1800, // 30 minutes
		CreatedAt: time.Now().UTC(),
	}
	err = repos.Media.Create(ctx, media2)
	require.NoError(t, err)

	// Add both to playlist
	item1 := &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: ch.ID,
		MediaID:   media1.ID,
		Position:  0,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.PlaylistItems.Create(ctx, item1)
	require.NoError(t, err)

	item2 := &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: ch.ID,
		MediaID:   media2.ID,
		Position:  1,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.PlaylistItems.Create(ctx, item2)
	require.NoError(t, err)

	// Get current position (should be in second video since 45 minutes elapsed)
	position, err := service.GetCurrentPosition(ctx, ch.ID)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, position)
	assert.Equal(t, media2.ID, position.MediaID)
	assert.Equal(t, "Video 2", position.MediaTitle)
	// Should be about 15 minutes into the second video
	assert.GreaterOrEqual(t, position.OffsetSeconds, int64(800)) // At least ~13 minutes
	assert.LessOrEqual(t, position.OffsetSeconds, int64(1000))   // At most ~16 minutes
}

func TestGetCurrentPosition_LoopingChannel(t *testing.T) {
	service, database, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	repos := db.NewRepositories(database)

	// Create a looping channel that started 2 hours ago
	startTime := time.Now().UTC().Add(-2 * time.Hour)
	ch := &models.Channel{
		ID:        uuid.New(),
		Name:      "Looping Channel",
		StartTime: startTime,
		Loop:      true, // Looping enabled
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := repos.Channels.Create(ctx, ch)
	require.NoError(t, err)

	// Create media with 30 minutes of content
	const mediaDuration = 1800 // 30 minutes
	media := &models.Media{
		ID:        uuid.New(),
		FilePath:  "/test/video.mp4",
		Title:     "Looping Video",
		Duration:  mediaDuration,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.Media.Create(ctx, media)
	require.NoError(t, err)

	playlistItem := &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: ch.ID,
		MediaID:   media.ID,
		Position:  0,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.PlaylistItems.Create(ctx, playlistItem)
	require.NoError(t, err)

	// Get current position (should loop around)
	position, err := service.GetCurrentPosition(ctx, ch.ID)

	// Verify - should succeed with looping
	require.NoError(t, err)
	assert.NotNil(t, position)
	assert.Equal(t, media.ID, position.MediaID)
	assert.Equal(t, "Looping Video", position.MediaTitle)
	assert.GreaterOrEqual(t, position.OffsetSeconds, int64(0))
	assert.Less(t, position.OffsetSeconds, int64(mediaDuration))
}
