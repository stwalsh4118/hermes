# Product Backlog - Virtual TV Channel Service

This document contains all Product Backlog Items (PBIs) for the Virtual TV Channel Service project, ordered by priority.

## Backlog

| ID | Actor | User Story | Status | Conditions of Satisfaction (CoS) |
|----|-------|------------|--------|----------------------------------|
| 1 | Developer | As a developer, I want to set up the project foundation with database schema and models so that we have a solid base for building features | Agreed | - Go project structure created with proper organization<br>- SQLite database schema implemented with all required tables<br>- Data models defined matching PRD specifications<br>- Database migrations framework in place<br>- Basic CRUD operations tested<br>- Configuration management set up (Viper)<br>- Logging framework configured<br>- Project builds and runs successfully |
| 2 | User | As a user, I want the system to scan and manage my media library so that I can use my video files in channels | Agreed | - Media directory scanning implemented<br>- FFprobe integration for metadata extraction (duration, codecs, resolution)<br>- Media CRUD API endpoints functional<br>- Media validation (readable files, codec detection)<br>- Database stores all media metadata<br>- Support for MP4, MKV, AVI, MOV formats<br>- Show/series organization capability<br>- Season and episode metadata support |
| 3 | User | As a user, I want to create and manage TV channels so that I can organize my content into continuous broadcasts | Agreed | - Channel CRUD API endpoints functional<br>- Channel creation with name, icon, start time, loop setting<br>- Playlist management (add, remove, reorder items)<br>- Playlist validation (media exists, proper ordering)<br>- Channel deletion with cleanup<br>- List all channels with current status<br>- Database relationships properly maintained<br>- Input validation on all endpoints |
| 4 | System | As the system, I want to calculate the virtual timeline accurately so that channels show the correct content at any given time | Proposed | - Virtual timeline algorithm implemented<br>- Accurate position calculation based on channel start time and current time<br>- Playlist duration calculation<br>- Loop handling when playlist ends<br>- Edge case handling (empty playlists, time before start)<br>- Position accurate to ±1 second<br>- Efficient calculation (< 100ms)<br>- Unit tests for various scenarios |
| 5 | User | As a user, I want to view an electronic program guide so that I can see what's currently playing and what's coming up | Proposed | - EPG calculation algorithm for current and future programs<br>- API endpoint for "now playing" on each channel<br>- API endpoint for channel schedule (next 24 hours minimum)<br>- 7-day EPG generation capability<br>- XMLTV format support<br>- EPG updates automatically when playlist changes<br>- Program metadata includes title, start/end time, description, thumbnail<br>- EPG generation completes in < 5 seconds |
| 6 | User | As a user, I want to stream channels on any device so that I can watch my content anywhere | Proposed | - HLS stream generation from current timeline position<br>- FFmpeg transcoding pipeline (H.264 + AAC)<br>- Adaptive bitrate streams (1080p, 720p, 480p)<br>- Hardware acceleration support (NVENC, QSV, VAAPI, VideoToolbox)<br>- Stream lifecycle management (start, stop, cleanup)<br>- Multi-client support with stream sharing<br>- Stream cleanup when last client disconnects (30s grace period)<br>- Segment file cleanup<br>- Synchronization across clients (±2 seconds)<br>- Graceful error handling and recovery<br>- Stream startup < 10 seconds |
| 7 | Developer | As a developer, I want to set up the frontend foundation so that we can build the user interface | Agreed | - Next.js 14+ project initialized with App Router<br>- TypeScript configured<br>- Tailwind CSS + shadcn/ui integrated<br>- Routing structure implemented (/channels, /library, /settings)<br>- API client module created<br>- Type definitions matching backend models<br>- Basic layouts and navigation<br>- Responsive design foundation<br>- Error handling patterns established<br>- Loading states components |
| 8 | User | As a user, I want to manage channels through a web interface so that I don't need to use APIs directly | Proposed | - Channel list page showing all channels with status<br>- Channel card component with icon, name, current program, viewer count<br>- Channel creation form (name, icon, start time, loop setting)<br>- Channel editor with all settings<br>- Playlist editor with drag-and-drop reordering<br>- Media browser integration for adding content<br>- Delete channel with confirmation dialog<br>- Visual feedback for all operations<br>- Real-time preview of "what's playing now"<br>- Responsive design for mobile/tablet/desktop |
| 9 | User | As a user, I want to browse and organize my media library through a web interface so that I can easily manage my content | Proposed | - Media library page with table/grid view<br>- Display columns: title, show, season, episode, duration, resolution, actions<br>- Filter by show name<br>- Search by title functionality<br>- Scan library button triggering directory scan<br>- Progress indicator during scanning<br>- Media metadata editor<br>- Media browser component for playlist selection<br>- Thumbnail display where available<br>- Responsive design |
| 10 | User | As a user, I want to watch my channels through a web player so that I can enjoy my content | Proposed | - HLS video player integration (HLS.js or Video.js)<br>- Full-width responsive player (16:9 aspect ratio)<br>- "Now playing" display with program title and description<br>- Progress bar showing position in current program<br>- "Next up" preview section<br>- Channel schedule sidebar/bottom sheet<br>- Schedule shows next 24 hours with thumbnails<br>- Auto-refresh current program info<br>- Quality selection interface<br>- Live indicator<br>- Error handling and reconnection logic<br>- Mobile and TV browser compatibility |
| 11 | User | As a user, I want to configure system settings and deploy easily so that I can customize and maintain the service | Proposed | - Settings page UI for all configuration options<br>- Media library path configuration<br>- Streaming quality presets (High/Medium/Low)<br>- Hardware acceleration selection<br>- Custom FFmpeg parameters (advanced)<br>- Server port and host settings<br>- Settings API integration<br>- Hardware encoder detection and display<br>- Docker Compose configuration<br>- Comprehensive README with setup instructions<br>- Deployment guide documentation<br>- Error messages with actionable suggestions<br>- Loading states throughout application<br>- Confirmation dialogs for destructive actions<br>- End-to-end testing suite |

