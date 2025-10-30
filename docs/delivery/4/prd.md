# PBI-4: Virtual Timeline Calculation

[View in Backlog](../backlog.md#user-content-4)

## Overview

Implement the core algorithm that calculates what should be playing on a channel at any given moment, creating the illusion of a continuously broadcasting television channel that has been running since its configured start time.

## Problem Statement

The virtual timeline is the heart of the "always-on TV channel" experience. Given:
- A channel's start time (when it "began broadcasting")
- The current time
- The channel's playlist and each item's duration
- Whether the channel loops

The system must calculate:
- Which media item should be playing right now
- At what position within that media item (timestamp)
- When the current item started
- When the current item will end

This calculation must be accurate, efficient, and handle edge cases like empty playlists, times before channel start, and loop boundaries.

## User Stories

**As a user, I want to:**
- Tune into a channel and see what's "currently playing" as if it's been broadcasting continuously
- Have all my devices show the exact same content at the same time (synchronized)
- See accurate time information (how long the current show has been playing, when it ends)

**As the system, I need to:**
- Calculate timeline position in under 100ms
- Handle channels with looping enabled and disabled
- Work correctly across playlist boundaries
- Return accurate results for any valid timestamp

## Technical Approach

### Algorithm Overview

1. **Input:**
   - Channel start time: `startTime`
   - Current time: `currentTime`
   - Playlist: `[]PlaylistItem` (ordered by position, each with `duration`)
   - Loop enabled: `loop`

2. **Process:**
   ```
   elapsed = currentTime - startTime
   
   if elapsed < 0:
       return "channel not started yet"
   
   totalPlaylistDuration = sum(item.duration for item in playlist)
   
   if totalPlaylistDuration == 0:
       return "empty playlist"
   
   if loop:
       position = elapsed % totalPlaylistDuration
   else:
       if elapsed >= totalPlaylistDuration:
           return "playlist finished"
       position = elapsed
   
   # Find which item contains this position
   accumulated = 0
   for item in playlist:
       if position < accumulated + item.duration:
           offsetInItem = position - accumulated
           return (item, offsetInItem)
       accumulated += item.duration
   ```

3. **Output:**
   ```go
   type TimelinePosition struct {
       MediaID      string    `json:"media_id"`
       MediaTitle   string    `json:"media_title"`
       OffsetSeconds int64    `json:"offset_seconds"`
       StartedAt    time.Time `json:"started_at"`
       EndsAt       time.Time `json:"ends_at"`
       Duration     int64     `json:"duration"`
   }
   ```

### Edge Cases to Handle

1. **Empty Playlist:** Return error or special state
2. **Time Before Start:** Return "not started" state
3. **Non-looping Past End:** Return "finished" state
4. **Playlist Modified:** Recalculate on next request
5. **Single Item Playlist:** Works correctly with loop on/off
6. **Very Long Playlists:** Algorithm must be O(n) worst case
7. **Leap Seconds/DST:** Use UTC internally

### Performance Considerations

- Cache playlist duration calculation
- Use integer arithmetic where possible (seconds, not nanoseconds)
- Pre-sort playlist by position
- Consider caching current position if called frequently

## UX/UI Considerations

The timeline calculation impacts:
- "Now Playing" displays
- EPG generation
- Stream start position
- Progress indicators

Results should include both technical data (offset in seconds) and user-friendly data (formatted times, human-readable durations).

## Acceptance Criteria

- [ ] Virtual timeline algorithm implemented and functional
- [ ] Accurate position calculation based on channel start time and current time
- [ ] Playlist duration calculation correct (sum of all item durations)
- [ ] Loop handling works correctly (wraps around at playlist end)
- [ ] Non-loop handling works correctly (returns "finished" state)
- [ ] Edge case: empty playlist returns appropriate error
- [ ] Edge case: time before channel start returns appropriate state
- [ ] Edge case: single-item playlist works correctly
- [ ] Position accuracy within ±1 second of expected value
- [ ] Calculation performance under 100ms for playlists up to 1000 items
- [ ] Returns complete TimelinePosition struct with all required fields
- [ ] Unit tests covering all scenarios:
  - Normal operation (mid-playlist)
  - First item in playlist
  - Last item in playlist
  - Loop boundary crossing
  - Non-loop past end
  - Empty playlist
  - Time before start
  - Single item
- [ ] Benchmark tests demonstrating performance requirements met

## Dependencies

**PBI Dependencies:**
- PBI-1: Project Setup & Database Foundation (REQUIRED)
- PBI-3: Channel Management Backend (REQUIRED - needs channel and playlist data)

**External Dependencies:**
- Go time package for datetime calculations

## Open Questions

- Should we cache the current position for each channel or recalculate every time?
- How should we handle playlist changes while something is "playing"?
- Do we need to support timezones or always use UTC?
- Should we log timeline calculations for debugging?
- Do we need an API endpoint specifically for timeline calculation, or is it internal only?

## Related Tasks

See [tasks.md](./tasks.md) for the complete task list for this PBI.


