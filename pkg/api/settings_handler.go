package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/db"
)

type updateSettingsRequest struct {
	AIProvider           string  `json:"ai_provider"`
	AutomationTimezone   string  `json:"automation_timezone"`
	OpenAIAPIKey         *string `json:"openai_api_key"`
	AnthropicAPIKey      *string `json:"anthropic_api_key"`
	TelegramToken        *string `json:"telegram_token"`
	DiscordToken         *string `json:"discord_token"`
	ClearOpenAIAPIKey    bool    `json:"clear_openai_api_key"`
	ClearAnthropicAPIKey bool    `json:"clear_anthropic_api_key"`
	ClearTelegramToken   bool    `json:"clear_telegram_token"`
	ClearDiscordToken    bool    `json:"clear_discord_token"`
}

type settingsResponse struct {
	AIProvider                string `json:"ai_provider"`
	AutomationTimezone        string `json:"automation_timezone"`
	OpenAIAPIKeyConfigured    bool   `json:"openai_api_key_configured"`
	AnthropicAPIKeyConfigured bool   `json:"anthropic_api_key_configured"`
	TelegramTokenConfigured   bool   `json:"telegram_token_configured"`
	DiscordTokenConfigured    bool   `json:"discord_token_configured"`
}

func (h *Handler) HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.Repo.GetAppSettings()
	if err != nil {
		http.Error(w, "failed to load settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := buildSettingsResponse(settings)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	provider := strings.ToLower(strings.TrimSpace(req.AIProvider))
	if provider != "" && provider != "openai" && provider != "anthropic" {
		http.Error(w, "ai_provider must be one of: openai, anthropic, or empty", http.StatusBadRequest)
		return
	}

	tz := strings.TrimSpace(req.AutomationTimezone)
	if tz == "" {
		tz = "UTC"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		http.Error(w, "invalid automation_timezone", http.StatusBadRequest)
		return
	}

	settings, err := h.Repo.GetAppSettings()
	if err != nil {
		http.Error(w, "failed to load settings: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if settings == nil {
		settings = &db.AppSettings{}
	}

	settings.AIProvider = provider
	settings.AutomationTimezone = tz

	settings.OpenAIAPIKey = applySecretUpdate(settings.OpenAIAPIKey, req.OpenAIAPIKey, req.ClearOpenAIAPIKey)
	settings.AnthropicAPIKey = applySecretUpdate(settings.AnthropicAPIKey, req.AnthropicAPIKey, req.ClearAnthropicAPIKey)
	settings.TelegramToken = applySecretUpdate(settings.TelegramToken, req.TelegramToken, req.ClearTelegramToken)
	settings.DiscordToken = applySecretUpdate(settings.DiscordToken, req.DiscordToken, req.ClearDiscordToken)

	if err := h.Repo.UpsertAppSettings(settings); err != nil {
		http.Error(w, "failed to save settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, buildSettingsResponse(settings))
}

func buildSettingsResponse(settings *db.AppSettings) settingsResponse {
	if settings == nil {
		return settingsResponse{
			AIProvider:         "",
			AutomationTimezone: "UTC",
		}
	}

	return settingsResponse{
		AIProvider:                settings.AIProvider,
		AutomationTimezone:        settings.AutomationTimezone,
		OpenAIAPIKeyConfigured:    strings.TrimSpace(settings.OpenAIAPIKey) != "",
		AnthropicAPIKeyConfigured: strings.TrimSpace(settings.AnthropicAPIKey) != "",
		TelegramTokenConfigured:   strings.TrimSpace(settings.TelegramToken) != "",
		DiscordTokenConfigured:    strings.TrimSpace(settings.DiscordToken) != "",
	}
}

func applySecretUpdate(current string, incoming *string, clear bool) string {
	if clear {
		return ""
	}
	if incoming == nil {
		return current
	}
	trimmed := strings.TrimSpace(*incoming)
	if trimmed == "" {
		return current
	}
	return trimmed
}
