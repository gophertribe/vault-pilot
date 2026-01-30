package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// Bot wraps the Discord session and dependencies
type Bot struct {
	Session    *discordgo.Session
	VaultPath  string
	TmplEngine *vault.TemplateEngine
	Git        *sync.GitManager
}

// NewBot creates a new Discord bot
func NewBot(token string, vaultPath string, tmplEngine *vault.TemplateEngine, git *sync.GitManager) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	bot := &Bot{
		Session:    dg,
		VaultPath:  vaultPath,
		TmplEngine: tmplEngine,
		Git:        git,
	}

	dg.AddHandler(bot.messageCreate)

	return bot, nil
}

// Start opens the websocket connection
func (b *Bot) Start() error {
	return b.Session.Open()
}

// Stop closes the websocket connection
func (b *Bot) Stop() error {
	return b.Session.Close()
}

func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from self
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Simple commands
	if strings.HasPrefix(m.Content, "!inbox ") {
		content := strings.TrimPrefix(m.Content, "!inbox ")
		b.handleInbox(s, m, content)
	} else if m.Content == "!status" {
		b.handleStatus(s, m)
	}
}

func (b *Bot) handleInbox(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	// Create Inbox Item
	title := "Discord Entry"
	// We could use AI here to extract a better title, but for MVP just use generic + content
	if len(content) > 20 {
		title = content[:20] + "..."
	} else {
		title = content
	}

	err := vault.CreateInboxItem(b.VaultPath, b.TmplEngine, title, content)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error creating item: %v", err))
		return
	}

	// Sync
	if b.Git != nil {
		go func() {
			b.Git.Sync("Add Discord item: " + title)
		}()
	}

	s.ChannelMessageSend(m.ChannelID, "✅ Added to Inbox")
}

func (b *Bot) handleStatus(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check active projects (simplified logic, ideally reuse from API/Service)
	// For MVP, just say "Online"
	s.ChannelMessageSend(m.ChannelID, "🤖 Vault Pilot is Online. Ready to capture.")
}
