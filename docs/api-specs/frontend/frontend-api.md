# Frontend Infrastructure API

Last Updated: 2025-10-28 (Layout components and routing added)

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

