package calendar

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// mockCalendarAPI is a test double for CalendarAPI.
type mockCalendarAPI struct {
	events       []Event
	createdIDs   map[string]string // summary -> id
	updatedCalls []updateCall
	nextID       int
}

type updateCall struct {
	EventID string
	Event   Event
}

func newMockCalendarAPI(events []Event) *mockCalendarAPI {
	return &mockCalendarAPI{
		events:     events,
		createdIDs: make(map[string]string),
		nextID:     100,
	}
}

func (m *mockCalendarAPI) FetchUpcoming(_ context.Context, _ time.Duration) ([]Event, error) {
	return m.events, nil
}

func (m *mockCalendarAPI) CreateEvent(_ context.Context, e Event) (string, error) {
	m.nextID++
	id := "mock-" + e.Summary
	m.createdIDs[e.Summary] = id
	return id, nil
}

func (m *mockCalendarAPI) UpdateEvent(_ context.Context, eventID string, e Event) error {
	m.updatedCalls = append(m.updatedCalls, updateCall{EventID: eventID, Event: e})
	return nil
}

func setupTestDB(t *testing.T) *db.Repository {
	t.Helper()
	database, err := db.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := database.InitSchema(); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}
	return db.NewRepository(database)
}

func setupVault(t *testing.T) (string, *vault.TemplateEngine) {
	t.Helper()
	vaultDir := t.TempDir()

	// Create required dirs
	dirs := []string{
		filepath.Join(vaultDir, "0. GTD System", "Templates"),
		filepath.Join(vaultDir, "1. Inbox"),
		filepath.Join(vaultDir, "2. Next Actions"),
		filepath.Join(vaultDir, "3. Projects"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	// Write a minimal inbox template
	tmplContent := "---\ncreated: {{date:YYYY-MM-DD}}\nstatus: inbox\n---\n# {{title}}\n"
	os.WriteFile(filepath.Join(vaultDir, "0. GTD System", "Templates", "Inbox Item Template.md"), []byte(tmplContent), 0644)

	tmplEngine := vault.NewTemplateEngine(filepath.Join(vaultDir, "0. GTD System", "Templates"))
	return vaultDir, tmplEngine
}

func TestPullNewEvent(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	start := time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 5, 11, 0, 0, 0, time.UTC)

	mock := newMockCalendarAPI([]Event{
		{ID: "evt-1", Summary: "Team Standup", StartTime: start, EndTime: end},
	})

	syncer := NewSyncer(mock, repo, vaultDir, tmplEngine, nil, time.Hour, 14*24*time.Hour)

	modified, err := syncer.pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !modified {
		t.Error("expected modified=true for new event")
	}

	// Verify note was created
	notePath := filepath.Join(vaultDir, "2. Next Actions", "@calendar", "Team Standup.md")
	if _, err := os.Stat(notePath); os.IsNotExist(err) {
		t.Fatalf("expected note at %s", notePath)
	}

	note, err := vault.ReadNote(notePath)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	fm := note.Frontmatter.(map[string]interface{})
	if fm["calendar_id"] != "evt-1" {
		t.Errorf("calendar_id = %v", fm["calendar_id"])
	}
	if fm["context"] != "@calendar" {
		t.Errorf("context = %v", fm["context"])
	}

	// Verify DB record
	rec, _ := repo.GetCalendarSyncByEventID("evt-1")
	if rec == nil {
		t.Fatal("expected sync record")
	}
	if rec.Direction != "pull" {
		t.Errorf("direction = %q", rec.Direction)
	}
}

func TestPullUpdatedEvent(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	start := time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 5, 11, 0, 0, 0, time.UTC)
	evt := Event{ID: "evt-1", Summary: "Team Standup", StartTime: start, EndTime: end}

	mock := newMockCalendarAPI([]Event{evt})
	syncer := NewSyncer(mock, repo, vaultDir, tmplEngine, nil, time.Hour, 14*24*time.Hour)

	// First pull creates the note
	syncer.pull(context.Background())

	// Update the event
	newStart := start.Add(2 * time.Hour)
	newEnd := end.Add(2 * time.Hour)
	mock.events = []Event{
		{ID: "evt-1", Summary: "Team Standup Updated", StartTime: newStart, EndTime: newEnd},
	}

	modified, err := syncer.pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !modified {
		t.Error("expected modified=true for updated event")
	}

	// Read updated note
	notePath := filepath.Join(vaultDir, "2. Next Actions", "@calendar", "Team Standup.md")
	note, err := vault.ReadNote(notePath)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(note.Content, "Team Standup Updated") {
		t.Errorf("content not updated: %s", note.Content)
	}
}

