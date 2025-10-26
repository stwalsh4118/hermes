# Product Requirements Document (PRD)
## Virtual TV Channel Service - "Saturday Morning Cartoons"

**Version:** 1.0  
**Date:** October 26, 2025  
**Status:** Ready for Development

---

## 1. Executive Summary

### 1.1 Product Overview
A self-hosted streaming service that recreates the "Saturday morning cartoons" experience by creating virtual TV channels that continuously play content from the user's personal media library. Users can tune in at any time on any device (phone, TV, computer) and catch shows mid-episode, just like traditional broadcast television.

### 1.2 Core Value Proposition
- **Nostalgia:** Recreates the serendipitous "tune in and see what's on" experience
- **Simplicity:** No choosing what to watch - just turn it on
- **Multi-device:** Seamless access from any device
- **Always-on:** Channels run 24/7, synchronized across all clients

### 1.3 Target User
Home media enthusiasts who:
- Have personal media libraries (legally obtained shows/movies)
- Want a passive viewing experience
- Use multiple devices to consume media
- Enjoy the traditional TV channel experience

---

## 2. Goals and Objectives

### 2.1 Primary Goals
1. Create virtual TV channels that simulate continuous broadcast
2. Support playback on web, mobile, and TV devices
3. Synchronize playback across all clients (everyone sees the same thing)
4. Require no manual intervention once configured

### 2.2 Success Metrics
- System can maintain 5+ concurrent streams without performance degradation
- Channels stay synchronized within ±2 seconds across all clients
- Stream startup time < 10 seconds
- Zero manual restarts required for normal operation

---

## 3. User Stories

### 3.1 Core User Stories

**As a user, I want to:**
- Create custom TV channels from my media library
- Configure channel schedules (what plays when)
- Access channels from my phone, computer, and TV
- See what's currently playing on each channel
- View an electronic program guide (EPG) showing upcoming content
- Have the channel "always on" without manually starting streams

**As a user, I should NOT be able to:**
- Pause or rewind (this is live TV)
- Skip ahead (defeats the purpose)
- Choose specific episodes (defeats the purpose)

---

## 4. Functional Requirements

### 4.1 Channel Management

**FR-1.1: Create Channel**
- User can create a new channel with a name and icon
- User specifies a start date/time (when the channel "began broadcasting")
- User can enable/disable looping (restart playlist when finished)

**FR-1.2: Configure Channel Playlist**
- User can add shows/movies to a channel's playlist
- User can specify episode order (sequential, random, custom)
- User can view total playlist duration
- System automatically detects video duration and metadata

**FR-1.3: Edit Channel**
- User can modify channel name, icon, and playlist
- User can reorder playlist items
- User can remove items from playlist
- Changes take effect on next episode transition (not mid-stream)

**FR-1.4: Delete Channel**
- User can delete a channel
- System confirms deletion
- Active streams are gracefully terminated

**FR-1.5: List Channels**
- User can view all configured channels
- Display shows: name, icon, current program, viewer count

### 4.2 Media Library Management

**FR-2.1: Scan Media Directory**
- System can scan a directory for video files
- Supported formats: MP4, MKV, AVI, MOV
- System extracts metadata: duration, resolution, codecs, file size

**FR-2.2: Organize Media**
- User can organize media into shows/series
- User can specify show name, season, episode information
- System stores media metadata in database

**FR-2.3: Media Validation**
- System validates video files are readable
- System warns if codec/format requires transcoding
- System shows which files will need transcoding

### 4.3 Streaming

**FR-3.1: Virtual Timeline Calculation**
- System calculates current position in playlist based on:
  - Channel start time
  - Current time
  - Playlist durations
  - Loop setting
- Calculation accurate to ±1 second

**FR-3.2: Stream Generation**
- System generates HLS stream starting from current position
- System transcodes to H.264 + AAC if needed
- System creates adaptive bitrate streams (1080p, 720p, 480p)

**FR-3.3: Multi-Client Support**
- Multiple clients can watch the same channel simultaneously
- All clients see the exact same frame (synchronized)
- System shares transcoding process across clients on same channel

**FR-3.4: Stream Cleanup**
- System terminates transcoding when last client disconnects
- System keeps stream alive 30 seconds after disconnect (quick rejoin)
- System cleans up old segment files

