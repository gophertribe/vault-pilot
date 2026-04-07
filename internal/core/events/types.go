package events

import (
	"context"
	"encoding/json"
	"time"
)

type Event struct {
	ID            string            `json:"id"`
	CorrelationID string            `json:"correlation_id"`
	Source        string            `json:"source"`
	Type          string            `json:"type"`
	OccurredAt    time.Time         `json:"occurred_at"`
	Payload       json.RawMessage   `json:"payload"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type EventHandler func(ctx context.Context, evt Event) error

type Bus interface {
	Publish(ctx context.Context, evt Event) error
	Subscribe(eventType string, handler EventHandler) error
	Close() error
}
