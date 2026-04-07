package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mklimuk/vault-pilot/internal/core/ai"
	"github.com/mklimuk/vault-pilot/internal/core/events"
	"github.com/mklimuk/vault-pilot/internal/core/jobs"
	"github.com/mklimuk/vault-pilot/internal/core/scheduler"
	"github.com/mklimuk/vault-pilot/internal/core/store"
)

func TestScheduledEventTriggersJob(t *testing.T) {
	eventStore := store.NewInMemoryEventStore()
	bus := events.NewInMemoryBus(eventStore)

	jobStore := store.NewInMemoryJobStore()
	orchestrator := jobs.NewOrchestrator(jobs.DefaultOrchestratorConfig(), jobStore, bus)

	var receivedEvents []events.Event
	var eventsMu sync.Mutex
	bus.Subscribe(events.TypeJobCompleted, func(ctx context.Context, evt events.Event) error {
		eventsMu.Lock()
		receivedEvents = append(receivedEvents, evt)
		eventsMu.Unlock()
		return nil
	})

	var executedJobs []jobs.Job
	var jobsMu sync.Mutex
	orchestrator.RegisterHandler("test_task", func(ctx context.Context, job jobs.Job) (json.RawMessage, error) {
		jobsMu.Lock()
		executedJobs = append(executedJobs, job)
		jobsMu.Unlock()
		return json.RawMessage(`{"status": "success"}`), nil
	})

	scheduleStore := store.NewInMemoryScheduleStore()
	sched := scheduler.NewScheduler(scheduler.ServiceConfig{
		PollInterval: 50 * time.Millisecond,
	}, scheduleStore, bus)

	var scheduleRuns int
	var runsMu sync.Mutex
	err := sched.Add(scheduler.ScheduleSpec{
		ID:      "test-schedule",
		Kind:    scheduler.KindInterval,
		Expr:    "100ms",
		Enabled: true,
	}, func(ctx context.Context, spec scheduler.ScheduleSpec) error {
		runsMu.Lock()
		scheduleRuns++
		runsMu.Unlock()

		job := jobs.Job{
			ID:       fmt.Sprintf("job-%d", time.Now().UnixNano()),
			PluginID: "test-plugin",
			Type:     "test_task",
		}
		return orchestrator.Submit(ctx, job)
	})
	if err != nil {
		t.Fatalf("add schedule: %v", err)
	}

	if err := orchestrator.Start(); err != nil {
		t.Fatalf("start orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	if err := sched.Start(); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer sched.Stop()

	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for job completion. runs=%d, jobs=%d, events=%d",
				scheduleRuns, len(executedJobs), len(receivedEvents))
		case <-ticker.C:
			runsMu.Lock()
			runs := scheduleRuns
			runsMu.Unlock()

			jobsMu.Lock()
			jobCount := len(executedJobs)
			jobsMu.Unlock()

			eventsMu.Lock()
			evtCount := len(receivedEvents)
			eventsMu.Unlock()

			if runs > 0 && jobCount > 0 && evtCount > 0 {
				t.Logf("integration test passed: %d schedule runs, %d jobs executed, %d events received",
					runs, jobCount, evtCount)
				return
			}
		}
	}
}

func TestEventBusPublishSubscribe(t *testing.T) {
	ctx := context.Background()
	eventStore := store.NewInMemoryEventStore()
	bus := events.NewInMemoryBus(eventStore)

	var received []events.Event
	var mu sync.Mutex

	bus.Subscribe(events.TypeMailReceived, func(ctx context.Context, evt events.Event) error {
		mu.Lock()
		received = append(received, evt)
		mu.Unlock()
		return nil
	})

	evt := events.Event{
		CorrelationID: "test-correlation",
		Source:        "test",
		Type:          events.TypeMailReceived,
		Payload:       json.RawMessage(`{"subject": "Test Email"}`),
	}

	if err := bus.Publish(ctx, evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Type != events.TypeMailReceived {
		t.Fatalf("expected type %s, got %s", events.TypeMailReceived, received[0].Type)
	}
}

func TestJobOrchestratorRetries(t *testing.T) {
	ctx := context.Background()
	eventStore := store.NewInMemoryEventStore()
	bus := events.NewInMemoryBus(eventStore)
	jobStore := store.NewInMemoryJobStore()

	config := jobs.DefaultOrchestratorConfig()
	config.RetryDelay = 10 * time.Millisecond
	config.MaxRetryDelay = 50 * time.Millisecond

	orchestrator := jobs.NewOrchestrator(config, jobStore, bus)

	attempts := 0
	orchestrator.RegisterHandler("flaky_task", func(ctx context.Context, job jobs.Job) (json.RawMessage, error) {
		attempts++
		if attempts < 3 {
			return nil, fmt.Errorf("transient error")
		}
		return json.RawMessage(`{"status": "success"}`), nil
	})

	if err := orchestrator.Start(); err != nil {
		t.Fatalf("start orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	job := jobs.Job{
		ID:          "retry-job",
		PluginID:    "test-plugin",
		Type:        "flaky_task",
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
	}

	if err := orchestrator.Submit(ctx, job); err != nil {
		t.Fatalf("submit job: %v", err)
	}

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for job completion, attempts=%d", attempts)
		case <-ticker.C:
			stored, _ := jobStore.GetByID(ctx, job.ID)
			if stored != nil && stored.Status == jobs.StatusDone {
				if attempts != 3 {
					t.Fatalf("expected 3 attempts, got %d", attempts)
				}
				t.Logf("retry test passed: job completed after %d attempts", attempts)
				return
			}
		}
	}
}

func TestAIRuntimeBudget(t *testing.T) {
	ctx := context.Background()

	usageStore := ai.NewInMemoryUsageStore()
	budget := ai.NewInMemoryBudgetManager(1000)

	runtime := ai.NewRuntime(ai.DefaultRuntimeConfig())
	runtime.SetUsageStore(usageStore)
	runtime.SetBudgetManager(budget)

	mockProvider := &mockProvider{name: "mock-model"}
	runtime.RegisterProvider(ai.ModelClassFast, mockProvider)

	_, err := runtime.Execute(ctx, ai.PromptRequest{
		PluginID:   "test-plugin",
		TaskType:   "test",
		ModelClass: ai.ModelClassFast,
		Input:      "Hello",
	})
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}

	records, err := usageStore.GetUsageByPlugin(ctx, "test-plugin", time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 usage record, got %d", len(records))
	}
}

type mockProvider struct {
	name string
}

func (m *mockProvider) Generate(ctx context.Context, req ai.PromptRequest) (ai.PromptResult, error) {
	return ai.PromptResult{
		Output:     "mock response",
		ModelUsed:  m.name,
		TokensUsed: 100,
	}, nil
}

func (m *mockProvider) IsAvailable() bool { return true }
func (m *mockProvider) Name() string      { return m.name }
