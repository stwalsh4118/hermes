package models

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNewStreamSession tests StreamSession creation
func TestNewStreamSession(t *testing.T) {
	channelID := uuid.New()
	session := NewStreamSession(channelID)

	if session == nil {
		t.Fatal("NewStreamSession() returned nil")
	}
	if session.ID == uuid.Nil {
		t.Error("Session ID not set")
	}
	if session.ChannelID != channelID {
		t.Errorf("ChannelID = %v, want %v", session.ChannelID, channelID)
	}
	if session.ClientCount != 0 {
		t.Errorf("ClientCount = %d, want 0", session.ClientCount)
	}
	if session.FFmpegPID != 0 {
		t.Errorf("FFmpegPID = %d, want 0", session.FFmpegPID)
	}
	if session.State != "idle" {
		t.Errorf("State = %s, want idle", session.State)
	}
	if session.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want 0", session.ErrorCount)
	}
	if session.LastError != "" {
		t.Errorf("LastError = %s, want empty", session.LastError)
	}
	if session.Qualities == nil {
		t.Error("Qualities slice not initialized")
	}
	if time.Since(session.StartedAt) > time.Second {
		t.Error("StartedAt not set to recent time")
	}
	if time.Since(session.LastAccessTime) > time.Second {
		t.Error("LastAccessTime not set to recent time")
	}
}

// TestStreamSession_ClientCounting tests increment and decrement operations
func TestStreamSession_ClientCounting(t *testing.T) {
	session := NewStreamSession(uuid.New())

	// Test increment
	session.IncrementClients()
	if count := session.GetClientCount(); count != 1 {
		t.Errorf("After increment, count = %d, want 1", count)
	}

	session.IncrementClients()
	if count := session.GetClientCount(); count != 2 {
		t.Errorf("After second increment, count = %d, want 2", count)
	}

	// Test decrement
	session.DecrementClients()
	if count := session.GetClientCount(); count != 1 {
		t.Errorf("After decrement, count = %d, want 1", count)
	}

	session.DecrementClients()
	if count := session.GetClientCount(); count != 0 {
		t.Errorf("After second decrement, count = %d, want 0", count)
	}

	// Test decrement at zero (should not go negative)
	session.DecrementClients()
	if count := session.GetClientCount(); count != 0 {
		t.Errorf("After decrement at zero, count = %d, want 0", count)
	}
}

// TestStreamSession_ConcurrentClientCounting tests thread-safety of client counting
func TestStreamSession_ConcurrentClientCounting(t *testing.T) {
	session := NewStreamSession(uuid.New())
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent increments
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			session.IncrementClients()
		}()
	}

	// Concurrent decrements
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			session.DecrementClients()
		}()
	}

	wg.Wait()

	// Final count should be 0 or positive (decrements don't go negative)
	if count := session.GetClientCount(); count < 0 {
		t.Errorf("Final count = %d, should not be negative", count)
	}
}

// TestStreamSession_IsActive tests the IsActive method
func TestStreamSession_IsActive(t *testing.T) {
	session := NewStreamSession(uuid.New())

	if session.IsActive() {
		t.Error("New session should not be active")
	}

	session.IncrementClients()
	if !session.IsActive() {
		t.Error("Session with clients should be active")
	}

	session.DecrementClients()
	if session.IsActive() {
		t.Error("Session without clients should not be active")
	}
}

// TestStreamSession_FFmpegPID tests FFmpeg PID operations
func TestStreamSession_FFmpegPID(t *testing.T) {
	session := NewStreamSession(uuid.New())

	if pid := session.GetFFmpegPID(); pid != 0 {
		t.Errorf("Initial PID = %d, want 0", pid)
	}

	session.SetFFmpegPID(12345)
	if pid := session.GetFFmpegPID(); pid != 12345 {
		t.Errorf("After SetFFmpegPID(12345), got %d", pid)
	}
}

// TestStreamSession_State tests state management
func TestStreamSession_State(t *testing.T) {
	session := NewStreamSession(uuid.New())

	if state := session.GetState(); state != "idle" {
		t.Errorf("Initial state = %s, want idle", state)
	}

	session.SetState("starting")
	if state := session.GetState(); state != "starting" {
		t.Errorf("After SetState(starting), got %s", state)
	}

	session.SetState("active")
	if state := session.GetState(); state != "active" {
		t.Errorf("After SetState(active), got %s", state)
	}
}

