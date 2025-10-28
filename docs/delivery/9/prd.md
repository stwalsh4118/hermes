# PBI-9: Media Library UI

[View in Backlog](../backlog.md#user-content-9)

## Overview

Build the user interface for browsing, organizing, and managing the media library, including viewing all media in table or grid format, searching and filtering, triggering library scans, editing metadata, and integrating a media browser for playlist selection.

## Problem Statement

Users need an efficient interface to:
- View all media in their library
- Scan directories to import new content
- Search for specific titles or shows
- Filter media by show name, season, format, etc.
- Edit metadata when automatic detection is incorrect
- See media details (duration, resolution, codecs)
- Know which files need transcoding
- Select media when building channel playlists

The interface should:
- Handle large libraries (1000+ files) efficiently
- Provide both overview (grid) and detail (table) views
- Show scanning progress
- Display media in an organized, sortable manner

## User Stories

**As a user, I want to:**
- Browse my entire media library in an organized view
- Click "Scan Library" and have it import new files automatically
- See progress while scanning is happening
- Search for a specific show or episode quickly
- Filter my library by show name to see all episodes
- Sort by title, date added, duration, etc.
- Click on media to see full details (codecs, file path, etc.)
- Edit show name, season, or episode if it's wrong
- See which files need transcoding and why
- View my library as thumbnails (grid) or details (table)
- Select media from a browser when building playlists

## Technical Approach

### Components to Build

1. **Media Library Page** (`app/library/page.tsx`)
   - View toggle (table vs grid)
   - Search bar
   - Filter dropdown(s)
   - "Scan Library" button
   - Media display area (table or grid)
   - Pagination or infinite scroll

2. **Media Table Component** (`components/media-table.tsx`)
   - Columns: Thumbnail, Title, Show, Season, Episode, Duration, Resolution, Codec, Actions
   - Sortable columns
   - Row click to view details
   - Action buttons: Edit, Remove
   - Responsive: Fewer columns on mobile
   - Use shadcn/ui Table components

3. **Media Grid Component** (`components/media-grid.tsx`)
   - Card-based layout
   - Shows: Thumbnail, title, duration badge
   - Hover shows: Show name, season/episode, resolution
   - Click to view details

4. **Media Detail Modal** (`components/media-detail-modal.tsx`)
   - Full metadata display
   - File information
   - Transcoding requirements
   - Edit button to switch to edit mode

5. **Media Editor Modal** (`components/media-editor-modal.tsx`)
   - Edit fields: Title, Show Name, Season, Episode
   - Save/Cancel buttons
   - Validation

6. **Media Browser Component** (`components/media-browser.tsx`) *[Already needed for PBI-8]*
   - Modal/drawer for selecting media
   - Search and filter
   - Multi-select option (add multiple to playlist)
   - Show which items already in current playlist

7. **Library Scanner Component** (`components/library-scanner.tsx`)
   - Scan button
   - Progress bar during scan
   - Stats: Files found, Files processed, Errors
   - Cancel button
   - Results summary when complete

### Data Fetching and State

```typescript
// hooks/use-media.ts
export function useMedia() {
  const [media, setMedia] = useState<Media[]>([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({
    search: '',
    showName: '',
  });

  useEffect(() => {
    fetchMedia();
  }, [filters]);

  const fetchMedia = async () => {
    setLoading(true);
    const data = await api.media.list(filters);
    setMedia(data);
    setLoading(false);
  };

  return { media, loading, filters, setFilters, refetch: fetchMedia };
}
```

### Search and Filter

- **Search**: Client-side filter on title/show name (or server-side for large libraries)
- **Filters**: 
  - Show name (dropdown of unique shows)
  - Season (dropdown)
  - Needs transcoding (checkbox)
  - Format (MP4, MKV, etc.)

### Scanning Flow

1. User clicks "Scan Library"
2. Show modal with progress
3. POST to `/api/media/scan`
4. Poll for progress or use WebSocket for real-time updates
5. Show results: X files added, Y files skipped, Z errors
6. Refresh media list

```typescript
const startScan = async () => {
  setScanProgress({ status: 'scanning', processed: 0, total: 0 });
  
  try {
    const result = await api.media.scan();
    setScanProgress({ status: 'complete', ...result });
    refetchMedia();
    toast.success(`Scan complete! ${result.added} files added.`);
  } catch (error) {
    toast.error('Scan failed');
    setScanProgress({ status: 'error' });
  }
};
```

### Pagination/Virtual Scrolling

For large libraries, implement:
- **Server-side pagination**: Limit/offset parameters
- **Infinite scroll**: Load more as user scrolls
- **Virtual scrolling**: Render only visible rows (using react-virtual or similar)

### Sorting

Support sorting by:
- Title (A-Z, Z-A)
- Show name
- Date added (newest first, oldest first)
- Duration (longest first, shortest first)
- Season/Episode

## UX/UI Considerations

### Visual Design
- Table view: Dense, information-rich, professional
- Grid view: Visual, thumbnail-focused, browsable
- Consistent with overall app aesthetic
- Transcoding warning icon (badge on items needing transcode)

### Interaction Design
- Instant search feedback (debounced)
- Smooth view transitions (table ↔ grid)
- Loading skeletons while fetching
- Empty states ("No media found", "Library is empty")
- Tooltips for technical info (codec names, etc.)

### Performance
- Paginate or virtualize for 1000+ items
- Debounce search input (300ms)
- Cache results where appropriate
- Show loading indicators

### Mobile Experience
- Table: Show only essential columns
- Grid: 2 columns on mobile, 3-4 on tablet
- Touch-friendly tap targets
- Simplified filters (drawer/bottom sheet)

## Acceptance Criteria

- [ ] Media library page renders with table and grid view options
- [ ] View toggle switches between table and grid
- [ ] Media table displays all columns correctly
- [ ] Media grid displays thumbnail cards correctly
- [ ] Search bar filters media by title or show name
- [ ] Filter by show name works correctly
- [ ] Sorting works for all sortable columns
- [ ] "Scan Library" button triggers scan operation
- [ ] Scan progress modal shows real-time progress
- [ ] Scan results displayed when complete
- [ ] Media list refreshes after successful scan
- [ ] Click on media opens detail modal
- [ ] Media detail modal shows all metadata
- [ ] Edit button opens media editor
- [ ] Media editor allows updating title, show, season, episode
- [ ] Saving edits updates the media in the list
- [ ] Delete media shows confirmation dialog
- [ ] Confirming deletion removes media from list
- [ ] Transcoding requirements displayed for each media item
- [ ] Pagination or infinite scroll works for large libraries
- [ ] Loading states shown during data fetching
- [ ] Empty state shown when library is empty
- [ ] Error messages displayed for failed operations
- [ ] Responsive design works on mobile, tablet, desktop
- [ ] Table view readable on mobile (fewer columns)
- [ ] Performance acceptable with 1000+ media items

## Dependencies

**PBI Dependencies:**
- PBI-2: Media Library Management Backend (REQUIRED - provides APIs)
- PBI-7: Frontend Foundation (REQUIRED - provides base setup)

**External Dependencies:**
- @tanstack/react-table (optional, for advanced table features)
- react-virtual (optional, for virtual scrolling)
- react-dropzone (optional, for drag-drop file upload in future)

## Open Questions

- Should we support uploading files directly through the UI?
- Do we need to show folder structure or just a flat list?
- Should we generate thumbnails from videos or use placeholders?
- How should we handle duplicate files detected during scan?
- Should scan be queue-based to prevent multiple concurrent scans?
- Do we need bulk operations (bulk edit, bulk delete)?
- Should we support exporting media list (CSV, JSON)?
- Do we need a "Recently Added" view?

## Related Tasks

See [tasks.md](./tasks.md) for the complete list of tasks for this PBI.


