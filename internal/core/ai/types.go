package ai

import (
	"context"
	"encoding/json"
	"time"
)

type ModelClass string

const (
	ModelClassFast    ModelClass = "fast"
	ModelClassDefault ModelClass = "default"
	ModelClassStrong  ModelClass = "strong"
)

type PromptRequest struct {
	PluginID   string          `json:"plugin_id"`
	TaskType   string          `json:"task_type"`
	ModelClass ModelClass      `json:"model_class"`
	Input      string          `json:"input"`
	Context    json.RawMessage `json:"context,omitempty"`
	BudgetKey  string          `json:"budget_key,omitempty"`
	MaxTokens  int             `json:"max_tokens,omitempty"`
}

type PromptResult struct {
	Output       string          `json:"output"`
	ModelUsed    string          `json:"model_used"`
	TokensUsed   int             `json:"tokens_used"`
	InputTokens  int             `json:"input_tokens"`
	OutputTokens int             `json:"output_tokens"`
	Duration     time.Duration   `json:"duration"`
	Cost         float64         `json:"cost"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

type Provider interface {
	Generate(ctx context.Context, req PromptRequest) (PromptResult, error)
	IsAvailable() bool
	Name() string
}

type UsageRecord struct {
	ID           string     `json:"id"`
	PluginID     string     `json:"plugin_id"`
	TaskType     string     `json:"task_type"`
	ModelClass   ModelClass `json:"model_class"`
	ModelUsed    string     `json:"model_used"`
	TokensUsed   int        `json:"tokens_used"`
	InputTokens  int        `json:"input_tokens"`
	OutputTokens int        `json:"output_tokens"`
	Cost         float64    `json:"cost"`
	Timestamp    time.Time  `json:"timestamp"`
}

type UsageStore interface {
	RecordUsage(ctx context.Context, record UsageRecord) error
	GetUsageByPlugin(ctx context.Context, pluginID string, since time.Time) ([]UsageRecord, error)
	GetTotalUsage(ctx context.Context, since time.Time) (int, float64, error)
}

type BudgetConfig struct {
	DailyTokenLimit int
	DailyCostLimit  float64
	Enabled         bool
}

type BudgetManager interface {
	CheckBudget(ctx context.Context, pluginID string) (allowed bool, remaining int, err error)
	RecordUsage(ctx context.Context, pluginID string, tokens int, cost float64) error
}