// TestStreamSession_Qualities tests quality management
func TestStreamSession_Qualities(t *testing.T) {
	session := NewStreamSession(uuid.New())

	// Test initial state
	qualities := session.GetQualities()
	if len(qualities) != 0 {
		t.Errorf("Initial qualities length = %d, want 0", len(qualities))
	}

	// Set qualities
	newQualities := []StreamQuality{
		{Level: "1080p", Bitrate: 5000, Resolution: "1920x1080"},
		{Level: "720p", Bitrate: 3000, Resolution: "1280x720"},
	}
	session.SetQualities(newQualities)

	// Get and verify
	retrieved := session.GetQualities()
	if len(retrieved) != 2 {
		t.Errorf("Retrieved qualities length = %d, want 2", len(retrieved))
	}
	if retrieved[0].Level != "1080p" {
		t.Errorf("First quality level = %s, want 1080p", retrieved[0].Level)
	}
	if retrieved[1].Level != "720p" {
		t.Errorf("Second quality level = %s, want 720p", retrieved[1].Level)
	}

	// Verify returned copy is independent
	retrieved[0].Level = "modified"
	retrieved2 := session.GetQualities()
	if retrieved2[0].Level != "1080p" {
		t.Error("Modifying returned slice affected internal state")
	}
}

// TestStreamSession_ErrorTracking tests error counting and messages
func TestStreamSession_ErrorTracking(t *testing.T) {
	session := NewStreamSession(uuid.New())

	// Test initial state
	if count := session.GetErrorCount(); count != 0 {
		t.Errorf("Initial error count = %d, want 0", count)
	}
	if msg := session.GetLastError(); msg != "" {
		t.Errorf("Initial last error = %s, want empty", msg)
	}

	// Test increment
	session.IncrementErrorCount()
	if count := session.GetErrorCount(); count != 1 {
		t.Errorf("After increment, error count = %d, want 1", count)
	}

	session.IncrementErrorCount()
	session.IncrementErrorCount()
	if count := session.GetErrorCount(); count != 3 {
		t.Errorf("After three increments, error count = %d, want 3", count)
	}

	// Test set last error
	testErr := errors.New("test error message")
	session.SetLastError(testErr)
	if msg := session.GetLastError(); msg != "test error message" {
		t.Errorf("Last error = %s, want 'test error message'", msg)
	}

	// Test set nil error
	session.SetLastError(nil)
	if msg := session.GetLastError(); msg != "" {
		t.Errorf("After SetLastError(nil), last error = %s, want empty", msg)
	}

	// Test reset
	session.IncrementErrorCount()
	session.SetLastError(testErr)
	session.ResetErrors()
	if count := session.GetErrorCount(); count != 0 {
		t.Errorf("After reset, error count = %d, want 0", count)
	}
	if msg := session.GetLastError(); msg != "" {
		t.Errorf("After reset, last error = %s, want empty", msg)
	}
}

// TestStreamSession_LastAccessTime tests access time tracking
func TestStreamSession_LastAccessTime(t *testing.T) {
	session := NewStreamSession(uuid.New())

	initialTime := session.GetLastAccessTime()
	if time.Since(initialTime) > time.Second {
		t.Error("Initial LastAccessTime not set to recent time")
	}

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Update access time
	session.UpdateLastAccess()
	newTime := session.GetLastAccessTime()

	if !newTime.After(initialTime) {
		t.Error("UpdateLastAccess() did not update the time")
	}
}

// TestStreamSession_IdleDuration tests idle duration calculation
func TestStreamSession_IdleDuration(t *testing.T) {
	session := NewStreamSession(uuid.New())

	// Initial idle duration should be very small
	if duration := session.IdleDuration(); duration > time.Second {
		t.Errorf("Initial idle duration = %v, expected < 1s", duration)
	}

	// Wait and check again
	time.Sleep(100 * time.Millisecond)
	duration := session.IdleDuration()
	if duration < 100*time.Millisecond {
		t.Errorf("Idle duration = %v, expected >= 100ms", duration)
	}

	// Update access and check reset
	session.UpdateLastAccess()
	duration = session.IdleDuration()
	if duration > 50*time.Millisecond {
		t.Errorf("Idle duration after update = %v, expected < 50ms", duration)
	}
}