**FR-3.5: Hardware Acceleration**
- System detects available hardware encoders (NVENC, QSV, VideoToolbox, VAAPI)
- User can select preferred encoder
- System falls back to software encoding if hardware unavailable

### 4.4 Program Guide (EPG)

**FR-4.1: Generate EPG Data**
- System generates program guide for next 7 days
- EPG shows: program title, start time, end time, description, thumbnail
- EPG updates automatically when channel playlist changes

**FR-4.2: View Current Programming**
- User can see what's currently playing on each channel
- Display shows: current program, elapsed time, remaining time

**FR-4.3: View Schedule**
- User can view upcoming programs for each channel
- Schedule shows next 24 hours minimum

### 4.5 Configuration

**FR-5.1: System Settings**
- User can configure media library paths
- User can configure output quality settings
- User can configure hardware acceleration options
- User can configure port and host settings

**FR-5.2: Transcoding Presets**
- User can select quality presets (High, Medium, Low)
- User can configure custom encoding parameters (advanced)

---

## 5. Technical Requirements

### 5.1 Technology Stack

**Backend:**
- Language: Go 1.21+
- Framework: Gin (github.com/gin-gonic/gin)
- Database: SQLite (github.com/mattn/go-sqlite3)
- Video Processing: FFmpeg (system dependency)
- HLS Generation: github.com/grafov/m3u8
- Configuration: github.com/spf13/viper

**Frontend:**
- Framework: Next.js 14+ (App Router)
- Language: TypeScript
- UI Library: Tailwind CSS + shadcn/ui
- Video Player: HLS.js or Video.js
- State Management: React Context or Zustand
- API Client: Native fetch or axios

**Infrastructure:**
- Container: Docker + Docker Compose (optional)
- Reverse Proxy: Nginx (optional, for production)

### 5.2 System Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Client Layer                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │   Web    │  │  Mobile  │  │    TV    │          │
│  │ (Next.js)│  │ (Browser)│  │ (Browser)│          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
                        │
                   HTTP/HLS
                        │
┌─────────────────────────────────────────────────────┐
│                   API Layer (Go/Gin)                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │   REST   │  │  Stream  │  │   EPG    │          │
│  │   API    │  │ Handler  │  │ Generator│          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
                        │
┌─────────────────────────────────────────────────────┐
│                  Business Logic                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ Channel  │  │  Stream  │  │  Media   │          │
│  │ Manager  │  │ Manager  │  │ Scanner  │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
                        │
┌─────────────────────────────────────────────────────┐
│                  Data Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │  SQLite  │  │  FFmpeg  │  │   File   │          │
│  │   DB     │  │  Process │  │  System  │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
```

### 5.3 Data Models

**Channel:**
```go
type Channel struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Icon        string    `json:"icon" db:"icon"`
    StartTime   time.Time `json:"start_time" db:"start_time"`
    Loop        bool      `json:"loop" db:"loop"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
```

**PlaylistItem:**
```go
type PlaylistItem struct {
    ID          string        `json:"id" db:"id"`
    ChannelID   string        `json:"channel_id" db:"channel_id"`
    MediaID     string        `json:"media_id" db:"media_id"`
    Position    int           `json:"position" db:"position"`
    CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}
```

**Media:**
```go
type Media struct {
    ID          string        `json:"id" db:"id"`
    FilePath    string        `json:"file_path" db:"file_path"`
    Title       string        `json:"title" db:"title"`
    ShowName    string        `json:"show_name" db:"show_name"`
    Season      int           `json:"season" db:"season"`
    Episode     int           `json:"episode" db:"episode"`
    Duration    int64         `json:"duration" db:"duration"` // seconds
    VideoCodec  string        `json:"video_codec" db:"video_codec"`
    AudioCodec  string        `json:"audio_codec" db:"audio_codec"`
    Resolution  string        `json:"resolution" db:"resolution"`
    FileSize    int64         `json:"file_size" db:"file_size"`
    CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}
```

**StreamSession:**
```go
type StreamSession struct {
    ID          string    `json:"id"`
    ChannelID   string    `json:"channel_id"`
    StartedAt   time.Time `json:"started_at"`
    ClientCount int       `json:"client_count"`
    FFmpegPID   int       `json:"ffmpeg_pid"`
}
```

