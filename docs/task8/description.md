# 🧯 Task 8 — Graceful Shutdown, Backpressure & Load Safety

## Objective

Make the system:

- shut down safely
- reject work predictably under load
- never lose in-flight jobs

This task focuses on:

- lifecycle ownership
- backpressure
- correctness during shutdown

This task is about **protecting the system under stress** and during deploys.

From here, we stop adding "features" and start **hardening the system** like a production backend.

---

## Scope

- Graceful shutdown
- Backpressure on job creation
- Safe worker termination

---

## Functional Requirements

### Graceful Shutdown

On shutdown signal:

- Stop accepting new jobs
- Finish processing in-flight jobs
- Exit cleanly without panics or leaks

### Backpressure

- Job queue must have a **max capacity**
- If queue is full:
  - `POST /jobs` must return `429 Too Many Requests`
  - Response must be JSON
- No blocking HTTP handlers indefinitely

### Worker Behavior

- Workers must:
  - finish current job
  - not pick up new jobs after shutdown starts
- No jobs should be left in `processing` state on exit

---

## Technical Constraints

### Context Propagation

- Use `context.Context` consistently
- Shutdown signal must propagate to:
  - HTTP server
  - worker pool
- No ad-hoc boolean flags

### Channel Handling

- Channel closing must be centralized
- No sends on closed channels
- No goroutine leaks

### Dependency Wiring

- Shutdown coordination lives in `main`
- Workers must respect context cancellation
- Handlers must check shutdown state

---

## Explicit Non-Goals

- Persistence
- Job draining to disk
- Leader election
- Rolling deployments
- Tests

---

## Review Criteria

**PR will be blocked if:**

- Jobs are dropped silently
- HTTP handlers block on full channels
- Workers ignore context cancellation
- Shutdown logic is scattered
- Race conditions during shutdown

**Will be commented on:**

- Backpressure enforcement approach
- Shutdown coordination design
- Context propagation pattern
- Channel closing strategy
- Edge cases considered

---

## Definition of Done

- `go build ./...` succeeds
- Ctrl+C triggers graceful shutdown
- In-flight jobs complete
- New jobs are rejected during shutdown
- No goroutine leaks

---

## Deliverables

1. Feature branch: `feature/graceful-shutdown`
2. Pull request into `main`
3. PR description must include:
   - How backpressure is enforced
   - How shutdown is coordinated
   - One edge case you worried about

---

## Notes

This task is where many "working" systems fail in prod.

Graceful shutdown is not optional.
Backpressure is not pessimism.
They are signs of maturity.

If this task feels _subtle_ — good.

Production bugs live here:

- half-processed jobs
- blocked handlers
- leaked goroutines

After this, the next phase will be:

- persistence boundaries
- idempotency guarantees
- testability refactor
- API versioning

Proceed carefully.
