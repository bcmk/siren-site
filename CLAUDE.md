# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## General
- Never use `git -C` — the working directory is already the repo root
- Prefer relative paths
- If you changed directory to run something, change back to the repo root after
- Use real em-dash (—) where grammar requires it

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
go build -o cmd/site/site ./cmd/site          # Build Go binary into cmd/site/
cd cmd/site && ./site -c site.ignore.yaml     # Run dev server locally
scripts/build-site                            # Build with version from git describe
scripts/build-and-push                        # Docker build and push (version from git tag)
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
- `cmd/site/main.go` — Server setup, routing, HTTP handlers
- `cmd/site/migrations.go` — Database schema migrations
- `cmd/site/request.go` — Query parameter parsing utilities
- `sitelib/config.go` — Viper configuration parsing
- `sitelib/packs.go` — Icon pack loading from S3

### Template System

Templates are in `cmd/site/pages/` with language-specific subdirectories:
- `en/` — English templates
- `ru/` — Russian templates
- `common/` — Shared templates

Routes are language-aware via subdomain detection (en.domain.com vs ru.domain.com).

### Icon Pack System

Packs are loaded dynamically from AWS S3 as JSON config files (`config_v2.json`).
Each pack contains metadata (name, scale, icons mapping) and can be enabled/disabled without redeployment.

### FontAwesome

This project uses **FontAwesome 6.7.2**. Use `fas`/`fab`/`far` class prefixes,
NOT the FontAwesome 6 `fa-solid`/`fa-brands`/`fa-regular` syntax.

### Frontend Structure

- `cmd/site/frontend/index.js` — Webpack entry point
- `cmd/site/frontend/styles/` — SCSS files (main.scss, chic.scss, switch.scss)
- `cmd/site/webpack.config.js` — Webpack configuration with PurgeCSSPlugin

### Main Routes

- `/` — Landing page
- `/streamer` — Streamer information
- `/chic` — Main icon feature
- `/chic/p/{pack}` — Individual pack page
- `/chic/code/{pack}` — Code generation for Chaturbate integration
- `/chic/like/{pack}` — POST endpoint for user preferences

## Database

PostgreSQL with a likes/preferences table tracking user interactions by IP address.
Migrations are versioned in `migrations.go`.

## Workflow

Run `yarn build` (from `cmd/site/`) after frontend changes without asking.
Change back to the repo root immediately after.
Run `gofmt` on changed Go files before committing.
Run `golangci-lint run ./...` on changed Go files before committing.

### Browser Testing

When testing UI changes in the browser,
resize the window to at least 10 different widths covering all Bootstrap breakpoints
and verify the layout at each size.
Suggested widths: 375, 576, 640, 768, 850, 996, 1024, 1200, 1400, 1600.

### PurgeCSS

PurgeCSS scans template files in `cmd/site/pages/` for CSS class names.
When adding new Bootstrap utility classes to templates,
always run `yarn build` afterward so PurgeCSS picks them up.
Do not add classes to the PurgeCSS safelist — just rebuild.

## Git Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/) (e.g. `feat:`, `fix:`, `refactor:`, `style:`, `docs:`).
Keep commit messages to a single line — no body or multi-line descriptions.
