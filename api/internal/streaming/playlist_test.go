package streaming

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestGenerateMasterPlaylist tests master playlist generation
func TestGenerateMasterPlaylist(t *testing.T) {
	tests := []struct {
		name        string
		variants    []PlaylistVariant
		wantErr     bool
		errContains string
		validate    func(t *testing.T, content string)
	}{
		{
			name: "valid single variant",
			variants: []PlaylistVariant{
				{
					Bandwidth:  5192000,
					Resolution: "1920x1080",
					Path:       "1080p.m3u8",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "#EXTM3U") {
					t.Error("Missing #EXTM3U tag")
				}
				if !strings.Contains(content, "#EXT-X-VERSION:3") {
					t.Error("Missing version tag")
				}
				if !strings.Contains(content, "BANDWIDTH=5192000") {
					t.Error("Missing or incorrect bandwidth")
				}
				if !strings.Contains(content, "RESOLUTION=1920x1080") {
					t.Error("Missing or incorrect resolution")
				}
				if !strings.Contains(content, "1080p.m3u8") {
					t.Error("Missing playlist path")
				}
			},
		},
		{
			name: "valid three variants",
			variants: []PlaylistVariant{
				{Bandwidth: 5192000, Resolution: "1920x1080", Path: "1080p.m3u8"},
				{Bandwidth: 3192000, Resolution: "1280x720", Path: "720p.m3u8"},
				{Bandwidth: 1692000, Resolution: "854x480", Path: "480p.m3u8"},
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if strings.Count(content, "#EXT-X-STREAM-INF:") != 3 {
					t.Error("Should have 3 stream variants")
				}
				if !strings.Contains(content, "1080p.m3u8") ||
					!strings.Contains(content, "720p.m3u8") ||
					!strings.Contains(content, "480p.m3u8") {
					t.Error("Missing one or more playlist paths")
				}
			},
		},
		{
			name:        "empty variants",
			variants:    []PlaylistVariant{},
			wantErr:     true,
			errContains: "at least one variant",
		},
		{
			name: "invalid bandwidth",
			variants: []PlaylistVariant{
				{Bandwidth: 0, Resolution: "1920x1080", Path: "1080p.m3u8"},
			},
			wantErr:     true,
			errContains: "bandwidth must be positive",
		},
		{
			name: "invalid resolution format",
			variants: []PlaylistVariant{
				{Bandwidth: 5192000, Resolution: "invalid", Path: "1080p.m3u8"},
			},
			wantErr:     true,
			errContains: "invalid resolution format",
		},
		{
			name: "empty path",
			variants: []PlaylistVariant{
				{Bandwidth: 5192000, Resolution: "1920x1080", Path: ""},
			},
			wantErr:     true,
			errContains: "path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GenerateMasterPlaylist(tt.variants)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error '%s' does not contain '%s'", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, content)
			}
		})
	}
}

