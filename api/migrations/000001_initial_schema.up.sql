-- Enable foreign key support
PRAGMA foreign_keys = ON;

-- Enable WAL mode for better concurrency
PRAGMA journal_mode = WAL;

-- Create channels table
CREATE TABLE IF NOT EXISTS channels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    icon TEXT,
    start_time DATETIME NOT NULL,
    loop INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create media table
CREATE TABLE IF NOT EXISTS media (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    show_name TEXT,
    season INTEGER,
    episode INTEGER,
    duration INTEGER NOT NULL,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    file_size INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create index for media queries
CREATE INDEX IF NOT EXISTS idx_media_show ON media(show_name, season, episode);
CREATE INDEX IF NOT EXISTS idx_media_path ON media(file_path);

-- Create playlist_items table
CREATE TABLE IF NOT EXISTS playlist_items (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    media_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE,
    UNIQUE (channel_id, position)
);

-- Create index for playlist queries
CREATE INDEX IF NOT EXISTS idx_playlist_channel_pos ON playlist_items(channel_id, position);

-- Create settings table (single row)
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    media_library_path TEXT NOT NULL,
    transcode_quality TEXT DEFAULT 'medium',
    hardware_accel TEXT DEFAULT 'none',
    server_port INTEGER DEFAULT 8080,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (id = 1)
);

-- Insert default settings row
INSERT OR IGNORE INTO settings (id, media_library_path) 
VALUES (1, './media');

