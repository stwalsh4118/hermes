# Frontend Infrastructure API

Last Updated: 2025-10-27

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

## Hooks

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

