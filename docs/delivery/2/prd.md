# PBI-2: Media Library Management Backend

[View in Backlog](../backlog.md#user-content-2)

## Overview

Implement the backend functionality to scan, analyze, and manage the user's media library. This includes directory scanning, FFprobe integration for metadata extraction, and RESTful APIs for media CRUD operations.

## Problem Statement

Users need to:
- Import their existing video files into the system
- Have the system automatically detect video metadata (duration, codecs, resolution)
- Organize media into shows, seasons, and episodes
- Validate that files are readable and identify transcoding requirements
- Manage their media library through APIs

## User Stories

**As a user, I want to:**
- Scan a directory and have all video files automatically imported
- See metadata for each video (duration, resolution, codec information)
- Organize my media into shows with season and episode numbers
- Know which files will require transcoding
- Edit media metadata if automatic detection is wrong
- Remove media from the library

## Technical Approach

### Components to Implement

1. **Media Scanner Service**
   - Recursively scan directory for video files
   - Support formats: MP4, MKV, AVI, MOV
   - Extract metadata using FFprobe
   - Parse filename patterns for show/season/episode detection

2. **FFprobe Integration**
   - Execute FFprobe to extract:
     - Duration (seconds)
     - Video codec
     - Audio codec
     - Resolution (width x height)
     - File size
   - Parse JSON output from FFprobe
   - Handle FFprobe errors gracefully

3. **Media Validation**
   - Verify file exists and is readable
   - Check if codecs are compatible (H.264 video, AAC audio = no transcode)
   - Flag files requiring transcoding
   - Validate file size is reasonable

4. **API Endpoints**
   - `GET /api/media` - List all media (with pagination, filtering)
   - `POST /api/media/scan` - Trigger directory scan
   - `GET /api/media/:id` - Get media details
   - `PUT /api/media/:id` - Update media metadata
   - `DELETE /api/media/:id` - Remove from library

### FFprobe Command Example
```bash
ffprobe -v quiet -print_format json -show_format -show_streams video.mp4
```

### Filename Pattern Detection
Support common patterns like:
- `Show Name - S01E01 - Episode Title.mp4`
- `Show.Name.S01E01.mp4`
- `Show Name/Season 1/01 - Episode Title.mp4`

## UX/UI Considerations

While this is a backend PBI, the API responses should be designed for UI consumption:
- Provide clear error messages for invalid files
- Return scan progress information (files processed, total files)
- Include transcoding requirements in media details
- Support filtering and search parameters in list endpoint

## Acceptance Criteria

- [ ] Media directory scanning implemented and functional
- [ ] FFprobe integration extracts all required metadata (duration, codecs, resolution)
- [ ] Media CRUD API endpoints implemented and documented
- [ ] Filename parsing detects show/season/episode from common patterns
- [ ] Media validation identifies files requiring transcoding
- [ ] Database stores all media metadata correctly
- [ ] Support for MP4, MKV, AVI, MOV formats verified
- [ ] Scan operation handles errors gracefully (corrupted files, permission issues)
- [ ] API returns appropriate HTTP status codes and error messages
- [ ] Pagination implemented for media list endpoint
- [ ] Filter by show name implemented
- [ ] Unit tests for scanner and FFprobe parsing
- [ ] Integration tests for API endpoints

## Dependencies

**PBI Dependencies:**
- PBI-1: Project Setup & Database Foundation (REQUIRED - needs database and models)

**External Dependencies:**
- FFmpeg/FFprobe installed on system
- Read access to media directory

**Go Packages:**
- os/exec for running FFprobe
- encoding/json for parsing FFprobe output
- path/filepath for file system operations

## Open Questions

- Should scanning be synchronous or asynchronous? (Recommend async with progress tracking)
- How should we handle very large libraries (10,000+ files)?
- Should we store thumbnails/screenshots of media?
- What should happen to playlist items when media is deleted?
- Should we support multiple media library paths or just one?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.