// TestStreamSession_ShouldCleanup tests cleanup decision logic
func TestStreamSession_ShouldCleanup(t *testing.T) {
	session := NewStreamSession(uuid.New())

	// With clients, should not cleanup even after grace period
	session.IncrementClients()
	time.Sleep(10 * time.Millisecond)
	if session.ShouldCleanup(5 * time.Millisecond) {
		t.Error("Should not cleanup with active clients")
	}

	// Without clients but within grace period, should not cleanup
	session.DecrementClients()
	if session.ShouldCleanup(1 * time.Second) {
		t.Error("Should not cleanup within grace period")
	}

	// Without clients and after grace period, should cleanup
	time.Sleep(20 * time.Millisecond)
	if !session.ShouldCleanup(10 * time.Millisecond) {
		t.Error("Should cleanup after grace period with no clients")
	}

	// Update access resets the timer
	session.UpdateLastAccess()
	if session.ShouldCleanup(10 * time.Millisecond) {
		t.Error("Should not cleanup after access update")
	}
}

// TestStreamSession_SegmentPath tests segment path operations
func TestStreamSession_SegmentPath(t *testing.T) {
	session := NewStreamSession(uuid.New())

	if path := session.GetSegmentPath(); path != "" {
		t.Errorf("Initial segment path = %s, want empty", path)
	}

	session.SetSegmentPath("/streams/channel1/segments")
	if path := session.GetSegmentPath(); path != "/streams/channel1/segments" {
		t.Errorf("Segment path = %s, want /streams/channel1/segments", path)
	}
}

// TestStreamSession_OutputDir tests output directory operations
func TestStreamSession_OutputDir(t *testing.T) {
	session := NewStreamSession(uuid.New())

	if dir := session.GetOutputDir(); dir != "" {
		t.Errorf("Initial output dir = %s, want empty", dir)
	}

	session.SetOutputDir("/streams/channel1")
	if dir := session.GetOutputDir(); dir != "/streams/channel1" {
		t.Errorf("Output dir = %s, want /streams/channel1", dir)
	}
}

// TestStreamSession_ConcurrentStateAccess tests thread-safety of state access
func TestStreamSession_ConcurrentStateAccess(t *testing.T) {
	session := NewStreamSession(uuid.New())
	const numGoroutines = 100

	states := []string{"idle", "starting", "active", "stopping", "failed"}

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			session.SetState(states[idx%len(states)])
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = session.GetState()
		}()
	}

	wg.Wait()

	// Verify state is one of the valid states
	finalState := session.GetState()
	validState := false
	for _, state := range states {
		if finalState == state {
			validState = true
			break
		}
	}
	if !validState {
		t.Errorf("Final state %s not in valid states", finalState)
	}
}

// TestStreamSession_ConcurrentErrorTracking tests thread-safety of error tracking
func TestStreamSession_ConcurrentErrorTracking(t *testing.T) {
	session := NewStreamSession(uuid.New())
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Concurrent increments
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			session.IncrementErrorCount()
		}()
	}

	// Concurrent error sets
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			err := errors.New("error " + string(rune(idx)))
			session.SetLastError(err)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = session.GetErrorCount()
			_ = session.GetLastError()
		}()
	}

	wg.Wait()

	// Verify error count is positive
	if count := session.GetErrorCount(); count <= 0 {
		t.Errorf("Final error count = %d, expected > 0", count)
	}
}

// TestStreamSession_ConcurrentAccessTimeUpdates tests thread-safety of access time updates
func TestStreamSession_ConcurrentAccessTimeUpdates(t *testing.T) {
	session := NewStreamSession(uuid.New())
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Concurrent updates
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			session.UpdateLastAccess()
		}()
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = session.GetLastAccessTime()
		}()
	}

	// Concurrent idle duration checks
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = session.IdleDuration()
		}()
	}

	wg.Wait()

	// Verify last access time is recent
	if time.Since(session.GetLastAccessTime()) > time.Second {
		t.Error("LastAccessTime not updated to recent time")
	}
}
