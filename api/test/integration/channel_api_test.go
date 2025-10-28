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
)

func TestChannelAPI(t *testing.T) {
	// Setup test database
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	channelService := channel.NewChannelService(repos)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	api.SetupChannelRoutes(apiGroup, channelService)

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