// TestGenerateMediaPlaylist tests media playlist generation
//
//nolint:gocyclo // Table-driven test with comprehensive coverage
func TestGenerateMediaPlaylist(t *testing.T) {
	tests := []struct {
		name        string
		segments    []Segment
		config      MediaPlaylistConfig
		wantErr     bool
		errContains string
		validate    func(t *testing.T, content string)
	}{
		{
			name:     "empty playlist EVENT type",
			segments: []Segment{},
			config: MediaPlaylistConfig{
				TargetDuration: 6,
				MediaSequence:  0,
				PlaylistType:   PlaylistTypeEvent,
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "#EXTM3U") {
					t.Error("Missing #EXTM3U tag")
				}
				if !strings.Contains(content, "#EXT-X-VERSION:3") {
					t.Error("Missing version tag")
				}
				if !strings.Contains(content, "#EXT-X-TARGETDURATION:6") {
					t.Error("Missing or incorrect target duration")
				}
				if !strings.Contains(content, "#EXT-X-MEDIA-SEQUENCE:0") {
					t.Error("Missing or incorrect media sequence")
				}
				if !strings.Contains(content, "#EXT-X-PLAYLIST-TYPE:EVENT") {
					t.Error("Missing playlist type")
				}
				if strings.Contains(content, "#EXT-X-ENDLIST") {
					t.Error("EVENT playlist should not have ENDLIST tag")
				}
			},
		},
		{
			name: "single segment",
			segments: []Segment{
				{Duration: 6.0, Path: "segment_000.ts"},
			},
			config: MediaPlaylistConfig{
				TargetDuration: 6,
				MediaSequence:  0,
				PlaylistType:   PlaylistTypeEvent,
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "#EXTINF:6.0,") {
					t.Error("Missing segment info")
				}
				if !strings.Contains(content, "segment_000.ts") {
					t.Error("Missing segment path")
				}
			},
		},
		{
			name: "multiple segments",
			segments: []Segment{
				{Duration: 6.0, Path: "segment_000.ts"},
				{Duration: 6.0, Path: "segment_001.ts"},
				{Duration: 6.0, Path: "segment_002.ts"},
			},
			config: MediaPlaylistConfig{
				TargetDuration: 6,
				MediaSequence:  0,
				PlaylistType:   PlaylistTypeEvent,
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if strings.Count(content, "#EXTINF:") != 3 {
					t.Error("Should have 3 segments")
				}
			},
		},
		{
			name: "VOD playlist type",
			segments: []Segment{
				{Duration: 6.0, Path: "segment_000.ts"},
			},
			config: MediaPlaylistConfig{
				TargetDuration: 6,
				MediaSequence:  0,
				PlaylistType:   PlaylistTypeVOD,
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if !strings.Contains(content, "#EXT-X-PLAYLIST-TYPE:VOD") {
					t.Error("Missing VOD playlist type")
				}
				if !strings.Contains(content, "#EXT-X-ENDLIST") {
					t.Error("VOD playlist should have ENDLIST tag")
				}
			},
		},
		{
			name: "sliding window with max segments",
			segments: []Segment{
				{Duration: 6.0, Path: "segment_000.ts"},
				{Duration: 6.0, Path: "segment_001.ts"},
				{Duration: 6.0, Path: "segment_002.ts"},
				{Duration: 6.0, Path: "segment_003.ts"},
				{Duration: 6.0, Path: "segment_004.ts"},
			},
			config: MediaPlaylistConfig{
				TargetDuration: 6,
				MediaSequence:  0,
				PlaylistType:   PlaylistTypeEvent,
				MaxSegments:    3,
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if strings.Count(content, "#EXTINF:") != 3 {
					t.Error("Should have only 3 segments (sliding window)")
				}
				// Should keep last 3 segments
				if !strings.Contains(content, "segment_002.ts") ||
					!strings.Contains(content, "segment_003.ts") ||
					!strings.Contains(content, "segment_004.ts") {
					t.Error("Should keep last 3 segments")
				}
				if strings.Contains(content, "segment_000.ts") ||
					strings.Contains(content, "segment_001.ts") {
					t.Error("Should not contain first 2 segments")
				}
				// MediaSequence should be updated to 2 (dropped 2 segments)
				if !strings.Contains(content, "#EXT-X-MEDIA-SEQUENCE:2") {
					t.Error("MediaSequence should be 2 after dropping first 2 segments")
				}
			},
		},
		{
			name: "sliding window with non-zero media sequence",
			segments: []Segment{
				{Duration: 6.0, Path: "segment_010.ts"},
				{Duration: 6.0, Path: "segment_011.ts"},
				{Duration: 6.0, Path: "segment_012.ts"},
				{Duration: 6.0, Path: "segment_013.ts"},
				{Duration: 6.0, Path: "segment_014.ts"},
				{Duration: 6.0, Path: "segment_015.ts"},
			},
			config: MediaPlaylistConfig{
				TargetDuration: 6,
				MediaSequence:  10, // Starting at sequence 10
				PlaylistType:   PlaylistTypeEvent,
				MaxSegments:    4,
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				if strings.Count(content, "#EXTINF:") != 4 {
					t.Error("Should have only 4 segments (sliding window)")
				}
				// Should keep last 4 segments (012-015)
				if !strings.Contains(content, "segment_012.ts") ||
					!strings.Contains(content, "segment_013.ts") ||
					!strings.Contains(content, "segment_014.ts") ||
					!strings.Contains(content, "segment_015.ts") {
					t.Error("Should keep last 4 segments")
				}
				// MediaSequence should be 12 (original 10 + 2 dropped segments)
				if !strings.Contains(content, "#EXT-X-MEDIA-SEQUENCE:12") {
					t.Error("MediaSequence should be 12 (10 + 2 dropped segments)")
				}
			},
		},
		{
			name: "auto calculate target duration",
			segments: []Segment{
				{Duration: 5.5, Path: "segment_000.ts"},
				{Duration: 6.2, Path: "segment_001.ts"},
				{Duration: 5.8, Path: "segment_002.ts"},
			},
			config: MediaPlaylistConfig{
				TargetDuration: 0, // Auto-calculate
				MediaSequence:  0,
				PlaylistType:   PlaylistTypeEvent,
			},
			wantErr: false,
			validate: func(t *testing.T, content string) {
				// Should be ceiling of max duration (6.2 -> 7)
				if !strings.Contains(content, "#EXT-X-TARGETDURATION:7") {
					t.Error("Target duration should be 7 (ceiling of 6.2)")
				}
			},
		},
		{
			name:     "invalid playlist type",
			segments: []Segment{},
			config: MediaPlaylistConfig{
				TargetDuration: 6,
				MediaSequence:  0,
				PlaylistType:   "INVALID",
			},
			wantErr:     true,
			errContains: "invalid playlist type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GenerateMediaPlaylist(tt.segments, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error '%s' does not contain '%s'", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, content)
			}
		})
	}
}

