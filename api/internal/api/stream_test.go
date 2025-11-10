package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/models"
	"github.com/stwalsh4118/hermes/internal/streaming"
)

// mockStreamManager is a test helper that implements streamManager interface
type mockStreamManager struct {
	startStreamFunc      func(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error)
	registerClientFunc   func(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error)
	unregisterClientFunc func(ctx context.Context, channelID uuid.UUID) error
	getStreamFunc        func(channelID uuid.UUID) (*models.StreamSession, bool)
}

func (m *mockStreamManager) StartStream(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error) {
	if m.startStreamFunc != nil {
		return m.startStreamFunc(ctx, channelID)
	}
	return nil, nil
}

func (m *mockStreamManager) RegisterClient(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error) {
	if m.registerClientFunc != nil {
		return m.registerClientFunc(ctx, channelID)
	}
	return nil, nil
}

func (m *mockStreamManager) UnregisterClient(ctx context.Context, channelID uuid.UUID) error {
	if m.unregisterClientFunc != nil {
		return m.unregisterClientFunc(ctx, channelID)
	}
	return nil
}

func (m *mockStreamManager) GetStream(channelID uuid.UUID) (*models.StreamSession, bool) {
	if m.getStreamFunc != nil {
		return m.getStreamFunc(channelID)
	}
	return nil, false
}

// setupStreamTestRouter creates a test Gin router with stream routes
func setupStreamTestRouter(manager *mockStreamManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a StreamHandler with the mock
	handler := &StreamHandler{
		streamManager: manager,
	}

	apiGroup := router.Group("/api")
	streamGroup := apiGroup.Group("/stream")

	streamGroup.GET("/:channel_id/master.m3u8", handler.GetMasterPlaylist)
	streamGroup.DELETE("/:channel_id/client", handler.UnregisterClient)
	streamGroup.POST("/:channel_id/position", handler.UpdatePosition)
	streamGroup.GET("/:channel_id/:quality/:segment", handler.GetSegment)
	streamGroup.GET("/:channel_id/:quality", handler.GetMediaPlaylist)

	return router
}

// createTestFiles creates temporary test files for streaming
func createTestFiles(t *testing.T, outputDir string) {
	t.Helper()

	// Create master playlist
	masterContent := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=5192000,RESOLUTION=1920x1080
1080p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=3192000,RESOLUTION=1280x720
720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1692000,RESOLUTION=854x480
480p.m3u8
`
	err := os.WriteFile(filepath.Join(outputDir, "master.m3u8"), []byte(masterContent), 0644)
	require.NoError(t, err)

	// Create quality directories and playlists
	qualities := []string{"1080p", "720p", "480p"}
	for _, quality := range qualities {
		qualityDir := filepath.Join(outputDir, quality)
		err := os.MkdirAll(qualityDir, 0755)
		require.NoError(t, err)

		playlistContent := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:EVENT
#EXTINF:6.0,
%s_segment_000.ts
#EXTINF:6.0,
%s_segment_001.ts
`, quality, quality)
		playlistPath := filepath.Join(qualityDir, fmt.Sprintf("%s.m3u8", quality))
		err = os.WriteFile(playlistPath, []byte(playlistContent), 0644)
		require.NoError(t, err)

		// Create dummy segment files
		segmentPath := filepath.Join(qualityDir, fmt.Sprintf("%s_segment_000.ts", quality))
		err = os.WriteFile(segmentPath, []byte("dummy video data"), 0644)
		require.NoError(t, err)
	}
}

func TestGetMasterPlaylist_Success(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "stream-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	session.SetOutputDir(tmpDir)

	// Create test files
	createTestFiles(t, tmpDir)

	// Setup mock manager
	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			// Return not found so it calls StartStream
			return nil, false
		},
		startStreamFunc: func(_ context.Context, id uuid.UUID) (*models.StreamSession, error) {
			if id == channelID {
				return session, nil
			}
			return nil, streaming.ErrStreamNotFound
		},
	}

	router := setupStreamTestRouter(mockManager)

	// Make request with session_id
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/master.m3u8?session_id=test-session", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/vnd.apple.mpegurl", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "no-cache")
	assert.Contains(t, w.Body.String(), "#EXTM3U")
	assert.Contains(t, w.Body.String(), "1080p.m3u8")
}

