# Hermes Web Client

Next.js frontend for the Virtual TV Channel Service.

## Overview

The web client provides a modern UI for:
- Creating and managing TV channels
- Browsing and organizing media library
- Watching channels via HLS streaming
- Configuring system settings
- Viewing the electronic program guide (EPG)

## Technology Stack (Planned)

- **Next.js 14+** with App Router
- **TypeScript**
- **Tailwind CSS** + **shadcn/ui**
- **React Query** for API state management
- **HLS.js** or **Video.js** for video playback

## Development Status

**Current Phase**: Not yet started

This frontend will be implemented in PBI-7 through PBI-11 after the backend foundation is complete.

## Planned Structure

```
web/
├── app/                 # Next.js App Router pages
│   ├── channels/       # Channel management UI
│   ├── library/        # Media library UI
│   ├── settings/       # Settings UI
│   └── watch/          # Video player UI
├── components/          # React components
├── lib/                # Utilities and API client
├── public/             # Static assets
└── package.json        # Node.js dependencies
```

## Future Quick Start

```bash
# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build
```

Stay tuned for implementation in PBI-7!

