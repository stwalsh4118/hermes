package models

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// StreamSession represents an active streaming session
// This is NOT persisted to database, only kept in memory
type StreamSession struct {
	ID          uuid.UUID `json:"id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	StartedAt   time.Time `json:"started_at"`
	ClientCount int       `json:"client_count"`
	FFmpegPID   int       `json:"ffmpeg_pid"`
	mu          sync.RWMutex
}

// NewStreamSession creates a new stream session
func NewStreamSession(channelID uuid.UUID) *StreamSession {
	return &StreamSession{
		ID:          uuid.New(),
		ChannelID:   channelID,
		StartedAt:   time.Now().UTC(),
		ClientCount: 0,
		FFmpegPID:   0,
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
