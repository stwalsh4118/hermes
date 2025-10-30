package media

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateMedia_Compatible tests validation of compatible media
func TestValidateMedia_Compatible(t *testing.T) {
	metadata := &VideoMetadata{
		Duration:   3600,
		VideoCodec: "h264",
		AudioCodec: "aac",
		Resolution: "1920x1080",
		FileSize:   1024000,
	}

	result := ValidateMedia(metadata)

	if !result.Compatible {
		t.Error("Expected compatible=true for H.264+AAC, got false")
	}
	if result.RequiresTranscode {
		t.Error("Expected requiresTranscode=false for H.264+AAC, got true")
	}
	if len(result.Reasons) > 0 {
		t.Errorf("Expected no reasons for compatible media, got: %v", result.Reasons)
	}
	if !result.Readable {
		t.Error("Expected readable=true when validating metadata, got false")
	}
}

// TestValidateMedia_CompatibleCaseInsensitive tests case-insensitive codec matching
func TestValidateMedia_CompatibleCaseInsensitive(t *testing.T) {
	testCases := []struct {
		name       string
		videoCodec string
		audioCodec string
	}{
		{"Uppercase", "H264", "AAC"},
		{"Mixed case", "H264", "aac"},
		{"Lowercase", "h264", "aac"},
		{"With dots", "h.264", "aac"}, // Should fail - we normalize but don't remove dots
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata := &VideoMetadata{
				Duration:   3600,
				VideoCodec: tc.videoCodec,
				AudioCodec: tc.audioCodec,
				Resolution: "1920x1080",
			}

			result := ValidateMedia(metadata)

			// Only exact matches (case-insensitive) should be compatible
			if tc.videoCodec == "h.264" {
				// This should fail because we're looking for exact "h264"
				if result.Compatible {
					t.Error("Expected compatible=false for h.264, got true")
				}
			} else {
				if !result.Compatible {
					t.Errorf("Expected compatible=true for %s+%s, got false", tc.videoCodec, tc.audioCodec)
				}
				if result.RequiresTranscode {
					t.Errorf("Expected requiresTranscode=false for %s+%s, got true", tc.videoCodec, tc.audioCodec)
				}
			}
		})
	}
}

// TestValidateMedia_IncompatibleVideoCodec tests incompatible video codecs
func TestValidateMedia_IncompatibleVideoCodec(t *testing.T) {
	testCases := []struct {
		codec string
	}{
		{"hevc"},
		{"vp9"},
		{"av1"},
		{"mpeg2"},
		{"mpeg4"},
	}

	for _, tc := range testCases {
		t.Run(tc.codec, func(t *testing.T) {
			metadata := &VideoMetadata{
				Duration:   3600,
				VideoCodec: tc.codec,
				AudioCodec: "aac", // Audio is compatible
				Resolution: "1920x1080",
			}

			result := ValidateMedia(metadata)

			if result.Compatible {
				t.Errorf("Expected compatible=false for %s video codec, got true", tc.codec)
			}
			if !result.RequiresTranscode {
				t.Errorf("Expected requiresTranscode=true for %s video codec, got false", tc.codec)
			}
			if len(result.Reasons) == 0 {
				t.Error("Expected reasons for incompatible video codec, got none")
			}

			// Check that reason mentions video codec
			foundVideoReason := false
			for _, reason := range result.Reasons {
				if containsIgnoreCase(reason, "video") && containsIgnoreCase(reason, tc.codec) {
					foundVideoReason = true
					break
				}
			}
			if !foundVideoReason {
				t.Errorf("Expected reason to mention video codec '%s', got: %v", tc.codec, result.Reasons)
			}
		})
	}
}

