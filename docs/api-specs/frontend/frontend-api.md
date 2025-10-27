# Frontend Infrastructure API

Last Updated: 2025-10-27 (TanStack Query and API client added)

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

