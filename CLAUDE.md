# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Vault Pilot is a Go backend for managing a GTD (Getting Things Done) Obsidian vault. It provides an HTTP API for AI-driven inbox processing and weekly reviews, Git-based sync, and integrations with Discord, Gmail, Google Calendar, and Google Drive. See DESIGN.md for the full architecture.

## Build & Test Commands

```bash
# Build (requires CGO for sqlite3)
go build -o vault-pilot ./cmd/server

# Run all tests
go test ./...

# Run a single test
go test ./pkg/vault/ -run TestReadWriteNote

# Run with coverage
go test -cover ./...

# Run the server with Gemini (default)
GEMINI_API_KEY=... ./vault-pilot -vault /path/to/vault -port 8080 -db vault-pilot.db

# Run the server with Moonshot (Kimi 2.5)
MOONSHOT_API_KEY=... ./vault-pilot -vault /path/to/vault -port 8080 -db vault-pilot.db -ai-provider moonshot

# Run with Google Calendar sync
GEMINI_API_KEY=... GOOGLE_SERVICE_ACCOUNT_KEY=/path/to/sa.json GOOGLE_CALENDAR_ID=primary \
  ./vault-pilot -vault /path/to/vault -port 8080 -db vault-pilot.db

# Run with Google Drive backup + watch
GEMINI_API_KEY=... GOOGLE_SERVICE_ACCOUNT_KEY=/path/to/sa.json \
  GOOGLE_DRIVE_BACKUP_FOLDER_ID=folder-id GOOGLE_DRIVE_WATCH_FOLDER_ID=folder-id \
  ./vault-pilot -vault /path/to/vault -port 8080 -db vault-pilot.db
```

## Architecture

The app follows a layered architecture where the HTTP API layer orchestrates between AI, vault file operations, database, and Git sync.

### Request Flow (e.g., POST /inbox)

1. `pkg/api/handler.go` receives the request
2. Calls `pkg/ai` to analyze content via Gemini, which returns structured JSON
3. Calls `pkg/vault` to write a markdown file from a template into the vault directory
4. Fires async Git sync via `pkg/sync` (goroutine)

### Key Layers

- **`pkg/api`** - HTTP handlers and routing using Go 1.22+ `net/http` method-based routing (`"POST /inbox"`, etc.). The `Handler` struct holds all dependencies (repo, AI client, template engine, vault path, git manager).
- **`pkg/ai`** - AI text generation behind the `Generator` interface (`GenerateText(ctx, prompt) (string, error)`). Implementations: `Client` (Google Gemini via `google/generative-ai-go`) and `MoonshotClient` (Moonshot/Kimi 2.5 via OpenAI-compatible HTTP API). Provider is selected with `-ai-provider` flag. Prompt templates live in `prompts.go`.
- **`pkg/vault`** - File-level operations on the Obsidian vault. `ReadNote` parses YAML frontmatter + markdown body into a `Note` struct. `WriteNote` serializes back. `TemplateEngine` loads `.md` templates from the vault's `0. GTD System/Templates/` directory and renders `{{title}}` and `{{date:FORMAT}}` placeholders (Moment.js format converted to Go time format).
- **`pkg/db`** - SQLite via `mattn/go-sqlite3` (CGO required). Schema is initialized inline in `InitSchema()` (no migration tool yet). `Repository` provides data access methods.
- **`pkg/sync`** - Git operations via `go-git`. `Sync()` stages all, commits, and pushes (SSH auth with fallback).
- **`pkg/integration/google`** - Shared Google service account auth (`NewHTTPClient` for OAuth-style APIs, `ClientOption` for service constructors). Used by Calendar, Drive, and Gmail integrations.
- **`pkg/integration/calendar`** - Bidirectional Google Calendar sync. `Service` wraps the Calendar API behind a `CalendarAPI` interface. `Syncer` runs a periodic loop that pulls events into `2. Next Actions/@calendar/` notes and pushes vault notes with `due_date` to Calendar.
- **`pkg/integration/drive`** - Google Drive backup and watch. `Service` wraps the Drive API behind a `DriveAPI` interface. `Backup` incrementally uploads `.md` files to a Drive folder. `Watcher` monitors a Drive folder for new files and creates inbox items.
- **`pkg/integration/discord`** - Discord bot via `bwmarrin/discordgo`. Commands: `!inbox <text>`, `!status`.
- **`pkg/integration/gmail`** - Gmail polling using service account auth. Fetches unread emails, analyzes with AI, and creates inbox items.

### Testing Pattern

Tests use `MockGenerator` (in `pkg/api/api_test.go`) to stub the `ai.Generator` interface. Vault tests create temp directories with fixture templates. No external services are needed to run the test suite.

### Vault Structure (operated on at runtime)

The server expects the vault at `-vault` path to follow this directory layout:
- `0. GTD System/Templates/` - Obsidian templates (Inbox Item Template.md, Weekly Review Template.md, etc.)
- `1. Inbox/` - Incoming items
- `2. Next Actions/` - Tasks by context
- `3. Projects/` - Active projects (filtered by `status: active` in frontmatter)
- `6. Weekly Reviews/` - Generated review files

### Frontmatter

All vault notes use YAML frontmatter. The `Note.Frontmatter` field is `interface{}` (typically `map[string]interface{}`). Typed structs (`InboxItem`, `Project`, `NextAction`, `WeeklyReview`) in `model.go` can be used via `ParseInboxItem()` / `ParseProject()` helpers but the handlers currently work with the raw map.

## Environment Variables

- `GEMINI_API_KEY` (required if `-ai-provider gemini`) - Google Gemini API key
- `MOONSHOT_API_KEY` (required if `-ai-provider moonshot`) - Moonshot API key for Kimi 2.5
- `GOOGLE_SERVICE_ACCOUNT_KEY` (optional) - Path to Google service account JSON key file. Required for Calendar, Drive, and Gmail integrations.
- `GOOGLE_CALENDAR_ID` (optional, requires `GOOGLE_SERVICE_ACCOUNT_KEY`) - Calendar ID for bidirectional sync (e.g., `primary`)
- `GOOGLE_DRIVE_BACKUP_FOLDER_ID` (optional, requires `GOOGLE_SERVICE_ACCOUNT_KEY`) - Drive folder ID for vault backup
- `GOOGLE_DRIVE_WATCH_FOLDER_ID` (optional, requires `GOOGLE_SERVICE_ACCOUNT_KEY`) - Drive folder ID to watch for incoming files
- `DISCORD_TOKEN` (optional) - enables Discord bot
- `TELEGRAM_TOKEN` (optional) - enables Telegram bot