**Settings:**
```go
type Settings struct {
    ID                  int       `json:"id" db:"id"`
    MediaLibraryPath    string    `json:"media_library_path" db:"media_library_path"`
    TranscodeQuality    string    `json:"transcode_quality" db:"transcode_quality"` // high, medium, low
    HardwareAccel       string    `json:"hardware_accel" db:"hardware_accel"` // nvenc, qsv, vaapi, none
    ServerPort          int       `json:"server_port" db:"server_port"`
    UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}
```

### 5.4 API Endpoints

**Channels:**
```
GET    /api/channels                    - List all channels
POST   /api/channels                    - Create channel
GET    /api/channels/:id                - Get channel details
PUT    /api/channels/:id                - Update channel
DELETE /api/channels/:id                - Delete channel
GET    /api/channels/:id/current        - Get current program
GET    /api/channels/:id/schedule       - Get channel schedule (EPG)
```

**Media Library:**
```
GET    /api/media                       - List all media
POST   /api/media/scan                  - Scan media directory
GET    /api/media/:id                   - Get media details
PUT    /api/media/:id                   - Update media metadata
DELETE /api/media/:id                   - Remove media from library
```

**Playlist:**
```
GET    /api/channels/:id/playlist       - Get channel playlist
POST   /api/channels/:id/playlist       - Add item to playlist
DELETE /api/channels/:id/playlist/:item_id - Remove item from playlist
PUT    /api/channels/:id/playlist/reorder - Reorder playlist
```

**Streaming:**
```
GET    /stream/:channel_id/master.m3u8  - HLS master playlist
GET    /stream/:channel_id/:quality.m3u8 - HLS media playlist (720p, 480p, etc)
GET    /stream/:channel_id/:segment.ts   - HLS video segment
```

**EPG:**
```
GET    /api/epg                         - Get EPG for all channels (XMLTV format)
GET    /api/epg/:channel_id             - Get EPG for specific channel
```

**Settings:**
```
GET    /api/settings                    - Get system settings
PUT    /api/settings                    - Update settings
GET    /api/settings/hardware           - Detect available hardware encoders
```

**System:**
```
GET    /api/health                      - Health check
GET    /api/stats                       - System statistics (active streams, CPU, etc)
```

### 5.5 Frontend Structure

```
/app
  /layout.tsx                 - Root layout
  /page.tsx                   - Home/dashboard
  /channels
    /page.tsx                 - Channel list
    /[id]
      /page.tsx               - Channel detail/player
      /edit/page.tsx          - Edit channel
    /new/page.tsx             - Create channel
  /library
    /page.tsx                 - Media library
  /settings
    /page.tsx                 - Settings page

/components
  /ui                         - shadcn components
  /channel-card.tsx           - Channel preview card
  /video-player.tsx           - HLS video player
  /epg-grid.tsx               - Program guide
  /media-browser.tsx          - Browse media files
  /playlist-editor.tsx        - Drag-drop playlist editor

/lib
  /api.ts                     - API client
  /types.ts                   - TypeScript types
  /utils.ts                   - Utility functions

/hooks
  /use-channels.ts            - Channel data hook
  /use-stream.ts              - Streaming state hook
```

---

## 6. UI/UX Requirements

### 6.1 Dashboard/Home Page
- Grid of channel cards showing:
  - Channel icon/logo
  - Channel name
  - Currently playing program with thumbnail
  - Live indicator (red dot)
  - Viewer count (if >0)
- "Create Channel" button prominently displayed
- Responsive grid (1 col mobile, 2-3 cols tablet, 4+ cols desktop)

### 6.2 Channel Player Page
- Full-width video player (16:9 aspect ratio)
- Below player:
  - Current program title and description
  - Progress bar showing position in current program
  - Next up: upcoming program preview
- Side panel (desktop) or bottom sheet (mobile):
  - Channel schedule for next 24 hours
  - Each item shows: time, title, thumbnail

### 6.3 Channel Management Page
- List of channels with edit/delete actions
- Each channel shows:
  - Icon, name
  - Playlist item count
  - Total duration
  - Last modified date
- Quick actions: Edit, Delete, View EPG

### 6.4 Channel Editor
- **Basic Info Section:**
  - Name input
  - Icon upload/URL
  - Start date/time picker
  - Loop checkbox
- **Playlist Section:**
  - Drag-and-drop list of playlist items
  - Add media button opens media browser
  - Each item shows: thumbnail, title, duration
  - Reorder by dragging
  - Remove button on each item
  - Total playlist duration displayed
