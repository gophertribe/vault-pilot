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

// AutomationDefinition represents a scheduled automation configuration.
type AutomationDefinition struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	ActionType   string     `json:"action_type"`
	ScheduleKind string     `json:"schedule_kind"`
	ScheduleExpr string     `json:"schedule_expr"`
	Timezone     string     `json:"timezone"`
	PayloadJSON  string     `json:"payload_json"`
	Enabled      bool       `json:"enabled"`
	NextRunAt    *time.Time `json:"next_run_at,omitempty"`
	LastRunAt    *time.Time `json:"last_run_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AutomationRun represents an execution attempt of an automation definition.
type AutomationRun struct {
	ID           int64      `json:"id"`
	AutomationID int64      `json:"automation_id"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Status       string     `json:"status"`
	Error        string     `json:"error,omitempty"`
	Output       string     `json:"output,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CreateAutomation inserts a new automation definition.
func (r *Repository) CreateAutomation(def *AutomationDefinition) (int64, error) {
	query := `
		INSERT INTO automations
			(name, action_type, schedule_kind, schedule_expr, timezone, payload_json, enabled, next_run_at, last_run_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var nextRun interface{}
	if def.NextRunAt != nil {
		nextRun = *def.NextRunAt
	}
	var lastRun interface{}
	if def.LastRunAt != nil {
		lastRun = *def.LastRunAt
	}
	result, err := r.db.Exec(
		query,
		def.Name,
		def.ActionType,
		def.ScheduleKind,
		def.ScheduleExpr,
		def.Timezone,
		def.PayloadJSON,
		boolToInt(def.Enabled),
		nextRun,
		lastRun,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create automation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to read automation id: %w", err)
	}
	return id, nil
}

// ListAutomations returns all automation definitions.
func (r *Repository) ListAutomations() ([]AutomationDefinition, error) {
	query := `
		SELECT id, name, action_type, schedule_kind, schedule_expr, timezone, payload_json,
		       enabled, next_run_at, last_run_at, created_at, updated_at
		FROM automations
		ORDER BY created_at DESC, id DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list automations: %w", err)
	}
	defer rows.Close()

	var out []AutomationDefinition
	for rows.Next() {
		def, err := scanAutomationDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *def)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list automations rows: %w", err)
	}
	return out, nil
}

// GetAutomationByID returns a single automation definition by ID.
func (r *Repository) GetAutomationByID(id int64) (*AutomationDefinition, error) {
	query := `
		SELECT id, name, action_type, schedule_kind, schedule_expr, timezone, payload_json,
		       enabled, next_run_at, last_run_at, created_at, updated_at
		FROM automations
		WHERE id = ?
	`
	row := r.db.QueryRow(query, id)
	def, err := scanAutomationDefinition(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get automation: %w", err)
	}
	return def, nil
}

// UpdateAutomation updates an existing automation definition by ID.
func (r *Repository) UpdateAutomation(def *AutomationDefinition) error {
	query := `
		UPDATE automations
		SET name = ?, action_type = ?, schedule_kind = ?, schedule_expr = ?, timezone = ?,
		    payload_json = ?, enabled = ?, next_run_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	var nextRun interface{}
	if def.NextRunAt != nil {
		nextRun = *def.NextRunAt
	}
	_, err := r.db.Exec(
		query,
		def.Name,
		def.ActionType,
		def.ScheduleKind,
		def.ScheduleExpr,
		def.Timezone,
		def.PayloadJSON,
		boolToInt(def.Enabled),
		nextRun,
		def.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update automation: %w", err)
	}
	return nil
}

// TriggerAutomationNow sets the next_run_at to now for a definition.
func (r *Repository) TriggerAutomationNow(id int64, now time.Time) error {
	query := `UPDATE automations SET next_run_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to trigger automation: %w", err)
	}
	return nil
}

// ClaimDueAutomations atomically claims due tasks and returns claimed definitions.
func (r *Repository) ClaimDueAutomations(now time.Time, limit int) ([]AutomationDefinition, error) {
	query := `
		WITH due AS (
			SELECT id
			FROM automations
			WHERE enabled = 1
			  AND next_run_at IS NOT NULL
			  AND next_run_at <= ?
			ORDER BY next_run_at
			LIMIT ?
		)
		UPDATE automations
		SET next_run_at = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id IN (SELECT id FROM due)
		RETURNING id, name, action_type, schedule_kind, schedule_expr, timezone, payload_json,
		          enabled, next_run_at, last_run_at, created_at, updated_at
	`
	rows, err := r.db.Query(query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim due automations: %w", err)
	}
	defer rows.Close()

	var out []AutomationDefinition
	for rows.Next() {
		def, err := scanAutomationDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *def)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed reading claimed automations: %w", err)
	}
	return out, nil
}

// InsertAutomationRun inserts a running automation execution record.
func (r *Repository) InsertAutomationRun(automationID int64, scheduledAt time.Time) (int64, error) {
	query := `INSERT INTO automation_runs (automation_id, scheduled_at, status) VALUES (?, ?, 'running')`
	res, err := r.db.Exec(query, automationID, scheduledAt)
	if err != nil {
		return 0, fmt.Errorf("failed to insert automation run: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to read automation run id: %w", err)
	}
	return id, nil
}

// CompleteAutomationRun finalizes an execution and updates definition schedule state.
func (r *Repository) CompleteAutomationRun(runID int64, automationID int64, status, runErr, output string, finishedAt time.Time, enabled bool, lastRunAt time.Time, nextRunAt *time.Time) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin automation complete tx: %w", err)
	}
	defer tx.Rollback()

	runQuery := `
		UPDATE automation_runs
		SET status = ?, error = ?, output = ?, finished_at = ?
		WHERE id = ?
	`
	if _, err := tx.Exec(runQuery, status, runErr, output, finishedAt, runID); err != nil {
		return fmt.Errorf("failed to update automation run: %w", err)
	}

	defQuery := `
		UPDATE automations
		SET enabled = ?, last_run_at = ?, next_run_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	var next interface{}
	if nextRunAt != nil {
		next = *nextRunAt
	}
	if _, err := tx.Exec(defQuery, boolToInt(enabled), lastRunAt, next, automationID); err != nil {
		return fmt.Errorf("failed to update automation definition: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit automation completion: %w", err)
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type automationRowScanner interface {
	Scan(dest ...interface{}) error
}

func scanAutomationDefinition(scanner automationRowScanner) (*AutomationDefinition, error) {
	var def AutomationDefinition
	var enabled int
	var nextRun sql.NullTime
	var lastRun sql.NullTime
	if err := scanner.Scan(
		&def.ID,
		&def.Name,
		&def.ActionType,
		&def.ScheduleKind,
		&def.ScheduleExpr,
		&def.Timezone,
		&def.PayloadJSON,
		&enabled,
		&nextRun,
		&lastRun,
		&def.CreatedAt,
		&def.UpdatedAt,
	); err != nil {
		return nil, err
	}
	def.Enabled = enabled == 1
	if nextRun.Valid {
		t := nextRun.Time
		def.NextRunAt = &t
	}
	if lastRun.Valid {
		t := lastRun.Time
		def.LastRunAt = &t
	}
	return &def, nil
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
