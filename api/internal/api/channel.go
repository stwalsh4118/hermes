package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/channel"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/models"
)

// Request/Response DTOs

// CreateChannelRequest represents a request to create a new channel
type CreateChannelRequest struct {
	Name      string     `json:"name" binding:"required"`
	Icon      *string    `json:"icon,omitempty"`
	StartTime *time.Time `json:"start_time" binding:"required"`
	Loop      *bool      `json:"loop,omitempty"`
}

// UpdateChannelRequest represents a request to update channel metadata (partial update)
type UpdateChannelRequest struct {
	Name      *string    `json:"name,omitempty"`
	Icon      *string    `json:"icon,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	Loop      *bool      `json:"loop,omitempty"`
}

// ChannelResponse represents a channel in API responses
type ChannelResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      *string   `json:"icon,omitempty"`
	StartTime time.Time `json:"start_time"`
	Loop      bool      `json:"loop"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChannelListResponse represents a list of channels
type ChannelListResponse struct {
	Channels []*ChannelResponse `json:"channels"`
}

// Playlist DTOs

// AddToPlaylistRequest represents a request to add media to a playlist
type AddToPlaylistRequest struct {
	MediaID  string `json:"media_id" binding:"required"`
	Position int    `json:"position" binding:"gte=0"`
}

// BulkAddToPlaylistRequest represents a request to add multiple media items to a playlist
type BulkAddToPlaylistRequest struct {
	Items []AddToPlaylistRequest `json:"items" binding:"required,min=1"`
}

// BulkRemoveFromPlaylistRequest represents a request to remove multiple items
type BulkRemoveFromPlaylistRequest struct {
	ItemIDs []string `json:"item_ids" binding:"required,min=1"`
}

// ReorderPlaylistRequest represents a request to reorder playlist items
type ReorderPlaylistRequest struct {
	Items []ReorderItem `json:"items" binding:"required,min=1"`
}

// ReorderItem represents an item position in reorder request
type ReorderItem struct {
	ItemID   string `json:"item_id" binding:"required"`
	Position int    `json:"position" binding:"gte=0"`
}

// PlaylistItemResponse represents a playlist item with embedded media details
type PlaylistItemResponse struct {
	ID        string        `json:"id"`
	ChannelID string        `json:"channel_id"`
	MediaID   string        `json:"media_id"`
	Position  int           `json:"position"`
	CreatedAt time.Time     `json:"created_at"`
	Media     *models.Media `json:"media,omitempty"`
}

// PlaylistResponse represents a channel's playlist
type PlaylistResponse struct {
	Items         []*PlaylistItemResponse `json:"items"`
	TotalDuration int64                   `json:"total_duration_seconds"`
}

// ChannelHandler handles channel-related API requests
type ChannelHandler struct {
	channelService  *channel.ChannelService
	playlistService *channel.PlaylistService
}

// NewChannelHandler creates a new channel handler instance
func NewChannelHandler(channelService *channel.ChannelService, playlistService *channel.PlaylistService) *ChannelHandler {
	return &ChannelHandler{
		channelService:  channelService,
		playlistService: playlistService,
	}
}

// toChannelResponse converts a channel model to API response format
func toChannelResponse(ch *models.Channel) *ChannelResponse {
	return &ChannelResponse{
		ID:        ch.ID.String(),
		Name:      ch.Name,
		Icon:      ch.Icon,
		StartTime: ch.StartTime,
		Loop:      ch.Loop,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}
}

// toPlaylistItemResponse converts a playlist item model to API response format
func toPlaylistItemResponse(item *models.PlaylistItem) *PlaylistItemResponse {
	return &PlaylistItemResponse{
		ID:        item.ID.String(),
		ChannelID: item.ChannelID.String(),
		MediaID:   item.MediaID.String(),
		Position:  item.Position,
		CreatedAt: item.CreatedAt,
		Media:     item.Media,
	}
}

// CreateChannel handles POST /api/channels
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Default loop to true if not specified
	loop := true
	if req.Loop != nil {
		loop = *req.Loop
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Call service to create channel
	newChannel, err := h.channelService.CreateChannel(ctx, req.Name, req.Icon, *req.StartTime, loop)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("name", req.Name).
			Msg("Failed to create channel")

		if errors.Is(err, channel.ErrDuplicateChannelName) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "duplicate_name",
				Message: "A channel with this name already exists",
			})
			return
		}

		if errors.Is(err, channel.ErrInvalidStartTime) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_start_time",
				Message: "Start time cannot be more than 1 year in the future",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "create_failed",
			Message: "Failed to create channel",
		})
		return
	}

	logger.Log.Info().
		Str("channel_id", newChannel.ID.String()).
		Str("name", newChannel.Name).
		Msg("Channel created successfully")

	c.JSON(http.StatusCreated, toChannelResponse(newChannel))
}

