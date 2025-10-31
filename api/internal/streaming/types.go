// Package streaming provides streaming-specific types and session management.
package streaming

import (
	"errors"
	"sync"

	"github.com/stwalsh4118/hermes/internal/models"
)

// StreamState represents the current state of a streaming session
type StreamState string

// Stream state constants
const (
	StateIdle     StreamState = "idle"     // No active stream
	StateStarting StreamState = "starting" // FFmpeg process launching
	StateActive   StreamState = "active"   // Stream running, generating segments
	StateStopping StreamState = "stopping" // Graceful shutdown in progress
	StateFailed   StreamState = "failed"   // Stream failed, needs recovery
)

// Common errors
var (
	ErrInvalidStateTransition = errors.New("invalid state transition")
)

// String returns the string representation of the stream state
func (s StreamState) String() string {
	return string(s)
}

// IsValid checks if the stream state is a known valid value
func (s StreamState) IsValid() bool {
	switch s {
	case StateIdle, StateStarting, StateActive, StateStopping, StateFailed:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if a transition from current state to newState is valid
func (s StreamState) CanTransitionTo(newState StreamState) bool {
	// Define valid state transitions
	switch s {
	case StateIdle:
		// From idle, can only start
		return newState == StateStarting
	case StateStarting:
		// From starting, can become active, failed, or stop
		return newState == StateActive || newState == StateFailed || newState == StateStopping
	case StateActive:
		// From active, can stop or fail
		return newState == StateStopping || newState == StateFailed
	case StateStopping:
		// From stopping, can only become idle
		return newState == StateIdle
	case StateFailed:
		// From failed, can restart or go idle
		return newState == StateStarting || newState == StateIdle
	default:
		return false
	}
}

// StreamQuality contains information about a quality variant for adaptive streaming
type StreamQuality struct {
	Level        string `json:"level"`         // Quality level: "1080p", "720p", "480p"
	Bitrate      int    `json:"bitrate"`       // Video bitrate in kbps
	Resolution   string `json:"resolution"`    // Resolution: "1920x1080"
	SegmentPath  string `json:"segment_path"`  // Path to segments for this quality
	PlaylistPath string `json:"playlist_path"` // Path to .m3u8 playlist file
}

// SessionManager manages a collection of active streaming sessions with thread-safe operations
type SessionManager struct {
	sessions        map[string]*models.StreamSession // key: channelID as string
	circuitBreakers map[string]*CircuitBreaker       // key: channelID as string
	mu              sync.RWMutex
}

// NewSessionManager creates a new SessionManager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:        make(map[string]*models.StreamSession),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// Get retrieves a session by channel ID (thread-safe)
func (m *SessionManager) Get(channelID string) (*models.StreamSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[channelID]
	return session, ok
}

// Set stores a session for a channel ID (thread-safe)
func (m *SessionManager) Set(channelID string, session *models.StreamSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[channelID] = session
}

// Delete removes a session by channel ID (thread-safe)
func (m *SessionManager) Delete(channelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, channelID)
}

// List returns all active sessions (thread-safe)
func (m *SessionManager) List() []*models.StreamSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*models.StreamSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetAll returns sessions that match the filter function (thread-safe)
func (m *SessionManager) GetAll(filter func(*models.StreamSession) bool) []*models.StreamSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*models.StreamSession, 0)
	for _, session := range m.sessions {
		if filter(session) {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// GetCircuitBreaker retrieves the circuit breaker for a channel (thread-safe)
func (m *SessionManager) GetCircuitBreaker(channelID string) (*CircuitBreaker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cb, ok := m.circuitBreakers[channelID]
	return cb, ok
}

// GetOrCreateCircuitBreaker gets or creates a circuit breaker for a channel (thread-safe)
func (m *SessionManager) GetOrCreateCircuitBreaker(channelID string) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, ok := m.circuitBreakers[channelID]; ok {
		return cb
	}

	cb := NewCircuitBreaker(CircuitBreakerThreshold, CircuitBreakerResetTimeout)
	m.circuitBreakers[channelID] = cb
	return cb
}

// DeleteCircuitBreaker removes a circuit breaker for a channel (thread-safe)
func (m *SessionManager) DeleteCircuitBreaker(channelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.circuitBreakers, channelID)
}
