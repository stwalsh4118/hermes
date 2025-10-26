# PBI-1: Project Setup & Database Foundation

[View in Backlog](../backlog.md#user-content-1)

## Overview

Establish the foundational architecture for the Virtual TV Channel Service by setting up the Go backend project structure, implementing the SQLite database schema, and creating core data models that will support all subsequent features.

## Problem Statement

Before any features can be implemented, we need a solid foundation that includes:
- Proper project organization following Go best practices
- A database schema that can handle channels, playlists, media, and settings
- Data models that match the PRD specifications
- Configuration management for flexible deployment
- Logging infrastructure for debugging and monitoring

## User Stories

**As a developer, I want to:**
- Have a well-organized Go project structure so that code is maintainable
- Use a database migration framework so that schema changes are tracked
- Have configuration management so that settings can be changed without code modifications
- Have logging in place so that I can debug issues effectively

## Technical Approach

### Technology Stack
- **Language:** Go 1.25+
- **Framework:** Gin (github.com/gin-gonic/gin) for HTTP routing
- **Database:** SQLite (github.com/mattn/go-sqlite3)
- **Configuration:** Viper (github.com/spf13/viper)
- **Logging:** zerolog (github.com/rs/zerolog)
- **Migrations:** golang-migrate (github.com/golang-migrate/migrate/v4)
- **UUID Generation:** google/uuid (github.com/google/uuid)

### Project Structure
```
hermes/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── api/                  # HTTP handlers
│   ├── models/               # Data models
│   ├── db/                   # Database operations
│   ├── config/               # Configuration management
│   └── middleware/           # HTTP middleware
├── migrations/               # Database migrations
├── pkg/                      # Public packages (if any)
├── go.mod
├── go.sum
└── README.md
```

### Database Schema

**Tables to create:**

1. **channels**
   - id (TEXT PRIMARY KEY)
   - name (TEXT NOT NULL)
   - icon (TEXT)
   - start_time (DATETIME NOT NULL)
   - loop (BOOLEAN NOT NULL DEFAULT 0)
   - created_at (DATETIME DEFAULT CURRENT_TIMESTAMP)
   - updated_at (DATETIME DEFAULT CURRENT_TIMESTAMP)

2. **media**
   - id (TEXT PRIMARY KEY)
   - file_path (TEXT NOT NULL UNIQUE)
   - title (TEXT NOT NULL)
   - show_name (TEXT)
   - season (INTEGER)
   - episode (INTEGER)
   - duration (INTEGER NOT NULL) -- seconds
   - video_codec (TEXT)
   - audio_codec (TEXT)
   - resolution (TEXT)
   - file_size (INTEGER)
   - created_at (DATETIME DEFAULT CURRENT_TIMESTAMP)

3. **playlist_items**
   - id (TEXT PRIMARY KEY)
   - channel_id (TEXT NOT NULL)
   - media_id (TEXT NOT NULL)
   - position (INTEGER NOT NULL)
   - created_at (DATETIME DEFAULT CURRENT_TIMESTAMP)
   - FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
   - FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE
   - UNIQUE (channel_id, position)

4. **settings**
   - id (INTEGER PRIMARY KEY DEFAULT 1)
   - media_library_path (TEXT NOT NULL)
   - transcode_quality (TEXT DEFAULT 'medium')
   - hardware_accel (TEXT DEFAULT 'none')
   - server_port (INTEGER DEFAULT 8080)
   - updated_at (DATETIME DEFAULT CURRENT_TIMESTAMP)

### Data Models

Define Go structs matching the PRD specifications:
- `Channel`
- `Media`
- `PlaylistItem`
- `Settings`
- `StreamSession` (in-memory, not persisted)

## UX/UI Considerations

This PBI is backend-only and has no direct UI impact. However, the API structure and data models will influence all future UI implementations.

## Acceptance Criteria

- [ ] Go project structure created with proper organization (cmd/, internal/, pkg/)
- [ ] SQLite database schema implemented with all required tables
- [ ] Data models defined matching PRD specifications with proper JSON tags
- [ ] Database migrations framework integrated and tested
- [ ] Basic database CRUD operations work for all models
- [ ] Configuration management set up using Viper (reads from config file/env vars)
- [ ] Logging framework configured with appropriate log levels
- [ ] Gin HTTP server starts successfully and responds to health check
- [ ] Project builds without errors (`go build`)
- [ ] Basic test suite passes (`go test ./...`)
- [ ] README.md updated with setup instructions

## Dependencies

**External:**
- Go 1.25+ installed
- SQLite3 installed on system
- No other PBIs block this work

**Packages:**
- github.com/gin-gonic/gin
- github.com/mattn/go-sqlite3
- github.com/spf13/viper
- github.com/rs/zerolog
- github.com/golang-migrate/migrate/v4
- github.com/google/uuid

## Technical Decisions

**✅ Resolved:**
- **Logging Library:** zerolog - Fast, zero-allocation JSON logging
- **Migration Tool:** golang-migrate - Industry standard with CLI support
- **Entity IDs:** UUIDs using google/uuid package
- **Go Version:** 1.25+ for latest features and performance

**Open Questions:**
- Do we need database connection pooling configuration? (To be determined during implementation)

## Related Tasks

Tasks for this PBI will be defined in [tasks.md](./tasks.md) once PBI moves to "Agreed" status.

