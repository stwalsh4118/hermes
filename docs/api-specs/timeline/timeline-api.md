# Virtual Timeline API

Last Updated: 2025-10-30

## Overview

The timeline package calculates what should be playing on a channel at any given moment based on the channel's start time, current time, playlist, and loop setting. This creates the illusion of a continuously broadcasting television channel.

## Data Contracts

### TimelinePosition (Go)

Location: `internal/timeline/types.go`

```go
type TimelinePosition struct {
    MediaID       uuid.UUID `json:"media_id"`
    MediaTitle    string    `json:"media_title"`
    OffsetSeconds int64     `json:"offset_seconds"`
    StartedAt     time.Time `json:"started_at"`
    EndsAt        time.Time `json:"ends_at"`
    Duration      int64     `json:"duration"`
}
```

**Fields:**
- `MediaID` - UUID of the currently playing media item
- `MediaTitle` - Title of the media for display
- `OffsetSeconds` - Position within the current media (seconds)
- `StartedAt` - When the current item started playing (UTC)
- `EndsAt` - When the current item will finish (UTC)
- `Duration` - Total duration of the current media item (seconds)

**JSON Example:**
```json
{
  "media_id": "550e8400-e29b-41d4-a716-446655440000",
  "media_title": "Episode 5 - The One with the Drama",
  "offset_seconds": 1234,
  "started_at": "2025-10-30T12:00:00Z",
  "ends_at": "2025-10-30T12:45:00Z",
  "duration": 2700
}
```

### TimelineState

Location: `internal/timeline/types.go`

```go
type TimelineState string

const (
    TimelineStateNotStarted TimelineState = "not_started"
    TimelineStatePlaying    TimelineState = "playing"
    TimelineStateFinished   TimelineState = "finished"
    TimelineStateEmpty      TimelineState = "empty"
)
```

**States:**
- `not_started` - Current time is before channel start time
- `playing` - Channel is actively playing content
- `finished` - Non-looping channel has completed its playlist
- `empty` - Channel has no playlist items

## Calculator Interface

### CalculatePosition Function

Location: `internal/timeline/calculator.go`

```go
func CalculatePosition(
    startTime time.Time,
    currentTime time.Time,
    playlist []*models.PlaylistItem,
    loop bool,
) (*TimelinePosition, error)
```

**Description:**
Pure calculation function that determines the current timeline position for a channel.
This is a stateless function with no I/O - all data is provided as parameters.

**Parameters:**
- `startTime` - When the channel started broadcasting (UTC)
- `currentTime` - The time to calculate position for (UTC)
- `playlist` - Ordered list of playlist items with populated Media field
- `loop` - Whether the channel loops the playlist

**Returns:**
- `*TimelinePosition` - Current playback position with all fields populated
- `error` - One of: ErrChannelNotStarted, ErrEmptyPlaylist, ErrPlaylistFinished, or nil

**Performance:**
- O(n) where n is playlist length
- Completes in < 1μs for typical playlists
- Completes in < 1ms for 1000-item playlists (requirement: < 100ms)
- Single allocation per call (96 bytes)

**Algorithm:**
1. Validate playlist is not empty
2. Calculate elapsed seconds since channel start
3. Check if channel hasn't started yet (elapsed < 0)
4. Calculate total playlist duration (sum of all media durations)
5. Apply loop logic: `position = elapsed % totalDuration` (or check for past-end)
6. Find containing playlist item via linear search
7. Build TimelinePosition with all calculated fields

**Edge Cases:**
- Empty playlist → ErrEmptyPlaylist
- Time before start → ErrChannelNotStarted
- Non-looping past end → ErrPlaylistFinished
- Single item playlist → Works correctly with both loop modes
- Loop boundary → Correctly wraps to first item

**Example Usage:**
```go
pos, err := timeline.CalculatePosition(
    channel.StartTime,
    time.Now().UTC(),
    playlistItems,
    channel.Loop,
)
if err != nil {
    // Handle error cases
    return err
}
// Use pos.MediaID, pos.OffsetSeconds, etc.
```

## Service Interfaces

### TimelineService (Go)

Location: `internal/timeline/service.go`

```go
type TimelineService struct {
    repos *db.Repositories
}

func NewTimelineService(repos *db.Repositories) *TimelineService
func (s *TimelineService) GetCurrentPosition(ctx context.Context, channelID uuid.UUID) (*TimelinePosition, error)
```

**Description:**
Service layer that integrates the timeline calculator with database repositories. Fetches channel and playlist data, performs validation, and delegates calculation to the pure calculator function.

**Methods:**

#### GetCurrentPosition

Calculates and returns the current timeline position for a channel.

**Signature:**
```go
func (s *TimelineService) GetCurrentPosition(ctx context.Context, channelID uuid.UUID) (*TimelinePosition, error)
```

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `channelID` - UUID of the channel to calculate position for

**Returns:**
- `*TimelinePosition` - Current playback position with all fields populated
- `error` - One of:
  - `channel.ErrChannelNotFound` - Channel doesn't exist
  - `ErrChannelNotStarted` - Current time before channel start
  - `ErrEmptyPlaylist` - Channel has no playlist items
  - `ErrPlaylistFinished` - Non-looping channel past end
  - Wrapped database errors

**Process:**
1. Fetches channel from database using `repos.Channels.GetByID`
2. Fetches playlist with media details using `repos.PlaylistItems.GetWithMedia`
3. Validates playlist is not empty
4. Calls `CalculatePosition` with current UTC time
5. Returns result or appropriate error

**Error Handling:**
- Database errors are wrapped with context
- Calculator errors are passed through unchanged
- "Not found" errors are converted to `channel.ErrChannelNotFound`

**Logging:**
- Debug: Calculation start with channel_id
- Info: Success with media_id, offset, duration
- Warn: Calculator errors or empty playlist
- Error: Database failures with context

**Example Usage:**
```go
import (
    "context"
    "github.com/stwalsh4118/hermes/internal/db"
    "github.com/stwalsh4118/hermes/internal/timeline"
)

// Create service
database, _ := db.New("./data/hermes.db")
repos := db.NewRepositories(database)
service := timeline.NewTimelineService(repos)

// Get current position
ctx := context.Background()
position, err := service.GetCurrentPosition(ctx, channelID)
if err != nil {
    // Handle error cases
    switch {
    case errors.Is(err, channel.ErrChannelNotFound):
        // Channel not found - return 404
    case errors.Is(err, timeline.ErrChannelNotStarted):
        // Channel hasn't started - return 409
    case errors.Is(err, timeline.ErrEmptyPlaylist):
        // No playlist items - return 409
    case errors.Is(err, timeline.ErrPlaylistFinished):
        // Playlist finished - return 409
    default:
        // Database error - return 500
    }
    return
}

// Use position data
fmt.Printf("Now playing: %s at %d seconds\n", 
    position.MediaTitle, position.OffsetSeconds)
```

## REST Endpoints

### GET /api/channels/:id/current

Get the current timeline position for a channel.

**Success Response (200 OK):**
```json
{
  "media_id": "uuid-here",
  "media_title": "Episode Title",
  "offset_seconds": 1234,
  "started_at": "2025-10-30T12:00:00Z",
  "ends_at": "2025-10-30T12:45:00Z",
  "duration": 2700
}
```

**Error Responses:**
- `400 Bad Request` - Invalid channel UUID format
- `404 Not Found` - Channel not found
- `409 Conflict` - Channel not started, empty playlist, or playlist finished
- `500 Internal Server Error` - Calculation failed

To be fully implemented during PBI 4.

