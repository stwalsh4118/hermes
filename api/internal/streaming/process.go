package streaming

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/stwalsh4118/hermes/internal/logger"
)

const (
	// Process termination timeouts
	terminationTimeout = 5 * time.Second
	killTimeout        = 2 * time.Second
)

// Process management errors
var (
	ErrProcessNotFound = errors.New("process not found")
	ErrProcessTimeout  = errors.New("process termination timeout")
)

// launchFFmpeg launches an FFmpeg process with the given command
func launchFFmpeg(cmd *FFmpegCommand) (*exec.Cmd, error) {
	launchStartTime := time.Now()

	if cmd == nil || len(cmd.Args) == 0 {
		return nil, errors.New("invalid FFmpeg command")
	}

	// Create exec command
	execCmd := exec.Command("ffmpeg", cmd.Args...)

	// Set up stdout and stderr pipes for logging
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	startTime := time.Now()
	if err := execCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start FFmpeg: %w", err)
	}
	startLatencyMs := time.Since(startTime).Milliseconds()
	totalLaunchLatencyMs := time.Since(launchStartTime).Milliseconds()

	// Capture output in background goroutines
	go captureFFmpegOutput(execCmd.Process.Pid, stdout, "stdout")
	go captureFFmpegOutput(execCmd.Process.Pid, stderr, "stderr")

	logger.Log.Info().
		Int("pid", execCmd.Process.Pid).
		Strs("args", cmd.Args[:minInt(5, len(cmd.Args))]).
		Int64("start_latency_ms", startLatencyMs).
		Int64("total_launch_latency_ms", totalLaunchLatencyMs).
		Time("launch_time", time.Now()).
		Msg("FFmpeg process launched")

	return execCmd, nil
}

// terminateProcess terminates a process gracefully (SIGTERM) then forcefully (SIGKILL) if needed
func terminateProcess(pid int) error {
	terminateStartTime := time.Now()

	if pid <= 0 {
		return ErrProcessNotFound
	}

	// Find process
	process, err := findProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try graceful termination first (SIGTERM)
	logger.Log.Debug().
		Int("pid", pid).
		Msg("Sending SIGTERM to FFmpeg process")

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		if errors.Is(err, syscall.ESRCH) {
			logger.Log.Debug().
				Int("pid", pid).
				Msg("Process already terminated")
			return nil
		}
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for process to exit gracefully
	exitChan := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		exitChan <- err
	}()

	select {
	case err := <-exitChan:
		// Process exited gracefully
		terminateLatencyMs := time.Since(terminateStartTime).Milliseconds()
		logger.Log.Info().
			Int("pid", pid).
			Int64("terminate_latency_ms", terminateLatencyMs).
			Msg("FFmpeg process terminated gracefully")
		return err
	case <-time.After(terminationTimeout):
		// Process didn't exit in time, force kill
		logger.Log.Warn().
			Int("pid", pid).
			Dur("timeout", terminationTimeout).
			Msg("FFmpeg process didn't exit gracefully, sending SIGKILL")

		if err := process.Kill(); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				logger.Log.Debug().
					Int("pid", pid).
					Msg("Process already terminated")
				return nil
			}
			return fmt.Errorf("failed to kill process: %w", err)
		}

		// Wait for kill to take effect
		select {
		case <-exitChan:
			terminateLatencyMs := time.Since(terminateStartTime).Milliseconds()
			logger.Log.Info().
				Int("pid", pid).
				Int64("terminate_latency_ms", terminateLatencyMs).
				Msg("FFmpeg process killed")
			return nil
		case <-time.After(killTimeout):
			terminateLatencyMs := time.Since(terminateStartTime).Milliseconds()
			logger.Log.Error().
				Int("pid", pid).
				Int64("terminate_latency_ms", terminateLatencyMs).
				Dur("kill_timeout", killTimeout).
				Msg("FFmpeg process did not die after SIGKILL")
			return fmt.Errorf("%w: process %d did not die after SIGKILL", ErrProcessTimeout, pid)
		}
	}
}

// findProcess finds a process by PID
func findProcess(pid int) (*os.Process, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}

	// On Unix, FindProcess always succeeds, so we need to check if process exists
	// by sending signal 0
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProcessNotFound, err)
	}

	return process, nil
}

// captureFFmpegOutput captures and logs output from FFmpeg process
func captureFFmpegOutput(pid int, reader io.Reader, streamName string) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		// FFmpeg outputs progress to stderr, so we log at debug level
		// Only log errors at error level
		if containsError(line) {
			logger.Log.Error().
				Int("ffmpeg_pid", pid).
				Str("stream", streamName).
				Str("output", line).
				Msg("FFmpeg error")
		} else {
			logger.Log.Debug().
				Int("ffmpeg_pid", pid).
				Str("stream", streamName).
				Str("output", line).
				Msg("FFmpeg output")
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Log.Warn().
			Err(err).
			Int("ffmpeg_pid", pid).
			Str("stream", streamName).
			Msg("Error reading FFmpeg output")
	}
}

// containsError checks if a log line contains error indicators
func containsError(line string) bool {
	errorKeywords := []string{
		"error",
		"Error",
		"ERROR",
		"failed",
		"Failed",
		"FAILED",
		"fatal",
		"Fatal",
		"FATAL",
	}

	for _, keyword := range errorKeywords {
		if contains(line, keyword) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

// containsAt checks if string contains substring at any position
func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
