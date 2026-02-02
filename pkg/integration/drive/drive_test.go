package drive

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// mockDriveAPI is a test double for DriveAPI.
type mockDriveAPI struct {
	files        []FileInfo
	uploadedIDs  map[string]string // localPath -> id
	updatedFiles map[string]bool   // fileID -> true
	downloads    map[string]string // fileID -> content
	nextID       int
}

func newMockDriveAPI() *mockDriveAPI {
	return &mockDriveAPI{
		uploadedIDs:  make(map[string]string),
		updatedFiles: make(map[string]bool),
		downloads:    make(map[string]string),
		nextID:       100,
	}
}

func (m *mockDriveAPI) ListFiles(_ context.Context) ([]FileInfo, error) {
	return m.files, nil
}

func (m *mockDriveAPI) UploadFile(_ context.Context, localPath, fileName, existingFileID string) (string, error) {
	if existingFileID != "" {
		m.updatedFiles[existingFileID] = true
		return existingFileID, nil
	}
	m.nextID++
	id := "drv-" + fileName
	m.uploadedIDs[localPath] = id
	return id, nil
}

func (m *mockDriveAPI) DownloadFile(_ context.Context, fileID string) (io.ReadCloser, error) {
	content := m.downloads[fileID]
	return io.NopCloser(strings.NewReader(content)), nil
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

	dirs := []string{
		filepath.Join(vaultDir, "0. GTD System", "Templates"),
		filepath.Join(vaultDir, "1. Inbox"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	tmplContent := "---\ncreated: {{date:YYYY-MM-DD}}\nstatus: inbox\n---\n# {{title}}\nBrief description of the item\n"
	os.WriteFile(filepath.Join(vaultDir, "0. GTD System", "Templates", "Inbox Item Template.md"), []byte(tmplContent), 0644)

	tmplEngine := vault.NewTemplateEngine(filepath.Join(vaultDir, "0. GTD System", "Templates"))
	return vaultDir, tmplEngine
}

// --- Backup tests ---

func TestBackupNewFile(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, _ := setupVault(t)

	// Create a .md file in the vault
	notePath := filepath.Join(vaultDir, "1. Inbox", "test-note.md")
	os.WriteFile(notePath, []byte("# Test"), 0644)

	mock := newMockDriveAPI()
	backup := NewBackup(mock, repo, vaultDir, time.Hour)

	if err := backup.backupOnce(); err != nil {
		t.Fatalf("backup: %v", err)
	}

	if len(mock.uploadedIDs) == 0 {
		t.Fatal("expected at least 1 upload")
	}

	// Verify DB record
	relPath := filepath.Join("1. Inbox", "test-note.md")
	rec, _ := repo.GetDriveSyncByLocalPath(relPath)
	if rec == nil {
		t.Fatal("expected sync record")
	}
	if rec.Direction != "upload" {
		t.Errorf("direction = %q", rec.Direction)
	}
}

func TestBackupUnmodifiedFile(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, _ := setupVault(t)

	notePath := filepath.Join(vaultDir, "1. Inbox", "test-note.md")
	os.WriteFile(notePath, []byte("# Test"), 0644)

	mock := newMockDriveAPI()
	backup := NewBackup(mock, repo, vaultDir, time.Hour)

	// First backup
	backup.backupOnce()
	uploadCount := len(mock.uploadedIDs)

	// Second backup — file not modified
	backup.backupOnce()

	if len(mock.updatedFiles) != 0 {
		t.Errorf("expected 0 updates for unmodified file, got %d", len(mock.updatedFiles))
	}
	if len(mock.uploadedIDs) != uploadCount {
		t.Errorf("expected no new uploads")
	}
}

func TestBackupModifiedFile(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, _ := setupVault(t)

	notePath := filepath.Join(vaultDir, "1. Inbox", "test-note.md")
	os.WriteFile(notePath, []byte("# Test"), 0644)

	mock := newMockDriveAPI()
	backup := NewBackup(mock, repo, vaultDir, time.Hour)

	// First backup
	backup.backupOnce()

	// Modify the file (change mod time to be after the recorded sync time)
	time.Sleep(time.Second) // ensure mod time changes
	os.WriteFile(notePath, []byte("# Updated"), 0644)

	// Second backup
	backup.backupOnce()

	if len(mock.updatedFiles) == 0 {
		t.Error("expected at least 1 update for modified file")
	}
}

func TestBackupSkipsHiddenDirs(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, _ := setupVault(t)

	// Create a file inside .git
	gitDir := filepath.Join(vaultDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config.md"), []byte("# config"), 0644)

	mock := newMockDriveAPI()
	backup := NewBackup(mock, repo, vaultDir, time.Hour)

	backup.backupOnce()

	// .git/config.md should not be uploaded
	for path := range mock.uploadedIDs {
		if strings.Contains(path, ".git") {
			t.Errorf("should not upload .git files, but uploaded %s", path)
		}
	}
}

// --- Watcher tests ---

func TestWatchNewFile(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	mock := newMockDriveAPI()
	mock.files = []FileInfo{
		{ID: "drv-1", Name: "meeting-notes.txt", MimeType: "text/plain"},
	}
	mock.downloads["drv-1"] = "Important meeting notes content"

	watcher := NewWatcher(mock, repo, vaultDir, tmplEngine, nil, time.Hour)

	if err := watcher.watchOnce(); err != nil {
		t.Fatalf("watch: %v", err)
	}

	// Verify inbox item was created
	inboxPath := filepath.Join(vaultDir, "1. Inbox", "meeting-notes.md")
	if _, err := os.Stat(inboxPath); os.IsNotExist(err) {
		t.Fatalf("expected inbox item at %s", inboxPath)
	}

	content, _ := os.ReadFile(inboxPath)
	if !strings.Contains(string(content), "Important meeting notes content") {
		t.Errorf("content missing expected text: %s", string(content))
	}

	// Verify DB record
	rec, _ := repo.GetDriveWatchByFileID("drv-1")
	if rec == nil {
		t.Fatal("expected watch record")
	}
	if rec.FileName != "meeting-notes.txt" {
		t.Errorf("file name = %q", rec.FileName)
	}
}

func TestWatchAlreadyProcessed(t *testing.T) {
	repo := setupTestDB(t)
	vaultDir, tmplEngine := setupVault(t)

	mock := newMockDriveAPI()
	mock.files = []FileInfo{
		{ID: "drv-1", Name: "meeting-notes.txt", MimeType: "text/plain"},
	}
	mock.downloads["drv-1"] = "Some content"

	watcher := NewWatcher(mock, repo, vaultDir, tmplEngine, nil, time.Hour)

	// First watch
	watcher.watchOnce()

	// Remove the created file to test that it's not recreated
	inboxPath := filepath.Join(vaultDir, "1. Inbox", "meeting-notes.md")
	os.Remove(inboxPath)

	// Second watch — should skip already-processed file
	watcher.watchOnce()

	if _, err := os.Stat(inboxPath); !os.IsNotExist(err) {
		t.Error("expected file to not be recreated for already-processed drive file")
	}
}
