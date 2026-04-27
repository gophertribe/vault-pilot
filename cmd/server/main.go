package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	"github.com/mklimuk/vault-pilot/web"
)

// Config holds all configuration for the application
type Config struct {
	VaultPath                 string
	DBPath                    string
	Port                      string
	AIProvider                string
	GoogleServiceAccountKey   string
	GoogleCalendarID          string
	GoogleDriveBackupFolderID string
	GoogleDriveWatchFolderID  string
	DiscordToken              string
	TelegramToken             string
	AutomationTimezone        string
}

var (
	cfgFile string
	config  Config
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "vault-pilot",
	Short: "GTD Obsidian vault management backend",
	Long:  `Vault Pilot is a Go backend for managing a GTD (Getting Things Done) Obsidian vault.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

func init() {
	// Config file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $XDG_CONFIG_HOME/life-pilot/config.yaml, $XDG_CONFIG_HOME=$HOME/.config if unset)")

	// Application flags (vault and db default to $XDG_DATA_HOME/life-pilot/...)
	rootCmd.Flags().StringP("vault", "v", "", "Path to Obsidian Vault (default $XDG_DATA_HOME/life-pilot/vault, $XDG_DATA_HOME=$HOME/.local/share if unset)")
	rootCmd.Flags().StringP("db", "d", "", "Path to SQLite DB (default $XDG_DATA_HOME/life-pilot/pilot.db, $XDG_DATA_HOME=$HOME/.local/share if unset)")
	rootCmd.Flags().StringP("port", "p", "8080", "HTTP Port")
	rootCmd.Flags().String("ai-provider", "", "AI provider: gemini, moonshot, openai, or anthropic (default: none, configure in Settings)")

	// Google integration flags
	rootCmd.Flags().String("google-service-account-key", "", "Path to Google service account JSON key file")
	rootCmd.Flags().String("google-calendar-id", "", "Google Calendar ID for bidirectional sync")
	rootCmd.Flags().String("google-drive-backup-folder-id", "", "Google Drive folder ID for vault backup")
	rootCmd.Flags().String("google-drive-watch-folder-id", "", "Google Drive folder ID to watch for incoming files")

	// Bot integration flags
	rootCmd.Flags().String("discord-token", "", "Discord bot token")
	rootCmd.Flags().String("telegram-token", "", "Telegram bot token")

	// Automation flags
	rootCmd.Flags().String("automation-timezone", "UTC", "Timezone for automation schedules")

	// Bind flags to Viper
	viper.BindPFlag("vault", rootCmd.Flags().Lookup("vault"))
	viper.BindPFlag("db", rootCmd.Flags().Lookup("db"))
	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("ai-provider", rootCmd.Flags().Lookup("ai-provider"))
	viper.BindPFlag("google.service_account_key", rootCmd.Flags().Lookup("google-service-account-key"))
	viper.BindPFlag("google.calendar_id", rootCmd.Flags().Lookup("google-calendar-id"))
	viper.BindPFlag("google.drive_backup_folder_id", rootCmd.Flags().Lookup("google-drive-backup-folder-id"))
	viper.BindPFlag("google.drive_watch_folder_id", rootCmd.Flags().Lookup("google-drive-watch-folder-id"))
	viper.BindPFlag("discord.token", rootCmd.Flags().Lookup("discord-token"))
	viper.BindPFlag("telegram.token", rootCmd.Flags().Lookup("telegram-token"))
	viper.BindPFlag("automation.timezone", rootCmd.Flags().Lookup("automation-timezone"))

	// Set environment variable prefix (e.g. VAULT_PILOT_VAULT, VAULT_PILOT_PORT)
	viper.SetEnvPrefix("VAULT_PILOT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			configDir = filepath.Join(home, ".config")
		}
		viper.SetConfigFile(filepath.Join(configDir, "life-pilot", "config.yaml"))
	}

	// Read config file if it exists (ignore errors if file doesn't exist)
	viper.ReadInConfig()

	// Load configuration into struct
	config = Config{
		VaultPath:                 viper.GetString("vault"),
		DBPath:                    viper.GetString("db"),
		Port:                      viper.GetString("port"),
		AIProvider:                viper.GetString("ai-provider"),
		GoogleServiceAccountKey:   viper.GetString("google.service_account_key"),
		GoogleCalendarID:          viper.GetString("google.calendar_id"),
		GoogleDriveBackupFolderID: viper.GetString("google.drive_backup_folder_id"),
		GoogleDriveWatchFolderID:  viper.GetString("google.drive_watch_folder_id"),
		DiscordToken:              viper.GetString("discord.token"),
		TelegramToken:             viper.GetString("telegram.token"),
		AutomationTimezone:        viper.GetString("automation.timezone"),
	}

	// Apply defaults for vault and db (XDG_DATA_HOME/life-pilot/...)
	dataDir := getDataDir()
	if config.VaultPath == "" {
		config.VaultPath = filepath.Join(dataDir, "life-pilot", "vault")
	}
	if config.DBPath == "" {
		config.DBPath = filepath.Join(dataDir, "life-pilot", "pilot.db")
	}

	return nil
}

// getDataDir returns XDG_DATA_HOME or $HOME/.local/share.
func getDataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "share")
}

func runServer() error {

	// Initialize DB
	database, err := db.NewDB(config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer database.Close()

	if err := database.InitSchema(); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	repo := db.NewRepository(database)

	storedSettings, err := repo.GetAppSettings()
	if err != nil {
		log.Printf("Failed to load persisted app settings: %v", err)
	}
	applyStoredSettingsFallback(&config, storedSettings)

	// Ensure vault and db parent directories exist
	if err := os.MkdirAll(filepath.Dir(config.DBPath), 0755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}
	if err := os.MkdirAll(config.VaultPath, 0755); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Initialize AI Client (optional; user configures in Settings)
	var aiClient ai.Generator = &ai.NoopGenerator{}
	openAIKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if openAIKey == "" && storedSettings != nil {
		openAIKey = strings.TrimSpace(storedSettings.OpenAIAPIKey)
	}
	anthropicKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if anthropicKey == "" && storedSettings != nil {
		anthropicKey = strings.TrimSpace(storedSettings.AnthropicAPIKey)
	}
	if config.AIProvider != "" {
		switch config.AIProvider {
		case "moonshot":
			if key := os.Getenv("MOONSHOT_API_KEY"); key != "" {
				aiClient = ai.NewMoonshotClient(key)
			}
		case "openai":
			if openAIKey != "" {
				aiClient = ai.NewOpenAIClient(openAIKey)
			}
		case "anthropic":
			if anthropicKey != "" {
				aiClient = ai.NewAnthropicClient(anthropicKey)
			}
		case "gemini":
			if key := os.Getenv("GEMINI_API_KEY"); key != "" {
				ctx := context.Background()
				geminiClient, err := ai.NewClient(ctx, key)
				if err != nil {
					log.Printf("Failed to create Gemini client: %v", err)
				} else {
					aiClient = geminiClient
				}
			}
		}
	}

	// Initialize Template Engine
	templateDir := filepath.Join(config.VaultPath, "0. GTD System", "Templates")
	tmplEngine := vault.NewTemplateEngine(templateDir)

	// Initialize Git Manager
	gitManager := sync.NewGitManager(config.VaultPath)

	// Initialize API Router
	apiRouter := api.NewRouter(repo, aiClient, tmplEngine, config.VaultPath, gitManager)

	// Initialize frontend handler (embedded assets)
	webHandler, err := web.NewHandler(web.Assets)
	if err != nil {
		return fmt.Errorf("frontend assets not found: run 'cd web && bun run build' first: %w", err)
	}

	// Main handler: /api/* -> API, /app -> frontend, / -> redirect to /app
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiRouter))
	mux.Handle("/app", webHandler)
	mux.Handle("/app/", webHandler)
	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/app/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	mux.Handle("/", rootHandler)

	var gmailSvc *gmail.Service

	// Initialize Google Calendar Sync (Optional)
	if config.GoogleServiceAccountKey != "" && config.GoogleCalendarID != "" {
		ctx := context.Background()
		calSvc, err := calendar.NewService(ctx, config.GoogleServiceAccountKey, config.GoogleCalendarID)
		if err != nil {
			log.Printf("Failed to create Calendar service: %v", err)
		} else {
			calSyncer := calendar.NewSyncer(calSvc, repo, config.VaultPath, tmplEngine, gitManager,
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
	if config.GoogleServiceAccountKey != "" && config.GoogleDriveBackupFolderID != "" {
		ctx := context.Background()
		drvSvc, err := drive.NewService(ctx, config.GoogleServiceAccountKey, config.GoogleDriveBackupFolderID)
		if err != nil {
			log.Printf("Failed to create Drive backup service: %v", err)
		} else {
			backup := drive.NewBackup(drvSvc, repo, config.VaultPath, 30*time.Minute)
			if err := backup.Start(); err != nil {
				log.Printf("Failed to start Drive backup: %v", err)
			} else {
				log.Println("Google Drive backup started")
				defer backup.Stop()
			}
		}
	}

	// Initialize Google Drive Watcher (Optional)
	if config.GoogleServiceAccountKey != "" && config.GoogleDriveWatchFolderID != "" {
		ctx := context.Background()
		drvSvc, err := drive.NewService(ctx, config.GoogleServiceAccountKey, config.GoogleDriveWatchFolderID)
		if err != nil {
			log.Printf("Failed to create Drive watch service: %v", err)
		} else {
			watcher := drive.NewWatcher(drvSvc, repo, config.VaultPath, tmplEngine, gitManager, 5*time.Minute)
			if err := watcher.Start(); err != nil {
				log.Printf("Failed to start Drive watcher: %v", err)
			} else {
				log.Println("Google Drive watcher started")
				defer watcher.Stop()
			}
		}
	}

	// Initialize Gmail Integration (Optional)
	if config.GoogleServiceAccountKey != "" {
		ctx := context.Background()
		httpClient, err := googleauth.NewHTTPClient(ctx, config.GoogleServiceAccountKey,
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
			if err := vault.CreateInboxItem(config.VaultPath, tmplEngine, subject, content); err != nil {
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
		path := filepath.Join(config.VaultPath, targetFolder, fileName)
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
	timezone := config.AutomationTimezone
	if timezone == "" {
		timezone = "UTC"
	}
	if err := ensureDefaultAutomations(repo, gmailSvc != nil, timezone); err != nil {
		log.Printf("Failed to seed default automations: %v", err)
	}
	automationService.Start()
	defer automationService.Stop()
	log.Println("Automation scheduler started")

	// Initialize Discord Bot (Optional)
	if config.DiscordToken != "" {
		bot, err := discord.NewBot(config.DiscordToken, config.VaultPath, tmplEngine, gitManager)
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
	if config.TelegramToken != "" {
		tgBot, err := telegram.NewBot(config.TelegramToken, config.VaultPath, tmplEngine, gitManager)
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

	log.Printf("Starting server on :%s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, mux); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
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

func applyStoredSettingsFallback(cfg *Config, settings *db.AppSettings) {
	if settings == nil {
		return
	}

	if strings.TrimSpace(cfg.AIProvider) == "" {
		cfg.AIProvider = strings.TrimSpace(settings.AIProvider)
	}
	if strings.TrimSpace(cfg.TelegramToken) == "" {
		cfg.TelegramToken = strings.TrimSpace(settings.TelegramToken)
	}
	if strings.TrimSpace(cfg.DiscordToken) == "" {
		cfg.DiscordToken = strings.TrimSpace(settings.DiscordToken)
	}
	if strings.TrimSpace(cfg.AutomationTimezone) == "" || strings.EqualFold(strings.TrimSpace(cfg.AutomationTimezone), "UTC") {
		if tz := strings.TrimSpace(settings.AutomationTimezone); tz != "" {
			cfg.AutomationTimezone = tz
		}
	}
}
