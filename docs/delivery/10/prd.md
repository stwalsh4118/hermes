# PBI-10: Video Player & Streaming UI

[View in Backlog](../backlog.md#user-content-10)

## Overview

Implement the video player interface that allows users to watch their virtual TV channels in a web browser, complete with HLS streaming, "now playing" information, program schedules, and responsive design for mobile, tablet, desktop, and TV browsers.

## Problem Statement

Users need a polished viewing experience that:
- Plays HLS streams smoothly
- Shows what's currently playing with details
- Displays upcoming programs
- Works on any device (phone, tablet, computer, TV)
- Feels like watching real television
- Provides quality selection for different connection speeds
- Handles errors gracefully and reconnects automatically

The player must:
- Start streaming quickly
- Maintain synchronization with the server
- Display program metadata
- Update "now playing" info automatically when programs change
- Provide intuitive controls (limited to volume, quality, fullscreen - no pause/rewind)
- Look professional and polished

## User Stories

**As a user, I want to:**
- Click on a channel and start watching immediately
- See what's currently playing (show title, episode, description)
- Know how long the current program has been playing and how much is left
- See what's coming up next
- View the schedule for the next several hours
- Choose video quality if my connection is slow
- Go fullscreen on any device
- Have the player work on my phone, tablet, computer, and smart TV browser
- Have playback automatically reconnect if my connection drops briefly
- See error messages if something goes wrong

**As a user, I should NOT be able to:**
- Pause the stream (it's live TV)
- Rewind or fast forward
- Choose specific episodes (defeats the purpose)

## Technical Approach

### Components to Build

1. **Channel Player Page** (`app/channels/[id]/page.tsx`)
   - Full-width video player
   - "Now Playing" section below player
   - Schedule sidebar (desktop) or bottom sheet (mobile)
   - Responsive layout

2. **HLS Video Player Component** (`components/video-player.tsx`)
   - Integrate HLS.js or Video.js
   - Custom controls (volume, quality, fullscreen)
   - No seek bar (or disabled seek bar)
   - Auto-play on load
   - Quality selector
   - Error handling with retry logic

3. **Now Playing Component** (`components/now-playing.tsx`)
   - Program title (show name + episode)
   - Episode description
   - Program thumbnail/poster
   - Progress bar (elapsed vs total duration)
   - Time remaining

4. **Next Up Component** (`components/next-up.tsx`)
   - Upcoming program preview
   - Thumbnail
   - Title and brief info
   - Starts in: countdown or time

5. **Schedule Sidebar** (`components/schedule-sidebar.tsx`)
   - Next 24 hours of programs
   - Each entry: time, thumbnail, title
   - Currently playing highlighted
   - Scrollable list
   - Auto-updates

### HLS Player Integration

**Option 1: HLS.js** (Recommended for flexibility)
```typescript
import Hls from 'hls.js';

const VideoPlayer = ({ channelId }: { channelId: string }) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    if (Hls.isSupported()) {
      const hls = new Hls({
        enableWorker: true,
        lowLatencyMode: true,
      });
      
      hls.loadSource(`/stream/${channelId}/master.m3u8`);
      hls.attachMedia(video);
      
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play();
      });

      hls.on(Hls.Events.ERROR, (event, data) => {
        handleError(data);
      });

      hlsRef.current = hls;
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      // Safari native HLS
      video.src = `/stream/${channelId}/master.m3u8`;
      video.play();
    }

    return () => {
      hlsRef.current?.destroy();
    };
  }, [channelId]);

  return <video ref={videoRef} className="w-full" />;
};
```

**Option 2: Video.js** (More complete UI out of the box)
```typescript
import videojs from 'video.js';
import 'video.js/dist/video-js.css';

useEffect(() => {
  const player = videojs(videoRef.current, {
    controls: true,
    autoplay: true,
    sources: [{
      src: `/stream/${channelId}/master.m3u8`,
      type: 'application/x-mpegURL'
    }]
  });

  return () => player.dispose();
}, [channelId]);
```

### Custom Controls

Build custom controls overlay:
- Volume slider
- Quality selector (1080p, 720p, 480p, Auto)
- Fullscreen button
- Live indicator (red dot + "LIVE")
- No seek bar (or disabled seek bar for clarity)

### Auto-Update "Now Playing"

```typescript
const useCurrentProgram = (channelId: string) => {
  const [program, setProgram] = useState<CurrentProgram | null>(null);
  
  useEffect(() => {
    const fetchProgram = async () => {
      const data = await api.channels.getCurrent(channelId);
      setProgram(data);
    };
    
    fetchProgram();
    const interval = setInterval(fetchProgram, 10000); // Poll every 10s
    
    return () => clearInterval(interval);
  }, [channelId]);
  
  return program;
};
```

### Schedule Display

Fetch schedule for next 24 hours:
```typescript
const useSchedule = (channelId: string) => {
  const [schedule, setSchedule] = useState<EPGProgram[]>([]);
  
  useEffect(() => {
    const fetchSchedule = async () => {
      const data = await api.channels.getSchedule(channelId, '24h');
      setSchedule(data);
    };
    
    fetchSchedule();
    const interval = setInterval(fetchSchedule, 60000); // Refresh every minute
    
    return () => clearInterval(interval);
  }, [channelId]);
  
  return schedule;
};
```

### Responsive Layout

**Desktop:**
```
┌─────────────────────────────────┬──────────┐
│                                 │          │
│         Video Player            │ Schedule │
│         (16:9)                  │ Sidebar  │
│                                 │          │
├─────────────────────────────────┤          │
│ Now Playing Info                │          │
├─────────────────────────────────┤          │
│ Next Up                         │          │
└─────────────────────────────────┴──────────┘
```

**Mobile:**
```
┌─────────────────────────────────┐
│                                 │
│         Video Player            │
│         (16:9)                  │
│                                 │
├─────────────────────────────────┤
│ Now Playing Info                │
├─────────────────────────────────┤
│ Next Up                         │
├─────────────────────────────────┤
│ Schedule (bottom sheet/collapsible) │
└─────────────────────────────────┘
```

### Error Handling

- Network error: Show "Connection lost, retrying..." and auto-retry
- Stream not available: Show "Stream temporarily unavailable"
- Browser not supported: Show message to use modern browser
- HLS errors: Attempt to recover with `hls.startLoad()`

## UX/UI Considerations

### Visual Design
- Minimal UI that fades during playback
- Controls visible on hover/tap, fade after 3 seconds
- Dark theme for player area (reduces eye strain)
- Program thumbnails high quality
- Professional, TV-like aesthetic

### Interaction Design
- Click/tap player to show controls
- Click/tap again to play/pause (but buffer doesn't pause, user catches up)
- Double-click for fullscreen
- Keyboard shortcuts (Space: play/pause, F: fullscreen, M: mute)
- Smooth transitions

### Performance
- Preload "now playing" data before stream starts
- Lazy load schedule data
- Use placeholder images while thumbnails load
- Optimize video player for low latency

### Mobile/TV Considerations
- Touch-friendly controls (large tap targets)
- Swipe gestures (optional: swipe up for schedule, swipe down to close)
- TV remote control support (arrow keys, OK button)
- Orientation lock for mobile (landscape recommended)

## Acceptance Criteria

- [ ] HLS video player integration complete (HLS.js or Video.js)
- [ ] Player loads and starts streaming within 10 seconds
- [ ] Video plays smoothly without buffering (on stable connection)
- [ ] Custom controls implemented (volume, quality, fullscreen)
- [ ] Quality selector allows choosing 1080p, 720p, 480p, or Auto
- [ ] No seek bar or seek bar is disabled (no rewind/fast-forward)
- [ ] Live indicator displayed (red dot + "LIVE" text)
- [ ] "Now Playing" section shows current program details
- [ ] Program title, episode info, and description displayed
- [ ] Progress bar shows elapsed time and remaining time
- [ ] Progress bar updates in real-time
- [ ] "Next Up" section shows upcoming program preview
- [ ] Schedule sidebar displays next 24 hours of programs
- [ ] Schedule updates automatically (every minute)
- [ ] Currently playing program highlighted in schedule
- [ ] Schedule scrollable and all programs accessible
- [ ] Full-width responsive player maintains 16:9 aspect ratio
- [ ] Desktop layout: player + sidebar side-by-side
- [ ] Mobile layout: player + bottom sheet for schedule
- [ ] Tablet layout: appropriate adaptation
- [ ] Fullscreen mode works correctly
- [ ] Player controls fade out after 3 seconds of inactivity
- [ ] Controls reappear on mouse move or tap
- [ ] Keyboard shortcuts work (space, F, M, arrows)
- [ ] Error handling displays user-friendly messages
- [ ] Auto-reconnect on connection loss
- [ ] Loading spinner shown while stream initializes
- [ ] Player works on Chrome, Firefox, Safari, Edge
- [ ] Player works on mobile browsers (iOS Safari, Chrome)
- [ ] Player works on TV browsers (basic compatibility)
- [ ] No console errors during normal operation
- [ ] Performance acceptable (60fps UI, minimal CPU usage)

## Dependencies

**PBI Dependencies:**
- PBI-3: Channel Management Backend (REQUIRED - provides channel data)
- PBI-5: EPG Generation (REQUIRED - provides "now playing" and schedule)
- PBI-6: Streaming Engine (REQUIRED - provides HLS streams)
- PBI-7: Frontend Foundation (REQUIRED - provides base setup)

**External Dependencies:**
- HLS.js or Video.js library
- Browser support for HLS or Media Source Extensions

## Open Questions

- Should we use HLS.js or Video.js? (HLS.js recommended for more control)
- Do we want picture-in-picture support?
- Should we show viewer count for the channel?
- Do we need Chromecast or AirPlay support?
- Should we implement a mini player that stays visible while browsing?
- Do we want to show "up next" countdown timer?
- Should we auto-advance to the channel list if stream ends (non-looping)?
- Do we need closed captions/subtitle support?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.


