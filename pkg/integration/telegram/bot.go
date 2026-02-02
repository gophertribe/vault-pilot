package telegram

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// Bot wraps the Telegram bot API and dependencies
type Bot struct {
	API        *tgbotapi.BotAPI
	VaultPath  string
	TmplEngine *vault.TemplateEngine
	Git        *sync.GitManager
	stopCh     chan struct{}
}

// NewBot creates a new Telegram bot
func NewBot(token string, vaultPath string, tmplEngine *vault.TemplateEngine, git *sync.GitManager) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("error creating Telegram bot: %w", err)
	}

	return &Bot{
		API:        api,
		VaultPath:  vaultPath,
		TmplEngine: tmplEngine,
		Git:        git,
		stopCh:     make(chan struct{}),
	}, nil
}

// Start begins polling for updates in a goroutine
func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.API.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-b.stopCh:
				return
			case update, ok := <-updates:
				if !ok {
					return
				}
				if update.Message != nil {
					b.handleMessage(update.Message)
				}
			}
		}
	}()

	return nil
}

// Stop stops polling for updates
func (b *Bot) Stop() {
	close(b.stopCh)
	b.API.StopReceivingUpdates()
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	text := msg.Text
	if strings.HasPrefix(text, "/inbox ") {
		content := strings.TrimPrefix(text, "/inbox ")
		b.handleInbox(msg, content)
	} else if text == "/status" {
		b.handleStatus(msg)
	}
}

func (b *Bot) handleInbox(msg *tgbotapi.Message, content string) {
	title := content
	if len(content) > 20 {
		title = content[:20] + "..."
	}

	err := vault.CreateInboxItem(b.VaultPath, b.TmplEngine, title, content)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error creating item: %v", err))
		if _, err := b.API.Send(reply); err != nil {
			log.Printf("Failed to send Telegram error reply: %v", err)
		}
		return
	}

	if b.Git != nil {
		go func() {
			b.Git.Sync("Add Telegram item: " + title)
		}()
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "Added to Inbox")
	if _, err := b.API.Send(reply); err != nil {
		log.Printf("Failed to send Telegram reply: %v", err)
	}
}

func (b *Bot) handleStatus(msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Vault Pilot is Online. Ready to capture.")
	if _, err := b.API.Send(reply); err != nil {
		log.Printf("Failed to send Telegram status reply: %v", err)
	}
}

// ParseCommand extracts the command and content from a message text.
// Returns the command (e.g. "/inbox", "/status") and the remaining content.
func ParseCommand(text string) (command, content string) {
	if strings.HasPrefix(text, "/inbox ") {
		return "/inbox", strings.TrimPrefix(text, "/inbox ")
	}
	if text == "/status" {
		return "/status", ""
	}
	return "", text
}

// TruncateTitle returns a title derived from content, truncated to 20 chars with "..." if needed.
func TruncateTitle(content string) string {
	if len(content) > 20 {
		return content[:20] + "..."
	}
	return content
}
