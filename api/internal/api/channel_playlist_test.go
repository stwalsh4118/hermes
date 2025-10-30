package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/channel"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/models"
)

// setupChannelTestRouter creates a test router with channel routes
func setupChannelTestRouter(database *db.DB, repos *db.Repositories) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")

	channelService := channel.NewChannelService(repos)
	playlistService := channel.NewPlaylistService(database, repos)
	SetupChannelRoutes(apiGroup, channelService, playlistService)

	return router
}

func TestBulkRemoveFromPlaylist(t *testing.T) {
	database, repos, cleanup := setupTestDB(t)
	defer cleanup()

	router := setupChannelTestRouter(database, repos)

	t.Run("Bulk remove multiple items and verify positions", func(t *testing.T) {
		// Create channel
		startTime := time.Now().UTC()
		channel := models.NewChannel("Test Channel", startTime, true)
		err := repos.Channels.Create(context.Background(), channel)
		require.NoError(t, err)

		// Create 5 media items
		mediaItems := make([]*models.Media, 5)
		for i := 0; i < 5; i++ {
			m := models.NewMedia(fmt.Sprintf("/test/video%d.mp4", i), fmt.Sprintf("Test Video %d", i), 1800)
			err := repos.Media.Create(context.Background(), m)
			require.NoError(t, err)
			mediaItems[i] = m
		}

		// Add all 5 media items to playlist
		playlistItems := make([]*models.PlaylistItem, 5)
		for i := 0; i < 5; i++ {
			item := &models.PlaylistItem{
				ID:        uuid.New(),
				ChannelID: channel.ID,
				MediaID:   mediaItems[i].ID,
				Position:  i,
			}
			err := repos.PlaylistItems.Create(context.Background(), item)
			require.NoError(t, err)
			playlistItems[i] = item
		}

		// Remove items at positions 1, 2, and 4 (keep 0 and 3)
		removeIDs := []string{
			playlistItems[1].ID.String(),
			playlistItems[2].ID.String(),
			playlistItems[4].ID.String(),
		}
		reqBody := BulkRemoveFromPlaylistRequest{
			ItemIDs: removeIDs,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/channels/"+channel.ID.String()+"/playlist/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(3), resp["removed"])

		// Verify remaining items have sequential positions (0, 1)
		remaining, err := repos.PlaylistItems.GetByChannelID(context.Background(), channel.ID)
		require.NoError(t, err)
		assert.Len(t, remaining, 2)

		// Check positions are sequential
		assert.Equal(t, 0, remaining[0].Position)
		assert.Equal(t, 1, remaining[1].Position)

		// Check correct media items remain (items 0 and 3)
		assert.Equal(t, mediaItems[0].ID, remaining[0].MediaID)
		assert.Equal(t, mediaItems[3].ID, remaining[1].MediaID)
	})

	t.Run("Bulk remove with empty array", func(t *testing.T) {
		// Create channel
		startTime := time.Now().UTC()
		channel := models.NewChannel("Empty Remove Channel", startTime, true)
		err := repos.Channels.Create(context.Background(), channel)
		require.NoError(t, err)

		reqBody := BulkRemoveFromPlaylistRequest{
			ItemIDs: []string{},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/channels/"+channel.ID.String()+"/playlist/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should fail validation due to min=1 binding
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Bulk remove with invalid channel ID", func(t *testing.T) {
		reqBody := BulkRemoveFromPlaylistRequest{
			ItemIDs: []string{uuid.New().String()},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/channels/invalid-uuid/playlist/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Bulk remove with non-existent item ID", func(t *testing.T) {
		// Create channel
		startTime := time.Now().UTC()
		channel := models.NewChannel("Test Channel", startTime, true)
		err := repos.Channels.Create(context.Background(), channel)
		require.NoError(t, err)

		// Try to remove non-existent item
		reqBody := BulkRemoveFromPlaylistRequest{
			ItemIDs: []string{uuid.New().String()},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/channels/"+channel.ID.String()+"/playlist/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Bulk remove item belonging to different channel", func(t *testing.T) {
		// Create two channels
		startTime := time.Now().UTC()
		channel1 := models.NewChannel("Channel 1", startTime, true)
		err := repos.Channels.Create(context.Background(), channel1)
		require.NoError(t, err)

		channel2 := models.NewChannel("Channel 2", startTime, true)
		err = repos.Channels.Create(context.Background(), channel2)
		require.NoError(t, err)

		// Create media item
		media := models.NewMedia("/test/video.mp4", "Test Video", 1800)
		err = repos.Media.Create(context.Background(), media)
		require.NoError(t, err)

		// Add item to channel1
		item := &models.PlaylistItem{
			ID:        uuid.New(),
			ChannelID: channel1.ID,
			MediaID:   media.ID,
			Position:  0,
		}
		err = repos.PlaylistItems.Create(context.Background(), item)
		require.NoError(t, err)

		// Try to remove it from channel2
		reqBody := BulkRemoveFromPlaylistRequest{
			ItemIDs: []string{item.ID.String()},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/channels/"+channel2.ID.String()+"/playlist/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "bulk_remove_failed", resp.Error)
	})

	t.Run("Bulk remove with invalid item UUID format", func(t *testing.T) {
		// Create channel
		startTime := time.Now().UTC()
		channel := models.NewChannel("Test Channel", startTime, true)
		err := repos.Channels.Create(context.Background(), channel)
		require.NoError(t, err)

		reqBody := BulkRemoveFromPlaylistRequest{
			ItemIDs: []string{"not-a-uuid"},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/channels/"+channel.ID.String()+"/playlist/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid_item_id", resp.Error)
	})

	t.Run("Bulk remove all items leaves empty playlist", func(t *testing.T) {
		// Create channel
		startTime := time.Now().UTC()
		channel := models.NewChannel("Test Channel Empty", startTime, true)
		err := repos.Channels.Create(context.Background(), channel)
		require.NoError(t, err)

		// Create 3 media items and add to playlist
		itemIDs := make([]string, 3)
		for i := 0; i < 3; i++ {
			m := models.NewMedia(fmt.Sprintf("/test/empty-playlist/video%d.mp4", i), fmt.Sprintf("Empty Test Video %d", i), 1800)
			err := repos.Media.Create(context.Background(), m)
			require.NoError(t, err)

			item := &models.PlaylistItem{
				ID:        uuid.New(),
				ChannelID: channel.ID,
				MediaID:   m.ID,
				Position:  i,
			}
			err = repos.PlaylistItems.Create(context.Background(), item)
			require.NoError(t, err)
			itemIDs[i] = item.ID.String()
		}

		// Remove all items
		reqBody := BulkRemoveFromPlaylistRequest{
			ItemIDs: itemIDs,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/channels/"+channel.ID.String()+"/playlist/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(3), resp["removed"])

		// Verify playlist is empty
		remaining, err := repos.PlaylistItems.GetByChannelID(context.Background(), channel.ID)
		require.NoError(t, err)
		assert.Len(t, remaining, 0)
	})
}
