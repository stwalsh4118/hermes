package channel

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/models"
)

// setupTestService creates a service with a test database
func setupTestService(t *testing.T) (*ChannelService, *db.DB, func()) {
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
	service := NewChannelService(repos)

	cleanup := func() {
		_ = database.Close()
	}

	return service, database, cleanup
}

func TestNewChannelService(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	assert.NotNil(t, service)
	assert.NotNil(t, service.repos)
}

func TestCreateChannel_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	name := "Test Channel"
	icon := "icon.png"
	startTime := time.Now().UTC()
	loop := true

	channel, err := service.CreateChannel(ctx, name, &icon, startTime, loop)

	require.NoError(t, err)
	assert.NotNil(t, channel)
	assert.NotEqual(t, uuid.Nil, channel.ID)
	assert.Equal(t, name, channel.Name)
	assert.Equal(t, &icon, channel.Icon)
	assert.True(t, channel.StartTime.Equal(startTime.UTC()))
	assert.Equal(t, loop, channel.Loop)
	assert.False(t, channel.CreatedAt.IsZero())
	assert.False(t, channel.UpdatedAt.IsZero())
}

func TestCreateChannel_DuplicateName(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	name := "Duplicate Channel"
	startTime := time.Now().UTC()

	// Create first channel
	_, err := service.CreateChannel(ctx, name, nil, startTime, true)
	require.NoError(t, err)

	// Try to create second channel with same name
	_, err = service.CreateChannel(ctx, name, nil, startTime, false)

	require.Error(t, err)
	assert.True(t, IsDuplicateName(err))
}

func TestCreateChannel_DuplicateNameCaseInsensitive(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	startTime := time.Now().UTC()

	// Create first channel
	_, err := service.CreateChannel(ctx, "Test Channel", nil, startTime, true)
	require.NoError(t, err)

	// Try to create second channel with different case
	_, err = service.CreateChannel(ctx, "test channel", nil, startTime, false)

	require.Error(t, err)
	assert.True(t, IsDuplicateName(err))
}

func TestCreateChannel_InvalidStartTime(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	name := "Future Channel"
	// Start time more than 1 year in the future
	startTime := time.Now().UTC().Add(400 * 24 * time.Hour)

	_, err := service.CreateChannel(ctx, name, nil, startTime, true)

	require.Error(t, err)
	assert.True(t, IsInvalidStartTime(err))
}

func TestCreateChannel_StartTimeExactlyOneYear(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	name := "One Year Future"
	// Exactly 1 year from now (should be valid)
	startTime := time.Now().UTC().Add(365 * 24 * time.Hour)

	channel, err := service.CreateChannel(ctx, name, nil, startTime, true)

	// This should succeed as it's at the boundary
	require.NoError(t, err)
	assert.NotNil(t, channel)
}

func TestGetByID_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a channel
	created, err := service.CreateChannel(ctx, "Test Channel", nil, time.Now().UTC(), true)
	require.NoError(t, err)

	// Get the channel by ID
	retrieved, err := service.GetByID(ctx, created.ID)

	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Name, retrieved.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	nonExistentID := uuid.New()

	_, err := service.GetByID(ctx, nonExistentID)

	require.Error(t, err)
	assert.True(t, IsChannelNotFound(err))
}

func TestList_Empty(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	channels, err := service.List(ctx)

	require.NoError(t, err)
	assert.NotNil(t, channels)
	assert.Equal(t, 0, len(channels))
}

func TestList_MultipleChannels(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	startTime := time.Now().UTC()

	// Create multiple channels
	names := []string{"Channel 1", "Channel 2", "Channel 3"}
	for _, name := range names {
		_, err := service.CreateChannel(ctx, name, nil, startTime, true)
		require.NoError(t, err)
	}

	// List all channels
	channels, err := service.List(ctx)

	require.NoError(t, err)
	assert.Equal(t, 3, len(channels))

	// Verify names exist (order may vary)
	channelNames := make(map[string]bool)
	for _, ch := range channels {
		channelNames[ch.Name] = true
	}
	for _, name := range names {
		assert.True(t, channelNames[name], "Expected channel %s to be in list", name)
	}
}

func TestUpdateChannel_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a channel
	channel, err := service.CreateChannel(ctx, "Original Name", nil, time.Now().UTC(), true)
	require.NoError(t, err)

	// Update the channel
	newIcon := "new-icon.png"
	channel.Name = "Updated Name"
	channel.Icon = &newIcon
	channel.Loop = false

	err = service.UpdateChannel(ctx, channel)

	require.NoError(t, err)

	// Retrieve and verify
	updated, err := service.GetByID(ctx, channel.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, &newIcon, updated.Icon)
	assert.False(t, updated.Loop)
}

