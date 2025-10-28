package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/channel"
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

// ChannelHandler handles channel-related API requests
type ChannelHandler struct {
	channelService *channel.ChannelService
}

// NewChannelHandler creates a new channel handler instance
func NewChannelHandler(channelService *channel.ChannelService) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
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

// SetupChannelRoutes registers channel-related routes
func SetupChannelRoutes(apiGroup *gin.RouterGroup, channelService *channel.ChannelService) {
	handler := NewChannelHandler(channelService)

	// Channel CRUD endpoints
	apiGroup.POST("/channels", handler.CreateChannel)
	apiGroup.GET("/channels", handler.ListChannels)
	apiGroup.GET("/channels/:id", handler.GetChannel)
	apiGroup.PUT("/channels/:id", handler.UpdateChannel)
	apiGroup.DELETE("/channels/:id", handler.DeleteChannel)

	// Current program placeholder (PBI 4)
	apiGroup.GET("/channels/:id/current", handler.GetCurrentProgram)
}
