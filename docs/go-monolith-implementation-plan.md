# Go Monolith Implementation Plan

## 1. Scope

This plan implements the architecture in a single Go codebase with internal plugins.

Deliverables:

1. Core runtime (AI, scheduler/hooks, communications, event bus, jobs).
2. Domain plugins (email, GTD, business, software).
3. Production-grade controls (idempotency, retries, token budgets, observability).

Non-goal for this phase:

- External binary plugin loading.

## 2. Target Milestones

1. M0 - Foundations and contracts
2. M1 - Runtime core operational
3. M2 - Communication channels and command routing
4. M3 - Email + GTD + Business scenario
5. M4 - Software plugin async job execution
6. M5 - Hardening: policy, cost controls, reliability
7. M6 - Documentation and operational runbooks

## 3. Work Breakdown by Milestone

## M0 - Foundations and Contracts

### Objectives

- Lock stable interfaces before implementation grows.

### Tasks

1. Create core contracts:
   - `Event`, `Command`, `Job`, `ScheduleSpec`, `Plugin`.
2. Define event naming conventions.
3. Define plugin registration API (`Registrar`).
4. Add repository-level ADR describing why internal plugins were chosen.

### Output

- Compiling interface package.
- Contract documentation in `/docs`.

### Acceptance criteria

- All plugins can compile against interfaces without depending on each other.

## M1 - Runtime Core Operational

### Objectives

- Make the core executable with event-driven workflows.

### Tasks

1. Implement in-memory event bus abstraction with pluggable store-backed implementation.
2. Implement job orchestrator:
   - queue
   - worker pool
   - retries with backoff
   - timeout and cancellation
3. Implement scheduler service:
   - periodic triggers
   - persistent last-run tracking
4. Implement AI runtime abstraction:
   - provider interface
   - prompt execution API
   - token accounting hooks

### Output

- Single process capable of emitting, consuming, and persisting workflow state.

### Acceptance criteria

- Integration test: scheduled event triggers plugin handler and completes a job.

## M2 - Communication and Commands

### Objectives

- Accept user messages and map them to domain commands.

### Tasks

1. Implement `comms.Adapter` interface.
2. Build Telegram adapter first, Discord second.
3. Implement command parser/router:
   - route by command namespace (`software`, `inbox`, etc.)
   - route by plugin registration
4. Implement outbound notifications and command acknowledgements.

### Output

- End-to-end command path from chat to plugin handler and back.

### Acceptance criteria

- Test command from adapter produces expected command event and response message.

## M3 - Scenario 1: Email + GTD + Business

### Objectives

- Deliver mailbox fanout flow with deterministic first-pass triage.

### Tasks

1. Email plugin:
   - mailbox polling schedule
   - dedup by message ID
   - emit `mail.received`
2. GTD plugin:
   - subscribe to `mail.received`
   - extract tasks/calendar/project updates
   - write to vault
   - emit `gtd.insight.created`
3. Business plugin:
   - subscribe to `mail.received`
   - invoice detection and extraction
   - push to invoicing endpoint
   - emit `business.invoice.processed`
4. Add user notifications for meaningful actions.

### Output

- Functional fanout pipeline for mail processing.

### Acceptance criteria

- Replaying the same mail event does not duplicate side effects.
- GTD and business processing can run independently.

## M4 - Scenario 2 and 3: Software + Inbox Commands

### Objectives

- Support software-agent workflows and lightweight inbox capture.

### Tasks

1. Software plugin:
   - `software` command handler
   - task discovery in configured projects
   - async job launching for coding-agent backend
   - progress/result events
2. GTD plugin command path:
   - `inbox` command parsing
   - note formatting and vault write
   - optional AI enrichment mode
3. Communication status updates:
   - immediate acknowledgement
   - completion notification

### Output

- Both command-driven scenarios work from chat.

### Acceptance criteria

- `software` command produces async completion message.
- `inbox` command stores normalized entry and confirms success.

