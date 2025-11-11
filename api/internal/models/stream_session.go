package models

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// StreamState is imported from streaming package to avoid circular dependency
// It represents the current state of a streaming session
type StreamState interface {
	String() string
	IsValid() bool
	CanTransitionTo(newState interface{}) bool
}

// StreamQuality contains information about a quality variant for adaptive streaming
type StreamQuality struct {
	Level        string `json:"level"`         // Quality level: "1080p", "720p", "480p"
	Bitrate      int    `json:"bitrate"`       // Video bitrate in kbps
	Resolution   string `json:"resolution"`    // Resolution: "1920x1080"
	SegmentPath  string `json:"segment_path"`  // Path to segments for this quality
	PlaylistPath string `json:"playlist_path"` // Path to .m3u8 playlist file
}

// BatchState tracks the state of a batch segment generation
type BatchState struct {
	BatchNumber       int       `json:"batch_number"`       // Current batch number (0, 1, 2, ...)
	StartSegment      int       `json:"start_segment"`      // First segment number in batch
	EndSegment        int       `json:"end_segment"`        // Last segment number in batch
	VideoSourcePath   string    `json:"video_source_path"`  // Media file being encoded
	VideoStartOffset  int64     `json:"video_start_offset"` // Starting position in source video (seconds)
	GenerationStarted time.Time `json:"generation_started"` // When batch generation began
	GenerationEnded   time.Time `json:"generation_ended"`   // When batch generation completed (zero value = not complete)
	IsComplete        bool      `json:"is_complete"`        // Whether batch finished generating
}

// ClientPosition tracks the playback position of a single client
type ClientPosition struct {
	SessionID     string    `json:"session_id"`     // Client session identifier
	SegmentNumber int       `json:"segment_number"` // Current segment being played
	Quality       string    `json:"quality"`        // Quality level (1080p, 720p, 480p)
	LastUpdated   time.Time `json:"last_updated"`   // When position was last updated
}

// StreamSession represents an active streaming session
// This is NOT persisted to database, only kept in memory
type StreamSession struct {
	ID                  uuid.UUID                  `json:"id"`
	ChannelID           uuid.UUID                  `json:"channel_id"`
	StartedAt           time.Time                  `json:"started_at"`
	ClientCount         int                        `json:"client_count"`
	FFmpegPID           int                        `json:"ffmpeg_pid"`
	State               string                     `json:"state"`                 // Current stream state (stored as string to avoid import cycle)
	Qualities           []StreamQuality            `json:"qualities"`             // Quality variants being generated
	LastAccessTime      time.Time                  `json:"last_access_time"`      // When last client interacted
	ErrorCount          int                        `json:"error_count"`           // Number of errors encountered
	LastError           string                     `json:"last_error"`            // Most recent error message
	SegmentPath         string                     `json:"segment_path"`          // Directory where segments are stored
	OutputDir           string                     `json:"output_dir"`            // Base directory for stream output
	RestartCount        int                        `json:"restart_count"`         // Number of restart attempts
	HardwareAccelFailed bool                       `json:"hardware_accel_failed"` // Whether hardware acceleration has failed
	RegisteredSessions  map[string]bool            `json:"registered_sessions"`   // Track unique client sessions for idempotent registration
	CurrentBatch        *BatchState                `json:"current_batch"`         // Current batch state (nil = no batch)
	ClientPositions     map[string]*ClientPosition `json:"client_positions"`      // Per-session client positions (key: session_id)
	FurthestSegment     int                        `json:"furthest_segment"`      // Furthest segment any client has reached
	mu                  sync.RWMutex
}

// NewStreamSession creates a new stream session
func NewStreamSession(channelID uuid.UUID) *StreamSession {
	now := time.Now().UTC()
	return &StreamSession{
		ID:                  uuid.New(),
		ChannelID:           channelID,
		StartedAt:           now,
		ClientCount:         0,
		FFmpegPID:           0,
		State:               "idle", // Start in idle state
		Qualities:           make([]StreamQuality, 0),
		LastAccessTime:      now,
		ErrorCount:          0,
		LastError:           "",
		SegmentPath:         "",
		OutputDir:           "",
		RestartCount:        0,
		HardwareAccelFailed: false,
		RegisteredSessions:  make(map[string]bool),
		CurrentBatch:        nil,
		ClientPositions:     make(map[string]*ClientPosition),
		FurthestSegment:     0,
	}
}

// IncrementClients increases the client count
func (s *StreamSession) IncrementClients() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ClientCount++
}

// DecrementClients decreases the client count
func (s *StreamSession) DecrementClients() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ClientCount > 0 {
		s.ClientCount--
	}
}

// GetClientCount returns the current client count (thread-safe)
func (s *StreamSession) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ClientCount
}

// SetFFmpegPID sets the FFmpeg process ID (thread-safe)
func (s *StreamSession) SetFFmpegPID(pid int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FFmpegPID = pid
}

// GetFFmpegPID returns the FFmpeg process ID (thread-safe)
func (s *StreamSession) GetFFmpegPID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.FFmpegPID
}

// IsActive returns true if session has active clients
func (s *StreamSession) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ClientCount > 0
}

// GetState returns the current stream state (thread-safe)
func (s *StreamSession) GetState() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// SetState sets the stream state (thread-safe)
// Note: State validation should be done by caller using streaming.StreamState type
func (s *StreamSession) SetState(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
}

