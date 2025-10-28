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

Location: `internal/channel/playlist.go`

```go
type PlaylistService struct {
    repos *db.Repositories
    db    *db.DB
}

func NewPlaylistService(database *db.DB, repos *db.Repositories) *PlaylistService

// Playlist Operations
func (s *PlaylistService) AddToPlaylist(ctx context.Context, channelID, mediaID uuid.UUID, position int) (*models.PlaylistItem, error)
func (s *PlaylistService) RemoveFromPlaylist(ctx context.Context, itemID uuid.UUID) error
func (s *PlaylistService) ReorderPlaylist(ctx context.Context, channelID uuid.UUID, items []db.ReorderItem) error
func (s *PlaylistService) GetPlaylist(ctx context.Context, channelID uuid.UUID) ([]*models.PlaylistItem, error)
func (s *PlaylistService) CalculateDuration(ctx context.Context, channelID uuid.UUID) (int64, error)
```

**Business Rules:**
- Position: 0-indexed, must be non-negative
- Add: Shifts items up if position occupied
- Remove: Reorders subsequent items down
- Reorder: Uses two-pass approach to avoid unique constraint violations
- Transactions: Multi-step operations use database transactions

**Errors:**
- `ErrMediaNotFound` - Media doesn't exist
- `ErrPlaylistItemNotFound` - Playlist item doesn't exist
- `ErrInvalidPosition` - Position is negative
- `ErrChannelNotFound` - Channel doesn't exist

## REST Endpoints

### POST /api/channels
Create a new channel

**Request:**
```json
{
  "name": "Comedy Central",
  "icon": "icon.png",
  "start_time": "2025-10-27T12:00:00Z",
  "loop": true
}
```

**Response (201 Created):**
```json
{
  "id": "uuid-here",
  "name": "Comedy Central",
  "icon": "icon.png",
  "start_time": "2025-10-27T12:00:00Z",
  "loop": true,
  "created_at": "2025-10-28T00:00:00Z",
  "updated_at": "2025-10-28T00:00:00Z"
}
```

**Errors:**
- `400 Bad Request` - Invalid request body or invalid start time
- `409 Conflict` - Channel name already exists
- `500 Internal Server Error` - Failed to create channel

### GET /api/channels
List all channels

**Response (200 OK):**
```json
{
  "channels": [
    {
      "id": "uuid-here",
      "name": "Comedy Central",
      "icon": "icon.png",
      "start_time": "2025-10-27T12:00:00Z",
      "loop": true,
      "created_at": "2025-10-28T00:00:00Z",
      "updated_at": "2025-10-28T00:00:00Z"
    }
  ]
}
```

**Errors:**
- `500 Internal Server Error` - Query failed

### GET /api/channels/:id
Get single channel details

**Response (200 OK):**
```json
{
  "id": "uuid-here",
  "name": "Comedy Central",
  "icon": "icon.png",
  "start_time": "2025-10-27T12:00:00Z",
  "loop": true,
  "created_at": "2025-10-28T00:00:00Z",
  "updated_at": "2025-10-28T00:00:00Z"
}
```

**Errors:**
- `400 Bad Request` - Invalid UUID format
- `404 Not Found` - Channel not found
- `500 Internal Server Error` - Query failed

### PUT /api/channels/:id
Update channel (partial update)

**Request:**
```json
{
  "name": "Updated Name",
  "icon": "new-icon.png",
  "start_time": "2025-10-27T15:00:00Z",
  "loop": false
}
```

All fields are optional - only provided fields will be updated.

**Response (200 OK):**
```json
{
  "id": "uuid-here",
  "name": "Updated Name",
  "icon": "new-icon.png",
  "start_time": "2025-10-27T15:00:00Z",
  "loop": false,
  "created_at": "2025-10-28T00:00:00Z",
  "updated_at": "2025-10-28T01:00:00Z"
}
```

**Errors:**
- `400 Bad Request` - Invalid UUID or request body
- `404 Not Found` - Channel not found
- `409 Conflict` - Channel name already exists
- `500 Internal Server Error` - Update failed

### DELETE /api/channels/:id
Delete channel and all associated playlist items (cascade)

**Response (200 OK):**
```json
{
  "message": "Channel deleted successfully"
}
```

**Errors:**
- `400 Bad Request` - Invalid UUID format
- `404 Not Found` - Channel not found
- `500 Internal Server Error` - Delete failed

### GET /api/channels/:id/current
Get currently playing program (placeholder for PBI 4)

**Response (501 Not Implemented):**
```json
{
  "error": "not_implemented",
  "message": "Current program feature will be implemented in PBI 4"
}
```

## Data Contracts

See database schema in `docs/api-specs/database/database-api.md` for:
- `Channel` model
- `PlaylistItem` model

