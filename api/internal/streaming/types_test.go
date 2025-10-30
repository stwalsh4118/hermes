package streaming

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stwalsh4118/hermes/internal/models"
)

// TestStreamState_String tests the String method
func TestStreamState_String(t *testing.T) {
	tests := []struct {
		name  string
		state StreamState
		want  string
	}{
		{"idle", StateIdle, "idle"},
		{"starting", StateStarting, "starting"},
		{"active", StateActive, "active"},
		{"stopping", StateStopping, "stopping"},
		{"failed", StateFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("StreamState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStreamState_IsValid tests the IsValid method
func TestStreamState_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		state StreamState
		want  bool
	}{
		{"idle is valid", StateIdle, true},
		{"starting is valid", StateStarting, true},
		{"active is valid", StateActive, true},
		{"stopping is valid", StateStopping, true},
		{"failed is valid", StateFailed, true},
		{"invalid state", StreamState("invalid"), false},
		{"empty state", StreamState(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.want {
				t.Errorf("StreamState.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStreamState_CanTransitionTo tests valid state transitions
func TestStreamState_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     StreamState
		to       StreamState
		expected bool
	}{
		// From Idle
		{"idle to starting", StateIdle, StateStarting, true},
		{"idle to active", StateIdle, StateActive, false},
		{"idle to stopping", StateIdle, StateStopping, false},
		{"idle to failed", StateIdle, StateFailed, false},
		{"idle to idle", StateIdle, StateIdle, false},

		// From Starting
		{"starting to active", StateStarting, StateActive, true},
		{"starting to failed", StateStarting, StateFailed, true},
		{"starting to stopping", StateStarting, StateStopping, true},
		{"starting to idle", StateStarting, StateIdle, false},
		{"starting to starting", StateStarting, StateStarting, false},

		// From Active
		{"active to stopping", StateActive, StateStopping, true},
		{"active to failed", StateActive, StateFailed, true},
		{"active to idle", StateActive, StateIdle, false},
		{"active to starting", StateActive, StateStarting, false},
		{"active to active", StateActive, StateActive, false},

		// From Stopping
		{"stopping to idle", StateStopping, StateIdle, true},
		{"stopping to starting", StateStopping, StateStarting, false},
		{"stopping to active", StateStopping, StateActive, false},
		{"stopping to failed", StateStopping, StateFailed, false},
		{"stopping to stopping", StateStopping, StateStopping, false},

		// From Failed
		{"failed to starting", StateFailed, StateStarting, true},
		{"failed to idle", StateFailed, StateIdle, true},
		{"failed to active", StateFailed, StateActive, false},
		{"failed to stopping", StateFailed, StateStopping, false},
		{"failed to failed", StateFailed, StateFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.expected {
				t.Errorf("StreamState.CanTransitionTo() from %s to %s = %v, want %v",
					tt.from, tt.to, got, tt.expected)
			}
		})
	}
}

// TestSessionManager_NewSessionManager tests SessionManager creation
func TestSessionManager_NewSessionManager(t *testing.T) {
	manager := NewSessionManager()
	if manager == nil {
		t.Fatal("NewSessionManager() returned nil")
	}
	if manager.sessions == nil {
		t.Error("SessionManager.sessions map not initialized")
	}
}

// TestSessionManager_SetAndGet tests basic set and get operations
func TestSessionManager_SetAndGet(t *testing.T) {
	manager := NewSessionManager()
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)

	// Set session
	manager.Set(channelID.String(), session)

	// Get session
	retrieved, ok := manager.Get(channelID.String())
	if !ok {
		t.Fatal("Get() returned false, expected true")
	}
	if retrieved.ID != session.ID {
		t.Errorf("Get() returned session with ID %v, want %v", retrieved.ID, session.ID)
	}
}

// TestSessionManager_GetNonExistent tests getting a non-existent session
func TestSessionManager_GetNonExistent(t *testing.T) {
	manager := NewSessionManager()
	channelID := uuid.New().String()

	_, ok := manager.Get(channelID)
	if ok {
		t.Error("Get() returned true for non-existent session, expected false")
	}
}

// TestSessionManager_Delete tests deleting a session
func TestSessionManager_Delete(t *testing.T) {
	manager := NewSessionManager()
	channelID := uuid.New()
	session := models.NewStreamSession(channelID)

	// Set and verify
	manager.Set(channelID.String(), session)
	if _, ok := manager.Get(channelID.String()); !ok {
		t.Fatal("Session not found after Set()")
	}

	// Delete and verify
	manager.Delete(channelID.String())
	if _, ok := manager.Get(channelID.String()); ok {
		t.Error("Session still exists after Delete()")
	}
}

// TestSessionManager_List tests listing all sessions
func TestSessionManager_List(t *testing.T) {
	manager := NewSessionManager()

	// Add multiple sessions
	channelIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, channelID := range channelIDs {
		session := models.NewStreamSession(channelID)
		manager.Set(channelID.String(), session)
	}

	// List all sessions
	sessions := manager.List()
	if len(sessions) != len(channelIDs) {
		t.Errorf("List() returned %d sessions, want %d", len(sessions), len(channelIDs))
	}
}

// TestSessionManager_GetAll tests filtered retrieval
func TestSessionManager_GetAll(t *testing.T) {
	manager := NewSessionManager()

	// Add sessions with different client counts
	channelID1 := uuid.New()
	session1 := models.NewStreamSession(channelID1)
	session1.IncrementClients()
	manager.Set(channelID1.String(), session1)

	channelID2 := uuid.New()
	session2 := models.NewStreamSession(channelID2)
	manager.Set(channelID2.String(), session2)

	channelID3 := uuid.New()
	session3 := models.NewStreamSession(channelID3)
	session3.IncrementClients()
	session3.IncrementClients()
	manager.Set(channelID3.String(), session3)

	// Filter for sessions with clients
	activeSessions := manager.GetAll(func(s *models.StreamSession) bool {
		return s.GetClientCount() > 0
	})

	if len(activeSessions) != 2 {
		t.Errorf("GetAll() returned %d active sessions, want 2", len(activeSessions))
	}

	// Filter for sessions without clients
	idleSessions := manager.GetAll(func(s *models.StreamSession) bool {
		return s.GetClientCount() == 0
	})

	if len(idleSessions) != 1 {
		t.Errorf("GetAll() returned %d idle sessions, want 1", len(idleSessions))
	}
}

// TestSessionManager_ConcurrentAccess tests thread-safety with concurrent operations
func TestSessionManager_ConcurrentAccess(t *testing.T) {
	manager := NewSessionManager()
	channelID := uuid.New().String()
	session := models.NewStreamSession(uuid.New())

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 operations per goroutine

	// Concurrent Set operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			manager.Set(channelID, session)
		}()
	}

	// Concurrent Get operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			manager.Get(channelID)
		}()
	}

	// Concurrent List operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			manager.List()
		}()
	}

	wg.Wait()

	// Verify final state
	retrieved, ok := manager.Get(channelID)
	if !ok {
		t.Error("Session not found after concurrent operations")
	}
	if retrieved.ID != session.ID {
		t.Error("Session ID mismatch after concurrent operations")
	}
}

