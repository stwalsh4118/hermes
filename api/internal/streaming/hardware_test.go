package streaming

import (
	"context"
	"errors"
	"testing"
)

func TestCheckFFmpegInstalled(t *testing.T) {
	// This test assumes FFmpeg is installed
	// In CI/CD, ensure FFmpeg is available or skip test
	err := CheckFFmpegInstalled()
	if err != nil {
		t.Skipf("FFmpeg not installed, skipping test: %v", err)
	}
}

func TestHardwareAccel_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		accel HardwareAccel
		want  bool
	}{
		{"none is valid", HardwareAccelNone, true},
		{"nvenc is valid", HardwareAccelNVENC, true},
		{"qsv is valid", HardwareAccelQSV, true},
		{"vaapi is valid", HardwareAccelVAAPI, true},
		{"videotoolbox is valid", HardwareAccelVideoToolbox, true},
		{"auto is valid", HardwareAccelAuto, true},
		{"invalid string is not valid", HardwareAccel("invalid"), false},
		{"empty string is not valid", HardwareAccel(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.accel.IsValid(); got != tt.want {
				t.Errorf("HardwareAccel.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHardwareAccel_String(t *testing.T) {
	tests := []struct {
		name  string
		accel HardwareAccel
		want  string
	}{
		{"none", HardwareAccelNone, "none"},
		{"nvenc", HardwareAccelNVENC, "nvenc"},
		{"qsv", HardwareAccelQSV, "qsv"},
		{"vaapi", HardwareAccelVAAPI, "vaapi"},
		{"videotoolbox", HardwareAccelVideoToolbox, "videotoolbox"},
		{"auto", HardwareAccelAuto, "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.accel.String(); got != tt.want {
				t.Errorf("HardwareAccel.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHardwareEncoders(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected map[HardwareAccel]bool // Use map for order-independent comparison
	}{
		{
			name: "NVIDIA encoder only",
			output: `Encoders:
 V..... h264_nvenc           NVIDIA NVENC H.264 encoder (codec h264)
 V..... libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10`,
			expected: map[HardwareAccel]bool{
				HardwareAccelNVENC: true,
				HardwareAccelNone:  true,
			},
		},
		{
			name: "Intel QSV only",
			output: `Encoders:
 V..... h264_qsv             H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (Intel Quick Sync Video acceleration)
 V..... libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10`,
			expected: map[HardwareAccel]bool{
				HardwareAccelQSV:  true,
				HardwareAccelNone: true,
			},
		},
		{
			name: "VAAPI only",
			output: `Encoders:
 V..... h264_vaapi           H.264/AVC (VAAPI) (codec h264)
 V..... libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10`,
			expected: map[HardwareAccel]bool{
				HardwareAccelVAAPI: true,
				HardwareAccelNone:  true,
			},
		},
		{
			name: "VideoToolbox only",
			output: `Encoders:
 V..... h264_videotoolbox    VideoToolbox H.264 Encoder (codec h264)
 V..... libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10`,
			expected: map[HardwareAccel]bool{
				HardwareAccelVideoToolbox: true,
				HardwareAccelNone:         true,
			},
		},
		{
			name: "Multiple encoders",
			output: `Encoders:
 V..... h264_nvenc           NVIDIA NVENC H.264 encoder (codec h264)
 V..... h264_qsv             H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (Intel Quick Sync Video acceleration)
 V..... h264_vaapi           H.264/AVC (VAAPI) (codec h264)
 V..... libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10`,
			expected: map[HardwareAccel]bool{
				HardwareAccelNVENC: true,
				HardwareAccelQSV:   true,
				HardwareAccelVAAPI: true,
				HardwareAccelNone:  true,
			},
		},
		{
			name: "No hardware encoders",
			output: `Encoders:
 V..... libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10
 V..... libx265              libx265 H.265 / HEVC`,
			expected: map[HardwareAccel]bool{
				HardwareAccelNone: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHardwareEncoders(tt.output)

			// Convert slice to map for comparison
			resultMap := make(map[HardwareAccel]bool)
			for _, encoder := range result {
				resultMap[encoder] = true
			}

			// Check all expected encoders are present
			for expectedEncoder := range tt.expected {
				if !resultMap[expectedEncoder] {
					t.Errorf("Expected encoder %s not found in result: %v", expectedEncoder, result)
				}
			}

			// Check no unexpected encoders are present
			for _, encoder := range result {
				if !tt.expected[encoder] {
					t.Errorf("Unexpected encoder %s found in result", encoder)
				}
			}
		})
	}
}

func TestValidateHardwareAccel(t *testing.T) {
	available := []HardwareAccel{HardwareAccelNVENC, HardwareAccelQSV, HardwareAccelNone}

	tests := []struct {
		name      string
		method    HardwareAccel
		available []HardwareAccel
		wantErr   bool
	}{
		{
			name:      "Auto is always valid",
			method:    HardwareAccelAuto,
			available: available,
			wantErr:   false,
		},
		{
			name:      "None is always valid",
			method:    HardwareAccelNone,
			available: available,
			wantErr:   false,
		},
		{
			name:      "Available encoder is valid",
			method:    HardwareAccelNVENC,
			available: available,
			wantErr:   false,
		},
		{
			name:      "Unavailable encoder is invalid",
			method:    HardwareAccelVAAPI,
			available: available,
			wantErr:   true,
		},
		{
			name:      "Invalid method is invalid",
			method:    HardwareAccel("invalid"),
			available: available,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHardwareAccel(tt.method, tt.available)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHardwareAccel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSelectBestEncoder(t *testing.T) {
	tests := []struct {
		name      string
		available []HardwareAccel
		want      HardwareAccel
	}{
		{
			name:      "NVENC has highest priority",
			available: []HardwareAccel{HardwareAccelVAAPI, HardwareAccelNVENC, HardwareAccelQSV, HardwareAccelNone},
			want:      HardwareAccelNVENC,
		},
		{
			name:      "QSV is second priority",
			available: []HardwareAccel{HardwareAccelVAAPI, HardwareAccelQSV, HardwareAccelNone},
			want:      HardwareAccelQSV,
		},
		{
			name:      "VideoToolbox is third priority",
			available: []HardwareAccel{HardwareAccelVAAPI, HardwareAccelVideoToolbox, HardwareAccelNone},
			want:      HardwareAccelVideoToolbox,
		},
		{
			name:      "VAAPI is fourth priority",
			available: []HardwareAccel{HardwareAccelVAAPI, HardwareAccelNone},
			want:      HardwareAccelVAAPI,
		},
		{
			name:      "None is fallback",
			available: []HardwareAccel{HardwareAccelNone},
			want:      HardwareAccelNone,
		},
		{
			name:      "Empty list defaults to none",
			available: []HardwareAccel{},
			want:      HardwareAccelNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectBestEncoder(tt.available)
			if got != tt.want {
				t.Errorf("SelectBestEncoder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectHardwareEncoders(t *testing.T) {
	// This is an integration test that requires FFmpeg to be installed
	ctx := context.Background()

	encoders, err := DetectHardwareEncoders(ctx)
	if err != nil {
		if errors.Is(err, ErrFFmpegNotFound) {
			t.Skip("FFmpeg not installed, skipping integration test")
		}
		t.Fatalf("DetectHardwareEncoders() unexpected error: %v", err)
	}

	// Should always have at least "none" (software encoding)
	if len(encoders) == 0 {
		t.Error("DetectHardwareEncoders() returned empty list, expected at least 'none'")
	}

	// Check that "none" is present
	hasNone := false
	for _, encoder := range encoders {
		if encoder == HardwareAccelNone {
			hasNone = true
			break
		}
	}
	if !hasNone {
		t.Error("DetectHardwareEncoders() missing 'none' encoder")
	}

	// Validate all returned encoders are known types (use IsValid method)
	for _, encoder := range encoders {
		if !encoder.IsValid() {
			t.Errorf("DetectHardwareEncoders() returned unknown encoder: %s", encoder)
		}
	}
}

func TestDetectHardwareEncoders_Timeout(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := DetectHardwareEncoders(ctx)
	if err == nil {
		t.Error("DetectHardwareEncoders() with cancelled context should return error")
	}
}
