package db

// Repositories provides access to all database repositories
type Repositories struct {
	Channels      *ChannelRepository
	Media         *MediaRepository
	PlaylistItems *PlaylistItemRepository
	Settings      *SettingsRepository
}

// NewRepositories creates a new repository collection
func NewRepositories(db *DB) *Repositories {
	return &Repositories{
		Channels:      NewChannelRepository(db),
		Media:         NewMediaRepository(db),
		PlaylistItems: NewPlaylistItemRepository(db),
		Settings:      NewSettingsRepository(db),
	}
}
