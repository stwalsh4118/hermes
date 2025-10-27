package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stwalsh4118/hermes/internal/db"
)

// HealthResponse represents the response from the health check endpoint
type HealthResponse struct {
	Status   string                 `json:"status"`
	Database string                 `json:"database"`
	Time     string                 `json:"time"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// HealthHandler handles health check requests
type HealthHandler struct {
	db *db.DB
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(database *db.DB) *HealthHandler {
	return &HealthHandler{db: database}
}

// Check handles the health check endpoint
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:  "ok",
		Time:    time.Now().UTC().Format(time.RFC3339),
		Details: make(map[string]interface{}),
	}

	// Check database connectivity
	if err := h.db.Health(ctx); err != nil {
		response.Status = "degraded"
		response.Database = "unhealthy"
		response.Details["database_error"] = err.Error()
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	response.Database = "healthy"
	c.JSON(http.StatusOK, response)
}

// SetupHealthRoutes registers health check routes
func SetupHealthRoutes(apiGroup *gin.RouterGroup, database *db.DB) {
	handler := NewHealthHandler(database)
	apiGroup.GET("/health", handler.Check)
}
