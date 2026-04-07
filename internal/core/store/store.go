package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mklimuk/vault-pilot/internal/core/events"
	"github.com/mklimuk/vault-pilot/internal/core/jobs"
	"github.com/mklimuk/vault-pilot/internal/core/scheduler"
)

type JobStore interface {
	Enqueue(ctx context.Context, job jobs.Job) error
	Dequeue(ctx context.Context) (*jobs.Job, error)
	UpdateStatus(ctx context.Context, jobID string, status jobs.Status, result json.RawMessage, errMsg string) error
	GetByID(ctx context.Context, jobID string) (*jobs.Job, error)
}

type ScheduleStore interface {
	Upsert(ctx context.Context, spec scheduler.ScheduleSpec) error
	GetByID(ctx context.Context, id string) (*scheduler.ScheduleSpec, error)
	ListEnabled(ctx context.Context) ([]scheduler.ScheduleSpec, error)
	UpdateLastRun(ctx context.Context, id string, lastRun time.Time, nextRun *time.Time) error
}

type InMemoryEventStore struct {
	mu          sync.RWMutex
	events      map[string]events.Event
	processed   map[string]bool
	correlation map[string][]string
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events:      make(map[string]events.Event),
		processed:   make(map[string]bool),
		correlation: make(map[string][]string),
	}
}

func (s *InMemoryEventStore) Append(ctx context.Context, evt events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[evt.ID] = evt
	s.correlation[evt.CorrelationID] = append(s.correlation[evt.CorrelationID], evt.ID)
	return nil
}

func (s *InMemoryEventStore) GetByID(ctx context.Context, id string) (*events.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	evt, ok := s.events[id]
	if !ok {
		return nil, nil
	}
	return &evt, nil
}

func (s *InMemoryEventStore) GetByCorrelationID(ctx context.Context, correlationID string) ([]events.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.correlation[correlationID]
	result := make([]events.Event, 0, len(ids))
	for _, id := range ids {
		if evt, ok := s.events[id]; ok {
			result = append(result, evt)
		}
	}
	return result, nil
}

func (s *InMemoryEventStore) MarkProcessed(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processed[id] = true
	return nil
}

func (s *InMemoryEventStore) IsProcessed(ctx context.Context, id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.processed[id], nil
}

type InMemoryJobStore struct {
	mu    sync.Mutex
	jobs  map[string]jobs.Job
	queue []string
}

func NewInMemoryJobStore() *InMemoryJobStore {
	return &InMemoryJobStore{
		jobs:  make(map[string]jobs.Job),
		queue: make([]string, 0),
	}
}

func (s *InMemoryJobStore) Enqueue(ctx context.Context, job jobs.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.Status = jobs.StatusPending
	s.jobs[job.ID] = job
	s.queue = append(s.queue, job.ID)
	return nil
}

func (s *InMemoryJobStore) Dequeue(ctx context.Context) (*jobs.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.queue) > 0 {
		id := s.queue[0]
		s.queue = s.queue[1:]
		job, ok := s.jobs[id]
		if !ok {
			continue
		}
		if job.Status == jobs.StatusPending {
			now := time.Now()
			job.Status = jobs.StatusRunning
			job.StartedAt = &now
			job.Attempts++
			s.jobs[id] = job
			return &job, nil
		}
	}
	return nil, nil
}

func (s *InMemoryJobStore) UpdateStatus(ctx context.Context, jobID string, status jobs.Status, result json.RawMessage, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}
	now := time.Now()
	job.Status = status
	job.Result = result
	job.Error = errMsg
	if status == jobs.StatusDone || status == jobs.StatusFailed || status == jobs.StatusCanceled {
		job.CompletedAt = &now
	}
	s.jobs[jobID] = job
	if status == jobs.StatusPending {
		s.queue = append(s.queue, jobID)
	}
	return nil
}

func (s *InMemoryJobStore) GetByID(ctx context.Context, jobID string) (*jobs.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return nil, nil
	}
	return &job, nil
}

type InMemoryScheduleStore struct {
	mu    sync.RWMutex
	items map[string]scheduler.ScheduleSpec
}

func NewInMemoryScheduleStore() *InMemoryScheduleStore {
	return &InMemoryScheduleStore{
		items: make(map[string]scheduler.ScheduleSpec),
	}
}

func (s *InMemoryScheduleStore) Upsert(ctx context.Context, spec scheduler.ScheduleSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[spec.ID] = spec
	return nil
}

func (s *InMemoryScheduleStore) GetByID(ctx context.Context, id string) (*scheduler.ScheduleSpec, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	spec, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	return &spec, nil
}

func (s *InMemoryScheduleStore) ListEnabled(ctx context.Context) ([]scheduler.ScheduleSpec, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []scheduler.ScheduleSpec
	for _, spec := range s.items {
		if spec.Enabled {
			result = append(result, spec)
		}
	}
	return result, nil
}

func (s *InMemoryScheduleStore) UpdateLastRun(ctx context.Context, id string, lastRun time.Time, nextRun *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, ok := s.items[id]
	if !ok {
		return fmt.Errorf("schedule %s not found", id)
	}
	_ = lastRun
	_ = nextRun
	s.items[id] = spec
	return nil
}

type SQLiteEventStore struct {
	db *sql.DB
}

