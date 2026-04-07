package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EventAppender interface {
	Append(ctx context.Context, evt Event) error
}

type InMemoryBus struct {
	mu     sync.RWMutex
	subs   map[string][]EventHandler
	store  EventAppender
	closed bool
}

func NewInMemoryBus(s EventAppender) Bus {
	return &InMemoryBus{
		subs:  make(map[string][]EventHandler),
		store: s,
	}
}

func (b *InMemoryBus) Publish(ctx context.Context, evt Event) error {
	if evt.ID == "" {
		evt.ID = uuid.NewString()
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now()
	}

	if b.store != nil {
		if err := b.store.Append(ctx, evt); err != nil {
			return fmt.Errorf("persist event: %w", err)
		}
	}

	b.mu.RLock()
	handlers := b.subs[evt.Type]
	wildcard := b.subs["*"]
	b.mu.RUnlock()

	allHandlers := append(handlers, wildcard...)

	for _, h := range allHandlers {
		go func(handler EventHandler) {
			handler(ctx, evt)
		}(h)
	}
	return nil
}

func (b *InMemoryBus) Subscribe(eventType string, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return fmt.Errorf("bus is closed")
	}
	b.subs[eventType] = append(b.subs[eventType], handler)
	return nil
}

func (b *InMemoryBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.subs = make(map[string][]EventHandler)
	return nil
}

func (b *InMemoryBus) Subscriptions() map[string]int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make(map[string]int)
	for k, v := range b.subs {
		result[k] = len(v)
	}
	return result
}
