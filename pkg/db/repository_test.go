package db

import (
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()
	database, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := database.InitSchema(); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}
	return NewRepository(database)
}

func TestCalendarSync(t *testing.T) {
	repo := setupTestDB(t)

	// Insert
	if err := repo.InsertCalendarSync("evt-1", "2. Next Actions/@calendar/Meeting.md", "key-1", "pull"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Get by event ID
	rec, err := repo.GetCalendarSyncByEventID("evt-1")
	if err != nil {
		t.Fatalf("get by event ID: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.VaultPath != "2. Next Actions/@calendar/Meeting.md" {
		t.Errorf("vault path = %q", rec.VaultPath)
	}
	if rec.SyncKey != "key-1" {
		t.Errorf("sync key = %q", rec.SyncKey)
	}
	if rec.Direction != "pull" {
		t.Errorf("direction = %q", rec.Direction)
	}

	// Get by vault path
	rec2, err := repo.GetCalendarSyncByVaultPath("2. Next Actions/@calendar/Meeting.md")
	if err != nil {
		t.Fatalf("get by vault path: %v", err)
	}
	if rec2 == nil || rec2.EventID != "evt-1" {
		t.Fatalf("expected event ID evt-1, got %+v", rec2)
	}

	// Update
	if err := repo.UpdateCalendarSync("evt-1", "key-2"); err != nil {
		t.Fatalf("update: %v", err)
	}
	rec3, _ := repo.GetCalendarSyncByEventID("evt-1")
	if rec3.SyncKey != "key-2" {
		t.Errorf("expected updated sync key key-2, got %q", rec3.SyncKey)
	}

	// Not found
	rec4, err := repo.GetCalendarSyncByEventID("nonexistent")
	if err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
	if rec4 != nil {
		t.Errorf("expected nil, got %+v", rec4)
	}
}

func TestDriveSync(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	// Insert
	if err := repo.InsertDriveSync("drv-1", "/vault/note.md", now, "upload"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Get by local path
	rec, err := repo.GetDriveSyncByLocalPath("/vault/note.md")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.DriveFileID != "drv-1" {
		t.Errorf("drive file ID = %q", rec.DriveFileID)
	}
	if rec.Direction != "upload" {
		t.Errorf("direction = %q", rec.Direction)
	}

	// Update
	later := now.Add(time.Hour)
	if err := repo.UpdateDriveSync("drv-1", later); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Not found
	rec2, err := repo.GetDriveSyncByLocalPath("/nonexistent")
	if err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
	if rec2 != nil {
		t.Errorf("expected nil, got %+v", rec2)
	}
}

func TestDriveWatch(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	// Insert
	if err := repo.InsertDriveWatch("drv-w-1", "document.pdf", now); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Get
	rec, err := repo.GetDriveWatchByFileID("drv-w-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.FileName != "document.pdf" {
		t.Errorf("file name = %q", rec.FileName)
	}

	// Not found
	rec2, err := repo.GetDriveWatchByFileID("nonexistent")
	if err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
	if rec2 != nil {
		t.Errorf("expected nil, got %+v", rec2)
	}
}

func TestReviewLog(t *testing.T) {
	repo := setupTestDB(t)

	// Empty at first
	rev, err := repo.GetLatestReview()
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if rev != nil {
		t.Errorf("expected nil, got %+v", rev)
	}

	// Insert
	if err := repo.LogReview("2026-W05"); err != nil {
		t.Fatalf("log review: %v", err)
	}

	rev, err = repo.GetLatestReview()
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if rev == nil || rev.WeekOf != "2026-W05" {
		t.Errorf("expected week 2026-W05, got %+v", rev)
	}
}
