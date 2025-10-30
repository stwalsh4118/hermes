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
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/media"
	"github.com/stwalsh4118/hermes/internal/models"
)

// setupTestDB creates a test database in memory
func setupTestDB(t *testing.T) (*db.DB, *db.Repositories, func()) {
	t.Helper()

	// Create in-memory database
	database, err := db.New(":memory:")
	require.NoError(t, err)

	// Run migrations
	sqlDB, err := database.GetSQLDB()
	require.NoError(t, err)
	err = db.RunMigrations(sqlDB, "file://../../migrations")
	require.NoError(t, err)

	repos := db.NewRepositories(database)

	cleanup := func() {
		_ = database.Close()
	}

	return database, repos, cleanup
}

// setupTestRouter creates a test Gin router with media routes
func setupTestRouter(scanner *media.Scanner, repos *db.Repositories) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	SetupMediaRoutes(apiGroup, scanner, repos)
	return router
}

// createTestMedia creates a media item in the database for testing
func createTestMedia(t *testing.T, repos *db.Repositories) *models.Media {
	t.Helper()

	mediaItem := models.NewMedia("/test/video.mp4", "Test Video", 3600)
	showName := "Test Show"
	season := 1
	episode := 1
	mediaItem.ShowName = &showName
	mediaItem.Season = &season
	mediaItem.Episode = &episode

	ctx := context.Background()
	err := repos.Media.Create(ctx, mediaItem)
	require.NoError(t, err)

	return mediaItem
}

func TestTriggerScan(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	t.Run("Missing path returns error", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/media/scan", bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "missing_path", resp.Error)
	})

	t.Run("Invalid directory returns error", func(t *testing.T) {
		reqBody := ScanRequest{Path: "/nonexistent/directory"}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/media/scan", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid_directory", resp.Error)
	})

	t.Run("Valid directory starts scan", func(t *testing.T) {
		// Create a temporary directory for testing
		tmpDir := t.TempDir()

		reqBody := ScanRequest{Path: tmpDir}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/media/scan", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp ScanResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.ScanID)
		assert.Equal(t, "Scan started", resp.Message)

		// Verify scan ID is valid UUID
		_, err = uuid.Parse(resp.ScanID)
		assert.NoError(t, err)
	})
}

func TestGetScanStatus(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	t.Run("Non-existent scan returns 404", func(t *testing.T) {
		scanID := uuid.New().String()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/media/scan/%s/status", scanID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "scan_not_found", resp.Error)
	})

	t.Run("Existing scan returns progress", func(t *testing.T) {
		// Start a scan
		tmpDir := t.TempDir()
		ctx := context.Background()
		scanID, err := scanner.StartScan(ctx, tmpDir)
		require.NoError(t, err)

		// Wait briefly for scan to initialize
		time.Sleep(100 * time.Millisecond)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/media/scan/%s/status", scanID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var progress media.ScanProgress
		err = json.Unmarshal(w.Body.Bytes(), &progress)
		require.NoError(t, err)
		assert.Equal(t, scanID, progress.ScanID)
		assert.Contains(t, []media.ScanStatus{
			media.ScanStatusRunning,
			media.ScanStatusCompleted,
		}, progress.Status)
	})
}

