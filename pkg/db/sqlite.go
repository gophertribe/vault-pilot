package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the sql.DB connection
type DB struct {
	*sql.DB
}

// NewDB creates a new SQLite database connection
func NewDB(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.DB.Close()
}

// InitSchema initializes the database schema
func (d *DB) InitSchema() error {
	// Basic schema for tracking reviews and jobs
	// In a real app, we'd use a migration tool like golang-migrate
	// For this MVP, we'll execute a simple CREATE TABLE string

	schema := `
	CREATE TABLE IF NOT EXISTS reviews (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		week_of TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'draft'
	);

	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		result TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS calendar_sync (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id TEXT NOT NULL UNIQUE,
		vault_path TEXT NOT NULL,
		sync_key TEXT NOT NULL,
		direction TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS drive_sync (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		drive_file_id TEXT NOT NULL UNIQUE,
		local_path TEXT NOT NULL,
		last_synced_at DATETIME NOT NULL,
		direction TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS drive_watch (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		drive_file_id TEXT NOT NULL UNIQUE,
		file_name TEXT NOT NULL,
		processed_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS automations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		action_type TEXT NOT NULL,
		schedule_kind TEXT NOT NULL,
		schedule_expr TEXT NOT NULL,
		timezone TEXT NOT NULL DEFAULT 'UTC',
		payload_json TEXT NOT NULL DEFAULT '{}',
		enabled INTEGER NOT NULL DEFAULT 1,
		next_run_at DATETIME,
		last_run_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS automation_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		automation_id INTEGER NOT NULL,
		scheduled_at DATETIME NOT NULL,
		started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME,
		status TEXT NOT NULL,
		error TEXT,
		output TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (automation_id) REFERENCES automations(id) ON DELETE CASCADE
	);
	`

	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	return nil
}