func NewSQLiteEventStore(db *sql.DB) *SQLiteEventStore {
	return &SQLiteEventStore{db: db}
}

func (s *SQLiteEventStore) InitSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			correlation_id TEXT NOT NULL,
			source TEXT NOT NULL,
			type TEXT NOT NULL,
			occurred_at TIMESTAMP NOT NULL,
			payload TEXT,
			metadata TEXT,
			processed INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_events_correlation_id ON events(correlation_id);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
		CREATE INDEX IF NOT EXISTS idx_events_processed ON events(processed);
	`)
	return err
}

func (s *SQLiteEventStore) Append(ctx context.Context, evt events.Event) error {
	var metadata []byte
	if evt.Metadata != nil {
		metadata, _ = json.Marshal(evt.Metadata)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO events (id, correlation_id, source, type, occurred_at, payload, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, evt.ID, evt.CorrelationID, evt.Source, evt.Type, evt.OccurredAt, evt.Payload, metadata)
	return err
}

func (s *SQLiteEventStore) GetByID(ctx context.Context, id string) (*events.Event, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, correlation_id, source, type, occurred_at, payload, metadata
		FROM events WHERE id = ?
	`, id)
	return scanEvent(row)
}

func (s *SQLiteEventStore) GetByCorrelationID(ctx context.Context, correlationID string) ([]events.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, correlation_id, source, type, occurred_at, payload, metadata
		FROM events WHERE correlation_id = ? ORDER BY occurred_at
	`, correlationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []events.Event
	for rows.Next() {
		evt, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *evt)
	}
	return result, rows.Err()
}

func (s *SQLiteEventStore) MarkProcessed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE events SET processed = 1 WHERE id = ?`, id)
	return err
}

func (s *SQLiteEventStore) IsProcessed(ctx context.Context, id string) (bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT processed FROM events WHERE id = ?`, id)
	var processed int
	err := row.Scan(&processed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return processed == 1, nil
}

func scanEvent(scanner interface{ Scan(...interface{}) error }) (*events.Event, error) {
	var evt events.Event
	var metadata []byte
	err := scanner.Scan(&evt.ID, &evt.CorrelationID, &evt.Source, &evt.Type, &evt.OccurredAt, &evt.Payload, &metadata)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(metadata) > 0 {
		json.Unmarshal(metadata, &evt.Metadata)
	}
	return &evt, nil
}

type SQLiteJobStore struct {
	db *sql.DB
}

func NewSQLiteJobStore(db *sql.DB) *SQLiteJobStore {
	return &SQLiteJobStore{db: db}
}

func (s *SQLiteJobStore) InitSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			correlation_id TEXT,
			plugin_id TEXT NOT NULL,
			type TEXT NOT NULL,
			payload TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER DEFAULT 0,
			max_attempts INTEGER DEFAULT 3,
			created_at TIMESTAMP NOT NULL,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			error TEXT,
			result TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
		CREATE INDEX IF NOT EXISTS idx_jobs_plugin_id ON jobs(plugin_id);
	`)
	return err
}

func (s *SQLiteJobStore) Enqueue(ctx context.Context, job jobs.Job) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (id, correlation_id, plugin_id, type, payload, status, attempts, max_attempts, created_at)
		VALUES (?, ?, ?, ?, ?, 'pending', 0, ?, ?)
	`, job.ID, job.CorrelationID, job.PluginID, job.Type, job.Payload, job.MaxAttempts, job.CreatedAt)
	return err
}

func (s *SQLiteJobStore) Dequeue(ctx context.Context) (*jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
		SELECT id, correlation_id, plugin_id, type, payload, status, attempts, max_attempts, created_at, started_at, completed_at, error, result
		FROM jobs WHERE status = 'pending' ORDER BY created_at LIMIT 1
	`)
	job, err := scanJob(row)
	if err != nil {
		return nil, nil
	}
	if job == nil {
		return nil, nil
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE jobs SET status = 'running', started_at = ?, attempts = attempts + 1 WHERE id = ?
	`, now, job.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	job.Status = jobs.StatusRunning
	job.StartedAt = &now
	job.Attempts++
	return job, nil
}

func (s *SQLiteJobStore) UpdateStatus(ctx context.Context, jobID string, status jobs.Status, result json.RawMessage, errMsg string) error {
	now := time.Now()
	var completedAt interface{}
	if status == jobs.StatusDone || status == jobs.StatusFailed || status == jobs.StatusCanceled {
		completedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE jobs SET status = ?, result = ?, error = ?, completed_at = ? WHERE id = ?
	`, status, result, errMsg, completedAt, jobID)
	return err
}

func (s *SQLiteJobStore) GetByID(ctx context.Context, jobID string) (*jobs.Job, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, correlation_id, plugin_id, type, payload, status, attempts, max_attempts, created_at, started_at, completed_at, error, result
		FROM jobs WHERE id = ?
	`, jobID)
	return scanJob(row)
}

func scanJob(scanner interface{ Scan(...interface{}) error }) (*jobs.Job, error) {
	var job jobs.Job
	var startedAt, completedAt sql.NullTime
	err := scanner.Scan(
		&job.ID, &job.CorrelationID, &job.PluginID, &job.Type, &job.Payload,
		&job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt,
		&startedAt, &completedAt, &job.Error, &job.Result,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	return &job, nil
}
