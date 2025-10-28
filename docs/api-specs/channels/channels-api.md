# Channel Management API

Last Updated: 2025-10-28

## Service Interfaces

### ChannelService (Go)

Location: `internal/channel/service.go`

```go
type ChannelService struct {
    repos *db.Repositories
}

func NewChannelService(repos *db.Repositories) *ChannelService

// CRUD Operations
func (s *ChannelService) CreateChannel(ctx context.Context, name string, icon *string, startTime time.Time, loop bool) (*models.Channel, error)
func (s *ChannelService) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error)
func (s *ChannelService) List(ctx context.Context) ([]*models.Channel, error)
func (s *ChannelService) UpdateChannel(ctx context.Context, channel *models.Channel) error
func (s *ChannelService) DeleteChannel(ctx context.Context, id uuid.UUID) error

// Validation Helpers
func (s *ChannelService) HasEmptyPlaylist(ctx context.Context, channelID uuid.UUID) (bool, error)
```

**Validation Rules:**
- Name: Must be unique (case-insensitive)
- Start Time: Cannot be more than 1 year in the future
- Cascade: Deleting channel deletes all playlist items

**Errors:**
- `ErrDuplicateChannelName` - Channel name already exists
- `ErrInvalidStartTime` - Start time > 1 year in future
- `ErrChannelNotFound` - Channel doesn't exist

### PlaylistService (Go)

Location: `internal/channel/playlist.go` (pending implementation)

## REST Endpoints

To be defined during API endpoint implementation.

## Data Contracts

See database schema in `docs/api-specs/database/database-api.md` for:
- `Channel` model
- `PlaylistItem` model