## PBI Details

- [PBI 1: Project Setup & Database Foundation](./1/prd.md)
- [PBI 2: Media Library Management Backend](./2/prd.md)
- [PBI 3: Channel Management Backend](./3/prd.md)
- [PBI 4: Virtual Timeline Calculation](./4/prd.md)
- [PBI 5: EPG Generation](./5/prd.md)
- [PBI 6: Streaming Engine](./6/prd.md)
- [PBI 7: Frontend Foundation](./7/prd.md)
- [PBI 8: Channel Management UI](./8/prd.md)
- [PBI 9: Media Library UI](./9/prd.md)
- [PBI 10: Video Player & Streaming UI](./10/prd.md)
- [PBI 11: Settings, Polish & Deployment](./11/prd.md)

## History Log

| Timestamp | PBI_ID | Event_Type | Details | User |
|-----------|--------|------------|---------|------|
| 20251026-000000 | ALL | create_pbi | Initial backlog created from PRD with 11 PBIs | AI_Agent |
| 20251026-120000 | 1 | propose_for_backlog | PBI-1 approved with technical decisions: Go 1.25+, zerolog, golang-migrate, UUIDs | User |
| 20251027-000000 | 2 | propose_for_backlog | PBI-2 approved with async scanning, single library path, MP4/MKV/AVI/MOV support | User |
| 20251027-140000 | 7 | propose_for_backlog | PBI-7 approved with technical decisions: Next.js 15.5, React 19.1.1, TanStack Query 5.85, Zustand 5.0.8, pnpm, dark mode enabled | User |
| 20251028-000000 | 3 | propose_for_backlog | PBI-3 approved with channel name uniqueness enforced, no channel limits, empty playlist validation, transaction-based concurrency | User |

