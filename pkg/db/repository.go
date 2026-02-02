package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Repository handles data access
type Repository struct {
	db *DB
}

// NewRepository creates a new Repository
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// ReviewLog represents a row in the reviews table
type ReviewLog struct {
	ID        int64
	WeekOf    string
	CreatedAt time.Time
	Status    string
}

// LogReview creates a new review log entry
func (r *Repository) LogReview(weekOf string) error {
	query := `INSERT INTO reviews (week_of, status) VALUES (?, 'draft')`
	_, err := r.db.Exec(query, weekOf)
	if err != nil {
		return fmt.Errorf("failed to log review: %w", err)
	}
	return nil
}

// GetLatestReview returns the most recent review log
func (r *Repository) GetLatestReview() (*ReviewLog, error) {
	query := `SELECT id, week_of, created_at, status FROM reviews ORDER BY created_at DESC LIMIT 1`
	row := r.db.QueryRow(query)

	var log ReviewLog
	err := row.Scan(&log.ID, &log.WeekOf, &log.CreatedAt, &log.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest review: %w", err)
	}
	return &log, nil
}

// --- Calendar sync ---

// CalendarSyncRecord represents a row in the calendar_sync table.
type CalendarSyncRecord struct {
	ID        int64
	EventID   string
	VaultPath string
	SyncKey   string
	Direction string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetCalendarSyncByEventID looks up a sync record by Google Calendar event ID.
func (r *Repository) GetCalendarSyncByEventID(eventID string) (*CalendarSyncRecord, error) {
	query := `SELECT id, event_id, vault_path, sync_key, direction, created_at, updated_at FROM calendar_sync WHERE event_id = ?`
	row := r.db.QueryRow(query, eventID)

	var rec CalendarSyncRecord
	err := row.Scan(&rec.ID, &rec.EventID, &rec.VaultPath, &rec.SyncKey, &rec.Direction, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get calendar sync by event ID: %w", err)
	}
	return &rec, nil
}

// GetCalendarSyncByVaultPath looks up a sync record by vault file path.
func (r *Repository) GetCalendarSyncByVaultPath(vaultPath string) (*CalendarSyncRecord, error) {
	query := `SELECT id, event_id, vault_path, sync_key, direction, created_at, updated_at FROM calendar_sync WHERE vault_path = ?`
	row := r.db.QueryRow(query, vaultPath)

	var rec CalendarSyncRecord
	err := row.Scan(&rec.ID, &rec.EventID, &rec.VaultPath, &rec.SyncKey, &rec.Direction, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get calendar sync by vault path: %w", err)
	}
	return &rec, nil
}

// InsertCalendarSync inserts a new calendar sync record.
func (r *Repository) InsertCalendarSync(eventID, vaultPath, syncKey, direction string) error {
	query := `INSERT INTO calendar_sync (event_id, vault_path, sync_key, direction) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, eventID, vaultPath, syncKey, direction)
	if err != nil {
		return fmt.Errorf("failed to insert calendar sync: %w", err)
	}
	return nil
}

// UpdateCalendarSync updates the sync_key and updated_at for an existing record.
func (r *Repository) UpdateCalendarSync(eventID, syncKey string) error {
	query := `UPDATE calendar_sync SET sync_key = ?, updated_at = CURRENT_TIMESTAMP WHERE event_id = ?`
	_, err := r.db.Exec(query, syncKey, eventID)
	if err != nil {
		return fmt.Errorf("failed to update calendar sync: %w", err)
	}
	return nil
}

// --- Drive sync (backup) ---

// DriveSyncRecord represents a row in the drive_sync table.
type DriveSyncRecord struct {
	ID           int64
	DriveFileID  string
	LocalPath    string
	LastSyncedAt time.Time
	Direction    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// GetDriveSyncByLocalPath looks up a drive sync record by local file path.
func (r *Repository) GetDriveSyncByLocalPath(localPath string) (*DriveSyncRecord, error) {
	query := `SELECT id, drive_file_id, local_path, last_synced_at, direction, created_at, updated_at FROM drive_sync WHERE local_path = ?`
	row := r.db.QueryRow(query, localPath)

	var rec DriveSyncRecord
	err := row.Scan(&rec.ID, &rec.DriveFileID, &rec.LocalPath, &rec.LastSyncedAt, &rec.Direction, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get drive sync by local path: %w", err)
	}
	return &rec, nil
}

// InsertDriveSync inserts a new drive sync record.
func (r *Repository) InsertDriveSync(driveFileID, localPath string, lastSyncedAt time.Time, direction string) error {
	query := `INSERT INTO drive_sync (drive_file_id, local_path, last_synced_at, direction) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, driveFileID, localPath, lastSyncedAt, direction)
	if err != nil {
		return fmt.Errorf("failed to insert drive sync: %w", err)
	}
	return nil
}

// UpdateDriveSync updates the last_synced_at and updated_at for an existing record.
func (r *Repository) UpdateDriveSync(driveFileID string, lastSyncedAt time.Time) error {
	query := `UPDATE drive_sync SET last_synced_at = ?, updated_at = CURRENT_TIMESTAMP WHERE drive_file_id = ?`
	_, err := r.db.Exec(query, lastSyncedAt, driveFileID)
	if err != nil {
		return fmt.Errorf("failed to update drive sync: %w", err)
	}
	return nil
}

// --- Drive watch ---

// DriveWatchRecord represents a row in the drive_watch table.
type DriveWatchRecord struct {
	ID          int64
	DriveFileID string
	FileName    string
	ProcessedAt time.Time
	CreatedAt   time.Time
}

// GetDriveWatchByFileID looks up a drive watch record by Drive file ID.
func (r *Repository) GetDriveWatchByFileID(driveFileID string) (*DriveWatchRecord, error) {
	query := `SELECT id, drive_file_id, file_name, processed_at, created_at FROM drive_watch WHERE drive_file_id = ?`
	row := r.db.QueryRow(query, driveFileID)

	var rec DriveWatchRecord
	err := row.Scan(&rec.ID, &rec.DriveFileID, &rec.FileName, &rec.ProcessedAt, &rec.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get drive watch by file ID: %w", err)
	}
	return &rec, nil
}

// InsertDriveWatch inserts a new drive watch record.
func (r *Repository) InsertDriveWatch(driveFileID, fileName string, processedAt time.Time) error {
	query := `INSERT INTO drive_watch (drive_file_id, file_name, processed_at) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, driveFileID, fileName, processedAt)
	if err != nil {
		return fmt.Errorf("failed to insert drive watch: %w", err)
	}
	return nil
}
