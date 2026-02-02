package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMoonshotGenerateText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req moonshotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "kimi-k2.5" {
			t.Errorf("expected model kimi-k2.5, got %s", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		resp := moonshotResponse{
			Choices: []moonshotChoice{
				{Message: moonshotMessage{Role: "assistant", Content: "world"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMoonshotClient("test-key")
	client.baseURL = server.URL

	result, err := client.GenerateText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "world" {
		t.Errorf("expected 'world', got %q", result)
	}
}

func TestMoonshotGenerateTextAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key","type":"auth_error"}}`))
	}))
	defer server.Close()

	client := NewMoonshotClient("bad-key")
	client.baseURL = server.URL

	_, err := client.GenerateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMoonshotGenerateTextEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := moonshotResponse{Choices: []moonshotChoice{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMoonshotClient("test-key")
	client.baseURL = server.URL

	_, err := client.GenerateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
}

func TestMoonshotGenerateTextMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewMoonshotClient("test-key")
	client.baseURL = server.URL

	_, err := client.GenerateText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestMoonshotClose(t *testing.T) {
	client := NewMoonshotClient("test-key")
	if err := client.Close(); err != nil {
		t.Fatalf("expected nil error from Close, got %v", err)
	}
}
