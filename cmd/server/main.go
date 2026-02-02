package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/ai"
	"github.com/mklimuk/vault-pilot/pkg/api"
	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/integration/calendar"
	"github.com/mklimuk/vault-pilot/pkg/integration/discord"
	"github.com/mklimuk/vault-pilot/pkg/integration/drive"
	"github.com/mklimuk/vault-pilot/pkg/integration/gmail"
	googleauth "github.com/mklimuk/vault-pilot/pkg/integration/google"
	"github.com/mklimuk/vault-pilot/pkg/integration/telegram"
	"github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

func main() {
	vaultPath := flag.String("vault", "", "Path to Obsidian Vault")
	dbPath := flag.String("db", "vault-pilot.db", "Path to SQLite DB")
	port := flag.String("port", "8080", "HTTP Port")
	aiProvider := flag.String("ai-provider", "gemini", "AI provider: gemini or moonshot")
	flag.Parse()

	if *vaultPath == "" {
		log.Fatal("Please provide -vault path")
	}

	// Initialize DB
	database, err := db.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer database.Close()

	if err := database.InitSchema(); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	repo := db.NewRepository(database)

	// Initialize AI Client
	var aiClient ai.Generator
	switch *aiProvider {
	case "moonshot":
		key := os.Getenv("MOONSHOT_API_KEY")
		if key == "" {
			log.Fatal("MOONSHOT_API_KEY environment variable is required when using moonshot provider")
		}
		aiClient = ai.NewMoonshotClient(key)
	case "gemini":
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			log.Fatal("GEMINI_API_KEY environment variable is required when using gemini provider")
		}
		ctx := context.Background()
		geminiClient, err := ai.NewClient(ctx, key)
		if err != nil {
			log.Fatalf("Failed to create AI client: %v", err)
		}
		defer geminiClient.Close()
		aiClient = geminiClient
	default:
		log.Fatalf("Unknown AI provider: %s", *aiProvider)
	}

	// Initialize Template Engine
	templateDir := filepath.Join(*vaultPath, "0. GTD System", "Templates")
	tmplEngine := vault.NewTemplateEngine(templateDir)

	// Initialize Git Manager
	gitManager := sync.NewGitManager(*vaultPath)

	// Initialize Router
	router := api.NewRouter(repo, aiClient, tmplEngine, *vaultPath, gitManager)

	// Google service account key — shared by Calendar, Drive, and Gmail
	googleKeyFile := os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")

	// Initialize Google Calendar Sync (Optional)
	calendarID := os.Getenv("GOOGLE_CALENDAR_ID")
	if googleKeyFile != "" && calendarID != "" {
		ctx := context.Background()
		calSvc, err := calendar.NewService(ctx, googleKeyFile, calendarID)
		if err != nil {
			log.Printf("Failed to create Calendar service: %v", err)
		} else {
			calSyncer := calendar.NewSyncer(calSvc, repo, *vaultPath, tmplEngine, gitManager,
				15*time.Minute, 14*24*time.Hour)
			if err := calSyncer.Start(); err != nil {
				log.Printf("Failed to start Calendar syncer: %v", err)
			} else {
				log.Println("Google Calendar sync started")
				defer calSyncer.Stop()
			}
		}
	}

	// Initialize Google Drive Backup (Optional)
	driveBackupFolderID := os.Getenv("GOOGLE_DRIVE_BACKUP_FOLDER_ID")
	if googleKeyFile != "" && driveBackupFolderID != "" {
		ctx := context.Background()
		drvSvc, err := drive.NewService(ctx, googleKeyFile, driveBackupFolderID)
		if err != nil {
			log.Printf("Failed to create Drive backup service: %v", err)
		} else {
			backup := drive.NewBackup(drvSvc, repo, *vaultPath, 30*time.Minute)
			if err := backup.Start(); err != nil {
				log.Printf("Failed to start Drive backup: %v", err)
			} else {
				log.Println("Google Drive backup started")
				defer backup.Stop()
			}
		}
	}

	// Initialize Google Drive Watcher (Optional)
	driveWatchFolderID := os.Getenv("GOOGLE_DRIVE_WATCH_FOLDER_ID")
	if googleKeyFile != "" && driveWatchFolderID != "" {
		ctx := context.Background()
		drvSvc, err := drive.NewService(ctx, googleKeyFile, driveWatchFolderID)
		if err != nil {
			log.Printf("Failed to create Drive watch service: %v", err)
		} else {
			watcher := drive.NewWatcher(drvSvc, repo, *vaultPath, tmplEngine, gitManager, 5*time.Minute)
			if err := watcher.Start(); err != nil {
				log.Printf("Failed to start Drive watcher: %v", err)
			} else {
				log.Println("Google Drive watcher started")
				defer watcher.Stop()
			}
		}
	}

	// Initialize Gmail Integration (Optional)
	if googleKeyFile != "" {
		ctx := context.Background()
		httpClient, err := googleauth.NewHTTPClient(ctx, googleKeyFile,
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.modify")
		if err != nil {
			log.Printf("Failed to create Gmail HTTP client: %v", err)
		} else {
			gmailSvc, err := gmail.NewService(ctx, httpClient)
			if err != nil {
				log.Printf("Failed to create Gmail service: %v", err)
			} else {
				poller := gmail.NewPoller(gmailSvc, 5*time.Minute, func(subject, body string) error {
					log.Printf("Received email: %s", subject)

					prompt := ai.AnalyzeInboxPrompt(fmt.Sprintf("Subject: %s\nBody: %s", subject, body))
					analysisJSON, err := aiClient.GenerateText(context.Background(), prompt)
					if err != nil {
						return fmt.Errorf("AI analysis failed: %w", err)
					}

					// Use subject as title, analysis as content
					title := subject
					if title == "" {
						title = "Email Item"
					}
					content := fmt.Sprintf("AI Analysis:\n%s\n\nOriginal:\n%s", analysisJSON, body)

					if err := vault.CreateInboxItem(*vaultPath, tmplEngine, title, content); err != nil {
						return fmt.Errorf("create inbox item: %w", err)
					}

					go gitManager.Sync("Add email item: " + title)
					return nil
				})
				go poller.Start()
				defer poller.Stop()
				log.Println("Gmail poller started")
			}
		}
	}

	// Initialize Discord Bot (Optional)
	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken != "" {
		bot, err := discord.NewBot(discordToken, *vaultPath, tmplEngine, gitManager)
		if err != nil {
			log.Printf("Failed to create Discord bot: %v", err)
		} else {
			if err := bot.Start(); err != nil {
				log.Printf("Failed to start Discord bot: %v", err)
			} else {
				log.Println("Discord Bot started")
				defer bot.Stop()
			}
		}
	}

	// Initialize Telegram Bot (Optional)
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken != "" {
		tgBot, err := telegram.NewBot(telegramToken, *vaultPath, tmplEngine, gitManager)
		if err != nil {
			log.Printf("Failed to create Telegram bot: %v", err)
		} else {
			if err := tgBot.Start(); err != nil {
				log.Printf("Failed to start Telegram bot: %v", err)
			} else {
				log.Println("Telegram Bot started")
				defer tgBot.Stop()
			}
		}
	}

	log.Printf("Starting server on :%s", *port)
	if err := http.ListenAndServe(":"+*port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