// TestValidateMedia_IncompatibleAudioCodec tests incompatible audio codecs
func TestValidateMedia_IncompatibleAudioCodec(t *testing.T) {
	testCases := []struct {
		codec string
	}{
		{"flac"},
		{"dts"},
		{"mp3"},
		{"opus"},
		{"vorbis"},
	}

	for _, tc := range testCases {
		t.Run(tc.codec, func(t *testing.T) {
			metadata := &VideoMetadata{
				Duration:   3600,
				VideoCodec: "h264", // Video is compatible
				AudioCodec: tc.codec,
				Resolution: "1920x1080",
			}

			result := ValidateMedia(metadata)

			if result.Compatible {
				t.Errorf("Expected compatible=false for %s audio codec, got true", tc.codec)
			}
			if !result.RequiresTranscode {
				t.Errorf("Expected requiresTranscode=true for %s audio codec, got false", tc.codec)
			}
			if len(result.Reasons) == 0 {
				t.Error("Expected reasons for incompatible audio codec, got none")
			}

			// Check that reason mentions audio codec
			foundAudioReason := false
			for _, reason := range result.Reasons {
				if containsIgnoreCase(reason, "audio") && containsIgnoreCase(reason, tc.codec) {
					foundAudioReason = true
					break
				}
			}
			if !foundAudioReason {
				t.Errorf("Expected reason to mention audio codec '%s', got: %v", tc.codec, result.Reasons)
			}
		})
	}
}

// TestValidateMedia_BothIncompatible tests both video and audio incompatible
func TestValidateMedia_BothIncompatible(t *testing.T) {
	metadata := &VideoMetadata{
		Duration:   3600,
		VideoCodec: "vp9",
		AudioCodec: "opus",
		Resolution: "1920x1080",
	}

	result := ValidateMedia(metadata)

	if result.Compatible {
		t.Error("Expected compatible=false for VP9+Opus, got true")
	}
	if !result.RequiresTranscode {
		t.Error("Expected requiresTranscode=true for VP9+Opus, got false")
	}
	if len(result.Reasons) < 2 {
		t.Errorf("Expected at least 2 reasons for both incompatible codecs, got %d: %v", len(result.Reasons), result.Reasons)
	}

	// Check that both video and audio reasons are present
	hasVideoReason := false
	hasAudioReason := false
	for _, reason := range result.Reasons {
		if containsIgnoreCase(reason, "video") {
			hasVideoReason = true
		}
		if containsIgnoreCase(reason, "audio") {
			hasAudioReason = true
		}
	}

	if !hasVideoReason {
		t.Errorf("Expected reason about video codec, got: %v", result.Reasons)
	}
	if !hasAudioReason {
		t.Errorf("Expected reason about audio codec, got: %v", result.Reasons)
	}
}

// TestValidateMedia_MissingVideoCodec tests missing video codec information
func TestValidateMedia_MissingVideoCodec(t *testing.T) {
	metadata := &VideoMetadata{
		Duration:   3600,
		VideoCodec: "", // Missing
		AudioCodec: "aac",
		Resolution: "1920x1080",
	}

	result := ValidateMedia(metadata)

	if result.Compatible {
		t.Error("Expected compatible=false for missing video codec, got true")
	}
	if !result.RequiresTranscode {
		t.Error("Expected requiresTranscode=true for missing video codec, got false")
	}
	if len(result.Reasons) == 0 {
		t.Error("Expected reasons for missing video codec, got none")
	}

	// Check for "missing" in reason
	foundMissingReason := false
	for _, reason := range result.Reasons {
		if containsIgnoreCase(reason, "missing") && containsIgnoreCase(reason, "video") {
			foundMissingReason = true
			break
		}
	}
	if !foundMissingReason {
		t.Errorf("Expected reason to mention missing video codec, got: %v", result.Reasons)
	}
}

// TestValidateMedia_MissingAudioCodec tests missing audio codec information
func TestValidateMedia_MissingAudioCodec(t *testing.T) {
	metadata := &VideoMetadata{
		Duration:   3600,
		VideoCodec: "h264",
		AudioCodec: "", // Missing
		Resolution: "1920x1080",
	}

	result := ValidateMedia(metadata)

	if result.Compatible {
		t.Error("Expected compatible=false for missing audio codec, got true")
	}
	if !result.RequiresTranscode {
		t.Error("Expected requiresTranscode=true for missing audio codec, got false")
	}
	if len(result.Reasons) == 0 {
		t.Error("Expected reasons for missing audio codec, got none")
	}

	// Check for "missing" in reason
	foundMissingReason := false
	for _, reason := range result.Reasons {
		if containsIgnoreCase(reason, "missing") && containsIgnoreCase(reason, "audio") {
			foundMissingReason = true
			break
		}
	}
	if !foundMissingReason {
		t.Errorf("Expected reason to mention missing audio codec, got: %v", result.Reasons)
	}
}

