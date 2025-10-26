# PBI-11: Settings, Polish & Deployment

[View in Backlog](../backlog.md#user-content-11)

## Overview

Complete the application by building the settings interface, adding polish throughout the UI (loading states, error handling, confirmations), implementing comprehensive testing, creating Docker deployment configuration, and writing thorough documentation.

## Problem Statement

To make the application production-ready, we need:

**Settings Management:**
- User interface for configuring system settings
- Media library path configuration
- Streaming quality presets
- Hardware acceleration selection
- Server configuration options

**UI Polish:**
- Consistent loading states throughout
- Clear error messages with actionable suggestions
- Confirmation dialogs for destructive actions
- Smooth transitions and animations
- Accessibility improvements

**Testing:**
- End-to-end tests for critical workflows
- Integration tests for API endpoints
- Unit tests for complex logic

**Deployment:**
- Docker configuration for easy deployment
- Docker Compose for full stack
- Environment variable configuration
- Health checks and monitoring

**Documentation:**
- Comprehensive README
- Setup/installation guide
- User guide
- API documentation
- Troubleshooting guide

## User Stories

**As a user, I want to:**
- Configure the path to my media library through the UI
- Choose streaming quality presets (High/Medium/Low)
- Select hardware acceleration if my system supports it
- Configure advanced FFmpeg parameters if I'm a power user
- Change server port and host if needed
- See what hardware encoders are available on my system
- Have all my settings saved and applied immediately
- See helpful error messages when something goes wrong
- Be asked to confirm before deleting important things

**As a system administrator, I want to:**
- Deploy the application easily using Docker
- Configure everything via environment variables
- Monitor system health and active streams
- Access logs for troubleshooting
- Follow clear documentation for setup and maintenance

## Technical Approach

### Settings UI Components

1. **Settings Page** (`app/settings/page.tsx`)
   - Tabbed or sectioned layout:
     - Media Library
     - Streaming Quality
     - Hardware Acceleration
     - Server Configuration
     - Advanced Options
   - Save button (or auto-save)
   - Reset to defaults button

2. **Settings Form Sections**

```typescript
// Media Library Section
<FormSection title="Media Library">
  <Input 
    label="Media Directory Path"
    value={settings.mediaLibraryPath}
    onChange={...}
  />
  <Button onClick={testPath}>Test Path</Button>
  <Button onClick={scanNow}>Scan Now</Button>
</FormSection>

// Streaming Section
<FormSection title="Streaming Quality">
  <Select 
    label="Quality Preset"
    options={['high', 'medium', 'low']}
    value={settings.transcodeQuality}
  />
  <HelpText>
    High: 1080p @ 5Mbps, Medium: 720p @ 3Mbps, Low: 480p @ 1.5Mbps
  </HelpText>
</FormSection>

// Hardware Acceleration Section
<FormSection title="Hardware Acceleration">
  <Select 
    label="Encoder"
    options={['auto', 'nvenc', 'qsv', 'vaapi', 'none']}
    value={settings.hardwareAccel}
  />
  <Button onClick={detectHardware}>Detect Available Encoders</Button>
  <InfoBox>Available: {detectedEncoders.join(', ')}</InfoBox>
</FormSection>

// Advanced Section
<FormSection title="Advanced">
  <Textarea 
    label="Custom FFmpeg Parameters"
    value={settings.customFFmpegParams}
    placeholder="-preset faster -crf 23"
  />
  <Warning>For advanced users only!</Warning>
</FormSection>
```

3. **Hardware Detection**

API endpoint to detect available encoders:
```go
// GET /api/settings/hardware
{
  "available_encoders": ["nvenc", "qsv"],
  "recommended": "nvenc",
  "cpu_cores": 8,
  "gpu_info": "NVIDIA GeForce RTX 3080"
}
```

### UI Polish Tasks

1. **Loading States**
   - Skeleton loaders for content loading
   - Spinner overlays for actions
   - Progress bars for long operations
   - Disabled states for buttons during operations

2. **Error Handling**
   - Toast notifications for errors
   - Inline validation errors
   - Error boundaries for React components
   - Retry buttons where appropriate
   - Helpful error messages:
     - ❌ "Error" → ✅ "Could not save channel: Channel name already exists"

3. **Confirmation Dialogs**
   - Delete channel: "Are you sure? This will delete the channel and its playlist."
   - Delete media: "Remove this media from library? This will also remove it from any playlists."
   - Reset settings: "Reset all settings to defaults?"
   - Use shadcn/ui AlertDialog

4. **Animations & Transitions**
   - Fade transitions between pages
   - Smooth modal open/close
   - Loading spinners
   - Success checkmarks
   - Keep subtle and fast (200-300ms)

5. **Accessibility**
   - ARIA labels on all interactive elements
   - Keyboard navigation works everywhere
   - Focus visible indicators
   - Screen reader announcements for dynamic content
   - Color contrast meets WCAG AA standards

### Testing Implementation

1. **Backend Tests**
   ```go
   // Unit tests for timeline calculation
   func TestTimelineCalculation(t *testing.T) { ... }
   
   // Integration tests for APIs
   func TestChannelCRUD(t *testing.T) { ... }
   
   // FFmpeg integration tests
   func TestStreamGeneration(t *testing.T) { ... }
   ```

2. **Frontend Tests**
   ```typescript
   // Component tests (Jest + React Testing Library)
   describe('ChannelCard', () => {
     it('displays channel name', () => { ... });
   });
   
   // E2E tests (Playwright or Cypress)
   test('create channel workflow', async () => {
     // Navigate to create channel
     // Fill out form
     // Submit
     // Verify channel appears in list
   });
   ```

3. **Load Tests**
   - Test 5+ concurrent streams
   - Measure CPU/memory usage
   - Verify synchronization across clients

### Docker Configuration

1. **Backend Dockerfile**
```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ffmpeg sqlite3
COPY --from=builder /app/server /usr/local/bin/
EXPOSE 8080
CMD ["server"]
```

2. **Frontend Dockerfile**
```dockerfile
FROM node:18 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:18-slim
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./
EXPOSE 3000
CMD ["npm", "start"]
```

3. **Docker Compose**
```yaml
version: '3.8'
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      - /path/to/media:/media:ro
    environment:
      - MEDIA_LIBRARY_PATH=/media
      - DATABASE_PATH=/data/hermes.db
  
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_BASE=http://localhost:8080
    depends_on:
      - backend
```

### Documentation Files

1. **README.md**
   - Project overview
   - Features list
   - Quick start guide
   - Screenshots
   - License

2. **INSTALL.md**
   - System requirements
   - Installation steps (Docker, manual)
   - Configuration options
   - First-time setup

3. **USER_GUIDE.md**
   - How to scan media library
   - How to create channels
   - How to build playlists
   - How to watch channels
   - Troubleshooting common issues

4. **API.md**
   - API endpoint reference
   - Request/response examples
   - Authentication (if implemented)
   - Error codes

## UX/UI Considerations

- Settings should be easy to find and understand
- Technical terms should have help text
- Test/validate buttons provide immediate feedback
- Changes should be validated before saving
- Success/error states clearly communicated

## Acceptance Criteria

**Settings UI:**
- [ ] Settings page implemented with all configuration sections
- [ ] Media library path input with validation
- [ ] Quality preset selector (High/Medium/Low)
- [ ] Hardware acceleration dropdown
- [ ] Custom FFmpeg parameters textarea (advanced)
- [ ] Server port and host configuration
- [ ] "Detect Hardware" button shows available encoders
- [ ] Settings API integration working
- [ ] Settings saved and applied correctly
- [ ] Settings persist across server restarts

**UI Polish:**
- [ ] Loading states implemented throughout application
- [ ] Skeleton loaders for all data-loading pages
- [ ] Error messages displayed clearly with actionable suggestions
- [ ] Confirmation dialogs for all destructive actions (delete channel, delete media)
- [ ] Toast notifications for success/error feedback
- [ ] Smooth page transitions
- [ ] All buttons have hover/active states
- [ ] Form validation with inline error display
- [ ] Accessibility: ARIA labels on interactive elements
- [ ] Accessibility: Keyboard navigation works
- [ ] Accessibility: Focus indicators visible
- [ ] Color contrast meets WCAG AA standards

**Testing:**
- [ ] Backend unit tests for timeline calculation
- [ ] Backend integration tests for all API endpoints
- [ ] Frontend component tests for key components
- [ ] End-to-end test for create channel workflow
- [ ] End-to-end test for create and watch channel workflow
- [ ] Load test with 5+ concurrent streams passes
- [ ] All tests pass in CI environment

**Docker/Deployment:**
- [ ] Backend Dockerfile created and builds successfully
- [ ] Frontend Dockerfile created and builds successfully
- [ ] Docker Compose configuration complete
- [ ] Environment variables documented
- [ ] Health check endpoint implemented (`/api/health`)
- [ ] System stats endpoint implemented (`/api/stats`)
- [ ] Volume mounts configured correctly
- [ ] Application runs successfully in Docker

**Documentation:**
- [ ] README.md complete with overview and quick start
- [ ] INSTALL.md with detailed setup instructions
- [ ] USER_GUIDE.md with step-by-step usage instructions
- [ ] API.md with endpoint documentation
- [ ] TROUBLESHOOTING.md with common issues and solutions
- [ ] All configuration options documented
- [ ] Screenshots included in documentation
- [ ] License file included

## Dependencies

**PBI Dependencies:**
- All previous PBIs (1-10) - This is the final integration PBI

**External Dependencies:**
- Docker and Docker Compose for deployment
- Testing frameworks (Jest, React Testing Library, Playwright/Cypress)

## Open Questions

- Should we include a setup wizard for first-time users?
- Do we need database backup/restore functionality?
- Should settings be exportable/importable?
- Do we want analytics or usage tracking (optional, privacy-respecting)?
- Should we implement automatic updates or update notifications?
- Do we need a system health dashboard?
- Should we include sample media or demo mode?

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.

