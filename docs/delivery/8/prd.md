# PBI-8: Channel Management UI

[View in Backlog](../backlog.md#user-content-8)

## Overview

Build the user interface for managing TV channels, including listing all channels, creating new channels, editing channel configuration, managing playlists with drag-and-drop reordering, and deleting channels.

## Problem Statement

Users need an intuitive interface to:
- View all their channels at a glance
- Create new channels with custom configuration
- Edit existing channels and their playlists
- Reorder playlist items easily (drag and drop)
- Add media to playlists from their library
- Delete channels they no longer want

The interface should be:
- Visually appealing and modern
- Easy to use without training
- Responsive across devices
- Provide immediate feedback for actions
- Show real-time preview of "what's playing now"

## User Stories

**As a user, I want to:**
- See all my channels on one page with their current status
- Click a button to create a new channel
- Fill out a simple form to configure my channel
- Browse my media library and add items to my channel's playlist
- Drag and drop to reorder shows in my playlist
- See how long my total playlist is
- Preview what would be playing right now based on my settings
- Save my channel and see it appear in my channel list
- Edit my channel later if I want to make changes
- Delete a channel with a confirmation to prevent accidents

## Technical Approach

### Components to Build

1. **Channel List Page** (`app/channels/page.tsx`)
   - Grid layout of channel cards
   - Shows: icon, name, "now playing", live indicator, viewer count
   - "Create Channel" button
   - Responsive grid (1 col mobile, 2-3 tablet, 4+ desktop)

2. **Channel Card Component** (`components/channel-card.tsx`)
   ```typescript
   interface ChannelCardProps {
     channel: Channel;
     currentProgram?: CurrentProgram;
     viewerCount: number;
   }
   ```
   - Channel icon/logo
   - Channel name
   - Current program with thumbnail
   - Live indicator (red dot + "LIVE")
   - Click to watch
   - Context menu for edit/delete

3. **Channel Creation Page** (`app/channels/new/page.tsx`)
   - Form with sections:
     - Basic Info (name, icon, start date/time, loop toggle)
     - Playlist Builder
   - "Save" and "Cancel" buttons

4. **Channel Editor Page** (`app/channels/[id]/edit/page.tsx`)
   - Same as creation page but pre-filled with existing data
   - Additional "Delete Channel" button

5. **Playlist Editor Component** (`components/playlist-editor.tsx`)
   - Drag-and-drop list using `@dnd-kit` or similar
   - Each item shows: thumbnail, title (show name + episode), duration
   - "Add Media" button opens media browser
   - Remove button (X) on each item
   - Total duration display at bottom
   - Real-time reorder with visual feedback

6. **Media Browser Component** (`components/media-browser.tsx`)
   - Modal/drawer that opens when adding media
   - Lists all media from library
   - Search and filter by show name
   - Click to add to playlist
   - Shows which items are already in playlist (disabled/checked)

7. **Channel Preview Component** (`components/channel-preview.tsx`)
   - Shows "Currently Playing" based on form settings
   - Updates when start time or playlist changes
   - Visual indicator of what program would be airing

### Form Handling

Use React Hook Form or similar for:
- Input validation
- Error display
- Form state management

**Validation Rules:**
- Name: Required, max 100 characters
- Start Time: Required, valid datetime
- Icon: Optional, valid URL format
- Playlist: At least 1 item recommended (warning if empty)

### Drag and Drop

Use `@dnd-kit/core` and `@dnd-kit/sortable`:
```typescript
const sensors = useSensors(
  useSensor(PointerSensor),
  useSensor(KeyboardSensor, {
    coordinateGetter: sortableKeyboardCoordinates,
  })
);

function handleDragEnd(event: DragEndEvent) {
  const {active, over} = event;
  // Reorder playlist items
  // Call API to update positions
}
```

### State Management

- Channel list: Fetch on mount, refresh after create/edit/delete
- Form state: Local component state or React Hook Form
- Playlist state: Local array, sync with backend on save
- Optimistic updates for better UX (update UI immediately, sync with backend)

### API Integration

```typescript
// Create channel
const createChannel = async (data: CreateChannelRequest) => {
  const response = await api.channels.create(data);
  toast.success('Channel created!');
  router.push('/channels');
};

// Update playlist order
const reorderPlaylist = async (channelId: string, items: PlaylistItem[]) => {
  await api.channels.reorderPlaylist(channelId, items);
};
```

## UX/UI Considerations

### Visual Design
- Cards with shadows/borders for depth
- Channel icons prominent and large
- Current program thumbnails visible
- Live indicator stands out (red/orange)
- Clear CTAs (Call To Action buttons)

### Interaction Design
- Hover states on all interactive elements
- Smooth drag animations
- Immediate feedback on actions
- Loading spinners during API calls
- Success toasts after saves
- Confirmation dialog before delete

### Responsive Behavior
- Mobile: Single column, simplified layout, touch-friendly targets
- Tablet: 2-3 columns, drawer for media browser
- Desktop: 4+ columns, modal for media browser, more information density

### Error Handling
- Show validation errors inline
- Display API errors in toasts
- Provide helpful messages ("Channel name is required")
- Don't lose form data on error

## Acceptance Criteria

- [ ] Channel list page displays all channels in responsive grid
- [ ] Channel card component shows icon, name, current program, live status
- [ ] "Create Channel" button navigates to creation form
- [ ] Channel creation form includes all required fields
- [ ] Icon upload/URL input functional
- [ ] Start date/time picker works correctly
- [ ] Loop toggle works and displays appropriately
- [ ] Playlist editor displays all playlist items
- [ ] Drag-and-drop reordering works smoothly
- [ ] "Add Media" button opens media browser
- [ ] Media browser displays all library media with search/filter
- [ ] Adding media to playlist updates the list immediately
- [ ] Remove item from playlist works correctly
- [ ] Total playlist duration calculated and displayed
- [ ] Channel preview shows "what's playing now" based on settings
- [ ] Preview updates when start time or playlist changes
- [ ] Form validation prevents invalid submissions
- [ ] Validation errors displayed clearly
- [ ] Save creates channel and navigates to channel list
- [ ] New channel appears in channel list immediately
- [ ] Edit channel page loads with existing data
- [ ] Editing channel saves changes correctly
- [ ] Delete channel shows confirmation dialog
- [ ] Confirming deletion removes channel and returns to list
- [ ] Loading states shown during API operations
- [ ] Success/error toasts appear for user actions
- [ ] Responsive design works on mobile, tablet, desktop
- [ ] Touch gestures work on mobile (tap, drag)
- [ ] Keyboard navigation works for accessibility

## Dependencies

**PBI Dependencies:**
- PBI-3: Channel Management Backend (REQUIRED - provides APIs)
- PBI-7: Frontend Foundation (REQUIRED - provides base setup)

**External Dependencies:**
- @dnd-kit/core and @dnd-kit/sortable for drag-drop
- date-fns or dayjs for date handling
- React Hook Form (optional, for form management)

## Open Questions

- Should we support uploading icons or only URL input?
- How should we handle very long playlists (100+ items)?
- Should drag-drop work on mobile or use alternative UI (up/down buttons)?
- Do we need bulk operations (add multiple media at once)?
- Should we show a progress bar when saving large playlists?
- Do we need undo/redo for playlist changes?
- Should changes auto-save or require explicit save button?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.

