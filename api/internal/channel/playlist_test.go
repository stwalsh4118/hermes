package channel

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
)

func setupPlaylistTest(t *testing.T) (*PlaylistService, *db.Repositories, func()) {
	t.Helper()

	// Initialize logger for tests
	logger.Init("error", false)

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)

	// Run migrations
	sqlDB, err := database.GetSQLDB()
	require.NoError(t, err)
	err = db.RunMigrations(sqlDB, "file://../../migrations")
	require.NoError(t, err)

	// Create repositories
	repos := db.NewRepositories(database)

	// Create service
	service := NewPlaylistService(database, repos)

	cleanup := func() {
		_ = database.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return service, repos, cleanup
}

func createTestChannel(t *testing.T, repos *db.Repositories, name string) *models.Channel {
	t.Helper()

	channel := &models.Channel{
		ID:        uuid.New(),
		Name:      name,
		StartTime: time.Now().UTC(),
		Loop:      true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := repos.Channels.Create(context.Background(), channel)
	require.NoError(t, err)

	return channel
}

func createTestMedia(t *testing.T, repos *db.Repositories, title string, duration int64) *models.Media {
	t.Helper()

	media := &models.Media{
		ID:        uuid.New(),
		FilePath:  "/test/" + title + ".mp4",
		Title:     title,
		Duration:  duration,
		CreatedAt: time.Now().UTC(),
	}

	err := repos.Media.Create(context.Background(), media)
	require.NoError(t, err)

	return media
}

func TestAddToPlaylist_Success(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media
	channel := createTestChannel(t, repos, "Test Channel")
	media := createTestMedia(t, repos, "Test Video", 3600)

	// Add media to playlist at position 0
	item, err := service.AddToPlaylist(ctx, channel.ID, media.ID, 0)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, channel.ID, item.ChannelID)
	assert.Equal(t, media.ID, item.MediaID)
	assert.Equal(t, 0, item.Position)

	// Verify item exists in database
	dbItem, err := repos.PlaylistItems.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, dbItem.Position)
}

func TestAddToPlaylist_PositionConflict(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and multiple media items
	channel := createTestChannel(t, repos, "Test Channel")
	media1 := createTestMedia(t, repos, "Video 1", 1800)
	media2 := createTestMedia(t, repos, "Video 2", 1800)
	media3 := createTestMedia(t, repos, "Video 3", 1800)

	// Add first two items
	item1, err := service.AddToPlaylist(ctx, channel.ID, media1.ID, 0)
	require.NoError(t, err)
	item2, err := service.AddToPlaylist(ctx, channel.ID, media2.ID, 1)
	require.NoError(t, err)

	// Insert new item at position 1 (should shift item2 to position 2)
	item3, err := service.AddToPlaylist(ctx, channel.ID, media3.ID, 1)
	require.NoError(t, err)

	// Verify positions
	assert.Equal(t, 1, item3.Position)

	// Get all items and verify ordering
	items, err := repos.PlaylistItems.GetByChannelID(ctx, channel.ID)
	require.NoError(t, err)
	require.Len(t, items, 3)

	// Verify positions are correct
	assert.Equal(t, item1.ID, items[0].ID)
	assert.Equal(t, 0, items[0].Position)

	assert.Equal(t, item3.ID, items[1].ID)
	assert.Equal(t, 1, items[1].Position)

	assert.Equal(t, item2.ID, items[2].ID)
	assert.Equal(t, 2, items[2].Position)
}

func TestAddToPlaylist_MediaNotFound(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel but no media
	channel := createTestChannel(t, repos, "Test Channel")
	fakeMediaID := uuid.New()

	// Try to add non-existent media
	item, err := service.AddToPlaylist(ctx, channel.ID, fakeMediaID, 0)

	// Assert error
	assert.Error(t, err)
	assert.Nil(t, item)
	assert.True(t, IsMediaNotFound(err))
}

func TestAddToPlaylist_ChannelNotFound(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create media but no channel
	media := createTestMedia(t, repos, "Test Video", 3600)
	fakeChannelID := uuid.New()

	// Try to add to non-existent channel
	item, err := service.AddToPlaylist(ctx, fakeChannelID, media.ID, 0)

	// Assert error
	assert.Error(t, err)
	assert.Nil(t, item)
	assert.True(t, IsChannelNotFound(err))
}

func TestAddToPlaylist_InvalidPosition(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media
	channel := createTestChannel(t, repos, "Test Channel")
	media := createTestMedia(t, repos, "Test Video", 3600)

	// Try to add with negative position
	item, err := service.AddToPlaylist(ctx, channel.ID, media.ID, -1)

	// Assert error
	assert.Error(t, err)
	assert.Nil(t, item)
	assert.True(t, IsInvalidPosition(err))
}

