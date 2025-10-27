package media

import (
	"os"
	"strings"
)

// ValidationResult contains the result of media validation
type ValidationResult struct {
	Compatible        bool     // true if H.264 + AAC (no transcode needed)
	RequiresTranscode bool     // true if transcoding required
	Reasons           []string // Human-readable incompatibility reasons
	Readable          bool     // File exists and is accessible
}

// Codec constants for compatibility checks
const (
	compatibleVideoCodec = "h264"
	compatibleAudioCodec = "aac"
)

// ValidateMedia checks codec compatibility from VideoMetadata
// Returns ValidationResult indicating if the media requires transcoding
func ValidateMedia(metadata *VideoMetadata) ValidationResult {
	result := ValidationResult{
		Compatible:        true,
		RequiresTranscode: false,
		Reasons:           []string{},
		Readable:          true, // Assumed readable if we have metadata
	}

	// Normalize codecs to lowercase for comparison
	videoCodec := strings.ToLower(metadata.VideoCodec)
	audioCodec := strings.ToLower(metadata.AudioCodec)

	// Check video codec compatibility
	if videoCodec != compatibleVideoCodec {
		result.Compatible = false
		result.RequiresTranscode = true
		if videoCodec == "" {
			result.Reasons = append(result.Reasons, "video codec information missing")
		} else {
			result.Reasons = append(result.Reasons, "video codec '"+metadata.VideoCodec+"' is not H.264")
		}
	}

	// Check audio codec compatibility
	if audioCodec != compatibleAudioCodec {
		result.Compatible = false
		result.RequiresTranscode = true
		if audioCodec == "" {
			result.Reasons = append(result.Reasons, "audio codec information missing")
		} else {
			result.Reasons = append(result.Reasons, "audio codec '"+metadata.AudioCodec+"' is not AAC")
		}
	}

	return result
}

// ValidateFile checks if a file exists and is readable
// Returns ValidationResult with Readable field set appropriately
func ValidateFile(filePath string) ValidationResult {
	result := ValidationResult{
		Compatible:        false, // Unknown until metadata is extracted
		RequiresTranscode: false, // Unknown until metadata is extracted
		Reasons:           []string{},
		Readable:          false,
	}

	// Check if file exists and get info
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Reasons = append(result.Reasons, "file does not exist")
		} else if os.IsPermission(err) {
			result.Reasons = append(result.Reasons, "file is not readable (permission denied)")
		} else {
			result.Reasons = append(result.Reasons, "file access error: "+err.Error())
		}
		return result
	}

	// Check if it's a regular file (not a directory)
	if info.IsDir() {
		result.Reasons = append(result.Reasons, "path is a directory, not a file")
		return result
	}

	// Actually try to open the file to verify read permissions
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsPermission(err) {
			result.Reasons = append(result.Reasons, "file is not readable (permission denied)")
		} else {
			result.Reasons = append(result.Reasons, "cannot open file: "+err.Error())
		}
		return result
	}
	file.Close()

	// File exists and is readable
	result.Readable = true
	return result
}
