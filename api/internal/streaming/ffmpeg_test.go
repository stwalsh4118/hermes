package streaming

import (
	"errors"
	"strings"
	"testing"
)

// TestBuildHLSCommand_1080p_Software tests basic 1080p software encoding
func TestBuildHLSCommand_1080p_Software(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	if cmd == nil {
		t.Fatal("Expected command, got nil")
	}

	// Verify args are not empty
	if len(cmd.Args) == 0 {
		t.Fatal("Expected non-empty args")
	}

	// Verify input file is present
	if !containsArg(cmd.Args, "/media/video.mp4") {
		t.Error("Input file not found in args")
	}

	// Verify software encoding codec
	if !containsConsecutiveArgs(cmd.Args, "-c:v", "libx264") {
		t.Error("Expected libx264 codec")
	}

	// Verify 1080p bitrate
	if !containsConsecutiveArgs(cmd.Args, "-b:v", "5000k") {
		t.Error("Expected 5000k bitrate for 1080p")
	}

	// Verify 1080p resolution
	if !containsConsecutiveArgs(cmd.Args, "-s", "1920x1080") {
		t.Error("Expected 1920x1080 resolution for 1080p")
	}

	// Verify audio encoding
	if !containsConsecutiveArgs(cmd.Args, "-c:a", "aac") {
		t.Error("Expected aac audio codec")
	}

	// Verify HLS format
	if !containsConsecutiveArgs(cmd.Args, "-f", "hls") {
		t.Error("Expected hls format")
	}

	// Verify output path
	if !containsArg(cmd.Args, "/streams/channel1/1080p.m3u8") {
		t.Error("Output path not found in args")
	}
}

// TestBuildHLSCommand_720p_NVENC tests 720p with NVIDIA hardware encoding
func TestBuildHLSCommand_720p_NVENC(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/720p.m3u8",
		Quality:         Quality720p,
		HardwareAccel:   HardwareAccelNVENC,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify NVENC codec
	if !containsConsecutiveArgs(cmd.Args, "-c:v", "h264_nvenc") {
		t.Error("Expected h264_nvenc codec")
	}

	// Verify NVENC preset
	if !containsConsecutiveArgs(cmd.Args, "-preset", "p1") {
		t.Error("Expected p1 preset for NVENC")
	}

	// Verify 720p bitrate
	if !containsConsecutiveArgs(cmd.Args, "-b:v", "3000k") {
		t.Error("Expected 3000k bitrate for 720p")
	}

	// Verify 720p resolution
	if !containsConsecutiveArgs(cmd.Args, "-s", "1280x720") {
		t.Error("Expected 1280x720 resolution for 720p")
	}
}

// TestBuildHLSCommand_480p_QSV tests 480p with Intel QSV hardware encoding
func TestBuildHLSCommand_480p_QSV(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/480p.m3u8",
		Quality:         Quality480p,
		HardwareAccel:   HardwareAccelQSV,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify QSV codec
	if !containsConsecutiveArgs(cmd.Args, "-c:v", "h264_qsv") {
		t.Error("Expected h264_qsv codec")
	}

	// Verify 480p bitrate
	if !containsConsecutiveArgs(cmd.Args, "-b:v", "1500k") {
		t.Error("Expected 1500k bitrate for 480p")
	}

	// Verify 480p resolution
	if !containsConsecutiveArgs(cmd.Args, "-s", "854x480") {
		t.Error("Expected 854x480 resolution for 480p")
	}
}

// TestBuildHLSCommand_VAAPI tests VAAPI hardware encoding
func TestBuildHLSCommand_VAAPI(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelVAAPI,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify VAAPI codec
	if !containsConsecutiveArgs(cmd.Args, "-c:v", "h264_vaapi") {
		t.Error("Expected h264_vaapi codec")
	}
}

// TestBuildHLSCommand_VideoToolbox tests Apple VideoToolbox hardware encoding
func TestBuildHLSCommand_VideoToolbox(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelVideoToolbox,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify VideoToolbox codec
	if !containsConsecutiveArgs(cmd.Args, "-c:v", "h264_videotoolbox") {
		t.Error("Expected h264_videotoolbox codec")
	}
}