// ListChannels handles GET /api/channels
func (h *ChannelHandler) ListChannels(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	channels, err := h.channelService.List(ctx)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to list channels")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve channel list",
		})
		return
	}

	// Convert to response format
	responses := make([]*ChannelResponse, len(channels))
	for i, ch := range channels {
		responses[i] = toChannelResponse(ch)
	}

	c.JSON(http.StatusOK, ChannelListResponse{
		Channels: responses,
	})
}

// GetChannel handles GET /api/channels/:id
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	ch, err := h.channelService.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Channel not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("channel_id", id.String()).
			Msg("Failed to get channel by ID")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve channel",
		})
		return
	}

	c.JSON(http.StatusOK, toChannelResponse(ch))
}

// UpdateChannel handles PUT /api/channels/:id
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Load existing channel
	ch, err := h.channelService.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Channel not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("channel_id", id.String()).
			Msg("Failed to get channel for update")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve channel",
		})
		return
	}

	// Apply partial updates
	if req.Name != nil {
		ch.Name = *req.Name
	}
	if req.Icon != nil {
		ch.Icon = req.Icon
	}
	if req.StartTime != nil {
		ch.StartTime = *req.StartTime
	}
	if req.Loop != nil {
		ch.Loop = *req.Loop
	}

	// Save updates
	if err := h.channelService.UpdateChannel(ctx, ch); err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", id.String()).
			Msg("Failed to update channel")

		if errors.Is(err, channel.ErrDuplicateChannelName) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "duplicate_name",
				Message: "A channel with this name already exists",
			})
			return
		}

		if errors.Is(err, channel.ErrInvalidStartTime) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_start_time",
				Message: "Start time cannot be more than 1 year in the future",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update channel",
		})
		return
	}

	logger.Log.Info().
		Str("channel_id", id.String()).
		Msg("Channel updated successfully")

	c.JSON(http.StatusOK, toChannelResponse(ch))
}

// DeleteChannel handles DELETE /api/channels/:id
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Delete the channel
	if err := h.channelService.DeleteChannel(ctx, id); err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Channel not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("channel_id", id.String()).
			Msg("Failed to delete channel")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete channel",
		})
		return
	}

	logger.Log.Info().
		Str("channel_id", id.String()).
		Msg("Channel deleted successfully")

	c.JSON(http.StatusOK, DeleteResponse{
		Message: "Channel deleted successfully",
	})
}

// GetCurrentProgram handles GET /api/channels/:id/current (Placeholder)
func (h *ChannelHandler) GetCurrentProgram(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Verify channel exists
	_, err = h.channelService.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Channel not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("channel_id", id.String()).
			Msg("Failed to get channel for current program")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve channel",
		})
		return
	}

	// Return placeholder response - full implementation in PBI 4
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Error:   "not_implemented",
		Message: "Current program feature will be implemented in PBI 4",
	})
}

// GetPlaylist handles GET /api/channels/:id/playlist
func (h *ChannelHandler) GetPlaylist(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID
	channelID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Verify channel exists
	_, err = h.channelService.GetByID(ctx, channelID)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Channel not found",
			})
			return
		}

		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to verify channel existence")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve channel",
		})
		return
	}

	// Get playlist items
	items, err := h.playlistService.GetPlaylist(ctx, channelID)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Msg("Failed to get playlist")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve playlist",
		})
		return
	}

	// Calculate total duration from the items we already fetched
	duration := h.playlistService.CalculateDuration(items)

	// Convert to response format
	responses := make([]*PlaylistItemResponse, len(items))
	for i, item := range items {
		responses[i] = toPlaylistItemResponse(item)
	}

	c.JSON(http.StatusOK, PlaylistResponse{
		Items:         responses,
		TotalDuration: duration,
	})
}

