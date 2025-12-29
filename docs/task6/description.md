# 💥 Task 6 — Failure Handling, Retries & Job States

## Objective

Extend the job system to handle **processing failures** and **retries** safely.

This task introduces:

- failure as a first-class concept
- retry logic
- idempotent state transitions
- clearer domain modeling

This task focuses on:

- explicit job state machine
- retry limits
- idempotent transitions
- concurrency-safe updates

This is where most queues break.

---

## Scope

- Add failure states
- Implement retry logic
- Track retry attempts
- Prevent infinite retries

---

## Functional Requirements

### Job States

Extend job statuses to:

```
pending → processing → completed
↘ failed
```

Retry flow:

```
processing → failed → pending
```

### Retry Rules

- Each job has `max_retries`
- Track `attempts`
- If `attempts >= max_retries` → permanent failure
- Retries must re-enqueue the job

### Failure Simulation

- Simulate failures deterministically:
  - e.g. fail every Nth job OR
  - fail based on job type
- No randomness allowed

### Job Model Changes

Add:

- `Attempts int`
- `MaxRetries int`
- `LastError string (optional)`

---

## Technical Constraints

### State Transitions

- All state changes must be validated
- Invalid transitions must be rejected
- Store enforces state rules (not workers)

### Retry Mechanics

- Worker signals failure
- Store updates state atomically
- Failed job is re-queued only if retryable
- No duplicate enqueueing

### Concurrency Rules

- No race conditions
- No double retries
- No lost jobs

---

## Explicit Non-Goals

- Dead-letter queues
- Backoff strategies
- Persistence
- Metrics
- Tests

---

## Review Criteria

**PR will be blocked if:**

- Workers directly mutate job state
- Retry logic is scattered
- Infinite retry is possible
- State transitions are implicit
- Failure logic is mixed with HTTP

**Will be commented on:**

- State transition validation approach
- Retry mechanism design
- Concurrency safety of retry logic
- Store enforcement of state rules
- Failure simulation approach

---

## Definition of Done

- `go build ./...` succeeds
- Failed jobs retry correctly
- Retry limit is enforced
- Completed jobs never retry
- `GET /jobs` shows accurate state

---

## Deliverables

1. Feature branch: `feature/retries-and-failures`
2. Pull request into `main`
3. PR description must include:
   - Where state transition rules live
   - How retries are prevented from duplicating work
   - One thing you'd redesign with persistence

---

## Notes

This task is **intentionally strict**.

A queue without failure handling is a demo.
A queue with bad failure handling is dangerous.

Do it carefully.

You will feel pressure to:

- "just update status"
- "just push back to channel"

Resist.

Think in terms of:

- invariants
- ownership
- allowed transitions

Once this is done, the next tasks will cover:

- backoff
- observability
- persistence boundaries
- testability