## M5 - Hardening and Operations

### Objectives

- Prevent runaway cost and improve reliability/safety.

### Tasks

1. Token budget module:
   - per-plugin/day caps
   - soft and hard limits
   - fallback behavior
2. Policy enforcement:
   - allowed actions by plugin
   - high-risk confirmation gates
3. Dead-letter queue and replay tooling.
4. Structured observability:
   - correlation IDs
   - metrics and dashboards
   - error alerting

### Output

- Production-safe runtime controls.

### Acceptance criteria

- Budget exceeded path degrades gracefully without crash.
- Failed handlers are inspectable and replayable.

## M6 - Docs and Runbooks

### Objectives

- Enable smooth operation and plugin onboarding.

### Tasks

1. Runbook for incident handling and retry/replay.
2. Plugin authoring guide for internal plugins.
3. Security checklist for new plugin integrations.
4. Cost-tuning guide (prompt and model optimization).

### Output

- Operational documentation ready for ongoing development.

### Acceptance criteria

- A new internal plugin can be added by following docs only.

## 4. Suggested Code Structure

```text
cmd/server/
internal/core/app/
internal/core/events/
internal/core/commands/
internal/core/scheduler/
internal/core/jobs/
internal/core/ai/
internal/core/comms/
internal/core/policy/
internal/core/store/
internal/plugins/registry/
internal/plugins/email/
internal/plugins/gtd/
internal/plugins/business/
internal/plugins/software/
```

## 5. Interface Sketches

```go
// internal/core/events/types.go
type Event struct {
    ID            string
    CorrelationID string
    Source        string
    Type          string
    OccurredAt    time.Time
    Payload       json.RawMessage
    Metadata      map[string]string
}

// internal/plugins/registry/contracts.go
type Plugin interface {
    ID() string
    Register(ctx context.Context, r Registrar) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

```go
// internal/core/ai/runtime.go
type Runtime interface {
    Execute(ctx context.Context, req PromptRequest) (PromptResult, error)
}

type PromptRequest struct {
    PluginID   string
    TaskType   string
    ModelClass string
    Input      string
    BudgetKey  string
}
```

## 6. Data and Migration Plan

### Phase 1

- Keep SQLite.
- Add tables for events, schedules, jobs, plugin_state, usage_cost.

### Phase 2 (if needed)

- Move to Postgres when worker concurrency and throughput require stronger locking and scaling.

## 7. Testing Strategy

1. Unit tests for each plugin handler and core modules.
2. Contract tests:
   - command routing
   - event schema compatibility
3. Integration tests for three key scenarios.
4. Chaos-style tests for retries, duplicate delivery, and adapter disconnects.
5. Golden tests for prompt shaping and deterministic parsing.

Minimum CI gates:

- `go test ./...`
- race detector on core packages
- lints for naming and complexity

## 8. Rollout Strategy

1. Deploy core + GTD plugin first (lowest risk).
2. Enable email plugin in read-only mode (no writes) and validate classifications.
3. Enable GTD writes and business invoice forwarding behind feature flags.
4. Enable software plugin with strict policy allowlist and low concurrency.

Feature flags should exist per plugin and per high-risk capability.

## 9. Risk Register

1. Event storms from fanout.
   - Mitigation: backpressure, queue limits, and rate-limited handlers.
2. Token cost spikes.
   - Mitigation: enforced per-plugin caps and model routing.
3. Duplicate side effects from retries.
   - Mitigation: idempotency keys and dedup tables.
4. Unsafe agent actions in software plugin.
   - Mitigation: policy engine + restricted execution profile.

## 10. Definition of Done

Implementation phase is complete when:

1. All three scenarios operate end-to-end.
2. Core services run in one Go binary with internal plugins.
3. Cost/safety controls are enforced at runtime.
4. Observability can answer: what happened, why, and what it cost.
5. Documentation is sufficient to add a new plugin with no architecture changes.
