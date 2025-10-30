# Frontend Infrastructure API

Last Updated: 2025-10-30 (Library scanner component and hook added)

## Utilities

### Class Name Utility

Location: `web/lib/utils.ts`

**cn() - Merge Tailwind Classes:**
```typescript
function cn(...inputs: ClassValue[]): string
```

Combines multiple class names and merges Tailwind classes intelligently, resolving conflicts.

**Usage Example:**
```typescript
import { cn } from "@/lib/utils";

// Basic usage
cn("text-red-500", "bg-blue-500");

// Conditional classes
cn("base-class", isActive && "active-class");

// Override Tailwind classes
cn("text-red-500", "text-blue-500"); // Results in text-blue-500

// With component props
<div className={cn("default-styles", className)} />
```

### Format Utilities

Location: `web/lib/utils/format.ts`

**formatDuration() - Convert Seconds to Human-Readable Duration:**
```typescript
function formatDuration(seconds: number): string
// Returns: "2h 30m", "45m", "1h", "0m"
```

**formatCount() - Pluralize Item Counts:**
```typescript
function formatCount(count: number, singular: string, plural?: string): string
// Returns: "1 item", "5 items", "1 episode", "3 episodes"
```

### Metadata Utility

Location: `web/lib/metadata.ts`

**createMetadata() - Generate Page Metadata:**
```typescript
interface PageMetadata {
  title: string;
  description: string;
  path?: string;
}

function createMetadata({ title, description, path = "" }: PageMetadata): Metadata
```

Creates consistent Next.js metadata including SEO, Open Graph, and Twitter Card tags.

**Usage Example:**
```typescript
import type { Metadata } from "next";
import { createMetadata } from "@/lib/metadata";

export const metadata: Metadata = createMetadata({
  title: "Channels",
  description: "Manage your virtual TV channels",
  path: "/channels",
});
```

**Generated Metadata:**
- Page title (uses template from root layout)
- Description
- Open Graph tags (title, description, url)
- Twitter Card tags (summary_large_image, title, description)
- Dynamic URL generation using `NEXT_PUBLIC_APP_URL` environment variable

## UI Components

### shadcn/ui Component Library

Location: `web/components/ui/`

All components are built on Radix UI primitives with full dark mode support.

**Available Components:**
- **Forms**: button, input, label, select, textarea, checkbox, radio-group, switch, form
- **Data Display**: card, table, badge, avatar, separator, skeleton
- **Feedback**: alert, alert-dialog, dialog, sonner (toast), spinner, progress
- **Navigation**: dropdown-menu, navigation-menu, tabs, breadcrumb, menubar
- **Layout**: sidebar, sheet, drawer, collapsible, resizable, scroll-area
- **Advanced**: calendar, carousel, command, context-menu, hover-card, popover, tooltip
- **Specialized**: input-otp, kbd, pagination, toggle, toggle-group, accordion, aspect-ratio

**Usage Example:**
```typescript
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

<Card>
  <CardHeader>
    <CardTitle>Title</CardTitle>
  </CardHeader>
  <CardContent>
    <Button>Click me</Button>
  </CardContent>
</Card>
```

**Toast Notifications:**
```typescript
import { toast } from "sonner";

toast.success("Success message");
toast.error("Error message");
toast.info("Info message", { description: "Additional details" });
```

## Layout Components

### MainLayout

Location: `web/components/layout/main-layout.tsx`

Reusable wrapper providing consistent padding and responsive container for all pages.

**Props:**
```typescript
interface MainLayoutProps {
  children: React.ReactNode;
  className?: string;
}
```

**Usage:**
```typescript
import { MainLayout } from "@/components/layout/main-layout";

export default function MyPage() {
  return (
    <MainLayout>
      <h1>Page Content</h1>
    </MainLayout>
  );
}
```

### PageHeader

Location: `web/components/layout/page-header.tsx`

Reusable page header with title, optional description, and action buttons slot.

**Props:**
```typescript
interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: React.ReactNode;
}
```

**Usage:**
```typescript
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";

<PageHeader
  title="Channels"
  description="Manage your virtual TV channels"
  actions={<Button>New Channel</Button>}
/>
```

## Common Components

Location: `web/components/common/`

Reusable components for loading states, error messages, empty states, and data display patterns.

### LoadingSpinner

Location: `web/components/common/loading-spinner.tsx`

Animated loading spinner with size variants.

**Props:**
```typescript
interface LoadingSpinnerProps {
  size?: "sm" | "md" | "lg";
  className?: string;
}
```

**Usage:**
```typescript
import { LoadingSpinner } from "@/components/common/loading-spinner";

<LoadingSpinner size="md" />
```

**Features:**
- Three size variants: sm (4x4), md (8x8), lg (12x12)
- Uses Loader2 icon from lucide-react
- Smooth spin animation
- Centered in flex container with padding