// TestBuildHLSCommand_Auto tests auto hardware acceleration (should use software)
func TestBuildHLSCommand_Auto(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelAuto,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Auto should default to software encoding in command builder
	if !containsConsecutiveArgs(cmd.Args, "-c:v", "libx264") {
		t.Error("Expected libx264 codec for auto")
	}
}

// TestBuildHLSCommand_WithSeeking tests input seeking
func TestBuildHLSCommand_WithSeeking(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     3600, // Seek to 1 hour
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify -ss appears before -i
	ssIndex := findArgIndex(cmd.Args, "-ss")
	iIndex := findArgIndex(cmd.Args, "-i")

	if ssIndex == -1 {
		t.Error("Expected -ss flag for seeking")
	}

	if iIndex == -1 {
		t.Error("Expected -i flag for input")
	}

	if ssIndex >= iIndex {
		t.Error("Expected -ss flag to appear before -i flag for fast seeking")
	}

	// Verify seek value
	if ssIndex+1 >= len(cmd.Args) || cmd.Args[ssIndex+1] != "3600" {
		t.Error("Expected seek value of 3600")
	}
}

// TestBuildHLSCommand_NoSeeking tests that seeking is omitted when SeekSeconds is 0
func TestBuildHLSCommand_NoSeeking(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify -ss is not present
	if containsArg(cmd.Args, "-ss") {
		t.Error("Did not expect -ss flag when SeekSeconds is 0")
	}
}

// TestBuildHLSCommand_CustomSegmentDuration tests custom segment duration
func TestBuildHLSCommand_CustomSegmentDuration(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 10,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify custom segment duration
	if !containsConsecutiveArgs(cmd.Args, "-hls_time", "10") {
		t.Error("Expected hls_time of 10")
	}
}

// TestBuildHLSCommand_CustomPlaylistSize tests custom playlist size
func TestBuildHLSCommand_CustomPlaylistSize(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    20,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify custom playlist size
	if !containsConsecutiveArgs(cmd.Args, "-hls_list_size", "20") {
		t.Error("Expected hls_list_size of 20")
	}
}

// TestBuildHLSCommand_HLSParameters tests HLS-specific parameters
func TestBuildHLSCommand_HLSParameters(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify HLS flags
	if !containsConsecutiveArgs(cmd.Args, "-hls_flags", "delete_segments") {
		t.Error("Expected hls_flags delete_segments")
	}

	// Verify stream looping for 24/7 channel behavior (non-batch mode)
	if !containsConsecutiveArgs(cmd.Args, "-stream_loop", "-1") {
		t.Error("Expected stream_loop -1 for infinite looping in non-batch mode")
	}

	// Verify no hls_playlist_type (allows sliding window)
	if containsArg(cmd.Args, "-hls_playlist_type") {
		t.Error("Should not have hls_playlist_type for sliding window behavior")
	}

	// Verify segment filename pattern exists
	if !containsArg(cmd.Args, "-hls_segment_filename") {
		t.Error("Expected -hls_segment_filename flag")
	}
}

// TestBuildHLSCommand_SegmentPattern tests segment filename pattern generation
func TestBuildHLSCommand_SegmentPattern(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Find segment filename pattern
	idx := findArgIndex(cmd.Args, "-hls_segment_filename")
	if idx == -1 || idx+1 >= len(cmd.Args) {
		t.Fatal("Expected -hls_segment_filename with value")
	}

	pattern := cmd.Args[idx+1]
	if !strings.Contains(pattern, "_segment_") {
		t.Error("Expected segment pattern to contain '_segment_'")
	}

	if !strings.Contains(pattern, ".ts") {
		t.Error("Expected segment pattern to have .ts extension")
	}

	if !strings.Contains(pattern, "%03d") {
		t.Error("Expected segment pattern to contain %03d for numbering")
	}
}

