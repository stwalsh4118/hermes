# PBI-7: Frontend Foundation

[View in Backlog](../backlog.md#user-content-7)

## Overview

Set up the Next.js 14+ frontend application with TypeScript, establish the routing structure, integrate Tailwind CSS and shadcn/ui component library, create the API client layer, and build foundational layouts and components.

## Problem Statement

Before building specific features, we need a solid frontend foundation that includes:
- Modern React framework (Next.js 14 with App Router)
- Type-safe development (TypeScript)
- Beautiful, accessible UI components (shadcn/ui)
- Consistent styling system (Tailwind CSS)
- API communication layer
- Routing structure matching the app's information architecture
- Common layouts and navigation
- Error handling patterns
- Loading state patterns

## User Stories

**As a developer, I want to:**
- Use a modern, well-documented React framework
- Have type safety across the frontend codebase
- Use pre-built, accessible UI components
- Have a consistent API for backend communication
- Follow clear patterns for loading and error states

**As a user, I want to:**
- Navigate easily between different sections of the app
- See a consistent, professional interface
- Have clear feedback when operations are in progress
- See helpful error messages when something goes wrong

## Technical Approach

### Project Setup

1. **Initialize Next.js Project**
   ```bash
   npx create-next-app@latest frontend --typescript --tailwind --app
   ```

2. **Install Dependencies**
   ```bash
   npm install @radix-ui/react-* class-variance-authority clsx tailwind-merge
   npm install lucide-react # icon library
   npm install axios # or use native fetch
   npm install zustand # state management (if needed)
   ```

3. **Configure shadcn/ui**
   ```bash
   npx shadcn-ui@latest init
   ```

### Directory Structure

```
frontend/
├── app/
│   ├── layout.tsx              # Root layout
│   ├── page.tsx                # Home/dashboard
│   ├── channels/
│   │   ├── page.tsx            # Channel list
│   │   ├── [id]/
│   │   │   ├── page.tsx        # Channel player
│   │   │   └── edit/
│   │   │       └── page.tsx    # Edit channel
│   │   └── new/
│   │       └── page.tsx        # Create channel
│   ├── library/
│   │   └── page.tsx            # Media library
│   └── settings/
│       └── page.tsx            # Settings
├── components/
│   ├── ui/                     # shadcn components
│   ├── layout/
│   │   ├── header.tsx
│   │   ├── nav.tsx
│   │   └── footer.tsx
│   ├── common/
│   │   ├── loading.tsx
│   │   └── error-message.tsx
│   └── [feature-specific]/
├── lib/
│   ├── api.ts                  # API client
│   ├── types.ts                # TypeScript types
│   ├── utils.ts                # Utility functions
│   └── constants.ts            # App constants
├── hooks/
│   ├── use-channels.ts
│   ├── use-media.ts
│   └── use-settings.ts
└── public/
    └── assets/
```

### Type Definitions

Create TypeScript interfaces matching backend models:
```typescript
// lib/types.ts
export interface Channel {
  id: string;
  name: string;
  icon: string;
  start_time: string;
  loop: boolean;
  created_at: string;
  updated_at: string;
}

export interface Media {
  id: string;
  file_path: string;
  title: string;
  show_name: string;
  season?: number;
  episode?: number;
  duration: number;
  video_codec: string;
  audio_codec: string;
  resolution: string;
  file_size: number;
  created_at: string;
}

// ... other types
```

### API Client

```typescript
// lib/api.ts
const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export const api = {
  channels: {
    list: () => fetch(`${API_BASE}/api/channels`).then(r => r.json()),
    get: (id: string) => fetch(`${API_BASE}/api/channels/${id}`).then(r => r.json()),
    create: (data: CreateChannelRequest) => 
      fetch(`${API_BASE}/api/channels`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      }).then(r => r.json()),
    // ... other methods
  },
  // ... other resources
};
```

### Responsive Layout

- Mobile-first approach
- Breakpoints: sm (640px), md (768px), lg (1024px), xl (1280px)
- Navigation: hamburger menu on mobile, sidebar on desktop
- Use Tailwind's responsive utilities

### shadcn/ui Components to Install Initially

- Button
- Card
- Input
- Label
- Select
- Dialog
- Dropdown Menu
- Toast (for notifications)
- Skeleton (for loading states)

## UX/UI Considerations

- Clean, modern aesthetic
- High contrast for readability
- Consistent spacing using Tailwind's scale
- Accessible components (shadcn/ui provides this)
- Loading skeletons for better perceived performance
- Toast notifications for user feedback
- Responsive design tested on mobile, tablet, desktop

## Acceptance Criteria

- [ ] Next.js 14+ project initialized with App Router
- [ ] TypeScript configured and strict mode enabled
- [ ] Tailwind CSS integrated and custom configuration set up
- [ ] shadcn/ui configured and base components installed
- [ ] Routing structure implemented for all main sections
- [ ] API client module created with typed methods
- [ ] Type definitions created matching all backend models
- [ ] Root layout with navigation implemented
- [ ] Header component with logo and nav links
- [ ] Navigation works correctly between all routes
- [ ] Responsive design foundation in place (mobile/tablet/desktop)
- [ ] Loading component created and demonstrable
- [ ] Error message component created and demonstrable
- [ ] Environment variable configuration for API URL
- [ ] Project builds without errors (`npm run build`)
- [ ] Project runs in development mode (`npm run dev`)
- [ ] Basic home/dashboard page renders
- [ ] 404 page implemented
- [ ] Favicon and metadata configured

## Dependencies

**PBI Dependencies:**
- PBI-1: Project Setup & Database Foundation (soft dependency - helps to have API running for testing)

**External Dependencies:**
- Node.js 18+
- npm or yarn

**Packages:**
- next (14+)
- react
- react-dom
- typescript
- tailwindcss
- @radix-ui/react-* (via shadcn/ui)
- lucide-react
- axios or native fetch

## Open Questions

- Should we use Zustand, Context, or other state management?
- Do we want dark mode support from the start?
- Should we use axios or native fetch for API calls?
- Do we need authentication UI components (login, etc.)?
- Should we set up Storybook for component development?
- Do we want to use React Query / TanStack Query for data fetching?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.

