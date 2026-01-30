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
	`

	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	return nil
}
