# PBI-3: Channel Management Backend

[View in Backlog](../backlog.md#user-content-3)

## Overview

Implement comprehensive backend APIs for creating, reading, updating, and deleting TV channels, along with playlist management functionality that allows users to configure what content plays on each channel.

## Problem Statement

Users need to:
- Create virtual TV channels with custom names and branding
- Configure when each channel "started broadcasting"
- Build playlists by selecting media from their library
- Reorder playlist items to control playback sequence
- Enable/disable looping for channels
- Edit channel settings and playlists
- Delete channels when no longer needed

## User Stories

**As a user, I want to:**
- Create a new channel and give it a unique name and icon
- Set when my channel started so the virtual timeline is correct
- Add shows and movies to my channel's playlist
- Reorder content in my playlist by changing positions
- Remove items from my playlist
- Enable loop mode so content repeats when the playlist ends
- Edit my channel's name, icon, and settings
- Delete a channel I no longer want
- See a list of all my channels with their current status
- View what's currently scheduled to play on each channel

## Technical Approach

### Components to Implement

1. **Channel Service**
   - Business logic for channel operations
   - Validation rules (unique names, valid timestamps, etc.)
   - Channel state management

2. **Playlist Service**
   - Add media to channel playlist at specific position
   - Reorder playlist items
   - Remove playlist items
   - Validate media exists before adding
   - Calculate total playlist duration
   - Handle position conflicts and gaps

3. **API Endpoints**

**Channel Management:**
- `GET /api/channels` - List all channels
- `POST /api/channels` - Create new channel
- `GET /api/channels/:id` - Get channel details with playlist
- `PUT /api/channels/:id` - Update channel (name, icon, start_time, loop)
- `DELETE /api/channels/:id` - Delete channel and associated playlist
- `GET /api/channels/:id/current` - Get currently playing program (uses timeline)

**Playlist Management:**
- `GET /api/channels/:id/playlist` - Get channel's playlist
- `POST /api/channels/:id/playlist` - Add media to playlist
- `DELETE /api/channels/:id/playlist/:item_id` - Remove from playlist
- `PUT /api/channels/:id/playlist/reorder` - Reorder playlist items

### Validation Rules

**Channel Creation/Update:**
- Name: Required, max 100 characters
- Start Time: Required, cannot be in distant future (> 1 year)
- Icon: Optional URL or file path
- Loop: Boolean, defaults to true

**Playlist Operations:**
- Media ID must exist in media table
- Position must be non-negative
- No duplicate positions allowed
- Removing item should reorder remaining items

### Database Operations

- Use transactions for playlist reordering
- Cascade delete playlist items when channel is deleted
- Maintain referential integrity with foreign keys
- Update channel's updated_at timestamp on changes

## UX/UI Considerations

API design should support UI needs:
- Include media details when returning playlist (JOIN query)
- Return total playlist duration in channel details
- Provide current playback position in channel status
- Return channel list with "now playing" information
- Support sorting channels by name or creation date

## Acceptance Criteria

- [ ] Channel CRUD API endpoints implemented and functional
- [ ] Channel creation accepts name, icon, start_time, loop setting
- [ ] Channel update allows modifying all configurable fields
- [ ] Channel deletion removes channel and all playlist items (cascade)
- [ ] Playlist add operation validates media exists
- [ ] Playlist reorder endpoint handles position changes correctly
- [ ] Playlist remove operation reorders remaining items
- [ ] List channels endpoint includes basic status information
- [ ] Get channel detail includes full playlist with media info
- [ ] Database relationships properly maintained with foreign keys
- [ ] Input validation on all endpoints with meaningful error messages
- [ ] API returns appropriate HTTP status codes (200, 201, 400, 404, etc.)
- [ ] Concurrent playlist modifications handled safely
- [ ] Unit tests for channel and playlist services
- [ ] Integration tests for all API endpoints
- [ ] API documentation (comments or OpenAPI spec)

## Dependencies

**PBI Dependencies:**
- PBI-1: Project Setup & Database Foundation (REQUIRED - needs database and models)
- PBI-2: Media Library Management (REQUIRED - channels reference media)

**External Dependencies:**
- None beyond standard Go libraries

## Open Questions

- Should we limit the number of channels a user can create?
- Should channel names be unique?
- How should we handle playlist modifications when channel is actively streaming?
- Should we support copying playlists between channels?
- Do we need a "dry run" mode for testing channel configuration?
- Should we validate that playlist has content before allowing channel activation?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.


