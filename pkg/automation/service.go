package automation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/db"
)

// ActionFunc executes one automation.
type ActionFunc func(ctx context.Context, def db.AutomationDefinition) (string, error)

// Service runs persisted automations.
type Service struct {
	repo         *db.Repository
	pollInterval time.Duration
	claimLimit   int

	mu      sync.RWMutex
	actions map[string]ActionFunc

	stop chan struct{}
	wg   sync.WaitGroup
}

// NewService creates a new automation scheduler service.
func NewService(repo *db.Repository, pollInterval time.Duration, claimLimit int) *Service {
	if pollInterval <= 0 {
		pollInterval = 15 * time.Second
	}
	if claimLimit <= 0 {
		claimLimit = 10
	}
	return &Service{
		repo:         repo,
		pollInterval: pollInterval,
		claimLimit:   claimLimit,
		actions:      make(map[string]ActionFunc),
		stop:         make(chan struct{}),
	}
}

// RegisterAction registers a runnable automation action.
func (s *Service) RegisterAction(name string, fn ActionFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[name] = fn
}

// Start begins the polling loop.
func (s *Service) Start() {
	s.wg.Add(1)
	go s.loop()
}

// Stop stops the polling loop and waits for shutdown.
func (s *Service) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *Service) loop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Run one immediate tick on startup.
	s.runOnce(context.Background())

	for {
		select {
		case <-ticker.C:
			s.runOnce(context.Background())
		case <-s.stop:
			return
		}
	}
}

func (s *Service) runOnce(ctx context.Context) {
	now := time.Now().UTC()
	defs, err := s.repo.ClaimDueAutomations(now, s.claimLimit)
	if err != nil {
		log.Printf("automation: failed to claim due definitions: %v", err)
		return
	}
	for _, def := range defs {
		s.execute(ctx, def, now)
	}
}

func (s *Service) execute(ctx context.Context, def db.AutomationDefinition, now time.Time) {
	runID, err := s.repo.InsertAutomationRun(def.ID, now)
	if err != nil {
		log.Printf("automation: failed to create run for id=%d: %v", def.ID, err)
		return
	}

	s.mu.RLock()
	action := s.actions[def.ActionType]
	s.mu.RUnlock()

	status := "success"
	runErr := ""
	output := ""
	if action == nil {
		status = "failed"
		runErr = fmt.Sprintf("unknown action_type: %s", def.ActionType)
	} else {
		result, execErr := action(ctx, def)
		output = result
		if execErr != nil {
			status = "failed"
			runErr = execErr.Error()
		}
	}

	nextRun, nextErr := NextRun(def.ScheduleKind, def.ScheduleExpr, def.Timezone, now)
	if nextErr != nil {
		status = "failed"
		if runErr == "" {
			runErr = nextErr.Error()
		} else {
			runErr = runErr + "; next run calc failed: " + nextErr.Error()
		}
		nextRun = nil
	}

	enabled := def.Enabled
	if def.ScheduleKind == "oneshot" {
		enabled = false
	}

	if err := s.repo.CompleteAutomationRun(runID, def.ID, status, runErr, output, time.Now().UTC(), enabled, now, nextRun); err != nil {
		log.Printf("automation: failed to complete run=%d id=%d: %v", runID, def.ID, err)
	}
}
