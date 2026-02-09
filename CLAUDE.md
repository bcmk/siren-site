# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the website for the [SIREN Telegram bot](https://siren.chat),
which provides notifications for webcast alerts across multiple streaming platforms.

- **Backend**: Go 1.23
- **Frontend**: Node.js/Webpack with SCSS and Bootstrap 5
- **Database**: PostgreSQL with pgxpool

## Build Commands

### Frontend (from cmd/site/)
```bash
yarn install        # Install dependencies
yarn build          # Production webpack build
yarn watch          # Development mode with file watching
```

### Backend
```bash
go build -o cmd/site/site ./cmd/site              # Build Go binary into cmd/site/
scripts/build-site <VERSION>                  # Build with version injection
scripts/build-and-push <VERSION>              # Multi-platform Docker build and push
```

### Linting
```bash
golangci-lint run   # Go linting (uses revive)
```

## Architecture

### Backend Structure

The main server is in `cmd/site/main.go`:
- **HTTP Framework**: Gorilla Mux for routing, Gorilla Handlers for middleware
- **Configuration**: Viper with environment variables (prefix: `XRN_`)
- **Templating**: Go `html/template` with bilingual support (English/Russian)

Key files:
- `cmd/site/main.go` - Server setup, routing, HTTP handlers
- `cmd/site/migrations.go` - Database schema migrations
- `cmd/site/request.go` - Query parameter parsing utilities
- `sitelib/config.go` - Viper configuration parsing
- `sitelib/packs.go` - Icon pack loading from S3

### Template System

Templates are in `cmd/site/pages/` with language-specific subdirectories:
- `en/` - English templates
- `ru/` - Russian templates
- `common/` - Shared templates

Routes are language-aware via subdomain detection (en.domain.com vs ru.domain.com).

### Icon Pack System

Packs are loaded dynamically from AWS S3 as JSON config files (`config_v2.json`).
Each pack contains metadata (name, scale, icons mapping) and can be enabled/disabled without redeployment.

### FontAwesome

This project uses **FontAwesome 5.15.4**. Use `fas`/`fab`/`far` class prefixes, NOT the FontAwesome 6 `fa-solid`/`fa-brands`/`fa-regular` syntax.

### Frontend Structure

- `cmd/site/frontend/index.js` - Webpack entry point
- `cmd/site/frontend/styles/` - SCSS files (main.scss, chic.scss, switch.scss)
- `cmd/site/webpack.config.js` - Webpack configuration with PurgeCSSPlugin

### Main Routes

- `/` - Landing page
- `/streamer` - Streamer information
- `/chic` - Main icon feature
- `/chic/p/{pack}` - Individual pack page
- `/chic/code/{pack}` - Code generation for Chaturbate integration
- `/chic/like/{pack}` - POST endpoint for user preferences

## Database

PostgreSQL with a likes/preferences table tracking user interactions by IP address.
Migrations are versioned in `migrations.go`.

## Git Conventions

Use one-line commit messages. The first word must be a verb (e.g. "add", "fix", "update", "remove").