func TestListMedia(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media
	media1 := createTestMedia(t, repos)
	media2 := models.NewMedia("/test/video2.mp4", "Test Video 2", 7200)
	showName2 := "Another Show"
	media2.ShowName = &showName2
	err := repos.Media.Create(context.Background(), media2)
	require.NoError(t, err)

	t.Run("List all media", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Items), 2)
		assert.Equal(t, 20, resp.Limit) // default limit
		assert.Equal(t, 0, resp.Offset)
	})

	t.Run("List media with pagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media?limit=1&offset=0", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, 1, resp.Limit)
		assert.Equal(t, 0, resp.Offset)
	})

	t.Run("Filter by show name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media?show=Test+Show", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Items), 1)

		// Verify all items have the correct show name
		for _, item := range resp.Items {
			assert.Equal(t, "Test Show", *item.ShowName)
		}
	})

	t.Run("Limit maximum enforced", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media?limit=500", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 500, resp.Limit) // limit accepted as-is since it's under 10000
	})

	t.Run("Limit over 10000 is capped at 10000", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media?limit=15000", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 10000, resp.Limit) // max limit enforced at 10000
	})

	t.Run("Total count reflects all items not just current page", func(t *testing.T) {
		// Create 5 media items
		for i := 0; i < 5; i++ {
			m := models.NewMedia(fmt.Sprintf("/test/video%d.mp4", i+3), fmt.Sprintf("Video %d", i+3), 1800)
			err := repos.Media.Create(context.Background(), m)
			require.NoError(t, err)
		}

		// Request with limit=1, should still get total count of all items
		req := httptest.NewRequest("GET", "/api/media?limit=1&offset=0", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp.Items), "Should return 1 item")
		assert.GreaterOrEqual(t, resp.Total, 7, "Total should be at least 7 (2 from setup + 5 just created)")
		assert.Equal(t, 1, resp.Limit)
		assert.Equal(t, 0, resp.Offset)
	})

	t.Run("Total count for filtered show is correct", func(t *testing.T) {
		// Create 3 more items with same show name
		showName := "Test Show"
		for i := 0; i < 3; i++ {
			m := models.NewMedia(fmt.Sprintf("/test/testshow%d.mp4", i), fmt.Sprintf("Episode %d", i), 1800)
			m.ShowName = &showName
			season := 1
			episode := i + 2
			m.Season = &season
			m.Episode = &episode
			err := repos.Media.Create(context.Background(), m)
			require.NoError(t, err)
		}

		// Request with limit=1 and show filter
		req := httptest.NewRequest("GET", "/api/media?show=Test+Show&limit=1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp.Items), "Should return 1 item")
		assert.Equal(t, 4, resp.Total, "Total should be 4 (1 from setup + 3 just created)")
		assert.Equal(t, 1, resp.Limit)
	})

	_ = media1 // prevent unused variable error

	t.Run("Fetch all media with limit=-1", func(t *testing.T) {
		// Create 25 media items (more than default limit of 20)
		for i := 0; i < 25; i++ {
			m := models.NewMedia(fmt.Sprintf("/test/unlimited%d.mp4", i), fmt.Sprintf("Video %d", i), 1800)
			err := repos.Media.Create(context.Background(), m)
			require.NoError(t, err)
		}

		req := httptest.NewRequest("GET", "/api/media?limit=-1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Items), 25, "Should return all items")
		assert.Equal(t, resp.Total, resp.Limit, "Limit should equal total when fetching all")
	})

	t.Run("Raised maximum limit to 10000", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media?limit=10000", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 10000, resp.Limit, "Should accept 10000 as limit")
	})

	t.Run("Default limit remains 20 for backward compatibility", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 20, resp.Limit, "Default limit should remain 20")
	})

	t.Run("Unlimited fetch with show filter", func(t *testing.T) {
		// Create items with specific show name
		showName := "Unlimited Test Show"
		for i := 0; i < 15; i++ {
			m := models.NewMedia(fmt.Sprintf("/test/showtest%d.mp4", i), fmt.Sprintf("Episode %d", i), 1800)
			m.ShowName = &showName
			err := repos.Media.Create(context.Background(), m)
			require.NoError(t, err)
		}

		req := httptest.NewRequest("GET", "/api/media?limit=-1&show=Unlimited+Test+Show", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MediaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Items), 15, "Should return all items for show")
		assert.Equal(t, resp.Total, resp.Limit, "Limit should equal total when fetching all with filter")
	})
}

func TestGetMedia(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media
	mediaItem := createTestMedia(t, repos)

	t.Run("Get existing media", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp models.Media
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, mediaItem.ID, resp.ID)
		assert.Equal(t, mediaItem.Title, resp.Title)
		assert.Equal(t, *mediaItem.ShowName, *resp.ShowName)
	})

	t.Run("Get non-existent media returns 404", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/media/%s", nonExistentID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "not_found", resp.Error)
	})

	t.Run("Invalid UUID returns 400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/media/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid_id", resp.Error)
	})
}

func TestUpdateMedia(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	// Create test media
	mediaItem := createTestMedia(t, repos)

	const updatedTitle = "Updated Title"

	t.Run("Update media title", func(t *testing.T) {
		newTitle := updatedTitle
		reqBody := UpdateMediaRequest{Title: &newTitle}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp models.Media
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, newTitle, resp.Title)
	})

	t.Run("Update multiple fields", func(t *testing.T) {
		newShowName := "Updated Show"
		newSeason := 2
		newEpisode := 5
		reqBody := UpdateMediaRequest{
			ShowName: &newShowName,
			Season:   &newSeason,
			Episode:  &newEpisode,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp models.Media
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, newShowName, *resp.ShowName)
		assert.Equal(t, newSeason, *resp.Season)
		assert.Equal(t, newEpisode, *resp.Episode)
	})

	t.Run("Update non-existent media returns 404", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		newTitle := updatedTitle
		reqBody := UpdateMediaRequest{Title: &newTitle}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/media/%s", nonExistentID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "not_found", resp.Error)
	})

	t.Run("Invalid UUID returns 400", func(t *testing.T) {
		newTitle := updatedTitle
		reqBody := UpdateMediaRequest{Title: &newTitle}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/media/invalid-uuid", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid_id", resp.Error)
	})

	t.Run("Invalid request body returns 400", func(t *testing.T) {
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid_request", resp.Error)
	})
}

func TestDeleteMedia(t *testing.T) {
	_, repos, cleanup := setupTestDB(t)
	defer cleanup()

	scanner := media.NewScanner(repos)
	defer scanner.Stop()

	router := setupTestRouter(scanner, repos)

	t.Run("Delete existing media", func(t *testing.T) {
		// Create test media
		mediaItem := createTestMedia(t, repos)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/media/%s", mediaItem.ID.String()), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp DeleteResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Media deleted successfully", resp.Message)

		// Verify media is deleted
		_, err = repos.Media.GetByID(context.Background(), mediaItem.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, db.ErrNotFound)
	})

	t.Run("Delete non-existent media returns 404", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/media/%s", nonExistentID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "not_found", resp.Error)
	})

	t.Run("Invalid UUID returns 400", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/media/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid_id", resp.Error)
	})
}
