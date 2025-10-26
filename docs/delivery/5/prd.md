# PBI-5: EPG Generation

[View in Backlog](../backlog.md#user-content-5)

## Overview

Implement an Electronic Program Guide (EPG) system that generates and exposes schedule data for all channels, allowing users to see what's currently playing and what's coming up in the next 7 days.

## Problem Statement

Users need to:
- See what's currently playing on each channel
- View upcoming programs for the next 24 hours minimum
- Access a full 7-day program guide
- Integrate with EPG-compatible clients that expect XMLTV format

The system must:
- Generate EPG data efficiently
- Keep EPG data synchronized with playlist changes
- Provide both JSON APIs and XMLTV format
- Include program metadata (title, times, descriptions, thumbnails)

## User Stories

**As a user, I want to:**
- See what's playing right now on each channel without tuning in
- Browse what's coming up next on my channels
- View a full week's schedule to plan my viewing
- See program titles, start/end times, and episode information

**As a developer integrating with this service, I want to:**
- Fetch EPG data in standard XMLTV format
- Get JSON APIs for web/mobile interfaces
- Have EPG data automatically update when playlists change

## Technical Approach

### Components to Implement

1. **EPG Generator Service**
   - Use Virtual Timeline algorithm from PBI-4
   - Generate program entries for specified time range
   - Calculate program boundaries based on media durations
   - Handle loop boundaries correctly

2. **EPG Models**
   ```go
   type EPGProgram struct {
       ChannelID   string    `json:"channel_id" xml:"channel"`
       MediaID     string    `json:"media_id"`
       Title       string    `json:"title" xml:"title"`
       Description string    `json:"description" xml:"desc"`
       StartTime   time.Time `json:"start_time" xml:"start"`
       EndTime     time.Time `json:"end_time" xml:"stop"`
       Season      int       `json:"season,omitempty" xml:"episode-num>season,omitempty"`
       Episode     int       `json:"episode,omitempty" xml:"episode-num>episode,omitempty"`
       Icon        string    `json:"icon,omitempty" xml:"icon>src,omitempty"`
   }
   ```

3. **API Endpoints**
   - `GET /api/channels/:id/current` - Get currently playing program
   - `GET /api/channels/:id/schedule?duration=24h` - Get channel schedule
   - `GET /api/epg` - Get EPG for all channels (XMLTV format)
   - `GET /api/epg/:channel_id` - Get EPG for specific channel

4. **XMLTV Format Support**
   - Generate valid XMLTV XML structure
   - Include channel definitions
   - Include program listings with proper timing
   - Support XMLTV client compatibility

### EPG Generation Algorithm

```
function GenerateEPG(channel, startTime, duration):
    programs = []
    currentTime = startTime
    endTime = startTime + duration
    
    while currentTime < endTime:
        position = CalculateTimelinePosition(channel, currentTime)
        
        if position is valid:
            program = {
                title: position.MediaTitle,
                start: currentTime,
                end: position.EndsAt,
                ...media metadata
            }
            programs.append(program)
            currentTime = position.EndsAt
        else:
            break  # Channel ended or error
    
    return programs
```

### Caching Strategy

- Cache EPG data for each channel
- Invalidate cache when playlist changes
- Set reasonable cache TTL (e.g., 5 minutes)
- Generate full 7-day EPG on demand or via background job

### Performance Optimization

- Batch process EPG generation
- Use concurrent goroutines for multiple channels
- Limit maximum EPG generation to 7 days
- Set timeout for EPG generation (< 5 seconds)

## UX/UI Considerations

EPG data should be designed for UI consumption:
- Include all metadata needed for rich displays
- Provide thumbnail URLs when available
- Format times in ISO 8601 for JSON APIs
- Include program descriptions
- Mark "currently playing" programs in responses

## Acceptance Criteria

- [ ] EPG calculation algorithm generates accurate program schedules
- [ ] API endpoint for "now playing" returns current program for any channel
- [ ] API endpoint for channel schedule returns programs for specified duration (default 24h)
- [ ] 7-day EPG generation capability implemented
- [ ] XMLTV format generator produces valid XMLTV XML
- [ ] EPG includes all required metadata: title, start/end times, description, season/episode
- [ ] EPG includes thumbnail/icon URLs when available
- [ ] EPG automatically updates when channel playlist changes
- [ ] EPG generation completes within 5 seconds for all channels
- [ ] EPG correctly handles:
  - Looping channels (schedule wraps around)
  - Non-looping channels (schedule ends)
  - Empty playlists (returns empty or error)
  - Channels not yet started
- [ ] Caching implemented to avoid redundant calculations
- [ ] API returns appropriate HTTP status codes and error messages
- [ ] Unit tests for EPG generation algorithm
- [ ] Integration tests for EPG API endpoints
- [ ] XMLTV output validates against XMLTV schema

## Dependencies

**PBI Dependencies:**
- PBI-1: Project Setup & Database Foundation (REQUIRED)
- PBI-3: Channel Management Backend (REQUIRED - needs channel data)
- PBI-4: Virtual Timeline Calculation (REQUIRED - uses timeline algorithm)

**External Dependencies:**
- encoding/xml for XMLTV generation
- Time handling libraries

## Open Questions

- Should EPG generation be real-time or pre-generated via background job?
- How far into the future should the default EPG extend?
- Should we support EPG data for past programs (for DVR-like features in future)?
- Do we need to support XMLTV.se icon URL format?
- Should EPG include content ratings or other metadata?
- How should we handle very long media items (e.g., 3-hour movies) in the schedule display?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.

