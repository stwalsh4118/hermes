//go:build !windows

package streaming

import (
	"fmt"
	"syscall"
)

// getAvailableSpace returns available disk space in bytes for the given path (Unix implementation)
func getAvailableSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to stat filesystem: %w", err)
	}

	// Available space = Available blocks * Block size
	available := stat.Bavail * uint64(stat.Bsize)
	return available, nil
}
