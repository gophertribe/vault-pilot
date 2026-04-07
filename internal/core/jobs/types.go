package jobs

import (
	"context"
	"encoding/json"
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusDone     Status = "done"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

type Job struct {
	ID            string          `json:"id"`
	CorrelationID string          `json:"correlation_id"`
	PluginID      string          `json:"plugin_id"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
	Status        Status          `json:"status"`
	Attempts      int             `json:"attempts"`
	MaxAttempts   int             `json:"max_attempts"`
	CreatedAt     time.Time       `json:"created_at"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	Error         string          `json:"error,omitempty"`
	Result        json.RawMessage `json:"result,omitempty"`
}

type Handler func(ctx context.Context, job Job) (json.RawMessage, error)

type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context) (*Job, error)
	UpdateStatus(ctx context.Context, jobID string, status Status, result json.RawMessage, errMsg string) error
	GetByID(ctx context.Context, jobID string) (*Job, error)
}
