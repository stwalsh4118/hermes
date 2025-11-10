// Package api provides HTTP handlers for the REST API endpoints.
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
	"github.com/stwalsh4118/hermes/internal/streaming"
)

// streamManager defines the interface required by StreamHandler for stream management
type streamManager interface {
	StartStream(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error)
	RegisterClient(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error)
	UnregisterClient(ctx context.Context, channelID uuid.UUID) error
	GetStream(channelID uuid.UUID) (*models.StreamSession, bool)
}

// validQualities defines the allowed quality levels for streaming
var validQualities = map[string]bool{
	"1080p": true,
	"720p":  true,
	"480p":  true,
}

// rewriteSegmentPaths modifies playlist content to include quality directory in segment paths
// Converts "1080p_segment_000.ts" to "1080p/1080p_segment_000.ts"
func rewriteSegmentPaths(content, quality string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder

	for _, line := range lines {
		// Check if line is a segment reference (ends with .ts or .vtt and doesn't start with #)
		trimmedLine := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmedLine, "#") &&
			(strings.HasSuffix(trimmedLine, ".ts") || strings.HasSuffix(trimmedLine, ".vtt")) &&
			len(trimmedLine) > 0 {
			// Prepend quality directory to segment filename
			result.WriteString(quality + "/" + line)
		} else {
			result.WriteString(line)
		}
		result.WriteString("\n")
	}

	return result.String()
}

// StreamHandler handles streaming-related API requests
type StreamHandler struct {
	streamManager streamManager
}

// NewStreamHandler creates a new stream handler instance
func NewStreamHandler(manager *streaming.StreamManager) *StreamHandler {
	return &StreamHandler{
		streamManager: manager,
	}
}

// GetMasterPlaylist handles GET /stream/:channel_id/master.m3u8
// This endpoint serves the master playlist and registers the client with the stream
func (h *StreamHandler) GetMasterPlaylist(c *gin.Context) {
	channelIDStr := c.Param("channel_id")
	sessionID := c.Query("session_id") // Get unique client session ID

	// Validate UUID
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	// Validate session ID
	if sessionID == "" {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Msg("Master playlist request missing session_id")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_session_id",
			Message: "Session ID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Str("session_id", sessionID).
		Str("client_ip", c.ClientIP()).
		Msg("Client requesting master playlist")

	// Start stream if not already active (but don't register client yet)
	session, found := h.streamManager.GetStream(channelID)
	if !found {
		// Stream doesn't exist, start it
		var err error
		session, err = h.streamManager.StartStream(ctx, channelID)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("channel_id", channelID.String()).
				Msg("Failed to start stream")

			// Map errors to appropriate HTTP status codes
			if errors.Is(err, streaming.ErrStreamNotFound) {
				c.JSON(http.StatusNotFound, ErrorResponse{
					Error:   "channel_not_found",
					Message: "Channel not found",
				})
				return
			}

			if errors.Is(err, streaming.ErrManagerStopped) {
				c.JSON(http.StatusServiceUnavailable, ErrorResponse{
					Error:   "service_unavailable",
					Message: "Streaming service is unavailable",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "stream_failed",
				Message: "Failed to start stream",
			})
			return
		}
	}

	// Get output directory from session
	outputDir := session.GetOutputDir()
	if outputDir == "" {
		logger.Log.Error().
			Str("channel_id", channelID.String()).
			Msg("Session output directory not set")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "stream_error",
			Message: "Stream configuration error",
		})
		return
	}

	// Build path to master playlist
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")

	// Check if file exists - return 503 if not ready yet (don't register client)
	if _, err := os.Stat(masterPlaylistPath); os.IsNotExist(err) {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Str("path", masterPlaylistPath).
			Msg("Master playlist not yet generated")

		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "stream_starting",
			Message: "Stream is starting, please retry in a moment",
		})
		return
	}

	// Read the master playlist file
	content, err := os.ReadFile(masterPlaylistPath)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("path", masterPlaylistPath).
			Msg("Failed to read master playlist")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "read_failed",
			Message: "Failed to read master playlist",
		})
		return
	}

	// NOW register the client (only on successful playlist delivery)
	// This prevents counting retries and failed attempts
	// Use session ID to ensure idempotent registration
	wasNew := session.RegisterSession(sessionID)
	if wasNew {
		session.IncrementClients()
		logger.Log.Debug().
			Str("channel_id", channelID.String()).
			Str("session_id", sessionID).
			Int("client_count", session.GetClientCount()).
			Msg("New client session registered")
	} else {
		logger.Log.Debug().
			Str("channel_id", channelID.String()).
			Str("session_id", sessionID).
			Int("client_count", session.GetClientCount()).
			Msg("Existing client session reconnected")
	}
	session.UpdateLastAccess()

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Int("client_count", session.GetClientCount()).
		Msg("Serving master playlist")

	// Set appropriate headers
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "no-cache") // Don't cache to ensure client registration on each page load

	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", content)
}

