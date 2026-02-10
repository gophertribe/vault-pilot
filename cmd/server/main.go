package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/ai"
	"github.com/mklimuk/vault-pilot/pkg/api"
	"github.com/mklimuk/vault-pilot/pkg/automation"
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
	aiProvider := flag.String("ai-provider", "gemini", "AI provider: gemini, moonshot, openai, or anthropic")
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
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			log.Fatal("OPENAI_API_KEY environment variable is required when using openai provider")
		}
		aiClient = ai.NewOpenAIClient(key)
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			log.Fatal("ANTHROPIC_API_KEY environment variable is required when using anthropic provider")
		}
		aiClient = ai.NewAnthropicClient(key)
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
	var gmailSvc *gmail.Service

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
			gmailSvc, err = gmail.NewService(ctx, httpClient)
			if err != nil {
				log.Printf("Failed to create Gmail service: %v", err)
			} else {
				log.Println("Gmail service initialized for automation actions")
			}
		}
	}

	automationService := automation.NewService(repo, 15*time.Second, 10)
	automationService.RegisterAction("pull_gmail", func(ctx context.Context, def db.AutomationDefinition) (string, error) {
		if gmailSvc == nil {
			return "", fmt.Errorf("gmail service is not configured")
		}
		msgs, err := gmailSvc.FetchUnreadEmails(ctx)
		if err != nil {
			return "", fmt.Errorf("fetch unread emails: %w", err)
		}
		created := 0
		for _, msg := range msgs {
			subject := ""
			for _, h := range msg.Payload.Headers {
				if h.Name == "Subject" {
					subject = h.Value
					break
				}
			}
			if subject == "" {
				subject = "Email Item"
			}
			body := gmail.GetBody(msg)
			prompt := ai.AnalyzeInboxPrompt(fmt.Sprintf("Subject: %s\nBody: %s", subject, body))
			analysisJSON, err := aiClient.GenerateText(ctx, prompt)
			if err != nil {
				log.Printf("pull_gmail: AI analysis failed for subject=%q: %v", subject, err)
				continue
			}
			content := fmt.Sprintf("AI Analysis:\n%s\n\nOriginal:\n%s", analysisJSON, body)
			if err := vault.CreateInboxItem(*vaultPath, tmplEngine, subject, content); err != nil {
				log.Printf("pull_gmail: failed to create inbox item for subject=%q: %v", subject, err)
				continue
			}
			created++
		}
		if created > 0 && gitManager != nil {
			go gitManager.Sync(fmt.Sprintf("Automation: import %d email(s)", created))
		}
		return fmt.Sprintf("created %d inbox item(s)", created), nil
	})
	automationService.RegisterAction("generate_daily_summary", func(ctx context.Context, def db.AutomationDefinition) (string, error) {
		var payload struct {
			Folder string `json:"folder"`
			Title  string `json:"title"`
		}
		if strings.TrimSpace(def.PayloadJSON) != "" {
			if err := json.Unmarshal([]byte(def.PayloadJSON), &payload); err != nil {
				return "", fmt.Errorf("invalid payload_json: %w", err)
			}
		}
		targetFolder := payload.Folder
		if targetFolder == "" {
			targetFolder = "7. Daily Summaries"
		}
		title := payload.Title
		if title == "" {
			title = "Daily Vault Summary"
		}

		now := time.Now()
		prompt := fmt.Sprintf(
			"Generate a concise daily vault summary for %s with sections: Wins, Open Loops, Risks, and Top 3 Priorities.",
			now.Format("2006-01-02"),
		)
		summary, err := aiClient.GenerateText(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("generate summary: %w", err)
		}

		fileName := fmt.Sprintf("%s %s.md", now.Format("2006-01-02"), vault.SanitizeFilename(title))
		path := filepath.Join(*vaultPath, targetFolder, fileName)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", fmt.Errorf("create folder: %w", err)
		}
		content := fmt.Sprintf("# %s\n\nDate: %s\n\n%s\n", title, now.Format("2006-01-02"), summary)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("write summary: %w", err)
		}
		if gitManager != nil {
			go gitManager.Sync("Automation: add daily summary " + now.Format("2006-01-02"))
		}
		return "wrote " + fileName, nil
	})
	if err := ensureDefaultAutomations(repo, gmailSvc != nil, os.Getenv("AUTOMATION_TIMEZONE")); err != nil {
		log.Printf("Failed to seed default automations: %v", err)
	}
	automationService.Start()
	defer automationService.Stop()
	log.Println("Automation scheduler started")

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

func ensureDefaultAutomations(repo *db.Repository, hasGmail bool, tz string) error {
	if tz == "" {
		tz = "UTC"
	}
	existing, err := repo.ListAutomations()
	if err != nil {
		return err
	}
	hasAction := map[string]bool{}
	for _, def := range existing {
		hasAction[def.ActionType] = true
	}

	if hasGmail && !hasAction["pull_gmail"] {
		nextRun, err := automation.NextRun("interval", "5m", tz, time.Now().UTC())
		if err != nil {
			return err
		}
		_, err = repo.CreateAutomation(&db.AutomationDefinition{
			Name:         "Pull Gmail Inbox",
			ActionType:   "pull_gmail",
			ScheduleKind: "interval",
			ScheduleExpr: "5m",
			Timezone:     tz,
			PayloadJSON:  `{}`,
			Enabled:      true,
			NextRunAt:    nextRun,
		})
		if err != nil {
			return err
		}
		log.Println("Seeded default automation: pull_gmail")
	}

	if !hasAction["generate_daily_summary"] {
		nextRun, err := automation.NextRun("cron", "0 8 * * *", tz, time.Now().UTC())
		if err != nil {
			return err
		}
		_, err = repo.CreateAutomation(&db.AutomationDefinition{
			Name:         "Daily Vault Summary",
			ActionType:   "generate_daily_summary",
			ScheduleKind: "cron",
			ScheduleExpr: "0 8 * * *",
			Timezone:     tz,
			PayloadJSON:  `{"folder":"7. Daily Summaries","title":"Daily Vault Summary"}`,
			Enabled:      true,
			NextRunAt:    nextRun,
		})
		if err != nil {
			return err
		}
		log.Println("Seeded default automation: generate_daily_summary")
	}

	return nil
}
