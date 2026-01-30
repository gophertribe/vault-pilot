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