// GetMediaPlaylist handles GET /stream/:channel_id/:quality
// This endpoint serves quality-specific media playlists and updates client access time
// The quality parameter should include .m3u8 extension (e.g., "1080p.m3u8")
func (h *StreamHandler) GetMediaPlaylist(c *gin.Context) {
	channelIDStr := c.Param("channel_id")
	quality := c.Param("quality")

	// Remove .m3u8 extension if present
	quality = strings.TrimSuffix(quality, ".m3u8")

	// Validate UUID
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	// Validate quality parameter
	if !validQualities[quality] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_quality",
			Message: "Quality must be 1080p, 720p, or 480p",
		})
		return
	}

	// Get stream session
	session, found := h.streamManager.GetStream(channelID)
	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "stream_not_found",
			Message: "Stream not found or not active",
		})
		return
	}

	// Update last access time only if there are active clients
	// This prevents lingering HLS requests from keeping idle streams alive
	if session.GetClientCount() > 0 {
		session.UpdateLastAccess()
	}

	// Get output directory from session
	outputDir := session.GetOutputDir()
	if outputDir == "" {
		logger.Log.Error().
			Str("channel_id", channelID.String()).
			Msg("Session output directory not set")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "stream_error",
			Message: "Stream configuration error",
		})
		return
	}

	// Build path to quality-specific directory and playlist
	qualityDir := filepath.Join(outputDir, quality)
	playlistPath := filepath.Join(qualityDir, fmt.Sprintf("%s.m3u8", quality))

	// Check if file exists
	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Str("quality", quality).
			Str("path", playlistPath).
			Msg("Media playlist not yet generated")

		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "playlist_not_ready",
			Message: "Playlist not yet available, please retry",
		})
		return
	}

	// Read the media playlist file
	content, err := os.ReadFile(playlistPath)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("quality", quality).
			Str("path", playlistPath).
			Msg("Failed to read media playlist")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "read_failed",
			Message: "Failed to read media playlist",
		})
		return
	}

	// Rewrite segment paths to include quality directory
	// FFmpeg generates segments as "1080p_segment_000.ts" but we need "1080p/1080p_segment_000.ts"
	// to match our route structure /:channel_id/:quality/:segment
	modifiedContent := rewriteSegmentPaths(string(content), quality)

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Str("quality", quality).
		Msg("Serving media playlist")

	// Set appropriate headers - NO caching for live playlists
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")

	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", []byte(modifiedContent))
}

