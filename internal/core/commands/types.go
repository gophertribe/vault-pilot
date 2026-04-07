package commands

import (
	"context"
	"time"
)

type Command struct {
	ID            string            `json:"id"`
	CorrelationID string            `json:"correlation_id"`
	Channel       string            `json:"channel"`
	SenderID      string            `json:"sender_id"`
	Name          string            `json:"name"`
	Args          map[string]string `json:"args,omitempty"`
	RawText       string            `json:"raw_text"`
	ReceivedAt    time.Time         `json:"received_at"`
}

type CommandHandler func(ctx context.Context, cmd Command) error

type Router interface {
	Register(name string, handler CommandHandler) error
	Dispatch(ctx context.Context, cmd Command) error
}