// TestBuildHLSCommand_AudioParameters tests audio encoding parameters
func TestBuildHLSCommand_AudioParameters(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify audio codec
	if !containsConsecutiveArgs(cmd.Args, "-c:a", "aac") {
		t.Error("Expected aac audio codec")
	}

	// Verify audio bitrate
	if !containsConsecutiveArgs(cmd.Args, "-b:a", "192k") {
		t.Error("Expected 192k audio bitrate")
	}

	// Verify audio channels
	if !containsConsecutiveArgs(cmd.Args, "-ac", "2") {
		t.Error("Expected 2 audio channels")
	}
}

// TestBuildHLSCommand_InvalidQuality tests invalid quality level
func TestBuildHLSCommand_InvalidQuality(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         "4K", // Invalid
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	_, err := BuildHLSCommand(params)
	if err == nil {
		t.Fatal("Expected error for invalid quality")
	}

	if !strings.Contains(err.Error(), "invalid quality") {
		t.Errorf("Expected 'invalid quality' error, got: %v", err)
	}
}

// TestBuildHLSCommand_InvalidHardwareAccel tests invalid hardware acceleration
func TestBuildHLSCommand_InvalidHardwareAccel(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccel("invalid"),
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	_, err := BuildHLSCommand(params)
	if err == nil {
		t.Fatal("Expected error for invalid hardware acceleration")
	}

	if !strings.Contains(err.Error(), "invalid hardware acceleration") {
		t.Errorf("Expected 'invalid hardware acceleration' error, got: %v", err)
	}
}

// TestBuildHLSCommand_EmptyInputFile tests empty input file
func TestBuildHLSCommand_EmptyInputFile(t *testing.T) {
	params := StreamParams{
		InputFile:       "",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	_, err := BuildHLSCommand(params)
	if err == nil {
		t.Fatal("Expected error for empty input file")
	}

	if !errors.Is(err, ErrEmptyInputFile) {
		t.Errorf("Expected ErrEmptyInputFile, got: %v", err)
	}
}

// TestBuildHLSCommand_EmptyOutputPath tests empty output path
func TestBuildHLSCommand_EmptyOutputPath(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
	}

	_, err := BuildHLSCommand(params)
	if err == nil {
		t.Fatal("Expected error for empty output path")
	}

	if !errors.Is(err, ErrEmptyOutputPath) {
		t.Errorf("Expected ErrEmptyOutputPath, got: %v", err)
	}
}

// TestBuildHLSCommand_InvalidSegmentDuration tests invalid segment duration
func TestBuildHLSCommand_InvalidSegmentDuration(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 0,
		PlaylistSize:    10,
	}

	_, err := BuildHLSCommand(params)
	if err == nil {
		t.Fatal("Expected error for invalid segment duration")
	}

	if !errors.Is(err, ErrInvalidSegmentDuration) {
		t.Errorf("Expected ErrInvalidSegmentDuration, got: %v", err)
	}
}

// TestBuildHLSCommand_InvalidPlaylistSize tests invalid playlist size
func TestBuildHLSCommand_InvalidPlaylistSize(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    0,
	}

	_, err := BuildHLSCommand(params)
	if err == nil {
		t.Fatal("Expected error for invalid playlist size")
	}

	if !errors.Is(err, ErrInvalidPlaylistSize) {
		t.Errorf("Expected ErrInvalidPlaylistSize, got: %v", err)
	}
}