// AddToPlaylist handles POST /api/channels/:id/playlist
func (h *ChannelHandler) AddToPlaylist(c *gin.Context) {
	idStr := c.Param("id")

	// Validate channel UUID
	channelID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	var req AddToPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate media UUID
	mediaID, err := uuid.Parse(req.MediaID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_media_id",
			Message: "Invalid media ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Add to playlist via service
	item, err := h.playlistService.AddToPlaylist(ctx, channelID, mediaID, req.Position)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("media_id", mediaID.String()).
			Int("position", req.Position).
			Msg("Failed to add media to playlist")

		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "channel_not_found",
				Message: "Channel not found",
			})
			return
		}

		if errors.Is(err, channel.ErrMediaNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "media_not_found",
				Message: "Media not found",
			})
			return
		}

		if errors.Is(err, channel.ErrInvalidPosition) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_position",
				Message: "Position must be non-negative",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "add_failed",
			Message: "Failed to add media to playlist",
		})
		return
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Str("media_id", mediaID.String()).
		Str("item_id", item.ID.String()).
		Int("position", req.Position).
		Msg("Media added to playlist successfully")

	c.JSON(http.StatusCreated, toPlaylistItemResponse(item))
}

// BulkAddToPlaylist handles POST /api/channels/:id/playlist/bulk
func (h *ChannelHandler) BulkAddToPlaylist(c *gin.Context) {
	idStr := c.Param("id")

	// Validate channel UUID
	channelID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	var req BulkAddToPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Convert request items to service layer format
	bulkItems := make([]channel.BulkAddItem, 0, len(req.Items))
	for _, item := range req.Items {
		// Validate media UUID
		mediaID, err := uuid.Parse(item.MediaID)
		if err != nil {
			logger.Log.Warn().
				Str("channel_id", channelID.String()).
				Str("media_id", item.MediaID).
				Msg("Invalid media ID in bulk add request")
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_media_id",
				Message: fmt.Sprintf("Invalid media ID format: %s", item.MediaID),
			})
			return
		}

		bulkItems = append(bulkItems, channel.BulkAddItem{
			MediaID:  mediaID,
			Position: item.Position,
		})
	}

	// Use bulk add service method - single transaction with batch INSERT
	playlistItems, err := h.playlistService.BulkAddToPlaylist(ctx, channelID, bulkItems)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Int("item_count", len(req.Items)).
			Msg("Failed to bulk add media to playlist")

		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "channel_not_found",
				Message: "Channel not found",
			})
			return
		}

		if errors.Is(err, channel.ErrMediaNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "media_not_found",
				Message: "One or more media items not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "bulk_add_failed",
			Message: "Failed to add items to playlist",
		})
		return
	}

	// Convert to response format
	responses := make([]*PlaylistItemResponse, len(playlistItems))
	for i, item := range playlistItems {
		responses[i] = toPlaylistItemResponse(item)
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Int("item_count", len(responses)).
		Msg("Bulk add to playlist completed successfully")

	c.JSON(http.StatusCreated, gin.H{
		"items":  responses,
		"added":  len(responses),
		"failed": 0,
		"total":  len(responses),
	})
}

// RemoveFromPlaylist handles DELETE /api/channels/:id/playlist/:item_id
func (h *ChannelHandler) RemoveFromPlaylist(c *gin.Context) {
	channelIDStr := c.Param("id")
	itemIDStr := c.Param("item_id")

	// Validate channel UUID
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	// Validate item UUID
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_item_id",
			Message: "Invalid item ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Remove from playlist via service
	err = h.playlistService.RemoveFromPlaylist(ctx, itemID)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("item_id", itemID.String()).
			Msg("Failed to remove item from playlist")

		if errors.Is(err, channel.ErrPlaylistItemNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Playlist item not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "remove_failed",
			Message: "Failed to remove item from playlist",
		})
		return
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Str("item_id", itemID.String()).
		Msg("Item removed from playlist successfully")

	c.JSON(http.StatusOK, DeleteResponse{
		Message: "Playlist item removed successfully",
	})
}