### SkeletonCard

Location: `web/components/common/skeleton-card.tsx`

Card skeleton loader matching typical card layout.

**Usage:**
```typescript
import { SkeletonCard } from "@/components/common/skeleton-card";

<SkeletonCard />
```

**Features:**
- Matches shadcn Card structure (header + content)
- Placeholder for title, subtitle, image, and text lines
- Responsive sizing

### SkeletonList

Location: `web/components/common/skeleton-list.tsx`

List skeleton loader with configurable item count.

**Props:**
```typescript
interface SkeletonListProps {
  count?: number; // Default: 5
}
```

**Usage:**
```typescript
import { SkeletonList } from "@/components/common/skeleton-list";

<SkeletonList count={3} />
```

**Features:**
- Configurable number of items
- Avatar + two-line text pattern
- Consistent spacing

### ErrorMessage

Location: `web/components/common/error-message.tsx`

Error alert with optional retry button.

**Props:**
```typescript
interface ErrorMessageProps {
  title?: string;        // Default: "Error"
  message: string;
  onRetry?: () => void;
  showIcon?: boolean;    // Default: true
}
```

**Usage:**
```typescript
import { ErrorMessage } from "@/components/common/error-message";

<ErrorMessage
  message="Failed to load data"
  onRetry={() => refetch()}
/>
```

**Features:**
- Uses shadcn Alert component (destructive variant)
- AlertCircle icon
- Optional retry button with RefreshCw icon
- Accessible and semantic markup

### EmptyState

Location: `web/components/common/empty-state.tsx`

Empty state component with icon, text, and optional action.

**Props:**
```typescript
interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}
```

**Usage:**
```typescript
import { EmptyState } from "@/components/common/empty-state";
import { InboxIcon } from "lucide-react";

<EmptyState
  icon={InboxIcon}
  title="No channels yet"
  description="Get started by creating your first TV channel"
  action={{
    label: "Create Channel",
    onClick: () => router.push("/channels/new")
  }}
/>
```

**Features:**
- Icon in circular muted background
- Centered layout with max-width description
- Optional call-to-action button
- Flexible and composable

### DataWrapper

Location: `web/components/common/data-wrapper.tsx`

Generic wrapper component that handles loading, error, and empty states with render props pattern.

**Props:**
```typescript
interface DataWrapperProps<T> {
  data: T | undefined;
  isLoading: boolean;
  error: Error | null;
  isEmpty?: (data: T) => boolean;
  emptyState?: {
    icon?: LucideIcon;
    title: string;
    description: string;
    action?: {
      label: string;
      onClick: () => void;
    };
  };
  onRetry?: () => void;
  children: (data: T) => React.ReactNode;
}
```

**Usage:**
```typescript
import { DataWrapper } from "@/components/common/data-wrapper";
import { InboxIcon } from "lucide-react";

const { data, isLoading, error } = useChannels();

<DataWrapper
  data={data}
  isLoading={isLoading}
  error={error}
  isEmpty={(data) => data.length === 0}
  emptyState={{
    icon: InboxIcon,
    title: "No channels",
    description: "Create your first channel to get started"
  }}
  onRetry={refetch}
>
  {(channels) => (
    <div>
      {channels.map(channel => <ChannelCard key={channel.id} channel={channel} />)}
    </div>
  )}
</DataWrapper>
```

**Features:**
- Unified handling of loading/error/empty/data states
- Type-safe with TypeScript generics
- Render props pattern for flexible content rendering
- Composes LoadingSpinner, ErrorMessage, and EmptyState
- Custom isEmpty function for flexible empty detection

### ConfirmDialog

Location: `web/components/common/confirm-dialog.tsx`

Confirmation dialog for destructive or important actions.

**Props:**
```typescript
interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  confirmLabel?: string;    // Default: "Continue"
  cancelLabel?: string;     // Default: "Cancel"
  onConfirm: () => void;
  variant?: "default" | "destructive";  // Default: "default"
}
```

**Usage:**
```typescript
import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { useState } from "react";

const [open, setOpen] = useState(false);

<>
  <Button onClick={() => setOpen(true)}>Delete Channel</Button>
  
  <ConfirmDialog
    open={open}
    onOpenChange={setOpen}
    title="Delete channel?"
    description="This action cannot be undone. The channel will be permanently deleted."
    confirmLabel="Delete"
    variant="destructive"
    onConfirm={() => {
      deleteChannel(id);
      setOpen(false);
    }}
  />
</>
```

**Features:**
- Uses shadcn AlertDialog component
- Controlled open state
- Destructive variant with red styling
- Accessible with keyboard and screen reader support
- Backdrop overlay

### Common Components Index

Location: `web/components/common/index.ts`

Barrel export for all common components.

