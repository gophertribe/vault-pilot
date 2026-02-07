package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com/v1"
	anthropicDefaultModel   = "claude-3-5-haiku-latest"
	anthropicVersion        = "2023-06-01"
)

// AnthropicClient implements the Generator interface using the Anthropic messages API.
type AnthropicClient struct {
	httpClient *http.Client
	apiKey     string
	model      string
	baseURL    string
}

// Ensure AnthropicClient implements Generator.
var _ Generator = (*AnthropicClient)(nil)

// NewAnthropicClient creates a new Anthropic API client.
func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		model:      anthropicDefaultModel,
		baseURL:    anthropicDefaultBaseURL,
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
}

type anthropicMessage struct {
	Role    string               `json:"role"`
	Content []anthropicTextBlock `json:"content"`
}

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	Content []anthropicTextBlock `json:"content"`
	Error   *anthropicError      `json:"error,omitempty"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// GenerateText sends a prompt to Anthropic and returns the generated text.
func (c *AnthropicClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model: c.model,
		Messages: []anthropicMessage{
			{
				Role: "user",
				Content: []anthropicTextBlock{
					{Type: "text", Text: prompt},
				},
			},
		},
		MaxTokens: 1024,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("anthropic API error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content returned")
	}

	return result.Content[0].Text, nil
}

// Close is a no-op for the HTTP-based Anthropic client.
func (c *AnthropicClient) Close() error {
	return nil
}