// SetQualities sets the quality variants for this stream (thread-safe)
func (s *StreamSession) SetQualities(qualities []StreamQuality) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Qualities = qualities
}

// GetQualities returns the quality variants for this stream (thread-safe)
func (s *StreamSession) GetQualities() []StreamQuality {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to prevent external modification
	qualities := make([]StreamQuality, len(s.Qualities))
	copy(qualities, s.Qualities)
	return qualities
}

// IncrementErrorCount increments the error counter (thread-safe)
func (s *StreamSession) IncrementErrorCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
}

// GetErrorCount returns the current error count (thread-safe)
func (s *StreamSession) GetErrorCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ErrorCount
}

// SetLastError sets the most recent error message (thread-safe)
func (s *StreamSession) SetLastError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.LastError = err.Error()
	} else {
		s.LastError = ""
	}
}

// GetLastError returns the most recent error message (thread-safe)
func (s *StreamSession) GetLastError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError
}

// ResetErrors resets error count and clears last error (thread-safe)
func (s *StreamSession) ResetErrors() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount = 0
	s.LastError = ""
}

// UpdateLastAccess updates the last access time to now (thread-safe)
func (s *StreamSession) UpdateLastAccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastAccessTime = time.Now().UTC()
}

// GetLastAccessTime returns the last access time (thread-safe)
func (s *StreamSession) GetLastAccessTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastAccessTime
}

// IdleDuration returns the time since last access (thread-safe)
func (s *StreamSession) IdleDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastAccessTime)
}

// ShouldCleanup returns true if the session should be cleaned up
// Requires both zero clients AND idle duration exceeding grace period (thread-safe)
func (s *StreamSession) ShouldCleanup(gracePeriod time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ClientCount == 0 && time.Since(s.LastAccessTime) > gracePeriod
}

// GetSegmentPath returns the segment path (thread-safe)
func (s *StreamSession) GetSegmentPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SegmentPath
}

// SetSegmentPath sets the segment path (thread-safe)
func (s *StreamSession) SetSegmentPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SegmentPath = path
}

// GetOutputDir returns the output directory (thread-safe)
func (s *StreamSession) GetOutputDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.OutputDir
}

// SetOutputDir sets the output directory (thread-safe)
func (s *StreamSession) SetOutputDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OutputDir = dir
}

// GetRestartCount returns the restart count (thread-safe)
func (s *StreamSession) GetRestartCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RestartCount
}

// IncrementRestartCount increments the restart counter (thread-safe)
func (s *StreamSession) IncrementRestartCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RestartCount++
}

// ResetRestartCount resets the restart counter (thread-safe)
func (s *StreamSession) ResetRestartCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RestartCount = 0
}

// GetHardwareAccelFailed returns whether hardware acceleration has failed (thread-safe)
func (s *StreamSession) GetHardwareAccelFailed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.HardwareAccelFailed
}

// SetHardwareAccelFailed sets whether hardware acceleration has failed (thread-safe)
func (s *StreamSession) SetHardwareAccelFailed(failed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.HardwareAccelFailed = failed
}

// RegisterSession registers a client session ID and returns true if it's a new session (thread-safe)
func (s *StreamSession) RegisterSession(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.RegisteredSessions[sessionID] {
		// Session already registered
		return false
	}

	// New session
	s.RegisteredSessions[sessionID] = true
	return true
}

// UnregisterSession unregisters a client session ID and returns true if it was registered (thread-safe)
func (s *StreamSession) UnregisterSession(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.RegisteredSessions[sessionID] {
		// Session was not registered
		return false
	}

	// Remove session
	delete(s.RegisteredSessions, sessionID)
	return true
}

// UpdateClientPosition updates or creates a client position entry and updates FurthestSegment (thread-safe)
func (s *StreamSession) UpdateClientPosition(sessionID string, segment int, quality string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ClientPositions[sessionID] = &ClientPosition{
		SessionID:     sessionID,
		SegmentNumber: segment,
		Quality:       quality,
		LastUpdated:   time.Now().UTC(),
	}

	// Track furthest position across all clients
	if segment > s.FurthestSegment {
		s.FurthestSegment = segment
	}
}

// GetFurthestPosition returns the furthest segment any client has reached (thread-safe)
func (s *StreamSession) GetFurthestPosition() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.FurthestSegment
}

// ShouldGenerateNextBatch returns true if the next batch should be generated (thread-safe)
// Returns false if no batch exists, batch is not complete, or segments remaining > threshold
func (s *StreamSession) ShouldGenerateNextBatch(threshold int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.CurrentBatch == nil || !s.CurrentBatch.IsComplete {
		return false // Wait for current batch to finish
	}

	segmentsRemaining := s.CurrentBatch.EndSegment - s.FurthestSegment
	return segmentsRemaining <= threshold
}

// SetCurrentBatch sets the current batch state (thread-safe)
func (s *StreamSession) SetCurrentBatch(batch *BatchState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentBatch = batch
}

// GetCurrentBatch returns the current batch state (thread-safe)
// May return nil if no batch is set
func (s *StreamSession) GetCurrentBatch() *BatchState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentBatch
}

// UpdateBatchCompletion updates the batch completion state (thread-safe)
func (s *StreamSession) UpdateBatchCompletion(generationEnded time.Time, isComplete bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.CurrentBatch != nil {
		s.CurrentBatch.GenerationEnded = generationEnded
		s.CurrentBatch.IsComplete = isComplete
	}
}
