package config

import (
	"os"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	// Create a temporary config file (empty to test defaults)
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()
	_ = tmpFile.Close()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test server defaults
	if cfg.Server.Port != defaultServerPort {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, defaultServerPort)
	}
	if cfg.Server.Host != defaultServerHost {
		t.Errorf("Server.Host = %s, want %s", cfg.Server.Host, defaultServerHost)
	}

	// Test database defaults
	if cfg.Database.Path != defaultDatabasePath {
		t.Errorf("Database.Path = %s, want %s", cfg.Database.Path, defaultDatabasePath)
	}
	if cfg.Database.EnableWAL != defaultDatabaseEnableWAL {
		t.Errorf("Database.EnableWAL = %v, want %v", cfg.Database.EnableWAL, defaultDatabaseEnableWAL)
	}

	// Test logging defaults
	if cfg.Logging.Level != defaultLogLevel {
		t.Errorf("Logging.Level = %s, want %s", cfg.Logging.Level, defaultLogLevel)
	}
	if cfg.Logging.Pretty != defaultLogPretty {
		t.Errorf("Logging.Pretty = %v, want %v", cfg.Logging.Pretty, defaultLogPretty)
	}

	// Test streaming defaults
	if cfg.Streaming.HardwareAccel != defaultStreamingHardwareAccel {
		t.Errorf("Streaming.HardwareAccel = %s, want %s", cfg.Streaming.HardwareAccel, defaultStreamingHardwareAccel)
	}
	if cfg.Streaming.SegmentDuration != defaultStreamingSegmentDuration {
		t.Errorf("Streaming.SegmentDuration = %d, want %d", cfg.Streaming.SegmentDuration, defaultStreamingSegmentDuration)
	}
	if cfg.Streaming.PlaylistSize != defaultStreamingPlaylistSize {
		t.Errorf("Streaming.PlaylistSize = %d, want %d", cfg.Streaming.PlaylistSize, defaultStreamingPlaylistSize)
	}
	if cfg.Streaming.SegmentPath != defaultStreamingSegmentPath {
		t.Errorf("Streaming.SegmentPath = %s, want %s", cfg.Streaming.SegmentPath, defaultStreamingSegmentPath)
	}
	if cfg.Streaming.GracePeriodSeconds != defaultStreamingGracePeriod {
		t.Errorf("Streaming.GracePeriodSeconds = %d, want %d", cfg.Streaming.GracePeriodSeconds, defaultStreamingGracePeriod)
	}
	if cfg.Streaming.CleanupInterval != defaultStreamingCleanupInterval {
		t.Errorf("Streaming.CleanupInterval = %d, want %d", cfg.Streaming.CleanupInterval, defaultStreamingCleanupInterval)
	}
	if cfg.Streaming.BatchSize != defaultStreamingBatchSize {
		t.Errorf("Streaming.BatchSize = %d, want %d", cfg.Streaming.BatchSize, defaultStreamingBatchSize)
	}
	if cfg.Streaming.TriggerThreshold != defaultStreamingTriggerThreshold {
		t.Errorf("Streaming.TriggerThreshold = %d, want %d", cfg.Streaming.TriggerThreshold, defaultStreamingTriggerThreshold)
	}
	if cfg.Streaming.StreamSegmentDuration != defaultStreamSegmentDuration {
		t.Errorf("Streaming.StreamSegmentDuration = %d, want %d", cfg.Streaming.StreamSegmentDuration, defaultStreamSegmentDuration)
	}
	if cfg.Streaming.StreamSegmentFilenamePattern != defaultStreamSegmentFilenamePattern {
		t.Errorf("Streaming.StreamSegmentFilenamePattern = %s, want %s", cfg.Streaming.StreamSegmentFilenamePattern, defaultStreamSegmentFilenamePattern)
	}
	if cfg.Streaming.FPS != defaultFPS {
		t.Errorf("Streaming.FPS = %d, want %d", cfg.Streaming.FPS, defaultFPS)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
					EnableWAL:         true,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Pretty: false,
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid server port (too low)",
			config: Config{
				Server: ServerConfig{
					Port:         0,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid hardware acceleration",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "invalid",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid segment duration",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              0,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid playlist size",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 -1,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "empty segment path",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid cleanup interval",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              0,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "all hardware acceleration options valid",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "nvenc",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: false,
		},
		{
			name: "grace period can be zero",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           0,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid batch size (zero)",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    0,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid batch size (negative)",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    -1,
					TriggerThreshold:             5,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid trigger threshold (zero)",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             0,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid trigger threshold (negative)",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             -1,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid trigger threshold (greater than or equal to batch size)",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    20,
					TriggerThreshold:             20,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: true,
		},
		{
			name: "valid batch configuration",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  defaultReadTimeout,
					WriteTimeout: defaultWriteTimeout,
				},
				Database: DatabaseConfig{
					Path:              "./data/hermes.db",
					ConnectionTimeout: defaultDatabaseConnectionTimeout,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Streaming: StreamingConfig{
					HardwareAccel:                "auto",
					SegmentDuration:              6,
					PlaylistSize:                 10,
					SegmentPath:                  "./data/streams",
					GracePeriodSeconds:           30,
					CleanupInterval:              60,
					EncodingPreset:               "ultrafast",
					BatchSize:                    30,
					TriggerThreshold:             10,
					StreamSegmentDuration:        4,
					StreamSegmentFilenamePattern: "seg-%Y%m%dT%H%M%S.ts",
					FPS:                          30,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStreamingConfigEnvVars(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("HERMES_STREAMING_HARDWAREACCEL", "nvenc")
	_ = os.Setenv("HERMES_STREAMING_SEGMENTDURATION", "10")
	_ = os.Setenv("HERMES_STREAMING_PLAYLISTSIZE", "15")
	_ = os.Setenv("HERMES_STREAMING_SEGMENTPATH", "/custom/path")
	_ = os.Setenv("HERMES_STREAMING_GRACEPERIODSECONDS", "45")
	_ = os.Setenv("HERMES_STREAMING_CLEANUPINTERVAL", "90")
	_ = os.Setenv("HERMES_STREAMING_BATCHSIZE", "25")
	_ = os.Setenv("HERMES_STREAMING_TRIGGERTHRESHOLD", "8")
	defer func() {
		_ = os.Unsetenv("HERMES_STREAMING_HARDWAREACCEL")
		_ = os.Unsetenv("HERMES_STREAMING_SEGMENTDURATION")
		_ = os.Unsetenv("HERMES_STREAMING_PLAYLISTSIZE")
		_ = os.Unsetenv("HERMES_STREAMING_SEGMENTPATH")
		_ = os.Unsetenv("HERMES_STREAMING_GRACEPERIODSECONDS")
		_ = os.Unsetenv("HERMES_STREAMING_CLEANUPINTERVAL")
		_ = os.Unsetenv("HERMES_STREAMING_BATCHSIZE")
		_ = os.Unsetenv("HERMES_STREAMING_TRIGGERTHRESHOLD")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Streaming.HardwareAccel != "nvenc" {
		t.Errorf("Streaming.HardwareAccel = %s, want nvenc", cfg.Streaming.HardwareAccel)
	}
	if cfg.Streaming.SegmentDuration != 10 {
		t.Errorf("Streaming.SegmentDuration = %d, want 10", cfg.Streaming.SegmentDuration)
	}
	if cfg.Streaming.PlaylistSize != 15 {
		t.Errorf("Streaming.PlaylistSize = %d, want 15", cfg.Streaming.PlaylistSize)
	}
	if cfg.Streaming.SegmentPath != "/custom/path" {
		t.Errorf("Streaming.SegmentPath = %s, want /custom/path", cfg.Streaming.SegmentPath)
	}
	if cfg.Streaming.GracePeriodSeconds != 45 {
		t.Errorf("Streaming.GracePeriodSeconds = %d, want 45", cfg.Streaming.GracePeriodSeconds)
	}
	if cfg.Streaming.CleanupInterval != 90 {
		t.Errorf("Streaming.CleanupInterval = %d, want 90", cfg.Streaming.CleanupInterval)
	}
	if cfg.Streaming.BatchSize != 25 {
		t.Errorf("Streaming.BatchSize = %d, want 25", cfg.Streaming.BatchSize)
	}
	if cfg.Streaming.TriggerThreshold != 8 {
		t.Errorf("Streaming.TriggerThreshold = %d, want 8", cfg.Streaming.TriggerThreshold)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item exists",
			slice: []string{"one", "two", "three"},
			item:  "two",
			want:  true,
		},
		{
			name:  "item does not exist",
			slice: []string{"one", "two", "three"},
			item:  "four",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "one",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
