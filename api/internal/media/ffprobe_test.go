package media

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCheckFFprobeInstalled(t *testing.T) {
	// This test assumes FFprobe is installed on the test system
	// In CI, FFprobe should be installed as part of setup
	err := CheckFFprobeInstalled()
	if err != nil {
		t.Skip("FFprobe not installed, skipping tests")
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name           string
		result         *FFprobeResult
		wantErr        bool
		wantDuration   int64
		wantCodec      string
		wantResolution string
	}{
		{
			name: "complete metadata",
			result: &FFprobeResult{
				Streams: []Stream{
					{
						CodecType: "video",
						CodecName: "h264",
						Width:     1920,
						Height:    1080,
						Duration:  "120.5",
					},
					{
						CodecType: "audio",
						CodecName: "aac",
					},
				},
				Format: Format{
					Duration: "120.5",
					Size:     "104857600",
				},
			},
			wantErr:        false,
			wantDuration:   120,
			wantCodec:      "h264",
			wantResolution: "1920x1080",
		},
		{
			name: "duration from format only",
			result: &FFprobeResult{
				Streams: []Stream{
					{
						CodecType: "video",
						CodecName: "hevc",
						Width:     3840,
						Height:    2160,
					},
				},
				Format: Format{
					Duration: "300.123",
					Size:     "524288000",
				},
			},
			wantErr:        false,
			wantDuration:   300,
			wantCodec:      "hevc",
			wantResolution: "3840x2160",
		},
		{
			name: "no duration",
			result: &FFprobeResult{
				Streams: []Stream{
					{
						CodecType: "video",
						CodecName: "h264",
					},
				},
				Format: Format{},
			},
			wantErr: true,
		},
		{
			name: "audio only",
			result: &FFprobeResult{
				Streams: []Stream{
					{
						CodecType: "audio",
						CodecName: "mp3",
					},
				},
				Format: Format{
					Duration: "180.5",
					Size:     "5242880",
				},
			},
			wantErr:      false,
			wantDuration: 180,
		},
		{
			name: "multiple streams",
			result: &FFprobeResult{
				Streams: []Stream{
					{
						CodecType: "video",
						CodecName: "h264",
						Width:     1280,
						Height:    720,
						Duration:  "60.0",
					},
					{
						CodecType: "audio",
						CodecName: "aac",
					},
					{
						CodecType: "audio",
						CodecName: "ac3",
					},
				},
				Format: Format{
					Duration: "60.0",
					Size:     "52428800",
				},
			},
			wantErr:        false,
			wantDuration:   60,
			wantCodec:      "h264",
			wantResolution: "1280x720",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := extractMetadata(tt.result)

			if tt.wantErr {
				if err == nil {
					t.Errorf("extractMetadata() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("extractMetadata() unexpected error: %v", err)
				return
			}

			if metadata.Duration != tt.wantDuration {
				t.Errorf("Duration = %v, want %v", metadata.Duration, tt.wantDuration)
			}

			if tt.wantCodec != "" && metadata.VideoCodec != tt.wantCodec {
				t.Errorf("VideoCodec = %v, want %v", metadata.VideoCodec, tt.wantCodec)
			}

			if tt.wantResolution != "" && metadata.Resolution != tt.wantResolution {
				t.Errorf("Resolution = %v, want %v", metadata.Resolution, tt.wantResolution)
			}
		})
	}
}

func TestFFprobeJSONParsing(t *testing.T) {
	// Test that we can parse real FFprobe JSON output format
	sampleJSON := `{
		"streams": [
			{
				"index": 0,
				"codec_name": "h264",
				"codec_long_name": "H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10",
				"codec_type": "video",
				"width": 1920,
				"height": 1080,
				"duration": "120.000000",
				"bit_rate": "5000000"
			},
			{
				"index": 1,
				"codec_name": "aac",
				"codec_long_name": "AAC (Advanced Audio Coding)",
				"codec_type": "audio",
				"channels": 2,
				"sample_rate": "48000",
				"channel_layout": "stereo"
			}
		],
		"format": {
			"filename": "test.mp4",
			"nb_streams": 2,
			"format_name": "mov,mp4,m4a,3gp,3g2,mj2",
			"format_long_name": "QuickTime / MOV",
			"duration": "120.000000",
			"size": "75000000",
			"bit_rate": "5000000"
		}
	}`

	var result FFprobeResult
	err := json.Unmarshal([]byte(sampleJSON), &result)
	if err != nil {
		t.Fatalf("Failed to parse sample JSON: %v", err)
	}

	// Verify parsing
	if len(result.Streams) != 2 {
		t.Errorf("Expected 2 streams, got %d", len(result.Streams))
	}

	if result.Streams[0].CodecType != "video" {
		t.Errorf("Expected first stream to be video, got %s", result.Streams[0].CodecType)
	}

	if result.Streams[1].CodecType != "audio" {
		t.Errorf("Expected second stream to be audio, got %s", result.Streams[1].CodecType)
	}

	if result.Format.FormatName == "" {
		t.Error("Format name should not be empty")
	}

	// Test metadata extraction from parsed result
	metadata, err := extractMetadata(&result)
	if err != nil {
		t.Fatalf("extractMetadata failed: %v", err)
	}

	if metadata.Duration != 120 {
		t.Errorf("Duration = %v, want 120", metadata.Duration)
	}

	if metadata.VideoCodec != "h264" {
		t.Errorf("VideoCodec = %v, want h264", metadata.VideoCodec)
	}

	if metadata.AudioCodec != "aac" {
		t.Errorf("AudioCodec = %v, want aac", metadata.AudioCodec)
	}

	if metadata.Resolution != "1920x1080" {
		t.Errorf("Resolution = %v, want 1920x1080", metadata.Resolution)
	}

	if metadata.FileSize != 75000000 {
		t.Errorf("FileSize = %v, want 75000000", metadata.FileSize)
	}
}

func TestProbeFileErrors(t *testing.T) {
	// Skip if FFprobe not installed
	if err := CheckFFprobeInstalled(); err != nil {
		t.Skip("FFprobe not installed, skipping integration tests")
	}

	ctx := context.Background()

	tests := []struct {
		name     string
		filePath string
		wantErr  error
	}{
		{
			name:     "non-existent file",
			filePath: "/nonexistent/file.mp4",
			wantErr:  ErrFileNotFound,
		},
		{
			name:     "empty path",
			filePath: "",
			wantErr:  ErrFileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ProbeFile(ctx, tt.filePath)

			if err == nil {
				t.Errorf("ProbeFile() expected error, got nil")
				return
			}

			// Verify the specific expected error
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ProbeFile() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProbeFileTimeout(t *testing.T) {
	// Skip if FFprobe not installed
	if err := CheckFFprobeInstalled(); err != nil {
		t.Skip("FFprobe not installed, skipping integration tests")
	}

	// Test with pre-expired context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep to ensure context is expired before calling ProbeFile
	time.Sleep(10 * time.Millisecond)

	_, err := ProbeFile(ctx, "/tmp/nonexistent-file-for-timeout-test.mp4")
	if err == nil {
		t.Error("ProbeFile() expected error with expired context, got nil")
		return
	}

	// Verify the error is ErrTimeout (which is returned when context deadline exceeded)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("ProbeFile() with expired context error = %v, want ErrTimeout", err)
	}
}

func TestProbeFileContext(t *testing.T) {
	// Skip if FFprobe not installed
	if err := CheckFFprobeInstalled(); err != nil {
		t.Skip("FFprobe not installed, skipping integration tests")
	}

	// Test that context cancellation is respected
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := ProbeFile(ctx, "/nonexistent/file.mp4")
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}