func TestPullNoChange(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	start := time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 5, 11, 0, 0, 0, time.UTC)
	evt := Event{ID: "evt-1", Summary: "Team Standup", StartTime: start, EndTime: end}

	mock := newMockCalendarAPI([]Event{evt})
	syncer := NewSyncer(mock, repo, vaultDir, tmplEngine, nil, time.Hour, 14*24*time.Hour)

	// First pull
	syncer.pull(context.Background())

	// Second pull with same data
	modified, err := syncer.pull(context.Background())
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if modified {
		t.Error("expected modified=false when nothing changed")
	}
}

func TestPushNewItem(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	// Create a vault note with due_date
	notePath := filepath.Join(vaultDir, "2. Next Actions", "review-proposal.md")
	note := &vault.Note{
		Path: notePath,
		Frontmatter: map[string]interface{}{
			"created":  "2026-01-30",
			"status":   "next",
			"context":  "@computer",
			"due_date": "2026-02-10",
		},
		Content: "\n# Review Proposal\n",
	}
	vault.WriteNote(note)

	mock := newMockCalendarAPI(nil)
	syncer := NewSyncer(mock, repo, vaultDir, tmplEngine, nil, time.Hour, 14*24*time.Hour)

	modified, err := syncer.push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if !modified {
		t.Error("expected modified=true for new push")
	}

	// Verify event was created
	if len(mock.createdIDs) != 1 {
		t.Fatalf("expected 1 created event, got %d", len(mock.createdIDs))
	}

	// Verify DB record
	relPath := filepath.Join("2. Next Actions", "review-proposal.md")
	rec, _ := repo.GetCalendarSyncByVaultPath(relPath)
	if rec == nil {
		t.Fatal("expected sync record")
	}
	if rec.Direction != "push" {
		t.Errorf("direction = %q", rec.Direction)
	}
}

func TestPushSkipsCalendarItems(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	// Create a note that came from calendar pull (has calendar_id)
	calDir := filepath.Join(vaultDir, "2. Next Actions", "@calendar")
	os.MkdirAll(calDir, 0755)

	notePath := filepath.Join(calDir, "Team Standup.md")
	note := &vault.Note{
		Path: notePath,
		Frontmatter: map[string]interface{}{
			"created":     "2026-01-30",
			"status":      "scheduled",
			"context":     "@calendar",
			"calendar_id": "evt-1",
			"due_date":    "2026-02-05",
		},
		Content: "\n# Team Standup\n",
	}
	vault.WriteNote(note)

	mock := newMockCalendarAPI(nil)
	syncer := NewSyncer(mock, repo, vaultDir, tmplEngine, nil, time.Hour, 14*24*time.Hour)

	modified, _ := syncer.push(context.Background())
	if modified {
		t.Error("expected modified=false; should skip calendar items")
	}
	if len(mock.createdIDs) != 0 {
		t.Errorf("expected 0 created events, got %d", len(mock.createdIDs))
	}
}

func TestPushUpdatedItem(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	// Create a vault note
	notePath := filepath.Join(vaultDir, "2. Next Actions", "review-proposal.md")
	note := &vault.Note{
		Path: notePath,
		Frontmatter: map[string]interface{}{
			"created":  "2026-01-30",
			"status":   "next",
			"context":  "@computer",
			"due_date": "2026-02-10",
		},
		Content: "\n# Review Proposal\n",
	}
	vault.WriteNote(note)

	mock := newMockCalendarAPI(nil)
	syncer := NewSyncer(mock, repo, vaultDir, tmplEngine, nil, time.Hour, 14*24*time.Hour)

	// First push creates event
	syncer.push(context.Background())

	// Update due_date in the note
	note.Frontmatter = map[string]interface{}{
		"created":  "2026-01-30",
		"status":   "next",
		"context":  "@computer",
		"due_date": "2026-02-15",
	}
	vault.WriteNote(note)

	modified, err := syncer.push(context.Background())
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if !modified {
		t.Error("expected modified=true for updated push")
	}
	if len(mock.updatedCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(mock.updatedCalls))
	}
}