// TestSessionManager_ConcurrentSetDelete tests concurrent set and delete
func TestSessionManager_ConcurrentSetDelete(_ *testing.T) {
	manager := NewSessionManager()
	const numIterations = 1000

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine that sets sessions
	go func() {
		defer wg.Done()
		for i := 0; i < numIterations; i++ {
			channelID := uuid.New()
			session := models.NewStreamSession(channelID)
			manager.Set(channelID.String(), session)
		}
	}()

	// Goroutine that deletes sessions
	go func() {
		defer wg.Done()
		for i := 0; i < numIterations; i++ {
			sessions := manager.List()
			for _, session := range sessions {
				manager.Delete(session.ChannelID.String())
			}
		}
	}()

	wg.Wait()

	// No assertions needed - test passes if no race conditions detected
}

// TestStreamQuality_Creation tests StreamQuality struct creation
func TestStreamQuality_Creation(t *testing.T) {
	quality := StreamQuality{
		Level:        "1080p",
		Bitrate:      5000,
		Resolution:   "1920x1080",
		SegmentPath:  "/streams/channel1/1080p",
		PlaylistPath: "/streams/channel1/1080p.m3u8",
	}

	if quality.Level != "1080p" {
		t.Errorf("StreamQuality.Level = %v, want 1080p", quality.Level)
	}
	if quality.Bitrate != 5000 {
		t.Errorf("StreamQuality.Bitrate = %v, want 5000", quality.Bitrate)
	}
	if quality.Resolution != "1920x1080" {
		t.Errorf("StreamQuality.Resolution = %v, want 1920x1080", quality.Resolution)
	}
}