// GetSegment handles GET /stream/:channel_id/:quality/:segment
// This endpoint serves video segment files and updates client access time
func (h *StreamHandler) GetSegment(c *gin.Context) {
	channelIDStr := c.Param("channel_id")
	quality := c.Param("quality")
	segment := c.Param("segment")

	// Validate UUID
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	// Validate quality parameter
	if !validQualities[quality] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_quality",
			Message: "Quality must be 1080p, 720p, or 480p",
		})
		return
	}

	// Validate segment filename (must be .ts file)
	if !strings.HasSuffix(segment, ".ts") {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_segment",
			Message: "Segment must be a .ts file",
		})
		return
	}

	// Security: Prevent directory traversal attacks
	if strings.Contains(segment, "..") || strings.Contains(segment, "/") || strings.Contains(segment, "\\") {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Str("segment", segment).
			Msg("Directory traversal attempt detected")

		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_segment",
			Message: "Invalid segment filename",
		})
		return
	}

	// Get stream session
	session, found := h.streamManager.GetStream(channelID)
	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "stream_not_found",
			Message: "Stream not found or not active",
		})
		return
	}

	// Update last access time only if there are active clients
	// This prevents lingering HLS requests from keeping idle streams alive
	if session.GetClientCount() > 0 {
		session.UpdateLastAccess()
	}

	// Get output directory from session
	outputDir := session.GetOutputDir()
	if outputDir == "" {
		logger.Log.Error().
			Str("channel_id", channelID.String()).
			Msg("Session output directory not set")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "stream_error",
			Message: "Stream configuration error",
		})
		return
	}

	// Build path to segment file
	qualityDir := filepath.Join(outputDir, quality)
	segmentPath := filepath.Join(qualityDir, segment)

	// Security: Verify the resolved path is still within the expected directory
	absQualityDir, err := filepath.Abs(qualityDir)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("path", qualityDir).
			Msg("Failed to resolve quality directory path")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "path_error",
			Message: "Failed to validate segment path",
		})
		return
	}

	absSegmentPath, err := filepath.Abs(segmentPath)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("segment", segment).
			Msg("Failed to resolve segment path")

		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_path",
			Message: "Invalid segment path",
		})
		return
	}

	if !strings.HasPrefix(absSegmentPath, absQualityDir) {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Str("segment", segment).
			Msg("Path traversal attempt blocked")

		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_path",
			Message: "Invalid segment path",
		})
		return
	}

	// Check if file exists
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		logger.Log.Debug().
			Str("channel_id", channelID.String()).
			Str("quality", quality).
			Str("segment", segment).
			Msg("Segment not found")

		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "segment_not_found",
			Message: "Segment not found",
		})
		return
	}

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Str("quality", quality).
		Str("segment", segment).
		Msg("Serving video segment")

	// Set appropriate headers
	c.Header("Content-Type", "video/MP2T")
	c.Header("Cache-Control", "public, max-age=31536000, immutable") // Segments never change

	// Serve the file
	c.File(segmentPath)
}

// UnregisterClient handles DELETE /stream/:channel_id/client
// This endpoint allows clients to explicitly unregister from a stream
func (h *StreamHandler) UnregisterClient(c *gin.Context) {
	channelIDStr := c.Param("channel_id")
	sessionID := c.Query("session_id") // Get unique client session ID

	// Validate UUID
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	// Validate session ID
	if sessionID == "" {
		logger.Log.Warn().
			Str("channel_id", channelID.String()).
			Msg("Unregister request missing session_id")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_session_id",
			Message: "Session ID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Str("session_id", sessionID).
		Str("client_ip", c.ClientIP()).
		Msg("Client unregistering from stream")

	// Get stream session
	session, found := h.streamManager.GetStream(channelID)
	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "stream_not_found",
			Message: "Stream not found or already stopped",
		})
		return
	}

	// Unregister session
	wasRegistered := session.UnregisterSession(sessionID)
	if wasRegistered {
		// Decrement client count
		err = h.streamManager.UnregisterClient(ctx, channelID)
	} else {
		logger.Log.Debug().
			Str("channel_id", channelID.String()).
			Str("session_id", sessionID).
			Msg("Session was not registered, skipping unregister")
		// No error, just not registered
		err = nil
	}
	if err != nil {
		if errors.Is(err, streaming.ErrStreamNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "stream_not_found",
				Message: "Stream not found or already stopped",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to unregister client")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "unregister_failed",
			Message: "Failed to unregister client",
		})
		return
	}

	logger.Log.Debug().
		Str("channel_id", channelID.String()).
		Msg("Client unregistered successfully")

	c.JSON(http.StatusOK, DeleteResponse{
		Message: "Client unregistered successfully",
	})
}

// SetupStreamRoutes registers streaming-related routes
func SetupStreamRoutes(apiGroup *gin.RouterGroup, manager *streaming.StreamManager) {
	handler := NewStreamHandler(manager)

	// Create stream route group
	streamGroup := apiGroup.Group("/stream")

	// HLS streaming endpoints - order matters for Gin routing
	streamGroup.GET("/:channel_id/master.m3u8", handler.GetMasterPlaylist)
	streamGroup.DELETE("/:channel_id/client", handler.UnregisterClient)
	// More specific route (3 segments) must come before less specific (2 segments)
	streamGroup.GET("/:channel_id/:quality/:segment", handler.GetSegment)
	streamGroup.GET("/:channel_id/:quality", handler.GetMediaPlaylist)
}
