# Hermes - Virtual TV Channel Service

A self-hosted service that transforms your media library into continuous virtual TV channels with EPG support and HLS streaming.

## Overview

Hermes allows you to create and manage virtual TV channels from your personal media library. Each channel operates like a traditional broadcast TV channel with a continuous stream, electronic program guide (EPG), and support for multiple simultaneous viewers.

### Key Features (Planned)

- **Channel Management**: Create multiple virtual TV channels with customizable playlists
- **Media Library Integration**: Automatic scanning and metadata extraction from your video files
- **Virtual Timeline**: Accurate calculation of what's playing at any given time
- **EPG Generation**: Electronic program guide with current and upcoming shows
- **HLS Streaming**: Adaptive bitrate streaming compatible with any device
- **Hardware Acceleration**: Support for NVENC, QSV, VAAPI, and VideoToolbox
- **Web Interface**: Modern React-based UI for channel and library management

## Prerequisites

- **Go 1.25+**: Required for building the backend service
- **SQLite3**: Embedded database for storing channels and media metadata
- **FFmpeg**: Required for media transcoding (will be added in later phases)

## Quick Start

### Backend (Go API)
```bash
# Navigate to api directory
cd api

# Build the application
go build ./cmd/server

# Run the application
./server

# Development with hot reload (requires Air)
air
```

### Frontend (Next.js)
```bash
# To be added in PBI-7
cd web
# Setup instructions coming soon
```

## Project Structure

This is a monorepo containing both the backend API and frontend web client:

```
hermes/
├── api/                 # Go backend service
│   ├── cmd/server/     # Application entry point
│   ├── internal/       # Private application code
│   │   ├── api/        # HTTP handlers
│   │   ├── models/     # Data models
│   │   ├── db/         # Database operations
│   │   ├── config/     # Configuration management
│   │   └── middleware/ # HTTP middleware
│   ├── migrations/     # Database migration files
│   ├── pkg/            # Public packages
│   ├── .air.toml       # Hot reload configuration
│   ├── go.mod          # Go module definition
│   └── README.md       # API documentation
├── web/                # Next.js frontend (to be added)
│   └── README.md       # Web client docs
├── docs/               # Documentation and specifications
├── .gitignore          # Git exclusions
└── README.md           # This file
```

## Development Status

**Current Phase**: Foundation Setup (PBI-1)

The project is currently in the foundation phase. This initial setup establishes:
- Go project structure and module initialization ✓
- Directory hierarchy following Go best practices ✓

**Next Steps**:
- Add core dependencies (Gin, SQLite, Viper, zerolog)
- Implement configuration management
- Create database schema and migrations
- Build data models and database layer

## Contributing

This project follows a structured task-driven development approach. All work is organized into Product Backlog Items (PBIs) and tasks documented in the `docs/delivery/` directory.

## License

To be determined.
