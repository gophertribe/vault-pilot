package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mklimuk/vault-pilot/internal/core/events"
)

type OrchestratorConfig struct {
	WorkerCount   int
	PollInterval  time.Duration
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
}

func DefaultOrchestratorConfig() OrchestratorConfig {
	return OrchestratorConfig{
		WorkerCount:   4,
		PollInterval:  100 * time.Millisecond,
		RetryDelay:    1 * time.Second,
		MaxRetryDelay: 30 * time.Second,
	}
}

type HandlerRegistry interface {
	Register(jobType string, handler Handler)
	Get(jobType string) (Handler, bool)
}

type Orchestrator struct {
	config   OrchestratorConfig
	queue    Queue
	bus      events.Bus
	handlers map[string]Handler
	mu       sync.RWMutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
}

func NewOrchestrator(config OrchestratorConfig, queue Queue, bus events.Bus) *Orchestrator {
	return &Orchestrator{
		config:   config,
		queue:    queue,
		bus:      bus,
		handlers: make(map[string]Handler),
	}
}

func (o *Orchestrator) RegisterHandler(jobType string, handler Handler) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.handlers[jobType] = handler
}

func (o *Orchestrator) Submit(ctx context.Context, job Job) error {
	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.Status = StatusPending

	if err := o.queue.Enqueue(ctx, job); err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}

	if o.bus != nil {
		_ = o.bus.Publish(ctx, events.Event{
			CorrelationID: job.CorrelationID,
			Source:        "jobs.orchestrator",
			Type:          events.TypeJobQueued,
			Payload:       mustMarshal(job),
		})
	}
	return nil
}

func (o *Orchestrator) Start() error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator already running")
	}
	o.ctx, o.cancel = context.WithCancel(context.Background())
	o.running = true
	o.mu.Unlock()

	for i := 0; i < o.config.WorkerCount; i++ {
		o.wg.Add(1)
		go o.worker(i)
	}
	return nil
}

func (o *Orchestrator) Stop() error {
	o.mu.Lock()
	if !o.running {
		o.mu.Unlock()
		return nil
	}
	o.running = false
	o.cancel()
	o.mu.Unlock()

	o.wg.Wait()
	return nil
}

func (o *Orchestrator) worker(id int) {
	defer o.wg.Done()

	for {
		select {
		case <-o.ctx.Done():
			return
		default:
		}

		job, err := o.queue.Dequeue(o.ctx)
		if err != nil {
			time.Sleep(o.config.PollInterval)
			continue
		}
		if job == nil {
			time.Sleep(o.config.PollInterval)
			continue
		}

		o.executeJob(job)
	}
}

func (o *Orchestrator) executeJob(job *Job) {
	ctx := o.ctx

	if o.bus != nil {
		_ = o.bus.Publish(ctx, events.Event{
			CorrelationID: job.CorrelationID,
			Source:        "jobs.orchestrator",
			Type:          events.TypeJobStarted,
			Payload:       mustMarshal(job),
		})
	}

	o.mu.RLock()
	handler, ok := o.handlers[job.Type]
	o.mu.RUnlock()

	if !ok {
		o.failJob(job, fmt.Errorf("no handler registered for job type: %s", job.Type))
		return
	}

	timeout := 5 * time.Minute
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := handler(ctx, *job)
	if err != nil {
		if job.Attempts < job.MaxAttempts {
			o.retryJob(job, err)
		} else {
			o.failJob(job, err)
		}
		return
	}

	o.completeJob(job, result)
}

func (o *Orchestrator) retryJob(job *Job, jobErr error) {
	delay := o.config.RetryDelay * time.Duration(job.Attempts)
	if delay > o.config.MaxRetryDelay {
		delay = o.config.MaxRetryDelay
	}

	_ = o.queue.UpdateStatus(o.ctx, job.ID, StatusPending, nil, jobErr.Error())

	time.Sleep(delay)
}

func (o *Orchestrator) failJob(job *Job, jobErr error) {
	_ = o.queue.UpdateStatus(o.ctx, job.ID, StatusFailed, nil, jobErr.Error())

	if o.bus != nil {
		_ = o.bus.Publish(o.ctx, events.Event{
			CorrelationID: job.CorrelationID,
			Source:        "jobs.orchestrator",
			Type:          events.TypeJobFailed,
			Payload: mustMarshal(map[string]interface{}{
				"job_id": job.ID,
				"error":  jobErr.Error(),
			}),
		})
	}
}

func (o *Orchestrator) completeJob(job *Job, result json.RawMessage) {
	_ = o.queue.UpdateStatus(o.ctx, job.ID, StatusDone, result, "")

	if o.bus != nil {
		_ = o.bus.Publish(o.ctx, events.Event{
			CorrelationID: job.CorrelationID,
			Source:        "jobs.orchestrator",
			Type:          events.TypeJobCompleted,
			Payload: mustMarshal(map[string]interface{}{
				"job_id": job.ID,
				"result": result,
			}),
		})
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
