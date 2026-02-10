package ai

import (
	"context"
	"errors"
)

// ErrNotConfigured is returned when the AI provider is not configured.
var ErrNotConfigured = errors.New("AI provider not configured; configure in Settings")

// NoopGenerator is a Generator that returns ErrNotConfigured.
// Used when no AI provider is configured at startup.
type NoopGenerator struct{}

// Ensure NoopGenerator implements Generator.
var _ Generator = (*NoopGenerator)(nil)

// GenerateText returns ErrNotConfigured.
func (n *NoopGenerator) GenerateText(_ context.Context, _ string) (string, error) {
	return "", ErrNotConfigured
}