func TestUpdateChannel_DuplicateName(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	startTime := time.Now().UTC()

	// Create two channels
	_, err := service.CreateChannel(ctx, "Channel 1", nil, startTime, true)
	require.NoError(t, err)

	channel2, err := service.CreateChannel(ctx, "Channel 2", nil, startTime, true)
	require.NoError(t, err)

	// Try to update channel2 to have the same name as channel1
	channel2.Name = "Channel 1"
	err = service.UpdateChannel(ctx, channel2)

	require.Error(t, err)
	assert.True(t, IsDuplicateName(err))
}

func TestUpdateChannel_SameNameAllowed(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a channel
	channel, err := service.CreateChannel(ctx, "Test Channel", nil, time.Now().UTC(), true)
	require.NoError(t, err)

	// Update with same name (should be allowed)
	channel.Loop = false
	err = service.UpdateChannel(ctx, channel)

	require.NoError(t, err)
}

func TestUpdateChannel_InvalidStartTime(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a channel
	channel, err := service.CreateChannel(ctx, "Test Channel", nil, time.Now().UTC(), true)
	require.NoError(t, err)

	// Try to update with invalid start time
	channel.StartTime = time.Now().UTC().Add(400 * 24 * time.Hour)
	err = service.UpdateChannel(ctx, channel)

	require.Error(t, err)
	assert.True(t, IsInvalidStartTime(err))
}

func TestUpdateChannel_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to update non-existent channel
	channel := &models.Channel{
		ID:        uuid.New(),
		Name:      "Non-existent",
		StartTime: time.Now().UTC(),
		Loop:      true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := service.UpdateChannel(ctx, channel)

	require.Error(t, err)
	assert.True(t, IsChannelNotFound(err))
}

func TestDeleteChannel_Success(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a channel
	channel, err := service.CreateChannel(ctx, "To Delete", nil, time.Now().UTC(), true)
	require.NoError(t, err)

	// Delete the channel
	err = service.DeleteChannel(ctx, channel.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = service.GetByID(ctx, channel.ID)
	require.Error(t, err)
	assert.True(t, IsChannelNotFound(err))
}

func TestDeleteChannel_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	nonExistentID := uuid.New()

	err := service.DeleteChannel(ctx, nonExistentID)

	require.Error(t, err)
	assert.True(t, IsChannelNotFound(err))
}

func TestHasEmptyPlaylist_Empty(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create a channel
	channel, err := service.CreateChannel(ctx, "Empty Playlist", nil, time.Now().UTC(), true)
	require.NoError(t, err)

	// Check if playlist is empty
	isEmpty, err := service.HasEmptyPlaylist(ctx, channel.ID)

	require.NoError(t, err)
	assert.True(t, isEmpty)
}

func TestHasEmptyPlaylist_WithItems(t *testing.T) {
	service, database, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	repos := db.NewRepositories(database)

	// Create a channel
	channel, err := service.CreateChannel(ctx, "With Playlist", nil, time.Now().UTC(), true)
	require.NoError(t, err)

	// Create a media item
	media := &models.Media{
		ID:       uuid.New(),
		FilePath: "/test/video.mp4",
		Title:    "Test Video",
		Duration: 3600,
	}
	err = repos.Media.Create(ctx, media)
	require.NoError(t, err)

	// Add playlist item
	playlistItem := &models.PlaylistItem{
		ID:        uuid.New(),
		ChannelID: channel.ID,
		MediaID:   media.ID,
		Position:  0,
		CreatedAt: time.Now().UTC(),
	}
	err = repos.PlaylistItems.Create(ctx, playlistItem)
	require.NoError(t, err)

	// Check if playlist is empty
	isEmpty, err := service.HasEmptyPlaylist(ctx, channel.ID)

	require.NoError(t, err)
	assert.False(t, isEmpty)
}

func TestValidateNameUniqueness_WithWhitespace(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	startTime := time.Now().UTC()

	// Create channel with name
	_, err := service.CreateChannel(ctx, "Test Channel", nil, startTime, true)
	require.NoError(t, err)

	// Try to create channel with extra whitespace (should fail)
	_, err = service.CreateChannel(ctx, "  Test Channel  ", nil, startTime, false)

	require.Error(t, err)
	assert.True(t, IsDuplicateName(err))
}