**Usage:**
```typescript
import {
  LoadingSpinner,
  SkeletonCard,
  SkeletonList,
  ErrorMessage,
  EmptyState,
  DataWrapper,
  ConfirmDialog,
  ThemeToggle,
} from "@/components/common";
```

## Navigation Components

### Navigation Configuration

Location: `web/lib/config/navigation.ts`

Centralized navigation items configuration used by Header and MobileMenu components.

**Types:**
```typescript
interface NavItem {
  title: string;
  href: string;
  icon: LucideIcon;
  description?: string;
}
```

**Available Navigation Items:**
```typescript
export const navItems: NavItem[] = [
  { title: "Home", href: "/", icon: Home, description: "Dashboard and overview" },
  { title: "Channels", href: "/channels", icon: Play, description: "Manage TV channels" },
  { title: "Library", href: "/library", icon: Library, description: "Browse media files" },
  { title: "Settings", href: "/settings", icon: Settings, description: "Configure preferences" },
];
```

**Usage:**
```typescript
import { navItems } from "@/lib/config/navigation";

// Iterate over navigation items
navItems.map((item) => {
  const Icon = item.icon;
  return <Link href={item.href}><Icon /> {item.title}</Link>;
});
```

### AppShell

Location: `web/components/layout/app-shell.tsx`

Top-level layout wrapper that provides the application shell including header and mobile menu.

**Props:**
```typescript
interface AppShellProps {
  children: React.ReactNode;
}
```

**Usage:**
```typescript
import { AppShell } from "@/components/layout/app-shell";

// Typically used in root layout (app/layout.tsx)
<AppShell>
  {children}
</AppShell>
```

**Features:**
- Sticky header at top
- Flexible main content area
- Mobile menu overlay
- Responsive design (desktop/mobile)

### Header

Location: `web/components/layout/header.tsx`

Main navigation header with logo, navigation links, theme toggle, and mobile menu button.

**Usage:**
```typescript
import { Header } from "@/components/layout/header";

// Used automatically via AppShell, but can be imported directly if needed
<Header />
```

**Features:**
- Sticky positioning with backdrop blur
- Logo linking to home page
- Desktop navigation (visible at md+ breakpoints)
- Active route highlighting
- Theme toggle button
- Mobile menu button (visible below md breakpoint)
- Integrates with Zustand `useUIStore` for mobile menu state

**Active Route Logic:**
- Root path `/` matches exactly
- Other paths match if current path starts with the nav item path
- Example: `/channels/123` highlights "Channels" nav item

### MobileMenu

Location: `web/components/layout/mobile-menu.tsx`

Slide-in mobile navigation drawer with backdrop overlay.

**Usage:**
```typescript
import { MobileMenu } from "@/components/layout/mobile-menu";

// Used automatically via AppShell, but can be imported directly if needed
<MobileMenu />
```

**Features:**
- Slides in from right side
- Backdrop overlay (click to close)
- Close button in header
- Navigation items with icons and descriptions
- Active route highlighting
- Auto-closes on route change
- Prevents body scroll when open
- Controlled by Zustand `useUIStore.mobileMenuOpen` state

**State Management:**
```typescript
import { useUIStore } from "@/lib/stores";

const { mobileMenuOpen, setMobileMenuOpen, toggleMobileMenu } = useUIStore();

// Open menu
setMobileMenuOpen(true);
// Or toggle
toggleMobileMenu();
```

### Navigation Utilities

Location: `web/lib/utils/navigation.ts`

Helper functions for navigation-related logic.

**isActiveRoute() - Check if Route is Active:**
```typescript
function isActiveRoute(currentPath: string, itemPath: string): boolean
```

Determines if a navigation item should be highlighted as active.

**Usage:**
```typescript
import { isActiveRoute } from "@/lib/utils/navigation";
import { usePathname } from "next/navigation";

const pathname = usePathname();
const isActive = isActiveRoute(pathname, "/channels");
```

