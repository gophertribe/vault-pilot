package google

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHTTPClient_InvalidPath(t *testing.T) {
	_, err := NewHTTPClient(context.Background(), "/nonexistent/path.json", "https://www.googleapis.com/auth/calendar")
	if err == nil {
		t.Fatal("expected error for nonexistent credentials file")
	}
}

func TestNewHTTPClient_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewHTTPClient(context.Background(), path, "https://www.googleapis.com/auth/calendar")
	if err == nil {
		t.Fatal("expected error for invalid JSON credentials")
	}
}

func TestClientOption(t *testing.T) {
	opt := ClientOption("/some/path.json")
	if opt == nil {
		t.Fatal("expected non-nil ClientOption")
	}
}