// TestDiscoverSegments tests segment discovery from directory
func TestDiscoverSegments(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantCount   int
		wantErr     bool
		errContains string
		validate    func(t *testing.T, segments []Segment)
	}{
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				dir := filepath.Join(tempDir, "empty")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "directory with segments",
			setup: func(t *testing.T) string {
				dir := filepath.Join(tempDir, "with_segments")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create test segment files
				for i := 0; i < 5; i++ {
					path := filepath.Join(dir, "test_1080p_segment_"+padSequence(i)+".ts")
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				return dir
			},
			wantCount: 5,
			wantErr:   false,
			validate: func(t *testing.T, segments []Segment) {
				// Check segments are sorted by sequence
				for i, seg := range segments {
					expectedPath := "test_1080p_segment_" + padSequence(i) + ".ts"
					if seg.Path != expectedPath {
						t.Errorf("Segment %d: expected path %s, got %s", i, expectedPath, seg.Path)
					}
					if seg.Duration != defaultSegmentLength {
						t.Errorf("Segment %d: expected duration %.1f, got %.1f", i, defaultSegmentLength, seg.Duration)
					}
				}
			},
		},
		{
			name: "directory with non-sequential segments",
			setup: func(t *testing.T) string {
				dir := filepath.Join(tempDir, "non_sequential")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create segments out of order
				for _, i := range []int{2, 0, 4, 1, 3} {
					path := filepath.Join(dir, "channel_720p_segment_"+padSequence(i)+".ts")
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				return dir
			},
			wantCount: 5,
			wantErr:   false,
			validate: func(t *testing.T, segments []Segment) {
				// Check segments are sorted correctly
				for i := 0; i < 5; i++ {
					expectedPath := "channel_720p_segment_" + padSequence(i) + ".ts"
					if segments[i].Path != expectedPath {
						t.Errorf("Segment not sorted correctly at position %d", i)
					}
				}
			},
		},
		{
			name: "directory with mixed files",
			setup: func(t *testing.T) string {
				dir := filepath.Join(tempDir, "mixed_files")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create segment files
				for i := 0; i < 3; i++ {
					path := filepath.Join(dir, "test_segment_"+padSequence(i)+".ts")
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				// Create non-segment files (should be ignored)
				_ = os.WriteFile(filepath.Join(dir, "playlist.m3u8"), []byte("test"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("test"), 0644)
				return dir
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "non-existent directory",
			setup: func(_ *testing.T) string {
				return filepath.Join(tempDir, "does_not_exist")
			},
			wantCount: 0,
			wantErr:   false, // Should return empty list, not error
		},
		{
			name: "file instead of directory",
			setup: func(t *testing.T) string {
				path := filepath.Join(tempDir, "not_a_dir")
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr:     true,
			errContains: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			segments, err := DiscoverSegments(dir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error '%s' does not contain '%s'", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(segments) != tt.wantCount {
				t.Errorf("Expected %d segments, got %d", tt.wantCount, len(segments))
			}

			if tt.validate != nil {
				tt.validate(t, segments)
			}
		})
	}
}

// TestWritePlaylistAtomic tests atomic playlist writing
func TestWritePlaylistAtomic(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		content  string
		wantErr  bool
		validate func(t *testing.T, path string)
	}{
		{
			name:    "write new playlist",
			path:    filepath.Join(tempDir, "test1", "playlist.m3u8"),
			content: "#EXTM3U\n#EXT-X-VERSION:3\n",
			wantErr: false,
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if string(data) != "#EXTM3U\n#EXT-X-VERSION:3\n" {
					t.Error("Content mismatch")
				}
			},
		},
		{
			name:    "overwrite existing playlist",
			path:    filepath.Join(tempDir, "test2", "playlist.m3u8"),
			content: "new content\n",
			wantErr: false,
			validate: func(t *testing.T, path string) {
				// First write
				if err := WritePlaylistAtomic(path, "old content\n"); err != nil {
					t.Fatal(err)
				}
				// Second write should overwrite
				if err := WritePlaylistAtomic(path, "new content\n"); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "new content\n" {
					t.Error("File not overwritten correctly")
				}
			},
		},
		{
			name:    "create nested directories",
			path:    filepath.Join(tempDir, "nested", "dir", "structure", "playlist.m3u8"),
			content: "test\n",
			wantErr: false,
			validate: func(t *testing.T, path string) {
				if _, err := os.Stat(path); err != nil {
					t.Error("Nested directories not created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WritePlaylistAtomic(tt.path, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, tt.path)
			}
		})
	}
}

// TestWritePlaylistAtomicConcurrent tests concurrent writes are safe
func TestWritePlaylistAtomicConcurrent(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "concurrent.m3u8")

	var wg sync.WaitGroup
	numWriters := 10

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			content := "writer " + string(rune('0'+id)) + "\n"
			if err := WritePlaylistAtomic(path, content); err != nil {
				t.Errorf("Write failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// File should exist and contain valid content from one of the writers
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Content should be complete (not corrupted)
	if len(data) == 0 {
		t.Error("File is empty")
	}
}

// TestValidateMasterPlaylist tests master playlist validation
func TestValidateMasterPlaylist(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid master playlist",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=5192000,RESOLUTION=1920x1080
1080p.m3u8
`,
			wantErr: false,
		},
		{
			name: "valid with multiple variants",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=5192000,RESOLUTION=1920x1080
1080p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=3192000,RESOLUTION=1280x720
720p.m3u8
`,
			wantErr: false,
		},
		{
			name:        "missing EXTM3U",
			content:     "#EXT-X-VERSION:3\n",
			wantErr:     true,
			errContains: "#EXTM3U",
		},
		{
			name:        "missing version",
			content:     "#EXTM3U\n",
			wantErr:     true,
			errContains: "#EXT-X-VERSION",
		},
		{
			name: "missing STREAM-INF",
			content: `#EXTM3U
#EXT-X-VERSION:3
`,
			wantErr:     true,
			errContains: "#EXT-X-STREAM-INF",
		},
		{
			name: "missing BANDWIDTH in STREAM-INF",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:RESOLUTION=1920x1080
1080p.m3u8
`,
			wantErr:     true,
			errContains: "BANDWIDTH",
		},
		{
			name: "missing RESOLUTION in STREAM-INF",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=5192000
1080p.m3u8
`,
			wantErr:     true,
			errContains: "RESOLUTION",
		},
		{
			name:        "empty content",
			content:     "",
			wantErr:     true,
			errContains: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMasterPlaylist(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error '%s' does not contain '%s'", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestValidateMediaPlaylist tests media playlist validation
func TestValidateMediaPlaylist(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid media playlist",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:6.0,
segment_000.ts
`,
			wantErr: false,
		},
		{
			name: "valid empty playlist",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
`,
			wantErr: false,
		},
		{
			name:        "missing EXTM3U",
			content:     "#EXT-X-VERSION:3\n",
			wantErr:     true,
			errContains: "#EXTM3U",
		},
		{
			name: "missing version",
			content: `#EXTM3U
#EXT-X-TARGETDURATION:6
`,
			wantErr:     true,
			errContains: "#EXT-X-VERSION",
		},
		{
			name: "missing target duration",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:0
`,
			wantErr:     true,
			errContains: "#EXT-X-TARGETDURATION",
		},
		{
			name: "missing media sequence",
			content: `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
`,
			wantErr:     true,
			errContains: "#EXT-X-MEDIA-SEQUENCE",
		},
		{
			name:        "empty content",
			content:     "",
			wantErr:     true,
			errContains: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMediaPlaylist(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error '%s' does not contain '%s'", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestGetBandwidthForQuality tests bandwidth calculation
func TestGetBandwidthForQuality(t *testing.T) {
	tests := []struct {
		name    string
		quality string
		want    int
		wantErr bool
	}{
		{"1080p", Quality1080p, 5192000, false},
		{"720p", Quality720p, 3192000, false},
		{"480p", Quality480p, 1692000, false},
		{"invalid", "4k", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBandwidthForQuality(tt.quality)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("GetBandwidthForQuality() = %d, want %d", got, tt.want)
				}
			}
		})
	}
}

// TestGetResolutionForQuality tests resolution lookup
func TestGetResolutionForQuality(t *testing.T) {
	tests := []struct {
		name    string
		quality string
		want    string
		wantErr bool
	}{
		{"1080p", Quality1080p, "1920x1080", false},
		{"720p", Quality720p, "1280x720", false},
		{"480p", Quality480p, "854x480", false},
		{"invalid", "4k", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetResolutionForQuality(tt.quality)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("GetResolutionForQuality() = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

// Helper function to pad sequence numbers
func padSequence(seq int) string {
	return strings.Repeat("0", 3-len(strconv.Itoa(seq))) + strconv.Itoa(seq)
}