func TestGetMasterPlaylist_InvalidUUID(t *testing.T) {
	mockManager := &mockStreamManager{}
	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, "/api/stream/invalid-uuid/master.m3u8", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_id", response.Error)
}

func TestGetMasterPlaylist_ChannelNotFound(t *testing.T) {
	channelID := uuid.New()

	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			return nil, false
		},
		startStreamFunc: func(_ context.Context, _ uuid.UUID) (*models.StreamSession, error) {
			return nil, streaming.ErrStreamNotFound
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/master.m3u8?session_id=test-session", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "channel_not_found", response.Error)
}

func TestGetMasterPlaylist_StreamStarting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stream-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	session.SetOutputDir(tmpDir)
	// Don't create master playlist file - simulating stream starting

	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			return nil, false
		},
		startStreamFunc: func(_ context.Context, _ uuid.UUID) (*models.StreamSession, error) {
			return session, nil
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/master.m3u8?session_id=test-session", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "stream_starting", response.Error)
}

func TestGetMediaPlaylist_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stream-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	session.SetOutputDir(tmpDir)

	createTestFiles(t, tmpDir)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	// Test each quality
	qualities := []string{"1080p", "720p", "480p"}
	for _, quality := range qualities {
		t.Run(quality, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/%s.m3u8", channelID.String(), quality), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/vnd.apple.mpegurl", w.Header().Get("Content-Type"))
			assert.Contains(t, w.Header().Get("Cache-Control"), "no-cache")
			assert.Contains(t, w.Body.String(), "#EXTM3U")
			assert.Contains(t, w.Body.String(), quality+"_segment_000.ts")
		})
	}
}

func TestGetMediaPlaylist_InvalidQuality(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)

	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			return session, true
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/4K.m3u8", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_quality", response.Error)
}

func TestGetMediaPlaylist_StreamNotFound(t *testing.T) {
	channelID := uuid.New()

	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/1080p.m3u8", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "stream_not_found", response.Error)
}

func TestGetSegment_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stream-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	session.SetOutputDir(tmpDir)

	createTestFiles(t, tmpDir)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/1080p/1080p_segment_000.ts", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "video/MP2T", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "public")
	assert.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	assert.Equal(t, "dummy video data", w.Body.String())
}

func TestGetSegment_InvalidSegmentExtension(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)

	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			return session, true
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/1080p/segment.mp4", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_segment", response.Error)
}

func TestGetSegment_DirectoryTraversalAttempt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stream-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	session.SetOutputDir(tmpDir)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	// Test directory traversal patterns that contain ".."
	// These should be caught by validation and return 400
	simplePatterns := []string{
		"..segment.ts",
		"segment..ts",
		"seg..ment.ts",
	}

	for _, pattern := range simplePatterns {
		t.Run(pattern, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/1080p/%s", channelID.String(), pattern), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "invalid_segment", response.Error)
		})
	}
}

func TestGetSegment_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stream-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	session.SetOutputDir(tmpDir)

	// Create quality directory but no segment
	qualityDir := filepath.Join(tmpDir, "1080p")
	err = os.MkdirAll(qualityDir, 0755)
	require.NoError(t, err)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stream/%s/1080p/missing_segment_999.ts", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "segment_not_found", response.Error)
}

func TestUnregisterClient_Success(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	// Register the session ID first
	session.RegisterSession("test-session")
	session.IncrementClients()
	called := false

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
		unregisterClientFunc: func(_ context.Context, id uuid.UUID) error {
			if id == channelID {
				called = true
				return nil
			}
			return streaming.ErrStreamNotFound
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/stream/%s/client?session_id=test-session", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "UnregisterClient should have been called")

	var response DeleteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response.Message, "unregistered successfully")
}

func TestUnregisterClient_InvalidUUID(t *testing.T) {
	mockManager := &mockStreamManager{}
	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodDelete, "/api/stream/invalid-uuid/client", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_id", response.Error)
}

func TestUnregisterClient_NotFound(t *testing.T) {
	channelID := uuid.New()

	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/stream/%s/client?session_id=test-session", channelID.String()), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "stream_not_found", response.Error)
}

