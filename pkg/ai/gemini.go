package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Generator defines the interface for text generation
type Generator interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
}

// Client wraps the Gemini API client
type Client struct {
	genaiClient *genai.Client
	model       *genai.GenerativeModel
}

// Ensure Client implements Generator
var _ Generator = (*Client)(nil)

// NewClient creates a new Gemini client
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	// Use gemini-pro by default
	model := client.GenerativeModel("gemini-pro")

	return &Client{
		genaiClient: client,
		model:       model,
	}, nil
}

// Close closes the client
func (c *Client) Close() error {
	return c.genaiClient.Close()
}

// GenerateText generates text from a prompt
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			sb.WriteString(string(txt))
		}
	}

	return sb.String(), nil
}
