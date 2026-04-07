package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RuntimeConfig struct {
	DefaultModelClass ModelClass
	DefaultMaxTokens  int
	Timeout           time.Duration
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		DefaultModelClass: ModelClassDefault,
		DefaultMaxTokens:  4096,
		Timeout:           60 * time.Second,
	}
}

type Runtime struct {
	config    RuntimeConfig
	providers map[ModelClass]Provider
	usage     UsageStore
	budget    BudgetManager
	mu        sync.RWMutex
}

func NewRuntime(config RuntimeConfig) *Runtime {
	return &Runtime{
		config:    config,
		providers: make(map[ModelClass]Provider),
	}
}

func (r *Runtime) RegisterProvider(class ModelClass, provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[class]; exists {
		return fmt.Errorf("provider for class %s already registered", class)
	}
	r.providers[class] = provider
	return nil
}

func (r *Runtime) SetUsageStore(store UsageStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usage = store
}

func (r *Runtime) SetBudgetManager(budget BudgetManager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.budget = budget
}

func (r *Runtime) Execute(ctx context.Context, req PromptRequest) (PromptResult, error) {
	if req.ModelClass == "" {
		req.ModelClass = r.config.DefaultModelClass
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = r.config.DefaultMaxTokens
	}

	r.mu.RLock()
	provider, ok := r.providers[req.ModelClass]
	r.mu.RUnlock()

	if !ok {
		return PromptResult{}, fmt.Errorf("no provider for model class: %s", req.ModelClass)
	}

	if r.budget != nil && req.PluginID != "" {
		allowed, _, err := r.budget.CheckBudget(ctx, req.PluginID)
		if err != nil {
			return PromptResult{}, fmt.Errorf("budget check: %w", err)
		}
		if !allowed {
			return PromptResult{}, fmt.Errorf("budget exceeded for plugin: %s", req.PluginID)
		}
	}

	timeout := r.config.Timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	result, err := provider.Generate(ctx, req)
	if err != nil {
		return PromptResult{}, fmt.Errorf("generate: %w", err)
	}
	result.Duration = time.Since(start)

	if r.usage != nil && req.PluginID != "" {
		record := UsageRecord{
			ID:           uuid.NewString(),
			PluginID:     req.PluginID,
			TaskType:     req.TaskType,
			ModelClass:   req.ModelClass,
			ModelUsed:    result.ModelUsed,
			TokensUsed:   result.TokensUsed,
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
			Cost:         result.Cost,
			Timestamp:    time.Now(),
		}
		_ = r.usage.RecordUsage(ctx, record)
	}

	if r.budget != nil && req.PluginID != "" {
		_ = r.budget.RecordUsage(ctx, req.PluginID, result.TokensUsed, result.Cost)
	}

	return result, nil
}

func (r *Runtime) IsAvailable(class ModelClass) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[class]
	if !ok {
		return false
	}
	return provider.IsAvailable()
}
