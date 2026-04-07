package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mklimuk/vault-pilot/internal/core/events"
)

type ServiceConfig struct {
	PollInterval time.Duration
}

func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		PollInterval: 1 * time.Second,
	}
}

type Store interface {
	Upsert(ctx context.Context, spec ScheduleSpec) error
	GetByID(ctx context.Context, id string) (*ScheduleSpec, error)
	ListEnabled(ctx context.Context) ([]ScheduleSpec, error)
	UpdateLastRun(ctx context.Context, id string, lastRun time.Time, nextRun *time.Time) error
}

type Scheduler struct {
	config    ServiceConfig
	store     Store
	bus       events.Bus
	schedules map[string]scheduledCallback
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
}

type scheduledCallback struct {
	spec     ScheduleSpec
	callback ScheduledCallback
	nextRun  time.Time
}

func NewScheduler(config ServiceConfig, store Store, bus events.Bus) Service {
	return &Scheduler{
		config:    config,
		store:     store,
		bus:       bus,
		schedules: make(map[string]scheduledCallback),
	}
}

func (s *Scheduler) Add(spec ScheduleSpec, callback ScheduledCallback) error {
	if spec.ID == "" {
		return fmt.Errorf("schedule ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	nextRun, err := s.calculateNextRun(spec, time.Now())
	if err != nil {
		return fmt.Errorf("calculate next run: %w", err)
	}

	s.schedules[spec.ID] = scheduledCallback{
		spec:     spec,
		callback: callback,
		nextRun:  nextRun,
	}

	if s.store != nil {
		if err := s.store.Upsert(context.Background(), spec); err != nil {
			return fmt.Errorf("persist schedule: %w", err)
		}
	}
	return nil
}

func (s *Scheduler) Remove(specID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.schedules, specID)
	return nil
}

func (s *Scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.runLoop()
	return nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
	return nil
}

func (s *Scheduler) runLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	s.mu.RLock()
	schedules := make([]scheduledCallback, 0, len(s.schedules))
	for _, sc := range s.schedules {
		schedules = append(schedules, sc)
	}
	s.mu.RUnlock()

	now := time.Now()
	for _, sc := range schedules {
		if !sc.spec.Enabled {
			continue
		}
		if now.Before(sc.nextRun) {
			continue
		}

		go s.execute(sc)
	}
}

func (s *Scheduler) execute(sc scheduledCallback) {
	ctx := s.ctx

	_ = sc.callback(ctx, sc.spec)

	nextRun, err := s.calculateNextRun(sc.spec, time.Now())
	if err != nil {
		nextRun = time.Now().Add(1 * time.Minute)
	}

	s.mu.Lock()
	if existing, ok := s.schedules[sc.spec.ID]; ok {
		existing.nextRun = nextRun
		s.schedules[sc.spec.ID] = existing
	}
	s.mu.Unlock()

	if s.store != nil {
		_ = s.store.UpdateLastRun(ctx, sc.spec.ID, time.Now(), &nextRun)
	}
}

func (s *Scheduler) calculateNextRun(spec ScheduleSpec, from time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(spec.Timezone)
	if err != nil {
		loc = time.UTC
	}
	from = from.In(loc)

	switch spec.Kind {
	case KindInterval:
		d, err := time.ParseDuration(spec.Expr)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse interval: %w", err)
		}
		return from.Add(d), nil

	case KindCron:
		return parseCronNext(spec.Expr, from, loc)

	default:
		return time.Time{}, fmt.Errorf("unknown schedule kind: %s", spec.Kind)
	}
}

func parseCronNext(expr string, from time.Time, loc *time.Location) (time.Time, error) {
	expr = "CRON_TZ=" + loc.String() + " " + expr
	_ = expr
	return from.Add(1 * time.Hour), nil
}
