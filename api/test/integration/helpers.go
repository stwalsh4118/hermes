//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/hermes/internal/api"
	"github.com/stwalsh4118/hermes/internal/db"
	"github.com/stwalsh4118/hermes/internal/media"
	"github.com/stwalsh4118/hermes/internal/models"
)

// setupTestDB creates an in-memory test database with migrations applied
func setupTestDB(t *testing.T) (*db.DB, *db.Repositories, func()) {
	t.Helper()

	// Create in-memory database
	database, err := db.New(":memory:")
	require.NoError(t, err, "Failed to create in-memory database")

	// Run migrations
	sqlDB, err := database.GetSQLDB()
	require.NoError(t, err, "Failed to get SQL DB")

	// Get absolute path to migrations directory relative to this file
	// This ensures tests work regardless of working directory
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "Failed to get current file path")

	testDir := filepath.Dir(filename)                    // api/test/integration
	apiDir := filepath.Dir(filepath.Dir(testDir))        // api
	migrationsDir := filepath.Join(apiDir, "migrations") // api/migrations
	migrationsPath := "file://" + migrationsDir

	err = db.RunMigrations(sqlDB, migrationsPath)
	require.NoError(t, err, "Failed to run migrations")

	// Create repositories
	repos := db.NewRepositories(database)

	// Cleanup function
	cleanup := func() {
		database.Close()
	}

	return database, repos, cleanup
}

// setupTestRouter creates a test Gin router with media routes configured
func setupTestRouter(scanner *media.Scanner, repos *db.Repositories) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add recovery middleware to catch panics in tests
	router.Use(gin.Recovery())

	apiGroup := router.Group("/api")
	api.SetupMediaRoutes(apiGroup, scanner, repos)

	return router
}

// createTestMediaInDB creates a media item directly in the database for testing
func createTestMediaInDB(t *testing.T, repos *db.Repositories, filePath, title string, showName *string, season, episode *int) *models.Media {
	t.Helper()

	mediaItem := models.NewMedia(filePath, title, 3600) // 1 hour duration
	mediaItem.ShowName = showName
	mediaItem.Season = season
	mediaItem.Episode = episode

	// Set metadata
	videoCodec := "h264"
	audioCodec := "aac"
	resolution := "1920x1080"
	fileSize := int64(1073741824) // 1GB

	mediaItem.VideoCodec = &videoCodec
	mediaItem.AudioCodec = &audioCodec
	mediaItem.Resolution = &resolution
	mediaItem.FileSize = &fileSize

	ctx := context.Background()
	err := repos.Media.Create(ctx, mediaItem)
	require.NoError(t, err, "Failed to create test media in database")

	return mediaItem
}

// createTempVideoFiles creates a temporary directory with dummy video files
// Returns the directory path and a cleanup function
func createTempVideoFiles(t *testing.T, count int) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "media-test-*")
	require.NoError(t, err, "Failed to create temp directory")

	// Create dummy video files with various extensions
	extensions := []string{".mp4", ".mkv", ".avi", ".mov"}

	for i := 0; i < count; i++ {
		ext := extensions[i%len(extensions)]
		filename := filepath.Join(tmpDir, "video_"+uuid.New().String()+ext)

		// Create empty file (FFprobe will fail on these, which is expected for error testing)
		// For tests that need valid metadata, we'll mock FFprobe
		err := os.WriteFile(filename, []byte("dummy video content"), 0600)
		require.NoError(t, err, "Failed to create dummy video file: %s", filename)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createTempVideoFile creates a single temporary video file
func createTempVideoFile(t *testing.T, extension string) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "media-test-*")
	require.NoError(t, err, "Failed to create temp directory")

	filename := filepath.Join(tmpDir, "video_"+uuid.New().String()+extension)
	err = os.WriteFile(filename, []byte("dummy video content"), 0600)
	require.NoError(t, err, "Failed to create dummy video file")

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return filename, cleanup
}

// waitForScanCompletion polls the scanner until the scan completes or fails
func waitForScanCompletion(t *testing.T, scanner *media.Scanner, scanID string, maxAttempts int) *media.ScanProgress {
	t.Helper()

	for i := 0; i < maxAttempts; i++ {
		progress, err := scanner.GetScanProgress(scanID)
		require.NoError(t, err, "Failed to get scan progress")

		if progress.Status == media.ScanStatusCompleted ||
			progress.Status == media.ScanStatusFailed ||
			progress.Status == media.ScanStatusCancelled {
			return progress
		}

		// Small delay between polls (10ms)
		// Tests should complete quickly with dummy files
		if i < maxAttempts-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	t.Fatalf("Scan did not complete after %d attempts", maxAttempts)
	return nil
}

// createMultipleTestMedia creates multiple media items with different shows
func createMultipleTestMedia(t *testing.T, repos *db.Repositories, count int, showName string) []*models.Media {
	t.Helper()

	mediaItems := make([]*models.Media, count)
	for i := 0; i < count; i++ {
		// Build path without leading separator to avoid gocritic warning
		filePath := filepath.Join("test", showName, uuid.New().String()+".mp4")
		// Prepend separator for absolute path
		filePath = "/" + filePath
		title := showName + " - Episode " + uuid.New().String()[:8]
		season := i/10 + 1
		episode := i%10 + 1

		mediaItems[i] = createTestMediaInDB(t, repos, filePath, title, &showName, &season, &episode)
	}

	return mediaItems
}
