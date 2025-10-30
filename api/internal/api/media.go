package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/media"
	"github.com/stwalsh4118/hermes/internal/models"
)

// Request/Response DTOs

// ScanRequest represents a request to trigger a media library scan
type ScanRequest struct {
	Path string `json:"path"` // Optional: defaults to config if not provided
}

// ScanResponse represents the response after triggering a scan
type ScanResponse struct {
	ScanID  string `json:"scan_id"`
	Message string `json:"message"`
}

// MediaListResponse represents a paginated list of media items
type MediaListResponse struct {
	Items  []*models.Media `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// UpdateMediaRequest represents a request to update media metadata
type UpdateMediaRequest struct {
	Title    *string `json:"title,omitempty"`
	ShowName *string `json:"show_name,omitempty"`
	Season   *int    `json:"season,omitempty"`
	Episode  *int    `json:"episode,omitempty"`
}

// DeleteResponse represents a successful delete operation
type DeleteResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// MediaHandler handles media-related API requests
type MediaHandler struct {
	scanner *media.Scanner
	repos   *db.Repositories
}

// NewMediaHandler creates a new media handler instance
func NewMediaHandler(scanner *media.Scanner, repos *db.Repositories) *MediaHandler {
	return &MediaHandler{
		scanner: scanner,
		repos:   repos,
	}
}

// TriggerScan handles POST /api/media/scan
func (h *MediaHandler) TriggerScan(c *gin.Context) {
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is acceptable - use default path
		if c.Request.ContentLength > 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_request",
				Message: "Invalid request body",
			})
			return
		}
	}

	// If no path provided, this would use the default from config
	// For now, we require a path to be provided
	if req.Path == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_path",
			Message: "Media library path is required",
		})
		return
	}

	// Use background context for long-running scan operation
	// The scan runs asynchronously and should not be tied to the HTTP request lifecycle
	scanID, err := h.scanner.StartScan(context.Background(), req.Path)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("path", req.Path).
			Msg("Failed to start media scan")

		if errors.Is(err, media.ErrScanAlreadyRunning) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "scan_in_progress",
				Message: "A scan is already running",
			})
			return
		}

		if errors.Is(err, media.ErrInvalidDirectory) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_directory",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "scan_failed",
			Message: "Failed to start media scan",
		})
		return
	}

	logger.Log.Info().
		Str("scan_id", scanID).
		Str("path", req.Path).
		Msg("Media scan started")

	c.JSON(http.StatusCreated, ScanResponse{
		ScanID:  scanID,
		Message: "Scan started",
	})
}

// GetScanStatus handles GET /api/media/scan/:scanId/status
func (h *MediaHandler) GetScanStatus(c *gin.Context) {
	scanID := c.Param("scanId")

	progress, err := h.scanner.GetScanProgress(scanID)
	if err != nil {
		if errors.Is(err, media.ErrScanNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "scan_not_found",
				Message: "Scan not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("scan_id", scanID).
			Msg("Failed to get scan progress")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve scan progress",
		})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// ListMedia handles GET /api/media
func (h *MediaHandler) ListMedia(c *gin.Context) {
	// Parse pagination parameters
	limit := 20 // default
	unlimitedFetch := false

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l == -1 {
				// Special case: fetch all items
				unlimitedFetch = true
				limit = 0 // GORM uses 0 for no limit
			} else if l > 0 {
				limit = l
				if limit > 10000 {
					limit = 10000 // raised max limit for large libraries
				}
			}
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	showName := c.Query("show")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var mediaItems []*models.Media
	var totalCount int64
	var err error

	if showName != "" {
		// Filter by show name
		mediaItems, err = h.repos.Media.ListByShow(ctx, showName, limit, offset)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("show", showName).
				Int("limit", limit).
				Int("offset", offset).
				Msg("Failed to list media by show")

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "query_failed",
				Message: "Failed to retrieve media list",
			})
			return
		}

		// Get total count for the show
		totalCount, err = h.repos.Media.CountByShow(ctx, showName)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("show", showName).
				Msg("Failed to count media by show")

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "query_failed",
				Message: "Failed to retrieve media count",
			})
			return
		}
	} else {
		// List all media
		mediaItems, err = h.repos.Media.List(ctx, limit, offset)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Int("limit", limit).
				Int("offset", offset).
				Msg("Failed to list media")

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "query_failed",
				Message: "Failed to retrieve media list",
			})
			return
		}

		// Get total count of all media
		totalCount, err = h.repos.Media.Count(ctx)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Msg("Failed to count media")

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "query_failed",
				Message: "Failed to retrieve media count",
			})
			return
		}
	}

	// Calculate the limit to return in response
	responseLimit := limit
	if unlimitedFetch {
		responseLimit = int(totalCount)
	}

	c.JSON(http.StatusOK, MediaListResponse{
		Items:  mediaItems,
		Total:  int(totalCount),
		Limit:  responseLimit,
		Offset: offset,
	})
}

// GetMedia handles GET /api/media/:id
func (h *MediaHandler) GetMedia(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid media ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	mediaItem, err := h.repos.Media.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Media not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("id", id.String()).
			Msg("Failed to get media by ID")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve media",
		})
		return
	}

	c.JSON(http.StatusOK, mediaItem)
}

// UpdateMedia handles PUT /api/media/:id
func (h *MediaHandler) UpdateMedia(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid media ID format",
		})
		return
	}

	var req UpdateMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Load existing media
	mediaItem, err := h.repos.Media.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Media not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("id", id.String()).
			Msg("Failed to get media for update")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve media",
		})
		return
	}

	// Apply partial updates
	if req.Title != nil {
		mediaItem.Title = *req.Title
	}
	if req.ShowName != nil {
		mediaItem.ShowName = req.ShowName
	}
	if req.Season != nil {
		mediaItem.Season = req.Season
	}
	if req.Episode != nil {
		mediaItem.Episode = req.Episode
	}

	// Save updates
	if err := h.repos.Media.Update(ctx, mediaItem); err != nil {
		logger.Log.Error().
			Err(err).
			Str("id", id.String()).
			Msg("Failed to update media")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update media",
		})
		return
	}

	logger.Log.Info().
		Str("id", id.String()).
		Msg("Media updated successfully")

	c.JSON(http.StatusOK, mediaItem)
}

// DeleteMedia handles DELETE /api/media/:id
func (h *MediaHandler) DeleteMedia(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid media ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check if media exists first
	_, err = h.repos.Media.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Media not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("id", id.String()).
			Msg("Failed to check media existence")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to check media",
		})
		return
	}

	// Delete the media
	if err := h.repos.Media.Delete(ctx, id); err != nil {
		logger.Log.Error().
			Err(err).
			Str("id", id.String()).
			Msg("Failed to delete media")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete media",
		})
		return
	}

	logger.Log.Info().
		Str("id", id.String()).
		Msg("Media deleted successfully")

	c.JSON(http.StatusOK, DeleteResponse{
		Message: "Media deleted successfully",
	})
}

// SetupMediaRoutes registers media-related routes
func SetupMediaRoutes(apiGroup *gin.RouterGroup, scanner *media.Scanner, repos *db.Repositories) {
	handler := NewMediaHandler(scanner, repos)

	// Scan endpoints
	apiGroup.POST("/media/scan", handler.TriggerScan)
	apiGroup.GET("/media/scan/:scanId/status", handler.GetScanStatus)

	// Media CRUD endpoints
	apiGroup.GET("/media", handler.ListMedia)
	apiGroup.GET("/media/:id", handler.GetMedia)
	apiGroup.PUT("/media/:id", handler.UpdateMedia)
	apiGroup.DELETE("/media/:id", handler.DeleteMedia)
}
