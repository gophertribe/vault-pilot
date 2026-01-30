package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mklimuk/vault-pilot/pkg/ai"
	"github.com/mklimuk/vault-pilot/pkg/api"
	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/integration/discord"
	"github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

func main() {
	vaultPath := flag.String("vault", "", "Path to Obsidian Vault")
	dbPath := flag.String("db", "vault-pilot.db", "Path to SQLite DB")
	port := flag.String("port", "8080", "HTTP Port")
	flag.Parse()

	if *vaultPath == "" {
		log.Fatal("Please provide -vault path")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
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
	ctx := context.Background()
	aiClient, err := ai.NewClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create AI client: %v", err)
	}
	defer aiClient.Close()

	// Initialize Template Engine
	templateDir := filepath.Join(*vaultPath, "0. GTD System", "Templates")
	tmplEngine := vault.NewTemplateEngine(templateDir)

	// Initialize Git Manager
	gitManager := sync.NewGitManager(*vaultPath)

	// Initialize Router
	router := api.NewRouter(repo, aiClient, tmplEngine, *vaultPath, gitManager)

	// Initialize Gmail Integration (Optional)
	// For MVP, we'll just log if it fails or skip if no credentials
	// We need an authenticated HTTP client. In a real app, this comes from OAuth flow.
	// Here we just skip it to avoid crashing if not configured, or we could try to use a service account if path provided.
	// Since we don't have a ready-to-use client, we'll comment out the actual start but show the logic.
	// To make it work, user would need to implement the auth flow in client.go.

	// Placeholder for when we have a client:
	/*
		gmailClient, err := gmail.NewService(ctx, someHttpClient)
		if err == nil {
			poller := gmail.NewPoller(gmailClient, 5*time.Minute, func(subject, body string) error {
				log.Printf("Received email: %s", subject)

				// Analyze with AI
				prompt := ai.AnalyzeInboxPrompt(fmt.Sprintf("Subject: %s\nBody: %s", subject, body))
				analysisJSON, err := aiClient.GenerateText(context.Background(), prompt)
				if err != nil {
					return fmt.Errorf("AI analysis failed: %w", err)
				}

				// Parse JSON (simplified)
				// ... (reuse cleanJSON and unmarshal logic) ...

				// Create Item
				// err = vault.CreateInboxItem(*vaultPath, tmplEngine, title, description)

				// Sync
				// gitManager.Sync("Add email item")

				return nil
			})
			go poller.Start()
		}
	*/

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

	log.Printf("Starting server on :%s", *port)
	if err := http.ListenAndServe(":"+*port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
