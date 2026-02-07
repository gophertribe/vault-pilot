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
	openAIDefaultBaseURL = "https://api.openai.com/v1"
	openAIDefaultModel   = "gpt-4o-mini"
)

// OpenAIClient implements the Generator interface using the OpenAI chat completions API.
type OpenAIClient struct {
	httpClient *http.Client
	apiKey     string
	model      string
	baseURL    string
}

// Ensure OpenAIClient implements Generator.
var _ Generator = (*OpenAIClient)(nil)

// NewOpenAIClient creates a new OpenAI API client.
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		model:      openAIDefaultModel,
		baseURL:    openAIDefaultBaseURL,
	}
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// GenerateText sends a prompt to OpenAI and returns the generated text.
func (c *OpenAIClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
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
		return "", fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var result openAIResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("openai API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return result.Choices[0].Message.Content, nil
}

// Close is a no-op for the HTTP-based OpenAI client.
func (c *OpenAIClient) Close() error {
	return nil
}
