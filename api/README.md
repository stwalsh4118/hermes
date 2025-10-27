# Hermes API

Go backend service for the Virtual TV Channel Service.

## Overview

This is the backend API that handles:
- Channel management and scheduling
- Media library scanning and metadata extraction
- Virtual timeline calculation
- EPG (Electronic Program Guide) generation
- HLS stream orchestration and transcoding

## Technology Stack

- **Go 1.25+**
- **Gin** - HTTP web framework
- **SQLite** - Embedded database
- **Viper** - Configuration management
- **zerolog** - Structured logging
- **golang-migrate** - Database migrations
- **FFmpeg** - Media transcoding (to be added)

## Quick Start

```bash
# Build
go build ./cmd/server

# Run
./server

# Development with hot reload (requires Air)
air
```

## Project Structure

```
api/
├── cmd/server/          # Application entry point
├── internal/            # Private application code
│   ├── api/            # HTTP handlers
│   ├── models/         # Data models
│   ├── db/             # Database operations
│   ├── config/         # Configuration management
│   └── middleware/     # HTTP middleware
├── migrations/          # Database migration files
├── pkg/                # Public packages (if needed)
└── go.mod              # Go module definition
```

## Development Status

**Current Phase**: Foundation Setup (PBI-1)

✅ Project structure initialized
⏳ Dependencies to be added (task 1-2)
⏳ Database schema to be created (task 1-5)
⏳ Core services to be implemented (PBI-1 through PBI-6)

## API Documentation

API specifications are maintained in `../docs/api-specs/`:
- Database schema: `database/database-api.md`
- Channel management: `channels/channels-api.md`

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