// TestGetOutputPath tests output path generation
func TestGetOutputPath(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		quality  string
		expected string
	}{
		{
			name:     "basic path",
			baseDir:  "/streams",
			quality:  "1080p",
			expected: "/streams/1080p/1080p.m3u8",
		},
		{
			name:     "with trailing slash",
			baseDir:  "/streams/",
			quality:  "720p",
			expected: "/streams/720p/720p.m3u8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOutputPath(tt.baseDir, tt.quality)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestGetSegmentPattern tests segment pattern generation
func TestGetSegmentPattern(t *testing.T) {
	pattern := GetSegmentPattern("channel1", "1080p")

	if !strings.Contains(pattern, "channel1") {
		t.Error("Expected pattern to contain channel ID")
	}

	if !strings.Contains(pattern, "1080p") {
		t.Error("Expected pattern to contain quality")
	}

	if !strings.Contains(pattern, "_segment_") {
		t.Error("Expected pattern to contain '_segment_'")
	}

	if !strings.Contains(pattern, "%03d") {
		t.Error("Expected pattern to contain '%03d'")
	}

	if !strings.HasSuffix(pattern, ".ts") {
		t.Error("Expected pattern to end with .ts")
	}
}

// TestGetPlaylistFilename tests playlist filename generation
func TestGetPlaylistFilename(t *testing.T) {
	tests := []struct {
		quality  string
		expected string
	}{
		{Quality1080p, "1080p.m3u8"},
		{Quality720p, "720p.m3u8"},
		{Quality480p, "480p.m3u8"},
	}

	for _, tt := range tests {
		t.Run(tt.quality, func(t *testing.T) {
			result := GetPlaylistFilename(tt.quality)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestQualityConstants tests that quality constants are defined
func TestQualityConstants(t *testing.T) {
	if Quality1080p != "1080p" {
		t.Errorf("Expected Quality1080p to be '1080p', got '%s'", Quality1080p)
	}

	if Quality720p != "720p" {
		t.Errorf("Expected Quality720p to be '720p', got '%s'", Quality720p)
	}

	if Quality480p != "480p" {
		t.Errorf("Expected Quality480p to be '480p', got '%s'", Quality480p)
	}
}

// TestBuildHLSCommand_BatchMode tests batch mode command generation
func TestBuildHLSCommand_BatchMode(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 2,
		PlaylistSize:    10,
		BatchMode:       true,
		BatchSize:       20,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify -stream_loop is NOT present in batch mode
	if containsConsecutiveArgs(cmd.Args, "-stream_loop", "-1") {
		t.Error("Did not expect stream_loop -1 in batch mode")
	}

	// Verify -t flag is present with correct duration
	// 20 segments * 2 seconds = 40 seconds
	if !containsConsecutiveArgs(cmd.Args, "-t", "40") {
		t.Error("Expected -t flag with duration 40 in batch mode")
	}

	// Verify -re flag is NOT present in batch mode
	// Batch mode always uses fast encoding (no -re flag)
	if containsArg(cmd.Args, "-re") {
		t.Error("Did not expect -re flag in batch mode")
	}
}

// TestBuildHLSCommand_BatchMode_WithSeeking tests batch mode with seeking
func TestBuildHLSCommand_BatchMode_WithSeeking(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     3600, // Seek to 1 hour
		SegmentDuration: 2,
		PlaylistSize:    10,
		BatchMode:       true,
		BatchSize:       20,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify -ss appears before -i for fast seeking
	ssIndex := findArgIndex(cmd.Args, "-ss")
	iIndex := findArgIndex(cmd.Args, "-i")

	if ssIndex == -1 {
		t.Error("Expected -ss flag for seeking")
	}

	if iIndex == -1 {
		t.Error("Expected -i flag for input")
	}

	if ssIndex >= iIndex {
		t.Error("Expected -ss flag to appear before -i flag for fast seeking")
	}

	// Verify seek value
	if ssIndex+1 >= len(cmd.Args) || cmd.Args[ssIndex+1] != "3600" {
		t.Error("Expected seek value of 3600")
	}

	// Verify -stream_loop is NOT present
	if containsConsecutiveArgs(cmd.Args, "-stream_loop", "-1") {
		t.Error("Did not expect stream_loop -1 in batch mode")
	}
}

// TestBuildHLSCommand_BatchMode_DurationCalculation tests duration calculation
func TestBuildHLSCommand_BatchMode_DurationCalculation(t *testing.T) {
	tests := []struct {
		name             string
		batchSize        int
		segmentDuration  int
		expectedDuration string
	}{
		{
			name:             "20 segments at 2 seconds",
			batchSize:        20,
			segmentDuration:  2,
			expectedDuration: "40",
		},
		{
			name:             "10 segments at 6 seconds",
			batchSize:        10,
			segmentDuration:  6,
			expectedDuration: "60",
		},
		{
			name:             "5 segments at 4 seconds",
			batchSize:        5,
			segmentDuration:  4,
			expectedDuration: "20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := StreamParams{
				InputFile:       "/media/video.mp4",
				OutputPath:      "/streams/channel1/1080p.m3u8",
				Quality:         Quality1080p,
				HardwareAccel:   HardwareAccelNone,
				SeekSeconds:     0,
				SegmentDuration: tt.segmentDuration,
				PlaylistSize:    10,
				BatchMode:       true,
				BatchSize:       tt.batchSize,
			}

			cmd, err := BuildHLSCommand(params)
			if err != nil {
				t.Fatalf("BuildHLSCommand failed: %v", err)
			}

			if !containsConsecutiveArgs(cmd.Args, "-t", tt.expectedDuration) {
				t.Errorf("Expected -t flag with duration %s, got: %v", tt.expectedDuration, cmd.Args)
			}
		})
	}
}

// TestBuildHLSCommand_BatchMode_InvalidBatchSize tests validation rejects invalid batch size
func TestBuildHLSCommand_BatchMode_InvalidBatchSize(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
		BatchMode:       true,
		BatchSize:       0, // Invalid: must be > 0 when BatchMode is true
	}

	_, err := BuildHLSCommand(params)
	if err == nil {
		t.Fatal("Expected error for invalid batch size")
	}

	if !errors.Is(err, ErrInvalidBatchSize) {
		t.Errorf("Expected ErrInvalidBatchSize, got: %v", err)
	}
}

// TestBuildHLSCommand_BatchMode_BackwardCompatibility tests that non-batch mode still works
func TestBuildHLSCommand_BatchMode_BackwardCompatibility(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0,
		SegmentDuration: 6,
		PlaylistSize:    10,
		BatchMode:       false, // Non-batch mode (default behavior)
		BatchSize:       0,     // Can be 0 when BatchMode is false
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify -stream_loop is present in non-batch mode
	if !containsConsecutiveArgs(cmd.Args, "-stream_loop", "-1") {
		t.Error("Expected stream_loop -1 in non-batch mode")
	}

	// Verify -t flag is NOT present in non-batch mode
	if containsArg(cmd.Args, "-t") {
		t.Error("Did not expect -t flag in non-batch mode")
	}
}

// TestBuildHLSCommand_BatchMode_ZeroSeekSeconds tests that SeekSeconds=0 works in batch mode
func TestBuildHLSCommand_BatchMode_ZeroSeekSeconds(t *testing.T) {
	params := StreamParams{
		InputFile:       "/media/video.mp4",
		OutputPath:      "/streams/channel1/1080p.m3u8",
		Quality:         Quality1080p,
		HardwareAccel:   HardwareAccelNone,
		SeekSeconds:     0, // No seeking
		SegmentDuration: 2,
		PlaylistSize:    10,
		BatchMode:       true,
		BatchSize:       20,
	}

	cmd, err := BuildHLSCommand(params)
	if err != nil {
		t.Fatalf("BuildHLSCommand failed: %v", err)
	}

	// Verify -ss is NOT present when SeekSeconds is 0
	if containsArg(cmd.Args, "-ss") {
		t.Error("Did not expect -ss flag when SeekSeconds is 0")
	}

	// Verify batch mode flags are still correct
	if containsConsecutiveArgs(cmd.Args, "-stream_loop", "-1") {
		t.Error("Did not expect stream_loop -1 in batch mode")
	}

	if !containsConsecutiveArgs(cmd.Args, "-t", "40") {
		t.Error("Expected -t flag with duration 40 in batch mode")
	}
}

// Helper functions for testing

// containsArg checks if an argument exists in the args slice
func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

// containsConsecutiveArgs checks if two consecutive arguments exist in the args slice
func containsConsecutiveArgs(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

// findArgIndex returns the index of an argument in the args slice, or -1 if not found
func findArgIndex(args []string, target string) int {
	for i, arg := range args {
		if arg == target {
			return i
		}
	}
	return -1
}