// TestValidateMedia_BothMissing tests both codecs missing
func TestValidateMedia_BothMissing(t *testing.T) {
	metadata := &VideoMetadata{
		Duration:   3600,
		VideoCodec: "",
		AudioCodec: "",
		Resolution: "1920x1080",
	}

	result := ValidateMedia(metadata)

	if result.Compatible {
		t.Error("Expected compatible=false for missing codecs, got true")
	}
	if !result.RequiresTranscode {
		t.Error("Expected requiresTranscode=true for missing codecs, got false")
	}
	if len(result.Reasons) < 2 {
		t.Errorf("Expected at least 2 reasons for missing codecs, got %d: %v", len(result.Reasons), result.Reasons)
	}
}

// TestValidateFile_ValidFile tests validation of existing readable file
func TestValidateFile_ValidFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-video.mp4")

	err := os.WriteFile(tmpFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ValidateFile(tmpFile)

	if !result.Readable {
		t.Errorf("Expected readable=true for existing file, got false. Reasons: %v", result.Reasons)
	}
	if len(result.Reasons) > 0 {
		t.Errorf("Expected no reasons for valid file, got: %v", result.Reasons)
	}
}

// TestValidateFile_NonExistentFile tests validation of non-existent file
func TestValidateFile_NonExistentFile(t *testing.T) {
	result := ValidateFile("/nonexistent/path/to/video.mp4")

	if result.Readable {
		t.Error("Expected readable=false for non-existent file, got true")
	}
	if len(result.Reasons) == 0 {
		t.Error("Expected reasons for non-existent file, got none")
	}

	// Check that reason mentions file not existing
	foundNotExistReason := false
	for _, reason := range result.Reasons {
		if containsIgnoreCase(reason, "not exist") || containsIgnoreCase(reason, "does not exist") {
			foundNotExistReason = true
			break
		}
	}
	if !foundNotExistReason {
		t.Errorf("Expected reason to mention file not existing, got: %v", result.Reasons)
	}
}

// TestValidateFile_Directory tests validation when path is a directory
func TestValidateFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	result := ValidateFile(tmpDir)

	if result.Readable {
		t.Error("Expected readable=false for directory, got true")
	}
	if len(result.Reasons) == 0 {
		t.Error("Expected reasons for directory path, got none")
	}

	// Check that reason mentions directory
	foundDirectoryReason := false
	for _, reason := range result.Reasons {
		if containsIgnoreCase(reason, "directory") {
			foundDirectoryReason = true
			break
		}
	}
	if !foundDirectoryReason {
		t.Errorf("Expected reason to mention directory, got: %v", result.Reasons)
	}
}

// TestValidateFile_PermissionDenied tests validation when file is not readable
func TestValidateFile_PermissionDenied(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.PathSeparator == '\\' {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a temporary file with no read permissions
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "unreadable.mp4")

	err := os.WriteFile(tmpFile, []byte("test content"), 0000)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ValidateFile(tmpFile)

	if result.Readable {
		t.Error("Expected readable=false for unreadable file, got true")
	}
	if len(result.Reasons) == 0 {
		t.Error("Expected reasons for unreadable file, got none")
	}

	// Clean up - restore permissions before cleanup
	_ = os.Chmod(tmpFile, 0644)
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		findSubstringIgnoreCase(s, substr)
}

func findSubstringIgnoreCase(s, substr string) bool {
	sLower := ""
	substrLower := ""

	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			sLower += string(c + 32)
		} else {
			sLower += string(c)
		}
	}

	for _, c := range substr {
		if c >= 'A' && c <= 'Z' {
			substrLower += string(c + 32)
		} else {
			substrLower += string(c)
		}
	}

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}
