package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// MockGenerator implements ai.Generator for testing
type MockGenerator struct {
	Response string
	Err      error
}

func (m *MockGenerator) GenerateText(ctx context.Context, prompt string) (string, error) {
	return m.Response, m.Err
}

func TestHandleCreateInboxItem(t *testing.T) {
	// Setup Temp Vault
	tmpVault, err := ioutil.TempDir("", "vault-test-api")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpVault)

	// Setup Temp DB
	tmpDBPath := filepath.Join(tmpVault, "test.db")
	database, err := db.NewDB(tmpDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.InitSchema()
	repo := db.NewRepository(database)

	// Setup Template Engine with dummy template
	tmplDir := filepath.Join(tmpVault, "0. GTD System", "Templates")
	os.MkdirAll(tmplDir, 0755)
	ioutil.WriteFile(filepath.Join(tmplDir, "Inbox Item Template.md"), []byte("# {{title}}\n{{description}}"), 0644)
	tmplEngine := vault.NewTemplateEngine(tmplDir)

	// Setup Mock AI
	mockAI := &MockGenerator{
		Response: `{"title": "Test Item", "description": "Test Description"}`,
	}

	// Setup Router
	router := NewRouter(repo, mockAI, tmplEngine, tmpVault, nil)

	// Create Request
	reqBody := map[string]string{"content": "Buy milk"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/inbox", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Serve
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify File Created
	expectedPath := filepath.Join(tmpVault, "1. Inbox", "Test Item.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("File not created at %s", expectedPath)
	}
}
