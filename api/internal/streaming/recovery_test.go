package streaming

import (
	"testing"
	"time"
)

func TestCalculateBackoffDuration(t *testing.T) {
	tests := []struct {
		name         string
		attemptCount int
		expected     time.Duration
	}{
		{"Zero attempts", 0, 1 * time.Second},
		{"First attempt", 1, 2 * time.Second},
		{"Second attempt", 2, 4 * time.Second},
		{"Third attempt", 3, 8 * time.Second},
		{"Fourth attempt (capped)", 4, 8 * time.Second},
		{"Many attempts (capped)", 10, 8 * time.Second},
		{"Negative attempts", -1, 1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBackoffDuration(tt.attemptCount)
			if result != tt.expected {
				t.Errorf("calculateBackoffDuration(%d) = %v, want %v", tt.attemptCount, result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoffDuration_ExponentialGrowth(t *testing.T) {
	// Verify exponential growth pattern
	backoff0 := calculateBackoffDuration(0)
	backoff1 := calculateBackoffDuration(1)
	backoff2 := calculateBackoffDuration(2)

	if backoff1 != backoff0*2 {
		t.Errorf("Backoff should double: %v * 2 != %v", backoff0, backoff1)
	}
	if backoff2 != backoff1*2 {
		t.Errorf("Backoff should double: %v * 2 != %v", backoff1, backoff2)
	}
}

func TestCheckDiskSpace_Constants(t *testing.T) {
	// Verify constants are properly defined
	if MinDiskSpaceBytes != 5*1024*1024*1024 {
		t.Errorf("MinDiskSpaceBytes = %d, want %d (5GB)", MinDiskSpaceBytes, 5*1024*1024*1024)
	}
	if WarnDiskSpaceBytes != 10*1024*1024*1024 {
		t.Errorf("WarnDiskSpaceBytes = %d, want %d (10GB)", WarnDiskSpaceBytes, 10*1024*1024*1024)
	}
	if MaxRestartAttempts != 3 {
		t.Errorf("MaxRestartAttempts = %d, want 3", MaxRestartAttempts)
	}
	if CircuitBreakerThreshold != 3 {
		t.Errorf("CircuitBreakerThreshold = %d, want 3", CircuitBreakerThreshold)
	}
	if CircuitBreakerResetTimeout != 60*time.Second {
		t.Errorf("CircuitBreakerResetTimeout = %v, want 60s", CircuitBreakerResetTimeout)
	}
	if InitialBackoff != 1*time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", InitialBackoff)
	}
	if MaxBackoff != 8*time.Second {
		t.Errorf("MaxBackoff = %v, want 8s", MaxBackoff)
	}
}

func TestGetAvailableSpace(t *testing.T) {
	// Test with current directory
	space, err := getAvailableSpace(".")
	if err != nil {
		t.Fatalf("getAvailableSpace(\".\") error = %v", err)
	}

	if space == 0 {
		t.Error("Available space should not be 0")
	}

	// Test with invalid path
	_, err = getAvailableSpace("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("getAvailableSpace should return error for nonexistent path")
	}
}

func TestCheckDiskSpace_ValidPath(t *testing.T) {
	// Test with current directory - should have space
	err := checkDiskSpace(".")

	// Note: This test might fail on systems with very low disk space
	// In production, you might want to skip this test or mock the syscall
	if err != nil {
		t.Logf("checkDiskSpace(\".\") error = %v (this might be expected if disk is actually low)", err)
		// Not failing the test as it depends on actual disk space
	}
}

func TestCheckDiskSpace_InvalidPath(t *testing.T) {
	err := checkDiskSpace("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("checkDiskSpace should return error for nonexistent path")
	}
}

// Note: The following tests for restartStream, handleFileError, fallbackToSoftwareEncoding,
// and attemptRecovery are integration tests that require a full StreamManager setup.
// They will be tested in the integration test suite (task 6-9).

// Test helper function for backoff progression
func TestBackoffProgression(t *testing.T) {
	var backoffs []time.Duration
	for i := 0; i < 5; i++ {
		backoffs = append(backoffs, calculateBackoffDuration(i))
	}

	// Verify increasing pattern
	for i := 1; i < len(backoffs); i++ {
		if backoffs[i] < backoffs[i-1] {
			t.Errorf("Backoff should not decrease: backoffs[%d] = %v < backoffs[%d] = %v",
				i, backoffs[i], i-1, backoffs[i-1])
		}
	}

	// Verify capping
	for i := 3; i < len(backoffs); i++ {
		if backoffs[i] != MaxBackoff {
			t.Errorf("Backoff should be capped at MaxBackoff: backoffs[%d] = %v, want %v",
				i, backoffs[i], MaxBackoff)
		}
	}
}

func TestRecoveryConstants_Relationships(t *testing.T) {
	// MinDiskSpace should be less than WarnDiskSpace
	if MinDiskSpaceBytes >= WarnDiskSpaceBytes {
		t.Error("MinDiskSpaceBytes should be less than WarnDiskSpaceBytes")
	}

	// InitialBackoff should be less than MaxBackoff
	if InitialBackoff >= MaxBackoff {
		t.Error("InitialBackoff should be less than MaxBackoff")
	}

	// MaxRestartAttempts should be reasonable
	if MaxRestartAttempts < 1 || MaxRestartAttempts > 10 {
		t.Errorf("MaxRestartAttempts = %d should be between 1 and 10", MaxRestartAttempts)
	}

	// CircuitBreakerThreshold should be reasonable
	if CircuitBreakerThreshold < 1 || CircuitBreakerThreshold > 10 {
		t.Errorf("CircuitBreakerThreshold = %d should be between 1 and 10", CircuitBreakerThreshold)
	}
}

func TestBackoffDoesNotOverflow(t *testing.T) {
	// Test with a very large attempt count to ensure no overflow
	result := calculateBackoffDuration(1000)
	if result != MaxBackoff {
		t.Errorf("Very large attempt count should return MaxBackoff, got %v", result)
	}
}

func TestDiskSpaceThresholds(t *testing.T) {
	// Verify the thresholds are in bytes (not KB, MB, or GB)
	oneGB := uint64(1024 * 1024 * 1024)

	if uint64(MinDiskSpaceBytes) != 5*oneGB {
		t.Errorf("MinDiskSpaceBytes should be 5GB in bytes, got %d", MinDiskSpaceBytes)
	}

	if uint64(WarnDiskSpaceBytes) != 10*oneGB {
		t.Errorf("WarnDiskSpaceBytes should be 10GB in bytes, got %d", WarnDiskSpaceBytes)
	}
}