// BulkRemoveFromPlaylist handles DELETE /api/channels/:id/playlist/bulk
func (h *ChannelHandler) BulkRemoveFromPlaylist(c *gin.Context) {
	idStr := c.Param("id")

	// Validate channel UUID
	channelID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	var req BulkRemoveFromPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Convert string IDs to UUIDs
	itemIDs := make([]uuid.UUID, 0, len(req.ItemIDs))
	for _, idStr := range req.ItemIDs {
		itemID, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_item_id",
				Message: fmt.Sprintf("Invalid item ID format: %s", idStr),
			})
			return
		}
		itemIDs = append(itemIDs, itemID)
	}

	// Call service to bulk remove
	err = h.playlistService.BulkRemoveFromPlaylist(ctx, channelID, itemIDs)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Int("item_count", len(itemIDs)).
			Msg("Failed to bulk remove from playlist")

		if errors.Is(err, channel.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "channel_not_found",
				Message: "Channel not found",
			})
			return
		}

		if errors.Is(err, channel.ErrPlaylistItemNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "item_not_found",
				Message: "One or more playlist items not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "bulk_remove_failed",
			Message: "Failed to remove items from playlist",
		})
		return
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Int("item_count", len(itemIDs)).
		Msg("Bulk remove from playlist completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"removed": len(itemIDs),
		"message": "Items removed successfully",
	})
}

// ReorderPlaylist handles PUT /api/channels/:id/playlist/reorder
func (h *ChannelHandler) ReorderPlaylist(c *gin.Context) {
	idStr := c.Param("id")

	// Validate channel UUID
	channelID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid channel ID format",
		})
		return
	}

	var req ReorderPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Convert API DTOs to db.ReorderItem
	reorderItems := make([]db.ReorderItem, len(req.Items))
	for i, item := range req.Items {
		itemID, err := uuid.Parse(item.ItemID)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_item_id",
				Message: fmt.Sprintf("Invalid item ID format at index %d", i),
			})
			return
		}
		reorderItems[i] = db.ReorderItem{
			ID:       itemID,
			Position: item.Position,
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Reorder playlist via service
	err = h.playlistService.ReorderPlaylist(ctx, channelID, reorderItems)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Int("item_count", len(reorderItems)).
			Msg("Failed to reorder playlist")

		if errors.Is(err, channel.ErrPlaylistItemNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "One or more playlist items not found",
			})
			return
		}

		if errors.Is(err, channel.ErrInvalidPosition) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_position",
				Message: "Invalid position values provided",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "reorder_failed",
			Message: "Failed to reorder playlist",
		})
		return
	}

	logger.Log.Info().
		Str("channel_id", channelID.String()).
		Int("item_count", len(reorderItems)).
		Msg("Playlist reordered successfully")

	c.JSON(http.StatusOK, DeleteResponse{
		Message: "Playlist reordered successfully",
	})
}

// SetupChannelRoutes registers channel-related routes
func SetupChannelRoutes(apiGroup *gin.RouterGroup, channelService *channel.ChannelService, playlistService *channel.PlaylistService) {
	handler := NewChannelHandler(channelService, playlistService)

	// Channel CRUD endpoints
	apiGroup.POST("/channels", handler.CreateChannel)
	apiGroup.GET("/channels", handler.ListChannels)
	apiGroup.GET("/channels/:id", handler.GetChannel)
	apiGroup.PUT("/channels/:id", handler.UpdateChannel)
	apiGroup.DELETE("/channels/:id", handler.DeleteChannel)

	// Current program placeholder (PBI 4)
	apiGroup.GET("/channels/:id/current", handler.GetCurrentProgram)

	// Playlist endpoints
	apiGroup.GET("/channels/:id/playlist", handler.GetPlaylist)
	apiGroup.POST("/channels/:id/playlist/bulk", handler.BulkAddToPlaylist)
	apiGroup.POST("/channels/:id/playlist", handler.AddToPlaylist)
	apiGroup.DELETE("/channels/:id/playlist/bulk", handler.BulkRemoveFromPlaylist)
	apiGroup.DELETE("/channels/:id/playlist/:item_id", handler.RemoveFromPlaylist)
	apiGroup.PUT("/channels/:id/playlist/reorder", handler.ReorderPlaylist)
}
