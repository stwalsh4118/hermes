// Package server provides the HTTP server setup and routing configuration.
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stwalsh4118/hermes/internal/api"
	"github.com/stwalsh4118/hermes/internal/channel"
	"github.com/stwalsh4118/hermes/internal/config"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/logger"
	"github.com/stwalsh4118/hermes/internal/media"
	"github.com/stwalsh4118/hermes/internal/middleware"
	"github.com/stwalsh4118/hermes/internal/streaming"
	"github.com/stwalsh4118/hermes/internal/timeline"
)

// Server represents the HTTP server
type Server struct {
	config          *config.Config
	db              *db.DB
	repos           *db.Repositories
	scanner         *media.Scanner
	channelService  *channel.ChannelService
	playlistService *channel.PlaylistService
	timelineService *timeline.TimelineService
	streamManager   *streaming.StreamManager
	router          *gin.Engine
	server          *http.Server
}

// New creates a new server instance
func New(cfg *config.Config, database *db.DB) *Server {
	repos := db.NewRepositories(database)
	scanner := media.NewScanner(repos)
	channelService := channel.NewChannelService(repos)
	playlistService := channel.NewPlaylistService(database, repos)
	timelineService := timeline.NewTimelineService(repos)
	streamManager := streaming.NewStreamManager(repos, timelineService, &cfg.Streaming)

	return &Server{
		config:          cfg,
		db:              database,
		repos:           repos,
		scanner:         scanner,
		channelService:  channelService,
		playlistService: playlistService,
		timelineService: timelineService,
		streamManager:   streamManager,
	}
}

// setupRouter initializes the Gin router with middleware and routes
func (s *Server) setupRouter() {
	// Set Gin mode based on log level
	if s.config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create new Gin router
	s.router = gin.New()

	// Add middleware stack
	s.router.Use(middleware.RequestLogger()) // Custom zerolog request logger
	s.router.Use(gin.Recovery())             // Panic recovery
	s.router.Use(cors.Default())             // CORS support (allows all origins)

	// Create API route group
	apiGroup := s.router.Group("/api")

	// Register service routes
	api.SetupHealthRoutes(apiGroup, s.db)
	api.SetupMediaRoutes(apiGroup, s.scanner, s.repos)
	api.SetupChannelRoutes(apiGroup, s.channelService, s.playlistService, s.timelineService)
	api.SetupStreamRoutes(apiGroup, s.streamManager)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.setupRouter()

	// Start stream manager
	if err := s.streamManager.Start(); err != nil {
		return fmt.Errorf("failed to start stream manager: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	logger.Log.Info().
		Str("host", s.config.Server.Host).
		Int("port", s.config.Server.Port).
		Msg("Starting HTTP server")

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info().Msg("Shutting down server gracefully")

	// Stop the stream manager
	if s.streamManager != nil {
		s.streamManager.Stop()
	}

	// Stop the scanner cleanup goroutine
	if s.scanner != nil {
		s.scanner.Stop()
	}

	// Check if server was started before attempting shutdown
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
	}

	logger.Log.Info().Msg("Server stopped")
	return nil
}
