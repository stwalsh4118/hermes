// Package streaming provides video streaming functionality including hardware acceleration detection.
package streaming

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/stwalsh4118/hermes/internal/logger"
)

// Timeout for FFmpeg execution
const ffmpegTimeout = 30 * time.Second

// HardwareAccel represents a hardware acceleration method for video encoding
type HardwareAccel string

// Hardware acceleration method constants
const (
	HardwareAccelNone         HardwareAccel = "none"
	HardwareAccelNVENC        HardwareAccel = "nvenc"
	HardwareAccelQSV          HardwareAccel = "qsv"
	HardwareAccelVAAPI        HardwareAccel = "vaapi"
	HardwareAccelVideoToolbox HardwareAccel = "videotoolbox"
	HardwareAccelAuto         HardwareAccel = "auto"
)

// String returns the string representation of the hardware acceleration method
func (h HardwareAccel) String() string {
	return string(h)
}

// IsValid checks if the hardware acceleration method is a known valid value
func (h HardwareAccel) IsValid() bool {
	switch h {
	case HardwareAccelNone, HardwareAccelNVENC, HardwareAccelQSV,
		HardwareAccelVAAPI, HardwareAccelVideoToolbox, HardwareAccelAuto:
		return true
	default:
		return false
	}
}

// Common errors
var (
	ErrFFmpegNotFound = errors.New("ffmpeg not found in PATH")
	ErrTimeout        = errors.New("ffmpeg detection timed out")
)

// CheckFFmpegInstalled checks if FFmpeg is available in PATH
func CheckFFmpegInstalled() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ErrFFmpegNotFound
	}
	return nil
}

// DetectHardwareEncoders probes FFmpeg for available hardware encoders
func DetectHardwareEncoders(ctx context.Context) ([]HardwareAccel, error) {
	// Check FFmpeg is available
	if err := CheckFFmpegInstalled(); err != nil {
		return nil, err
	}

	logger.Log.Debug().Msg("Detecting available hardware encoders")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, ffmpegTimeout)
	defer cancel()

	// Build FFmpeg command to list encoders
	cmd := exec.CommandContext(ctx, "ffmpeg", "-encoders", "-hide_banner")

	// Execute command
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Log.Error().Msg("FFmpeg encoder detection timed out")
			return nil, ErrTimeout
		}

		logger.Log.Error().
			Err(err).
			Msg("FFmpeg encoder detection failed")
		return nil, fmt.Errorf("failed to detect encoders: %w", err)
	}

	// Parse output for H.264 hardware encoders
	encoders := parseHardwareEncoders(string(output))

	// Convert to string slice for logging
	encoderStrs := make([]string, len(encoders))
	for i, e := range encoders {
		encoderStrs[i] = e.String()
	}

	logger.Log.Info().
		Strs("encoders", encoderStrs).
		Msg("Detected hardware encoders")

	return encoders, nil
}

// parseHardwareEncoders extracts hardware encoder names from FFmpeg output
func parseHardwareEncoders(output string) []HardwareAccel {
	encoderSet := make(map[HardwareAccel]bool)

	// Map of encoder names to look for
	encoderMap := map[string]HardwareAccel{
		"h264_nvenc":        HardwareAccelNVENC,
		"h264_qsv":          HardwareAccelQSV,
		"h264_vaapi":        HardwareAccelVAAPI,
		"h264_videotoolbox": HardwareAccelVideoToolbox,
	}

	// Parse line by line
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// FFmpeg encoder output format: " V..... encodername description"
		// Match encoder name at the start of the line after the flags
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "V") {
			continue
		}
		// Extract encoder name (skip the "V..... " prefix)
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		encoderName := parts[1]

		for targetEncoder, accelType := range encoderMap {
			if encoderName == targetEncoder {
				encoderSet[accelType] = true
				logger.Log.Debug().
					Str("encoder", targetEncoder).
					Str("type", accelType.String()).
					Msg("Found hardware encoder")
				break
			}
		}
	}

	// Always include "none" as it's always available (software encoding)
	encoderSet[HardwareAccelNone] = true

	// Convert set to slice
	encoders := make([]HardwareAccel, 0, len(encoderSet))
	for encoder := range encoderSet {
		encoders = append(encoders, encoder)
	}
	return encoders
}

// ValidateHardwareAccel validates a hardware acceleration method against available encoders
func ValidateHardwareAccel(method HardwareAccel, available []HardwareAccel) error {
	// Auto and none are always valid
	if method == HardwareAccelAuto || method == HardwareAccelNone {
		return nil
	}

	// Check if the requested method is available
	for _, encoder := range available {
		if encoder == method {
			return nil
		}
	}

	return fmt.Errorf("hardware acceleration method '%s' not available (available: %v)", method, available)
}

// SelectBestEncoder selects the best available hardware encoder from the detected list
// Priority: nvenc > qsv > videotoolbox > vaapi > none
func SelectBestEncoder(available []HardwareAccel) HardwareAccel {
	// Define priority order
	priority := []HardwareAccel{
		HardwareAccelNVENC,
		HardwareAccelQSV,
		HardwareAccelVideoToolbox,
		HardwareAccelVAAPI,
		HardwareAccelNone,
	}

	// Find the first available encoder in priority order
	for _, preferred := range priority {
		for _, encoder := range available {
			if encoder == preferred {
				logger.Log.Info().
					Str("selected", encoder.String()).
					Msg("Auto-selected hardware encoder")
				return encoder
			}
		}
	}

	// Fallback to software encoding (should always be available)
	return HardwareAccelNone
}