**Logic:**
- Root path (`/`) matches exactly
- Other paths match if current path starts with item path
- Prevents false positives (e.g., `/settings` won't match `/set`)

## Theming

### Theme Provider

Location: `web/components/providers/theme-provider.tsx`

Wraps application to provide dark/light mode support via `next-themes`.

**Usage:**
```typescript
import { ThemeProvider } from "@/components/providers/theme-provider";

<ThemeProvider attribute="class" defaultTheme="system" enableSystem>
  {children}
</ThemeProvider>
```

### Theme Toggle

Location: `web/components/common/theme-toggle.tsx`

Button component for switching between dark and light themes.

**Usage:**
```typescript
import { ThemeToggle } from "@/components/common/theme-toggle";

<ThemeToggle />
```

## Data Fetching

### TanStack Query Setup

Location: `web/lib/query/client.ts`

**QueryClient Configuration:**
```typescript
import { queryClient } from "@/lib/query/client";
```

**Configuration:**
- staleTime: 5 minutes
- gcTime: 10 minutes (cache time)
- refetchOnWindowFocus: false
- retry: 1

**Query Provider:**
```typescript
import { QueryProvider } from "@/components/providers/query-provider";

// Already included in app/layout.tsx
<QueryProvider>
  {children}
</QueryProvider>
```

**React Query DevTools:**
- Automatically included in development builds
- Access via floating icon in bottom-left corner
- Shows query states, cache, and debugging info

### API Client

Location: `web/lib/api/client.ts`

**Singleton instance:**
```typescript
import { apiClient } from "@/lib/api/client";
```

**Environment Configuration:**
```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
```

**Available Methods:**

**Health:**
- `apiClient.health()` → `Promise<HealthResponse>`

**Channels:**
- `apiClient.getChannels()` → `Promise<Channel[]>`
- `apiClient.getChannel(id)` → `Promise<Channel>`
- `apiClient.createChannel(data)` → `Promise<Channel>`
- `apiClient.updateChannel(id, data)` → `Promise<Channel>`
- `apiClient.deleteChannel(id)` → `Promise<MessageResponse>`

**Playlist:**
- `apiClient.getPlaylist(channelId)` → `Promise<PlaylistResponse>`
- `apiClient.addToPlaylist(channelId, data)` → `Promise<PlaylistItem>`
- `apiClient.bulkAddToPlaylist(channelId, items)` → `Promise<BulkAddResponse>`
- `apiClient.removeFromPlaylist(channelId, itemId)` → `Promise<MessageResponse>`
- `apiClient.reorderPlaylist(channelId, items)` → `Promise<MessageResponse>`

**Media:**
- `apiClient.getMedia(params?)` → `Promise<PaginatedMediaResponse>`
- `apiClient.getMediaItem(id)` → `Promise<Media>`
- `apiClient.updateMedia(id, data)` → `Promise<Media>`
- `apiClient.deleteMedia(id)` → `Promise<MessageResponse>`
- `apiClient.scanMedia(path)` → `Promise<ScanResponse>`
- `apiClient.getScanStatus(scanId)` → `Promise<ScanProgress>`

**Error Handling:**
All methods throw `ApiError` on failure:
```typescript
interface ApiError {
  error: string;
  message: string;
  status: number;
}
```

## Hooks

### Channel Hooks

Location: `web/hooks/use-channels.ts`

**Query Hooks:**
```typescript
import { useChannels, useChannel } from "@/hooks/use-channels";

// List all channels
const { data, isLoading, error } = useChannels();

// Get single channel
const { data, isLoading, error } = useChannel(channelId);
```

**Mutation Hooks:**
```typescript
import {
  useCreateChannel,
  useUpdateChannel,
  useDeleteChannel,
} from "@/hooks/use-channels";

// Create channel
const createMutation = useCreateChannel();
createMutation.mutate({ name: "New Channel", ... });

// Update channel
const updateMutation = useUpdateChannel(channelId);
updateMutation.mutate({ name: "Updated Name" });

// Delete channel
const deleteMutation = useDeleteChannel();
deleteMutation.mutate(channelId);
```

**Query Keys:**
```typescript
import { channelKeys } from "@/hooks/use-channels";

channelKeys.all           // ["channels"]
channelKeys.lists()       // ["channels", "list"]
channelKeys.detail(id)    // ["channels", "detail", id]
```

**Features:**
- Automatic cache invalidation on mutations
- Toast notifications on success/error
- Type-safe request/response handling

### Playlist Hooks

Location: `web/hooks/use-playlist.ts`

**Query Hook:**
```typescript
import { usePlaylist } from "@/hooks/use-playlist";

const { data, isLoading, error } = usePlaylist(channelId);
```

**Mutation Hooks:**
```typescript
import {
  useAddToPlaylist,
  useBulkAddToPlaylist,
  useRemoveFromPlaylist,
  useReorderPlaylist,
} from "@/hooks/use-playlist";

// Add single item to playlist
const addMutation = useAddToPlaylist();
addMutation.mutate({ channelId, data: { media_id, position } });

// Bulk add multiple items (efficient for large selections)
const bulkAddMutation = useBulkAddToPlaylist();
bulkAddMutation.mutate({ 
  channelId, 
  items: [
    { media_id: "uuid-1", position: 0 },
    { media_id: "uuid-2", position: 1 }
  ]
});

// Remove from playlist
const removeMutation = useRemoveFromPlaylist();
removeMutation.mutate({ channelId, itemId });

// Reorder playlist
const reorderMutation = useReorderPlaylist();
reorderMutation.mutate({ channelId, data: { items } });
```

**Query Keys:**
```typescript
import { playlistKeys } from "@/hooks/use-playlist";

playlistKeys.all              // ["playlists"]
playlistKeys.lists()          // ["playlists", "list"]
playlistKeys.list(channelId)  // ["playlists", "list", channelId]
```

### Media Hooks

Location: `web/hooks/use-media.ts`

**Query Hooks:**
```typescript
import { useMedia, useMediaItem, useScanStatus } from "@/hooks/use-media";

// List media with optional filtering/pagination
const { data, isLoading, error } = useMedia({ 
  limit: 20, 
  offset: 0, 
  show: "Friends" 
});

// Get single media item
const { data, isLoading, error } = useMediaItem(mediaId);

// Track scan progress (auto-polling while running)
const { data, isLoading, error } = useScanStatus(scanId);
```

**Mutation Hooks:**
```typescript
import {
  useUpdateMedia,
  useDeleteMedia,
  useScanMedia,
} from "@/hooks/use-media";

// Update media metadata
const updateMutation = useUpdateMedia(mediaId);
updateMutation.mutate({ title: "New Title", season: 2 });

// Delete media
const deleteMutation = useDeleteMedia();
deleteMutation.mutate(mediaId);

// Start media scan
const scanMutation = useScanMedia();
scanMutation.mutate("/path/to/media");
```

**Query Keys:**
```typescript
import { mediaKeys } from "@/hooks/use-media";

mediaKeys.all                  // ["media"]
mediaKeys.lists()              // ["media", "list"]
mediaKeys.list(params)         // ["media", "list", params]
mediaKeys.detail(id)           // ["media", "detail", id]
mediaKeys.scan(scanId)         // ["media", "scan", scanId]
```

**Features:**
- Automatic cache invalidation on mutations
- Toast notifications on success/error
- Auto-polling for scan status (every 2s while running)
- Type-safe request/response handling

### Media Scan Hook

Location: `web/hooks/use-media-scan.ts`

**useMediaScan() - Orchestrate Scanning Workflow:**
```typescript
function useMediaScan(onComplete?: () => void): UseScanResult

interface UseScanResult {
  state: "idle" | "scanning" | "completed" | "failed" | "cancelled";
  scanId: string | null;
  progress: ScanProgress | null;
  error: string | null;
  startScan: (path: string) => Promise<void>;
  cancelScan: () => void;
  reset: () => void;
  elapsedTime: number;
}
```

**Usage:**
```typescript
import { useMediaScan } from "@/hooks/use-media-scan";

const { state, progress, startScan, cancelScan, elapsedTime } = useMediaScan(() => {
  console.log("Scan complete!");
  refetch();
});

// Start scan
await startScan("/media/videos");

// Cancel scan (client-side only)
cancelScan();
```

**Features:**
- State machine management (idle → scanning → completed/failed/cancelled)
- Automatic polling via `useScanStatus` (every 2s while running)
- Elapsed time tracking
- Error handling (409 Conflict, 400 Bad Request, network errors)
- Completion callback support

## State Management (Zustand)

### Store Configuration

Location: `web/lib/stores/`

All stores use Zustand v5 with TypeScript, devtools middleware, and selective persistence.

**Import Stores:**
```typescript
import {
  useUIStore,
  useFilterStore,
  usePreferencesStore,
  usePlayerStore,
} from "@/lib/stores";

// Import types
import type { ViewMode, SortOrder } from "@/lib/stores";
```

### UI Store

Location: `web/lib/stores/ui-store.ts`

Manages ephemeral UI state (not persisted).

**State & Actions:**
```typescript
interface UIState {
  // Sidebar
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;

  // Mobile menu
  mobileMenuOpen: boolean;
  setMobileMenuOpen: (open: boolean) => void;
  toggleMobileMenu: () => void;

  // Modals
  activeModal: string | null;
  openModal: (modalId: string) => void;
  closeModal: () => void;

  // Loading states
  isLoading: boolean;
  setLoading: (loading: boolean) => void;
}
```

**Usage Example:**
```typescript
import { useUIStore } from "@/lib/stores";

function Sidebar() {
  const { sidebarOpen, toggleSidebar } = useUIStore();

  return (
    <aside className={cn(sidebarOpen && "open")}>
      <button onClick={toggleSidebar}>Toggle</button>
    </aside>
  );
}
```

### Filter Store

Location: `web/lib/stores/filter-store.ts`

Manages filter and search state with selective persistence.

**State & Actions:**
```typescript
interface FilterState {
  // Media filters
  mediaSearch: string;
  mediaShowFilter: string | null;
  setMediaSearch: (search: string) => void;
  setMediaShowFilter: (show: string | null) => void;
  clearMediaFilters: () => void;

  // Channel filters
  channelSearch: string;
  setChannelSearch: (search: string) => void;
  clearChannelFilters: () => void;
}
```

**Persistence:**
- Persists: `mediaShowFilter` (only sticky filters)
- Does NOT persist: Search queries (for privacy/freshness)
- Storage: localStorage as `filter-storage` with explicit `createJSONStorage(() => localStorage)` configuration

**Usage Example:**
```typescript
import { useFilterStore } from "@/lib/stores";

function MediaFilters() {
  const { mediaSearch, mediaShowFilter, setMediaSearch, setMediaShowFilter } = useFilterStore();

  return (
    <>
      <Input value={mediaSearch} onChange={(e) => setMediaSearch(e.target.value)} />
      <Select value={mediaShowFilter} onValueChange={setMediaShowFilter}>
        {/* options */}
      </Select>
    </>
  );
}
```

### Preferences Store

Location: `web/lib/stores/preferences-store.ts`

Manages user preferences with full persistence.

**State & Actions:**
```typescript
type ViewMode = "grid" | "list";
type SortOrder = "name" | "date" | "duration";

interface PreferencesState {
  // View preferences
  mediaViewMode: ViewMode;
  channelViewMode: ViewMode;
  setMediaViewMode: (mode: ViewMode) => void;
  setChannelViewMode: (mode: ViewMode) => void;

  // Sort preferences
  mediaSortOrder: SortOrder;
  channelSortOrder: SortOrder;
  setMediaSortOrder: (order: SortOrder) => void;
  setChannelSortOrder: (order: SortOrder) => void;

  // Player preferences
  defaultVolume: number;
  setDefaultVolume: (volume: number) => void;

  // Reset
  resetPreferences: () => void;
}
```

**Defaults:**
- mediaViewMode: `"grid"`
- channelViewMode: `"grid"`
- mediaSortOrder: `"name"`
- channelSortOrder: `"name"`

**Persistence:**
- All preferences persisted to localStorage as `preferences-storage`
- Explicit storage configuration: `createJSONStorage(() => localStorage)`
- Survives page refreshes and browser restarts
- `resetPreferences()` clears both in-memory state and localStorage

**Usage Example:**
```typescript
import { usePreferencesStore } from "@/lib/stores";

function ViewModeToggle() {
  const { mediaViewMode, setMediaViewMode } = usePreferencesStore();

  return (
    <div>
      <button onClick={() => setMediaViewMode("grid")}>Grid</button>
      <button onClick={() => setMediaViewMode("list")}>List</button>
    </div>
  );
}
```

### Player Store

Location: `web/lib/stores/player-store.ts`

Manages player state with volume persistence.

**State & Actions:**
```typescript
interface PlayerState {
  // Current playback
  currentChannelId: string | null;
  isPlaying: boolean;
  volume: number;
  isMuted: boolean;

  // Actions
  setCurrentChannel: (channelId: string | null) => void;
  setPlaying: (playing: boolean) => void;
  setVolume: (volume: number) => void;
  setMuted: (muted: boolean) => void;
  toggleMute: () => void;

  // Player controls
  play: (channelId: string) => void;
  pause: () => void;
  stop: () => void;
}
```

**Initial State:**
- currentChannelId: `null`
- isPlaying: `false`
- volume: `80` (persisted to localStorage)
- isMuted: `false`

**Persistence:**
- Only `volume` is persisted (playback state is not)
- Storage: localStorage as `player-storage`
- Explicit storage configuration: `createJSONStorage(() => localStorage)`
- Prevents unwanted auto-resume while preserving user's volume preference

**Usage Example:**
```typescript
import { usePlayerStore } from "@/lib/stores";

function PlayerControls() {
  const { currentChannelId, isPlaying, volume, play, pause, setVolume } = usePlayerStore();

  return (
    <div>
      <button onClick={() => play("channel-123")}>Play</button>
      <button onClick={pause}>Pause</button>
      <input type="range" value={volume} onChange={(e) => setVolume(Number(e.target.value))} />
    </div>
  );
}
```

### Store Architecture Notes

**Zustand v5 Features:**
- Lightweight (~1KB)
- No providers needed
- Built-in TypeScript support
- React 19 compatible

**Middleware:**
- `devtools`: Redux DevTools integration (set name for each store)
- `persist`: localStorage persistence with `createJSONStorage(() => localStorage)` and selective partialize
- **Important**: Must use explicit `storage: createJSONStorage(() => localStorage)` for Next.js 15 App Router compatibility

**Best Practices:**
1. Use stores directly in components (no context/providers)
2. Selectively subscribe to minimize re-renders: `const name = useUIStore(state => state.sidebarOpen)`
3. Check Redux DevTools for debugging state changes
4. Don't persist sensitive or transient data
5. Use TanStack Query for server state, Zustand for client state
6. **Use `useHydration()` hook when displaying persisted values to prevent hydration flash**

**Test Page:**
Visit `/stores-test` to interact with all stores and verify persistence.

### Hydration Hook

Location: `web/hooks/use-hydration.ts`

**useHydration() - Wait for Store Hydration:**
```typescript
function useHydration(): boolean
```

Returns `true` after client-side hydration is complete. Use this to prevent hydration mismatches when using persisted Zustand stores.

**Usage Example:**
```typescript
import { useHydration } from "@/hooks/use-hydration";
import { usePlayerStore } from "@/lib/stores";

function PlayerComponent() {
  const hydrated = useHydration();
  const volume = usePlayerStore((state) => state.volume);

  if (!hydrated) {
    return <div>Loading...</div>;
  }

  return <div>Volume: {volume}%</div>;
}
```

**Why This Is Needed:**
- Prevents flash of default values before persisted state loads
- Ensures server and client render the same initial state
- Required for any component displaying persisted store values

### Mobile Detection

Location: `web/hooks/use-mobile.ts`

**useMobile() - Detect Mobile Viewport:**
```typescript
function useMobile(): boolean
```

Returns `true` if viewport width is below 768px (mobile/tablet).

**Usage Example:**
```typescript
import { useMobile } from "@/hooks/use-mobile";

function MyComponent() {
  const isMobile = useMobile();
  
  return isMobile ? <MobileView /> : <DesktopView />;
}
```

## Routing Structure

Next.js 15 App Router with file-system based routing.

**Route Map:**
- `/` - Home/Dashboard
- `/channels` - Channel list
- `/channels/[id]` - Channel player (dynamic)
- `/channels/[id]/edit` - Edit channel (nested dynamic)
- `/channels/new` - Create new channel
- `/library` - Media library
- `/settings` - Settings

**Special Files:**
- `app/not-found.tsx` - 404 error page
- `app/loading.tsx` - Root loading state
- `app/channels/loading.tsx` - Channels loading state

**Dynamic Routes:**
```typescript
// Access route params in page component
export default function ChannelPlayerPage({
  params,
}: {
  params: { id: string };
}) {
  // Use params.id
}
```

## Configuration

### Path Aliases

Configured in `tsconfig.json`:
- `@/components` → `web/components`
- `@/lib` → `web/lib`
- `@/hooks` → `web/hooks`
- `@/app` → `web/app`

### Component Configuration

Location: `web/components.json`

Defines shadcn/ui configuration including style, colors, and paths. Used by shadcn CLI for component installation.

### Environment Variables

**Required Variables:**
- `NEXT_PUBLIC_API_URL` - Backend API endpoint (default: http://localhost:8080)
- `NEXT_PUBLIC_APP_URL` - Frontend URL for metadata and SEO (default: http://localhost:3000)

**Configuration Files:**
- `.env.example` - Template with all required variables
- `.env.local` - Local development environment (gitignored)
- `.env.production` - Production deployment reference

**Usage:**
```typescript
const apiUrl = process.env.NEXT_PUBLIC_API_URL;
const appUrl = process.env.NEXT_PUBLIC_APP_URL;
```

## SEO and PWA Configuration

### Root Layout Metadata

Location: `web/app/layout.tsx`

Comprehensive metadata configured with:
- Title template: `"%s | Hermes"`
- Full description and keywords
- Open Graph tags (type, locale, url, title, description, siteName)
- Twitter Card metadata (summary_large_image)
- Robots configuration (index: true, follow: true)

### Page-Specific Metadata

All main pages export metadata using `createMetadata()` helper:
- `app/channels/page.tsx` - Channels page metadata
- `app/library/page.tsx` - Media library page metadata
- `app/settings/page.tsx` - Settings page metadata

### Dynamic Icons

**App Icon** (`app/icon.tsx`):
- Size: 32x32
- Edge runtime
- Dynamic generation using ImageResponse
- Blue background (#2563eb) with "H" letter

**Apple Touch Icon** (`app/apple-icon.tsx`):
- Size: 180x180
- Edge runtime
- Rounded corners for iOS home screen

### PWA Manifest

Location: `app/manifest.ts`

Generates `/manifest.webmanifest` with:
- App name and short name
- Description
- Start URL and display mode (standalone)
- Theme colors
- Icon references (192x192, 512x512)

### Robots.txt

Location: `app/robots.ts`

Generates `/robots.txt` with:
- Allow all user agents to crawl "/"
- Disallow: /api/, /api-test/, /stores-test/, /components/
- Sitemap reference

### Sitemap

Location: `app/sitemap.ts`

Generates `/sitemap.xml` with static routes:
- Home (priority: 1.0, changeFrequency: daily)
- Channels (priority: 0.8, changeFrequency: daily)
- Library (priority: 0.8, changeFrequency: weekly)
- Settings (priority: 0.5, changeFrequency: monthly)

## Channel Components

Location: `web/components/channel/`

### ChannelCard
Displays channel information in a card format with current status. Props: `channel`, `onEdit`, `onDelete`, `onView`.

### ChannelForm
Form for creating/editing channels with playlist management. Props: `mode`, `channel?`, `playlist?`, `onSubmit`, `onCancel`, `isSubmitting?`. Includes drag-drop reorder support (@dnd-kit).

### PlaylistEditor
Manages playlist items with drag-drop reordering and media browser. Props: `items`, `onReorder`, `onAdd`, `onRemove`.

### MediaBrowser
Modal for browsing and selecting media to add to playlist. Props: `open`, `onOpenChange`, `onSelect`.

### ChannelPreview
Real-time preview showing "what's playing now" based on start time and playlist.

## Media Components

Location: `web/components/media/`

### MediaTree

Hierarchical tree view for media organized by Show > Season > Episode with virtual scrolling and keyboard navigation.

**Props:**
```typescript
interface MediaTreeProps {
  media: Media[];
  searchQuery?: string;
  isLoading?: boolean;
  className?: string;
  height?: number;
  onSelectionChange?: (selectedMedia: Media[]) => void;
  disabledMediaIds?: string[];
  initialSelectedMediaIds?: string[];
  enableReordering?: boolean;
  showFilterToggle?: boolean;
}
```

**Features:**
- Automatic grouping by show/season/episode
- Checkbox selection with cascading (select parent = select all children)
- "Show Only Added" toggle to filter to selected items
- Playlist position indicators (#1, #2, #3) on selected items
- Total duration and item count display in toolbar
- Expand/collapse with state management
- Virtual scrolling for 1000+ items (@tanstack/react-virtual)
- Keyboard navigation (arrows, space, enter) with aria-activedescendant
- Search highlighting with auto-expand
- Disabled items support (shows "ALREADY ADDED" badge)
- Loading and empty states

**Usage:**
```typescript
import { MediaTree } from "@/components/media";

<MediaTree
  media={mediaItems}
  searchQuery={searchTerm}
  height={600}
  onSelectionChange={(selected) => setSelected(selected)}
  disabledMediaIds={existingPlaylistIds}
  initialSelectedMediaIds={playlistMediaIds}
  showFilterToggle={true}
  enableReordering={false}
/>
```

### LibraryScanner

Component for triggering media library scans and monitoring real-time progress.

**Props:**
```typescript
interface LibraryScannerProps {
  onScanComplete?: () => void;
  defaultPath?: string; // Default: "/media"
}
```

**Features:**
- Scan button with optional path input
- Progress modal with real-time updates (progress bar, stats, current file)
- Results modal with success summary and error list
- Elapsed time tracking
- Auto-refresh parent data on completion
- Error handling (409 Conflict, 400 Bad Request, network errors)
- Toast notifications
- Responsive design with proper overflow handling

**Usage:**
```typescript
import { LibraryScanner } from "@/components/media";

<LibraryScanner 
  onScanComplete={() => refetch()} 
  defaultPath="/media/videos" 
/>
```

### useMediaTree Hook

Location: `web/hooks/use-media-tree.ts`

**Signature:**
```typescript
function useMediaTree(options: {
  media: Media[];
  searchQuery?: string;
  showOnlySelected?: boolean;
  initialSelectedMediaIds?: string[];
  disabledMediaIds?: string[];
}): UseMediaTreeResult
```

**Returns:**
- `tree`: Hierarchical MediaTreeNode[]
- `flattenedNodes`: Flattened visible nodes for virtual scrolling (filtered if showOnlySelected)
- `toggleNode`, `selectNode`: Node manipulation
- `getSelectedMedia`, `getSelectedIds`: Selection queries
- `expandAll`, `collapseAll`, `clearSelection`, `selectAll`: Bulk operations

**Tree Structure Types:**

Location: `web/lib/types/media-tree.ts`

```typescript
interface MediaTreeNode {
  id: string;
  type: 'show' | 'season' | 'episode';
  label: string;
  children?: MediaTreeNode[];
  media?: Media;
  episodeCount?: number;
  expanded: boolean;
  selected: boolean;
  indeterminate: boolean;
  disabled: boolean;
  depth: number;
  parentId?: string;
  playlistPosition?: number;
}
```

## Best Practices

1. Always use `cn()` utility for combining class names
2. Import components from `@/components/ui/*` not node_modules
3. Use `toast()` from `sonner` for notifications (Toaster must be in layout)
4. All components support dark mode automatically via CSS variables
5. Prefer shadcn components over building custom equivalents
6. Use TanStack Query hooks (`useChannels`, `useMedia`) instead of calling `apiClient` directly
7. Follow query key patterns for proper cache invalidation
8. Mutations automatically handle toast notifications and cache updates
9. Set `NEXT_PUBLIC_API_URL` in `.env.local` for local development
10. Use TanStack Query for server state, Zustand for client state (don't mix)
11. Import stores from `@/lib/stores` and use hooks directly in components
12. Use selective subscriptions in Zustand to minimize re-renders
13. Only persist user preferences, not sensitive or transient data
14. Check Redux DevTools when debugging Zustand state
15. Use `useHydration()` hook to prevent hydration flashes with persisted stores

