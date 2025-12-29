# 🧠 Task 9 — Persistence Boundary & Startup Recovery

## Objective

Define clear recovery semantics when the process starts.

Up to now, your system assumes:

- memory is reliable
- process never crashes
- queue state is always fresh

That's not reality.

This task focuses on:

- defining the source of truth
- startup reconciliation
- safe re-enqueueing
- preventing stuck jobs

This task forces you to **formalize what happens when the process restarts**.

---

## Scope

- Define recovery behavior at startup
- Reconcile in-memory store and worker queue
- Ensure no job is permanently stuck

---

## Functional Requirements

### Recovery Rules

On application startup:

1. Jobs in `pending` state must be re-enqueued
2. Jobs in `processing` state must be moved back to `pending`
3. Jobs in `completed` state remain untouched
4. Jobs in permanent `failed` state remain untouched

### Re-enqueue Behavior

- Re-enqueueing must respect backpressure
- If queue is full:
  - recovery must pause and retry
  - no jobs may be dropped
- Recovery must finish before workers start processing

---

## Technical Constraints

### Ownership Rules

- Store is the **source of truth**
- Queue is a **delivery mechanism**
- Workers never scan the store directly

### Startup Order

1. Initialize store
2. Run recovery logic
3. Initialize queue
4. Start workers
5. Start HTTP server

### Recovery Logic Placement

- Recovery logic must live outside HTTP and workers
- Suggested location:
  ```
  internal/
  ├── recovery/
  │ └── recovery.go
  ```

### State Transitions

- Recovery must use the same state transition rules
- No ad-hoc mutations

---

## Explicit Non-Goals

- Disk persistence
- Snapshots
- WAL / journaling
- Distributed recovery
- Tests

---

## Review Criteria

**PR will be blocked if:**

- Workers start before recovery completes
- Recovery logic bypasses store invariants
- Jobs are dropped silently
- Processing jobs remain stuck
- Queue overflow causes loss

**Will be commented on:**

- Recovery coordination approach
- State transition handling
- Backpressure during recovery
- Source of truth design
- Edge cases considered

---

## Definition of Done

- `go build ./...` succeeds
- Restarting the app recovers jobs correctly
- No job is left stuck in `processing`
- Backpressure is respected during recovery
- System starts cleanly every time

---

## Deliverables

1. Feature branch: `feature/startup-recovery`
2. Pull request into `main`
3. PR description must include:
   - Why store is the source of truth
   - How recovery avoids job duplication
   - One limitation of in-memory recovery

---

## Notes

This task turns your system from:

> "works while running"

into:

> "survives restarts"

That's the difference between demos and systems.

This task is subtle and slow.

If you rush:

- you will introduce duplication
- or stuck jobs
- or silent loss

If you do it right:

- everything else becomes easier

Once this is done, the **final phase** begins:

- refactoring for testability
- interfaces vs concretes
- API versioning
- production hardening

Take your time.
