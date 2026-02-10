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
	"strconv"
	"testing"
	"time"

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

func TestAutomationEndpoints(t *testing.T) {
	tmpVault, err := ioutil.TempDir("", "vault-test-automation-api")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpVault)

	tmpDBPath := filepath.Join(tmpVault, "test.db")
	database, err := db.NewDB(tmpDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.InitSchema(); err != nil {
		t.Fatal(err)
	}
	repo := db.NewRepository(database)

	tmplDir := filepath.Join(tmpVault, "0. GTD System", "Templates")
	os.MkdirAll(tmplDir, 0755)
	ioutil.WriteFile(filepath.Join(tmplDir, "Inbox Item Template.md"), []byte("# {{title}}\n{{description}}"), 0644)
	tmplEngine := vault.NewTemplateEngine(tmplDir)

	router := NewRouter(repo, &MockGenerator{Response: "{}"}, tmplEngine, tmpVault, nil)

	createBody := map[string]interface{}{
		"name":          "Daily Summary",
		"action_type":   "generate_daily_summary",
		"schedule_kind": "cron",
		"schedule_expr": "0 8 * * *",
		"timezone":      "UTC",
		"payload": map[string]interface{}{
			"folder": "7. Daily Summaries",
		},
	}
	rawCreate, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest("POST", "/automations", bytes.NewBuffer(rawCreate))
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createResp.Code, createResp.Body.String())
	}

	var created struct {
		ID        int64      `json:"id"`
		NextRunAt *time.Time `json:"next_run_at"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("expected id > 0, got %d", created.ID)
	}
	if created.NextRunAt == nil {
		t.Fatal("expected next_run_at to be set")
	}

	listReq := httptest.NewRequest("GET", "/automations", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}

	runNowReq := httptest.NewRequest("POST", "/automations/"+strconv.FormatInt(created.ID, 10)+"/run-now", nil)
	runNowResp := httptest.NewRecorder()
	router.ServeHTTP(runNowResp, runNowReq)
	if runNowResp.Code != http.StatusOK {
		t.Fatalf("run-now status = %d body=%s", runNowResp.Code, runNowResp.Body.String())
	}
}