func TestUpdatePosition_Success(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	
	// Set up a batch state for testing
	batch := &models.BatchState{
		BatchNumber:  2,
		StartSegment: 20,
		EndSegment:   39,
		IsComplete:   true,
	}
	session.SetCurrentBatch(batch)
	
	// Set initial furthest segment
	session.UpdateClientPosition("existing-session", 25, "1080p")

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	// Create request body
	requestBody := `{
		"session_id": "test-session-123",
		"segment_number": 30,
		"quality": "1080p",
		"timestamp": "2025-10-31T12:34:56Z"
	}`

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["acknowledged"])
	assert.Equal(t, float64(2), response["current_batch"]) // JSON numbers are float64
	assert.Equal(t, float64(9), response["segments_remaining"]) // 39 - 30 = 9

	// Verify position was updated
	furthestSegment := session.GetFurthestPosition()
	assert.Equal(t, 30, furthestSegment) // Should be updated to 30
}

func TestUpdatePosition_Success_NoBatch(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	// No batch set yet (stream just starting)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	requestBody := `{
		"session_id": "test-session-123",
		"segment_number": 5,
		"quality": "720p"
	}`

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["acknowledged"])
	assert.Equal(t, float64(0), response["current_batch"])
	assert.Equal(t, float64(0), response["segments_remaining"])
}

func TestUpdatePosition_InvalidUUID(t *testing.T) {
	mockManager := &mockStreamManager{}
	router := setupStreamTestRouter(mockManager)

	requestBody := `{
		"session_id": "test-session-123",
		"segment_number": 5,
		"quality": "1080p"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/stream/invalid-uuid/position", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_id", response.Error)
}

func TestUpdatePosition_StreamNotFound(t *testing.T) {
	channelID := uuid.New()

	mockManager := &mockStreamManager{
		getStreamFunc: func(_ uuid.UUID) (*models.StreamSession, bool) {
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	requestBody := `{
		"session_id": "test-session-123",
		"segment_number": 5,
		"quality": "1080p"
	}`

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "stream_not_found", response.Error)
}

func TestUpdatePosition_MissingRequiredFields(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	testCases := []struct {
		name        string
		requestBody string
	}{
		{
			name:        "missing session_id",
			requestBody: `{"segment_number": 5, "quality": "1080p"}`,
		},
		{
			name:        "missing segment_number",
			requestBody: `{"session_id": "test-session", "quality": "1080p"}`,
		},
		{
			name:        "missing quality",
			requestBody: `{"session_id": "test-session", "segment_number": 5}`,
		},
		{
			name:        "empty body",
			requestBody: `{}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "invalid_request", response.Error)
		})
	}
}

func TestUpdatePosition_InvalidSegmentNumber(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	// Test negative segment number
	requestBody := `{
		"session_id": "test-session-123",
		"segment_number": -1,
		"quality": "1080p"
	}`

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

func TestUpdatePosition_InvalidQuality(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	testCases := []struct {
		name        string
		quality     string
		expectedErr string
	}{
		{"invalid quality 4K", "4K", "invalid_quality"},
		{"invalid quality 360p", "360p", "invalid_quality"},
		{"invalid quality empty", "", "invalid_request"}, // Empty string fails binding "required" check
		{"invalid quality random", "random", "invalid_quality"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody := fmt.Sprintf(`{
				"session_id": "test-session-123",
				"segment_number": 5,
				"quality": "%s"
			}`, tc.quality)

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), strings.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedErr, response.Error)
		})
	}
}

func TestUpdatePosition_SegmentsRemaining_ClientAhead(t *testing.T) {
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)
	
	// Set up a batch state
	batch := &models.BatchState{
		BatchNumber:  1,
		StartSegment: 0,
		EndSegment:   19,
		IsComplete:   true,
	}
	session.SetCurrentBatch(batch)
	
	// Client is ahead of batch end (edge case)
	session.UpdateClientPosition("test-session", 25, "1080p")

	mockManager := &mockStreamManager{
		getStreamFunc: func(id uuid.UUID) (*models.StreamSession, bool) {
			if id == channelID {
				return session, true
			}
			return nil, false
		},
	}

	router := setupStreamTestRouter(mockManager)

	requestBody := `{
		"session_id": "test-session",
		"segment_number": 30,
		"quality": "1080p"
	}`

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/stream/%s/position", channelID.String()), strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["acknowledged"])
	assert.Equal(t, float64(1), response["current_batch"])
	assert.Equal(t, float64(0), response["segments_remaining"]) // Client ahead, so 0 remaining
}