func TestRemoveFromPlaylist_Success(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media
	channel := createTestChannel(t, repos, "Test Channel")
	media1 := createTestMedia(t, repos, "Video 1", 1800)
	media2 := createTestMedia(t, repos, "Video 2", 1800)
	media3 := createTestMedia(t, repos, "Video 3", 1800)

	// Add three items
	item1, err := service.AddToPlaylist(ctx, channel.ID, media1.ID, 0)
	require.NoError(t, err)
	item2, err := service.AddToPlaylist(ctx, channel.ID, media2.ID, 1)
	require.NoError(t, err)
	item3, err := service.AddToPlaylist(ctx, channel.ID, media3.ID, 2)
	require.NoError(t, err)

	// Remove middle item
	err = service.RemoveFromPlaylist(ctx, item2.ID)
	require.NoError(t, err)

	// Verify item is deleted
	_, err = repos.PlaylistItems.GetByID(ctx, item2.ID)
	assert.True(t, db.IsNotFound(err))

	// Verify remaining items are reordered
	items, err := repos.PlaylistItems.GetByChannelID(ctx, channel.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	assert.Equal(t, item1.ID, items[0].ID)
	assert.Equal(t, 0, items[0].Position)

	assert.Equal(t, item3.ID, items[1].ID)
	assert.Equal(t, 1, items[1].Position) // Shifted down from 2 to 1
}

func TestRemoveFromPlaylist_ItemNotFound(t *testing.T) {
	service, _, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Try to remove non-existent item
	fakeItemID := uuid.New()
	err := service.RemoveFromPlaylist(ctx, fakeItemID)

	// Assert error
	assert.Error(t, err)
	assert.True(t, IsPlaylistItemNotFound(err))
}

func TestReorderPlaylist_Success(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media
	channel := createTestChannel(t, repos, "Test Channel")
	media1 := createTestMedia(t, repos, "Video 1", 1800)
	media2 := createTestMedia(t, repos, "Video 2", 1800)
	media3 := createTestMedia(t, repos, "Video 3", 1800)

	// Add three items
	item1, err := service.AddToPlaylist(ctx, channel.ID, media1.ID, 0)
	require.NoError(t, err)
	item2, err := service.AddToPlaylist(ctx, channel.ID, media2.ID, 1)
	require.NoError(t, err)
	item3, err := service.AddToPlaylist(ctx, channel.ID, media3.ID, 2)
	require.NoError(t, err)

	// Reorder: swap positions (0->2, 1->0, 2->1)
	reorderItems := []db.ReorderItem{
		{ID: item1.ID, Position: 2},
		{ID: item2.ID, Position: 0},
		{ID: item3.ID, Position: 1},
	}

	err = service.ReorderPlaylist(ctx, channel.ID, reorderItems)
	require.NoError(t, err)

	// Verify new ordering
	items, err := repos.PlaylistItems.GetByChannelID(ctx, channel.ID)
	require.NoError(t, err)
	require.Len(t, items, 3)

	assert.Equal(t, item2.ID, items[0].ID)
	assert.Equal(t, 0, items[0].Position)

	assert.Equal(t, item3.ID, items[1].ID)
	assert.Equal(t, 1, items[1].Position)

	assert.Equal(t, item1.ID, items[2].ID)
	assert.Equal(t, 2, items[2].Position)
}

func TestReorderPlaylist_ItemNotFound(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel
	channel := createTestChannel(t, repos, "Test Channel")

	// Try to reorder with non-existent item
	fakeItemID := uuid.New()
	reorderItems := []db.ReorderItem{
		{ID: fakeItemID, Position: 0},
	}

	err := service.ReorderPlaylist(ctx, channel.ID, reorderItems)

	// Assert error
	assert.Error(t, err)
	assert.True(t, IsPlaylistItemNotFound(err))
}

func TestReorderPlaylist_ItemFromDifferentChannel(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create two channels
	channel1 := createTestChannel(t, repos, "Channel 1")
	channel2 := createTestChannel(t, repos, "Channel 2")
	media := createTestMedia(t, repos, "Video", 1800)

	// Add item to channel1
	item, err := service.AddToPlaylist(ctx, channel1.ID, media.ID, 0)
	require.NoError(t, err)

	// Try to reorder it in channel2
	reorderItems := []db.ReorderItem{
		{ID: item.ID, Position: 0},
	}

	err = service.ReorderPlaylist(ctx, channel2.ID, reorderItems)

	// Assert error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to channel")
}

func TestGetPlaylist_Success(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media
	channel := createTestChannel(t, repos, "Test Channel")
	media1 := createTestMedia(t, repos, "Video 1", 1800)
	media2 := createTestMedia(t, repos, "Video 2", 3600)

	// Add items
	_, err := service.AddToPlaylist(ctx, channel.ID, media1.ID, 0)
	require.NoError(t, err)
	_, err = service.AddToPlaylist(ctx, channel.ID, media2.ID, 1)
	require.NoError(t, err)

	// Get playlist
	items, err := service.GetPlaylist(ctx, channel.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Verify items are ordered and include media details
	assert.Equal(t, 0, items[0].Position)
	assert.NotNil(t, items[0].Media)
	assert.Equal(t, "Video 1", items[0].Media.Title)
	assert.Equal(t, int64(1800), items[0].Media.Duration)

	assert.Equal(t, 1, items[1].Position)
	assert.NotNil(t, items[1].Media)
	assert.Equal(t, "Video 2", items[1].Media.Title)
	assert.Equal(t, int64(3600), items[1].Media.Duration)
}

func TestGetPlaylist_EmptyPlaylist(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel with no items
	channel := createTestChannel(t, repos, "Test Channel")

	// Get playlist
	items, err := service.GetPlaylist(ctx, channel.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestGetPlaylist_ChannelNotFound(t *testing.T) {
	service, _, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get playlist for non-existent channel
	fakeChannelID := uuid.New()
	items, err := service.GetPlaylist(ctx, fakeChannelID)

	// Assert error
	assert.Error(t, err)
	assert.Nil(t, items)
	assert.True(t, IsChannelNotFound(err))
}

func TestCalculateDuration_Success(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media with known durations
	channel := createTestChannel(t, repos, "Test Channel")
	media1 := createTestMedia(t, repos, "Video 1", 1800) // 30 minutes
	media2 := createTestMedia(t, repos, "Video 2", 3600) // 60 minutes
	media3 := createTestMedia(t, repos, "Video 3", 2700) // 45 minutes

	// Add items to playlist
	_, err := service.AddToPlaylist(ctx, channel.ID, media1.ID, 0)
	require.NoError(t, err)
	_, err = service.AddToPlaylist(ctx, channel.ID, media2.ID, 1)
	require.NoError(t, err)
	_, err = service.AddToPlaylist(ctx, channel.ID, media3.ID, 2)
	require.NoError(t, err)

	// Get playlist items
	items, err := service.GetPlaylist(ctx, channel.ID)
	require.NoError(t, err)

	// Calculate duration
	totalDuration := service.CalculateDuration(items)

	// Verify total (30 + 60 + 45 = 135 minutes = 8100 seconds)
	assert.Equal(t, int64(8100), totalDuration)
}

func TestCalculateDuration_EmptyPlaylist(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel with no items
	channel := createTestChannel(t, repos, "Test Channel")

	// Get empty playlist
	items, err := service.GetPlaylist(ctx, channel.ID)
	require.NoError(t, err)

	// Calculate duration
	totalDuration := service.CalculateDuration(items)

	// Should return 0 for empty playlist
	assert.Equal(t, int64(0), totalDuration)
}

func TestGetPlaylist_ChannelNotFound_PreventsDurationCalculation(t *testing.T) {
	service, _, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get playlist for non-existent channel
	fakeChannelID := uuid.New()
	items, err := service.GetPlaylist(ctx, fakeChannelID)

	// Assert error from GetPlaylist
	assert.Error(t, err)
	assert.Nil(t, items)
	assert.True(t, IsChannelNotFound(err))
}

func TestAddToPlaylist_DatabaseError(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media
	channel := createTestChannel(t, repos, "Test Channel")
	media := createTestMedia(t, repos, "Video", 1800)

	// Add first item successfully
	_, err := service.AddToPlaylist(ctx, channel.ID, media.ID, 0)
	require.NoError(t, err)

	// Close the database connection to force failure
	sqlDB, err := service.db.GetSQLDB()
	require.NoError(t, err)
	_ = sqlDB.Close()

	// Try to add item - should fail with database error
	_, err = service.AddToPlaylist(ctx, channel.ID, media.ID, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add media to playlist")
}

func TestRemoveFromPlaylist_DatabaseError(t *testing.T) {
	service, repos, cleanup := setupPlaylistTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test channel and media
	channel := createTestChannel(t, repos, "Test Channel")
	media := createTestMedia(t, repos, "Video", 1800)

	// Add item successfully
	item, err := service.AddToPlaylist(ctx, channel.ID, media.ID, 0)
	require.NoError(t, err)

	// Close the database connection to force failure
	sqlDB, err := service.db.GetSQLDB()
	require.NoError(t, err)
	_ = sqlDB.Close()

	// Try to remove item - should fail with database error
	err = service.RemoveFromPlaylist(ctx, item.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove from playlist")
}
