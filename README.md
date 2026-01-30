# Vault Pilot - GTD Obsidian Manager

A Go-based backend system for managing GTD (Getting Things Done) Obsidian vaults with AI-powered assistance, Git synchronization, and multi-channel integrations.

## Features

- 🤖 **AI-Powered**: Uses Google Gemini to analyze and process inbox items
- 📝 **Template Engine**: Automatic note creation with your GTD templates
- 🔄 **Git Sync**: Automatic commits and pushes after changes
- 💬 **Discord Bot**: Capture ideas directly from Discord
- 📧 **Gmail Integration**: Auto-process emails into tasks (architecture ready)
- 📊 **Weekly Reviews**: AI-generated summaries of your week

## Quick Start

### Prerequisites
- Go 1.22 or higher
- A Git-initialized Obsidian vault
- Google Gemini API key

### Installation

```bash
# Clone the repository
cd /Users/mklimuk/code/vault-pilot

# Build the server
go build -o vault-pilot ./cmd/server

# Set up environment
export GEMINI_API_KEY="your-gemini-api-key"

# Run the server
./vault-pilot -vault /path/to/your/vault -port 8080
```

### API Endpoints

#### Create Inbox Item
```bash
POST /inbox
Content-Type: application/json

{
  "content": "Your idea or task here"
}
```

#### List Active Projects
```bash
GET /projects
```

#### Generate Weekly Review
```bash
POST /review/weekly
```

## Discord Integration (Optional)

```bash
export DISCORD_TOKEN="your-discord-bot-token"
./vault-pilot -vault /path/to/vault
```

Commands:
- `!inbox <text>` - Add item to inbox
- `!status` - Check bot status

## Architecture

See [DESIGN.md](DESIGN.md) for detailed architecture documentation.

## Project Structure

- `cmd/server/` - Main application entry point
- `pkg/vault/` - Core vault operations (read/write/template)
- `pkg/api/` - HTTP handlers and routing
- `pkg/ai/` - Gemini API integration
- `pkg/db/` - SQLite database layer
- `pkg/sync/` - Git synchronization
- `pkg/integration/` - Gmail and Discord integrations

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Configuration

The server accepts the following flags:
- `-vault` - Path to your Obsidian vault (required)
- `-port` - HTTP port (default: 8080)
- `-db` - SQLite database path (default: vault-pilot.db)

Environment variables:
- `GEMINI_API_KEY` - Google Gemini API key (required)
- `DISCORD_TOKEN` - Discord bot token (optional)

## Development

Built with:
- Go 1.22+ (utilizing new routing features)
- `go-git` for Git operations
- `google/generative-ai-go` for Gemini
- `bwmarrin/discordgo` for Discord
- SQLite for state management

## License

[Add your license here]
