package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicGenerateText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected /messages, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != anthropicVersion {
			t.Errorf("expected %s, got %s", anthropicVersion, r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "claude-3-5-haiku-latest" {
			t.Errorf("expected model claude-3-5-haiku-latest, got %s", req.Model)
		}
		if len(req.Messages) != 1 || len(req.Messages[0].Content) != 1 || req.Messages[0].Content[0].Text != "hello" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		resp := anthropicResponse{
			Content: []anthropicTextBlock{{Type: "text", Text: "world"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key")
	client.baseURL = server.URL

	result, err := client.GenerateText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "world" {
		t.Errorf("expected 'world', got %q", result)
	}
}

func TestAnthropicGenerateTextAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key","type":"auth_error"}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient("bad-key")
	client.baseURL = server.URL

	_, err := client.GenerateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAnthropicGenerateTextEmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicResponse{Content: []anthropicTextBlock{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key")
	client.baseURL = server.URL

	_, err := client.GenerateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
}

func TestAnthropicGenerateTextMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key")
	client.baseURL = server.URL

	_, err := client.GenerateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestAnthropicClose(t *testing.T) {
	client := NewAnthropicClient("test-key")
	if err := client.Close(); err != nil {
		t.Fatalf("expected nil error from Close, got %v", err)
	}
}
