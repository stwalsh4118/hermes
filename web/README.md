# Hermes Web Frontend

Next.js 15 frontend for the Hermes Virtual TV Channel Service.

## Prerequisites

- Node.js 20+
- pnpm

## Development

1. Install dependencies:
   ```bash
   pnpm install
   ```

2. Set up environment variables:
   ```bash
   cp .env.example .env.local
   # Edit .env.local with your configuration
   ```

3. Run development server:
   ```bash
   pnpm dev
   ```

4. Open [http://localhost:3000](http://localhost:3000)

## Environment Variables

- `NEXT_PUBLIC_API_URL` - Backend API URL (default: http://localhost:8080)
- `NEXT_PUBLIC_APP_URL` - Frontend URL (default: http://localhost:3000)

## Scripts

- `pnpm dev` - Start development server
- `pnpm build` - Build for production
- `pnpm start` - Start production server
- `pnpm lint` - Run ESLint
- `pnpm type-check` - Run TypeScript compiler check
- `pnpm format` - Format code with Prettier

## Tech Stack

- Next.js 15.5
- React 19.1.1
- TypeScript
- Tailwind CSS
- shadcn/ui components
- TanStack Query (server state)
- Zustand (client state)
- Lucide icons

## Project Structure

```
web/
├── app/              # App Router pages
├── components/       # React components
│   ├── ui/          # shadcn/ui components
│   ├── layout/      # Layout components
│   └── common/      # Common/shared components
├── lib/             # Utility libraries
│   ├── api/         # API client
│   ├── stores/      # Zustand stores
│   └── types/       # TypeScript types
├── hooks/           # Custom React hooks
└── public/          # Static assets
```

## Deployment

1. Set production environment variables
2. Build the application: `pnpm build`
3. Start the server: `pnpm start`

For detailed deployment instructions, see the main project README.