- **Preview Section:**
  - Shows what would be playing "right now" based on settings
  - Updates in real-time as changes are made

### 6.5 Media Library Page
- Table or grid view of all media
- Columns: Title, Show, Season, Episode, Duration, Resolution, Actions
- Filter by show name
- Search by title
- "Scan Library" button triggers directory scan
- Edit button opens metadata editor

### 6.6 Settings Page
- **Media Library:**
  - Path to media directory
  - Scan now button
- **Streaming:**
  - Quality preset dropdown (High/Medium/Low)
  - Hardware acceleration dropdown (Auto/NVENC/QSV/VAAPI/None)
  - Advanced: Custom FFmpeg parameters (textarea)
- **Server:**
  - Port number
  - Host address
- Save button at bottom

### 6.7 Design Guidelines
- Clean, modern interface
- Dark mode support (optional for v1, recommended)
- Loading states for async operations
- Error messages displayed prominently
- Confirmation dialogs for destructive actions (delete)
- Responsive design (mobile-first)

---

## 7. Non-Functional Requirements

### 7.1 Performance
- Stream startup latency < 10 seconds
- Support 5+ concurrent streams on modest hardware (4-core CPU, 8GB RAM)
- Database queries < 100ms
- UI interactions feel instant (< 200ms)

### 7.2 Reliability
- System recovers gracefully from FFmpeg crashes
- Database transactions are atomic
- Corrupted video files don't crash the system

### 7.3 Scalability
- System handles media libraries with 1000+ files
- Channels can have 100+ playlist items
- EPG generation for 7 days completes in < 5 seconds

### 7.4 Usability
- User can create first channel in < 5 minutes
- No command-line interaction required for normal operation
- Clear error messages with actionable suggestions

### 7.5 Security
- Input validation on all API endpoints
- Path traversal prevention for media files
- SQL injection prevention (parameterized queries)
- CORS configuration for API
- Optional: Basic authentication for API access (phase 2)

### 7.6 Maintainability
- Code follows Go standard practices
- Comprehensive logging (info, warn, error levels)
- Configuration via environment variables or config file
- Docker support for easy deployment

---

## 8. Dependencies and Prerequisites

### 8.1 System Dependencies
- FFmpeg 4.4+ (with libx264, AAC encoder)
- FFprobe (usually bundled with FFmpeg)
- SQLite 3.35+

### 8.2 Optional Dependencies
- NVIDIA GPU drivers (for NVENC)
- Intel Media SDK (for QSV)
- VAAPI drivers (for AMD/Intel on Linux)

### 8.3 Development Tools
- Go 1.21+
- Node.js 18+
- npm or yarn
- Docker (optional)

---

## 9. Out of Scope (for v1)

The following features are explicitly out of scope for the initial version:

- ❌ User authentication and multi-user support
- ❌ DVR functionality (pause, rewind, record)
- ❌ Automatic media metadata scraping (TVDB, TMDB integration)
- ❌ Commercial break insertion
- ❌ Multiple audio tracks / subtitle selection
- ❌ Mobile native apps (iOS/Android) - web only for v1
- ❌ Social features (watch parties, chat)
- ❌ Cloud storage integration (S3, Google Drive)
- ❌ Automatic channel scheduling (AI-based)
- ❌ Advanced analytics (viewing history, statistics)

These features may be considered for future versions.

---

## 10. Development Phases

### Phase 1: Core Backend (Week 1-2)
**Tasks:**
- Set up Go project with Gin
- Implement database schema and models
- Create channel CRUD API endpoints
- Implement virtual timeline calculation logic
- Create media library scanner
- Basic FFmpeg integration (probe video info)

**Deliverable:** API that can create channels and scan media

### Phase 2: Streaming Engine (Week 2-3)
**Tasks:**
- Implement HLS stream generation
- Create stream manager (lifecycle, cleanup)
- Implement FFmpeg transcoding pipeline
- Add hardware acceleration detection
- Create segment cleanup worker
- Add stream synchronization logic

**Deliverable:** Backend can generate HLS streams

### Phase 3: EPG Generation (Week 3)
**Tasks:**
- Implement EPG calculation algorithm
- Create XMLTV generator
- Add "now playing" endpoint
- Add schedule endpoint

**Deliverable:** EPG data available via API

