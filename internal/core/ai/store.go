package ai

import (
	"context"
	"sync"
	"time"
)

type InMemoryUsageStore struct {
	mu      sync.RWMutex
	records []UsageRecord
}

func NewInMemoryUsageStore() *InMemoryUsageStore {
	return &InMemoryUsageStore{
		records: make([]UsageRecord, 0),
	}
}

func (s *InMemoryUsageStore) RecordUsage(ctx context.Context, record UsageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
	return nil
}

func (s *InMemoryUsageStore) GetUsageByPlugin(ctx context.Context, pluginID string, since time.Time) ([]UsageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []UsageRecord
	for _, r := range s.records {
		if r.PluginID == pluginID && r.Timestamp.After(since) {
			result = append(result, r)
		}
	}
	return result, nil
}

func (s *InMemoryUsageStore) GetTotalUsage(ctx context.Context, since time.Time) (int, float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var tokens int
	var cost float64
	for _, r := range s.records {
		if r.Timestamp.After(since) {
			tokens += r.TokensUsed
			cost += r.Cost
		}
	}
	return tokens, cost, nil
}

type InMemoryBudgetManager struct {
	mu         sync.RWMutex
	dailyLimit int
	used       map[string]int
	dayStart   time.Time
}

func NewInMemoryBudgetManager(dailyLimit int) *InMemoryBudgetManager {
	return &InMemoryBudgetManager{
		dailyLimit: dailyLimit,
		used:       make(map[string]int),
		dayStart:   startOfDay(time.Now()),
	}
}

func (b *InMemoryBudgetManager) CheckBudget(ctx context.Context, pluginID string) (allowed bool, remaining int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	currentDay := startOfDay(now)
	if !currentDay.Equal(b.dayStart) {
		b.dayStart = currentDay
		b.used = make(map[string]int)
	}

	used := b.used[pluginID]
	remaining = b.dailyLimit - used
	allowed = remaining > 0
	return allowed, remaining, nil
}

func (b *InMemoryBudgetManager) RecordUsage(ctx context.Context, pluginID string, tokens int, cost float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	currentDay := startOfDay(now)
	if !currentDay.Equal(b.dayStart) {
		b.dayStart = currentDay
		b.used = make(map[string]int)
	}

	b.used[pluginID] += tokens
	_ = cost
	return nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
