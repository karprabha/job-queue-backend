# 🔍 Task 7 — Observability: Structured Logging & Metrics

## Objective

Introduce **structured logging** and **basic metrics** so the system can be:

- debugged
- reasoned about
- monitored

This task focuses on:

- signal over noise
- consistent log structure
- clear operational insight

This task is about **making the system understandable under load**.

---

## Scope

- Add structured logging
- Add basic in-memory metrics
- Expose a metrics endpoint

---

## Functional Requirements

### Logging

Log events for:

- job created
- job enqueued
- job started
- job completed
- job failed
- job retried
- worker started/stopped

Each log entry must include:

- event name
- job_id (if applicable)
- worker_id (if applicable)
- timestamp

### Metrics

Track at minimum:

- total jobs created
- jobs completed
- jobs failed
- jobs retried
- jobs in progress (gauge)

### Metrics Endpoint

- Endpoint: `GET /metrics`
- Response format: JSON
- Example:
  ```json
  {
    "jobs_created": 120,
    "jobs_completed": 110,
    "jobs_failed": 5,
    "jobs_retried": 10,
    "jobs_in_progress": 2
  }
  ```

---

## Technical Constraints

### Logging Design

- Use Go's standard library only (`log/slog`)
- No global logger
- Logger must be injected where needed
- Consistent field keys across logs

### Metrics Design

- Metrics must be concurrency-safe
- No global mutable state
- Metrics logic must be separate from HTTP handlers
- Store metrics in memory only

### Dependency Wiring

- Logger and metrics must be initialized in `main`
- Passed into:
  - handlers
  - workers
  - store (if needed)

---

## Explicit Non-Goals

- Prometheus
- OpenTelemetry
- Histograms
- Persistence
- Distributed tracing

---

## Review Criteria

**PR will be blocked if:**

- Logs are unstructured strings
- Logger is accessed via global variables
- Metrics are updated from HTTP handlers directly
- Log spam (logging inside tight loops)
- Inconsistent field naming

**Will be commented on:**

- Logging design and field choices
- Metrics collection approach
- Concurrency safety of metrics
- Dependency injection pattern
- What is logged vs. what is not logged

---

## Definition of Done

- `go build ./...` succeeds
- Logs are structured and readable
- Metrics update correctly
- `/metrics` endpoint works
- No race conditions

---

## Deliverables

1. Feature branch: `feature/observability`
2. Pull request into `main`
3. PR description must include:
   - Why you chose specific log fields
   - How metrics avoid race conditions
   - One metric you would add later

---

## Notes

Observability is not optional in real systems.

If you can't explain what your system is doing, you don't own it.

This task will test your **taste**:

- what to log
- what _not_ to log
- where to collect metrics

Good engineers write code.
Great engineers make systems **explain themselves**.

Once this is done, the next tasks will cover:

- graceful backoff
- persistence boundaries
- testability & refactoring
- production hardening