### Phase 4: Frontend Foundation (Week 4)
**Tasks:**
- Set up Next.js project with TypeScript
- Implement routing structure
- Create API client
- Add Tailwind CSS + shadcn/ui
- Create basic layouts

**Deliverable:** Frontend shell with navigation

### Phase 5: Channel Management UI (Week 5)
**Tasks:**
- Build channel list page
- Create channel card component
- Build channel creation form
- Build channel editor with playlist management
- Implement drag-and-drop playlist ordering

**Deliverable:** Users can create and manage channels via UI

### Phase 6: Media Library UI (Week 5-6)
**Tasks:**
- Build media library page
- Create media table/grid component
- Add scan functionality UI
- Build media browser for playlist editor
- Implement search and filter

**Deliverable:** Users can view and organize media

### Phase 7: Video Player (Week 6)
**Tasks:**
- Integrate HLS.js or Video.js
- Build video player component
- Add "now playing" display
- Create schedule sidebar
- Implement responsive player layout

**Deliverable:** Users can watch channels

### Phase 8: Settings & Polish (Week 7)
**Tasks:**
- Build settings page
- Add loading states throughout UI
- Add error handling and user feedback
- Implement confirmation dialogs
- Add responsive design improvements
- Create README and documentation

**Deliverable:** Production-ready v1

### Phase 9: Testing & Deployment (Week 8)
**Tasks:**
- End-to-end testing
- Performance testing with multiple streams
- Create Docker configuration
- Write deployment guide
- Bug fixes and refinements

**Deliverable:** Deployed, tested system

---

## 11. Success Criteria

The project is considered successful when:

✅ User can create a channel with 10 episodes  
✅ User can stream the channel from web browser  
✅ Stream plays continuously without buffering (stable connection)  
✅ Two devices watching same channel see same content (±2s)  
✅ EPG accurately shows current and upcoming programs  
✅ System uses hardware acceleration when available  
✅ All CRUD operations work via UI (no database manipulation needed)  
✅ System documentation allows new users to deploy and configure  
✅ No manual intervention needed for 24+ hours of operation  

---

## 12. Risk and Mitigation

### Risk 1: FFmpeg Complexity
**Risk:** FFmpeg has many edge cases and can be fragile  
**Mitigation:** 
- Use conservative encoding settings
- Implement comprehensive error handling
- Validate media files during scan
- Provide fallback to software encoding

### Risk 2: Synchronization Drift
**Risk:** Clients may drift out of sync over time  
**Mitigation:**
- Use server-side timestamps in playlists
- Clients re-sync every segment fetch
- Keep segment duration short (6s)

### Risk 3: Performance with Multiple Streams
**Risk:** CPU/memory exhaustion with many concurrent streams  
**Mitigation:**
- Implement stream sharing for same channel
- Use hardware acceleration when available
- Set maximum concurrent stream limit
- Monitor system resources

### Risk 4: Media File Compatibility
**Risk:** Exotic codecs/containers may not work  
**Mitigation:**
- Document supported formats clearly
- Show codec info during media scan
- Warn user when transcoding required
- Provide media validation tool

---

## 13. Appendix

### 13.1 Glossary

- **HLS:** HTTP Live Streaming - Apple's streaming protocol
- **EPG:** Electronic Program Guide - TV schedule data
- **Transcode:** Converting video from one codec/format to another
- **Segment:** Small chunk of video (typically 6 seconds)
- **M3U8:** Playlist file format for HLS
- **MPEG-TS:** Transport stream format used in HLS segments
- **Virtual Timeline:** Mathematical calculation of what should be playing

### 13.2 References

- HLS Specification: https://datatracker.ietf.org/doc/html/rfc8216
- XMLTV Format: http://wiki.xmltv.org/index.php/XMLTVFormat
- FFmpeg Documentation: https://ffmpeg.org/documentation.html
- Gin Framework: https://gin-gonic.com/docs/
- Next.js Documentation: https://nextjs.org/docs

---

## 14. Approval

**Product Owner:** [Your Name]  
**Technical Lead:** [Coding Agent]  
**Date:** October 26, 2025  
**Status:** ✅ Approved for Development

---

**END OF PRD**

---

## Next Steps for Coding Agent

1. Review this PRD thoroughly
2. Break down each phase into specific implementation tasks
3. Create task list with estimates
4. Identify any ambiguities or questions
5. Begin Phase 1: Core Backend development
