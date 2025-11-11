//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stwalsh4118/hermes/internal/api"
	"github.com/stwalsh4118/hermes/internal/channel"
	"github.com/stwalsh4118/hermes/internal/models"
	"github.com/stwalsh4118/hermes/internal/timeline"
)

func TestChannelAPI(t *testing.T) {
	// Setup test database
	database, repos, cleanup := setupTestDB(t)
	defer cleanup()

	channelService := channel.NewChannelService(repos)
	playlistService := channel.NewPlaylistService(database, repos)
	timelineService := timeline.NewTimelineService(repos)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupChannelRoutes(apiGroup, channelService, playlistService, timelineService)

	t.Run("CreateChannel_Success", func(t *testing.T) {
		startTime := time.Now().Add(-24 * time.Hour)
		reqBody := map[string]interface{}{
			"name":       "Comedy Central",
			"start_time": startTime.Format(time.RFC3339),
			"loop":       true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response api.ChannelResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.ID)
		assert.Equal(t, "Comedy Central", response.Name)
		assert.Equal(t, true, response.Loop)
		assert.NotEmpty(t, response.CreatedAt)
	})

	t.Run("CreateChannel_DuplicateName", func(t *testing.T) {
		// Create first channel
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		_, err := channelService.CreateChannel(ctx, "Drama Channel", nil, startTime, true)
		require.NoError(t, err)

		// Try to create duplicate
		reqBody := map[string]interface{}{
			"name":       "Drama Channel",
			"start_time": startTime.Format(time.RFC3339),
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response api.ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "duplicate_name", response.Error)
	})

	t.Run("CreateChannel_InvalidStartTime", func(t *testing.T) {
		// Start time more than 1 year in future
		startTime := time.Now().Add(400 * 24 * time.Hour)
		reqBody := map[string]interface{}{
			"name":       "Future Channel",
			"start_time": startTime.Format(time.RFC3339),
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_start_time", response.Error)
	})

	t.Run("CreateChannel_InvalidBody", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Test Channel",
			// Missing start_time
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ListChannels", func(t *testing.T) {
		// Create test channels
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		_, err := channelService.CreateChannel(ctx, "Channel 1", nil, startTime, true)
		require.NoError(t, err)
		_, err = channelService.CreateChannel(ctx, "Channel 2", nil, startTime, false)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ChannelListResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have at least 2 channels (may have more from previous tests)
		assert.GreaterOrEqual(t, len(response.Channels), 2)
	})

	t.Run("GetChannel_Success", func(t *testing.T) {
		// Create test channel
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		icon := "icon.png"
		ch, err := channelService.CreateChannel(ctx, "Get Test Channel", &icon, startTime, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/channels/"+ch.ID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ChannelResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, ch.ID.String(), response.ID)
		assert.Equal(t, "Get Test Channel", response.Name)
		assert.NotNil(t, response.Icon)
		assert.Equal(t, "icon.png", *response.Icon)
		assert.Equal(t, true, response.Loop)
	})

	t.Run("GetChannel_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/channels/"+nonExistentID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "not_found", response.Error)
	})

	t.Run("GetChannel_InvalidUUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/channels/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_id", response.Error)
	})

	t.Run("UpdateChannel_Success", func(t *testing.T) {
		// Create test channel
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		ch, err := channelService.CreateChannel(ctx, "Update Test Channel", nil, startTime, true)
		require.NoError(t, err)

		// Update the channel
		newName := "Updated Channel Name"
		newLoop := false
		reqBody := map[string]interface{}{
			"name": newName,
			"loop": newLoop,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/channels/"+ch.ID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ChannelResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, newName, response.Name)
		assert.Equal(t, newLoop, response.Loop)
	})

	t.Run("UpdateChannel_DuplicateName", func(t *testing.T) {
		// Create two channels
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		_, err := channelService.CreateChannel(ctx, "Existing Channel", nil, startTime, true)
		require.NoError(t, err)
		ch2, err := channelService.CreateChannel(ctx, "Another Channel", nil, startTime, true)
		require.NoError(t, err)

		// Try to update ch2 to have the first channel's name
		reqBody := map[string]interface{}{
			"name": "Existing Channel",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/channels/"+ch2.ID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response api.ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "duplicate_name", response.Error)
	})

	t.Run("UpdateChannel_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		reqBody := map[string]interface{}{
			"name": "New Name",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/channels/"+nonExistentID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DeleteChannel_Success", func(t *testing.T) {
		// Create test channel
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		ch, err := channelService.CreateChannel(ctx, "Delete Test Channel", nil, startTime, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+ch.ID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.DeleteResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Channel deleted successfully", response.Message)

		// Verify channel is deleted
		_, err = channelService.GetByID(ctx, ch.ID)
		assert.Error(t, err)
	})

	t.Run("DeleteChannel_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+nonExistentID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetCurrentProgram_NotImplemented", func(t *testing.T) {
		// Create test channel
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		ch, err := channelService.CreateChannel(ctx, "Current Program Test", nil, startTime, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/channels/"+ch.ID.String()+"/current", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should return 501 Not Implemented
		assert.Equal(t, http.StatusNotImplemented, w.Code)

		var response api.ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "not_implemented", response.Error)
	})

	t.Run("DeleteChannel_WithPlaylistItems", func(t *testing.T) {
		// Create channel and add playlist items
		ctx := context.Background()
		startTime := time.Now().Add(-24 * time.Hour)
		ch, err := channelService.CreateChannel(ctx, "Delete With Playlist", nil, startTime, true)
		require.NoError(t, err)

		// Create media item
		media := models.NewMedia("/test/video.mp4", "Test Video", 3600)
		err = repos.Media.Create(ctx, media)
		require.NoError(t, err)

		// Add to playlist
		playlistItem := models.NewPlaylistItem(ch.ID, media.ID, 0)
		err = repos.PlaylistItems.Create(ctx, playlistItem)
		require.NoError(t, err)

		// Delete channel
		req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+ch.ID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify channel is deleted
		_, err = channelService.GetByID(ctx, ch.ID)
		assert.Error(t, err)

		// Verify playlist items are cascade deleted
		items, err := repos.PlaylistItems.GetByChannelID(ctx, ch.ID)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestPlaylistAPI(t *testing.T) {
	// Setup test database
	database, repos, cleanup := setupTestDB(t)
	defer cleanup()

	channelService := channel.NewChannelService(repos)
	playlistService := channel.NewPlaylistService(database, repos)
	timelineService := timeline.NewTimelineService(repos)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupChannelRoutes(apiGroup, channelService, playlistService, timelineService)

	// Create test channel and media for use across tests
	ctx := context.Background()
	startTime := time.Now().Add(-24 * time.Hour)
	testChannel, err := channelService.CreateChannel(ctx, "Test Playlist Channel", nil, startTime, true)
	require.NoError(t, err)

	testMedia1 := models.NewMedia("/test/video1.mp4", "Test Video 1", 3600)
	err = repos.Media.Create(ctx, testMedia1)
	require.NoError(t, err)

	testMedia2 := models.NewMedia("/test/video2.mp4", "Test Video 2", 2400)
	err = repos.Media.Create(ctx, testMedia2)
	require.NoError(t, err)

	t.Run("GetPlaylist_EmptyPlaylist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/channels/"+testChannel.ID.String()+"/playlist", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.PlaylistResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Empty(t, response.Items)
		assert.Equal(t, int64(0), response.TotalDuration)
	})

	t.Run("GetPlaylist_ChannelNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/channels/"+nonExistentID.String()+"/playlist", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "not_found", response.Error)
	})

	t.Run("AddToPlaylist_Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"media_id": testMedia1.ID.String(),
			"position": 0,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels/"+testChannel.ID.String()+"/playlist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response api.PlaylistItemResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.ID)
		assert.Equal(t, testChannel.ID.String(), response.ChannelID)
		assert.Equal(t, testMedia1.ID.String(), response.MediaID)
		assert.Equal(t, 0, response.Position)
	})

	t.Run("AddToPlaylist_MediaNotFound", func(t *testing.T) {
		nonExistentMediaID := uuid.New()

		reqBody := map[string]interface{}{
			"media_id": nonExistentMediaID.String(),
			"position": 0,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels/"+testChannel.ID.String()+"/playlist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "media_not_found", response.Error)
	})

	t.Run("AddToPlaylist_InvalidMediaID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"media_id": "invalid-uuid",
			"position": 0,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels/"+testChannel.ID.String()+"/playlist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_media_id", response.Error)
	})

	t.Run("AddToPlaylist_ChannelNotFound", func(t *testing.T) {
		nonExistentChannelID := uuid.New()

		reqBody := map[string]interface{}{
			"media_id": testMedia1.ID.String(),
			"position": 0,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/channels/"+nonExistentChannelID.String()+"/playlist", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "channel_not_found", response.Error)
	})

	t.Run("GetPlaylist_WithItems", func(t *testing.T) {
		// Create a new channel for this test
		ch, err := channelService.CreateChannel(ctx, "Playlist With Items", nil, startTime, true)
		require.NoError(t, err)

		// Add items to playlist
		item1, err := playlistService.AddToPlaylist(ctx, ch.ID, testMedia1.ID, 0)
		require.NoError(t, err)
		_, err = playlistService.AddToPlaylist(ctx, ch.ID, testMedia2.ID, 1)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/channels/"+ch.ID.String()+"/playlist", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.PlaylistResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Items, 2)
		assert.Equal(t, item1.ID.String(), response.Items[0].ID)
		assert.Equal(t, 0, response.Items[0].Position)
		assert.Equal(t, 1, response.Items[1].Position)
		assert.Equal(t, int64(6000), response.TotalDuration) // 3600 + 2400
	})

	t.Run("RemoveFromPlaylist_Success", func(t *testing.T) {
		// Create a new channel for this test
		ch, err := channelService.CreateChannel(ctx, "Remove From Playlist", nil, startTime, true)
		require.NoError(t, err)

		// Add items
		item1, err := playlistService.AddToPlaylist(ctx, ch.ID, testMedia1.ID, 0)
		require.NoError(t, err)
		item2, err := playlistService.AddToPlaylist(ctx, ch.ID, testMedia2.ID, 1)
		require.NoError(t, err)

		// Remove first item
		req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+ch.ID.String()+"/playlist/"+item1.ID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.DeleteResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Playlist item removed successfully", response.Message)

		// Verify item is removed and positions reordered
		items, err := playlistService.GetPlaylist(ctx, ch.ID)
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, item2.ID, items[0].ID)
		assert.Equal(t, 0, items[0].Position) // Should be reordered to position 0
	})

	t.Run("RemoveFromPlaylist_ItemNotFound", func(t *testing.T) {
		nonExistentItemID := uuid.New()

		req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+testChannel.ID.String()+"/playlist/"+nonExistentItemID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "not_found", response.Error)
	})

	t.Run("RemoveFromPlaylist_InvalidItemID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+testChannel.ID.String()+"/playlist/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_item_id", response.Error)
	})

	t.Run("ReorderPlaylist_Success", func(t *testing.T) {
		// Create a new channel for this test
		ch, err := channelService.CreateChannel(ctx, "Reorder Playlist", nil, startTime, true)
		require.NoError(t, err)

		// Add items
		item1, err := playlistService.AddToPlaylist(ctx, ch.ID, testMedia1.ID, 0)
		require.NoError(t, err)
		item2, err := playlistService.AddToPlaylist(ctx, ch.ID, testMedia2.ID, 1)
		require.NoError(t, err)

		// Reorder: swap positions
		reqBody := map[string]interface{}{
			"items": []map[string]interface{}{
				{"item_id": item1.ID.String(), "position": 1},
				{"item_id": item2.ID.String(), "position": 0},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/channels/"+ch.ID.String()+"/playlist/reorder", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.DeleteResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Playlist reordered successfully", response.Message)

		// Verify reordering
		items, err := playlistService.GetPlaylist(ctx, ch.ID)
		require.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, item2.ID, items[0].ID)
		assert.Equal(t, 0, items[0].Position)
		assert.Equal(t, item1.ID, items[1].ID)
		assert.Equal(t, 1, items[1].Position)
	})

	t.Run("ReorderPlaylist_InvalidItemID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"items": []map[string]interface{}{
				{"item_id": "invalid-uuid", "position": 0},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/channels/"+testChannel.ID.String()+"/playlist/reorder", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_item_id", response.Error)
	})

	t.Run("ReorderPlaylist_EmptyItems", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"items": []map[string]interface{}{},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/api/channels/"+testChannel.ID.String()+"/playlist/reorder", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_request", response.Error)
	})
}
