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
	moonshotDefaultBaseURL = "https://api.moonshot.ai/v1"
	moonshotDefaultModel   = "kimi-k2.5"
)

// MoonshotClient implements the Generator interface using the Moonshot API (Kimi 2.5).
// The Moonshot API is OpenAI-compatible, using the chat completions endpoint.
type MoonshotClient struct {
	httpClient *http.Client
	apiKey     string
	model      string
	baseURL    string
}

// Ensure MoonshotClient implements Generator
var _ Generator = (*MoonshotClient)(nil)

// NewMoonshotClient creates a new Moonshot API client
func NewMoonshotClient(apiKey string) *MoonshotClient {
	return &MoonshotClient{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		model:      moonshotDefaultModel,
		baseURL:    moonshotDefaultBaseURL,
	}
}

type moonshotRequest struct {
	Model    string            `json:"model"`
	Messages []moonshotMessage `json:"messages"`
}

type moonshotMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type moonshotResponse struct {
	Choices []moonshotChoice `json:"choices"`
	Error   *moonshotError   `json:"error,omitempty"`
}

type moonshotChoice struct {
	Message moonshotMessage `json:"message"`
}

type moonshotError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// GenerateText sends a prompt to the Moonshot API and returns the generated text
func (c *MoonshotClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	reqBody := moonshotRequest{
		Model: c.model,
		Messages: []moonshotMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return "", fmt.Errorf("moonshot API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var result moonshotResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("moonshot API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return result.Choices[0].Message.Content, nil
}

// Close is a no-op for the HTTP-based Moonshot client
func (c *MoonshotClient) Close() error {
	return nil
}
