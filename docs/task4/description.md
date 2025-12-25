# 🧠 Task 4 — Background Worker & Job Processing Loop

## Objective

Introduce an asynchronous **job processing system** using Go primitives.

This task focuses on:

- goroutines
- channels
- worker lifecycle
- state transitions
- avoiding race conditions

This is where Go starts to _matter_. You will introduce goroutines, channels, controlled concurrency, and state transitions. Still **no persistence, no retries, no complexity creep**.

---

## Scope

- Add a background worker
- Process jobs asynchronously
- Update job status safely
- Keep HTTP layer thin

---

## Functional Requirements

### Job Status Lifecycle

Jobs must transition through the following states:

```
pending → processing → completed
```

### Processing Rules

- Only `pending` jobs may be picked up
- Processing is simulated (sleep)
- Each job must be processed **exactly once**
- Job status must be updated atomically

### Worker Behavior

- Worker must start when the server starts
- Worker must stop when the server stops
- Worker must continuously poll or receive jobs
- Single worker only (for now)

---

## Technical Constraints

### Architecture

Introduce a **worker package**:

```
internal/
├── worker/
│   └── worker.go
```

Worker responsibilities:

- receive jobs
- process them
- update job status via the store

### Communication Model

- Use **channels** for job dispatch
- Do NOT have the worker scan the store blindly
- `POST /jobs` must enqueue the job

Example (conceptual):

```go
jobQueue <- job
```

### Concurrency Rules

- No data races
- No busy loops
- No sleeps inside infinite loops
- Store remains concurrency-safe

### Dependency Wiring

- Worker must receive:
  - job store
  - job channel
- Worker must NOT depend on HTTP packages
- No globals allowed

### Shutdown Handling

- Use `context.Context`
- Worker must stop gracefully on server shutdown
- No goroutine leaks

---

## Explicit Non-Goals

- Multiple workers
- Retry logic
- Failure states
- Dead-letter queues
- Persistence
- Metrics
- Tests

---

## Review Criteria

**PR will be blocked if:**

- Worker logic lives in HTTP handlers
- Worker polls store in a loop without coordination
- Job status updates are unsafe
- Goroutines are leaked
- Channel is unbuffered without justification
- Business logic is mixed with transport logic

**Will be commented on:**

- Worker design and structure
- Channel usage and buffering decisions
- Concurrency safety implementation
- Shutdown handling approach
- Goroutine ownership and lifecycle

---

## Definition of Done

- `go build ./...` succeeds
- `POST /jobs` enqueues a job
- Worker processes jobs asynchronously
- Job status transitions correctly
- `GET /jobs` reflects updated statuses
- Graceful shutdown stops worker

---

## Deliverables

1. Feature branch: `feature/background-worker`
2. Pull request into `main`
3. PR description must include:
   - Why you chose buffered vs unbuffered channel
   - How worker shutdown is handled
   - One concurrency concern you had

---

## Notes

This is your **first real Go concurrency task**.

Expect mistakes. That's the point. We will refactor later.

You are **not** building a perfect queue. You are learning:

- ownership of goroutines
- data flow via channels
- safe mutation under concurrency

If you feel slightly anxious about:

- "what owns this goroutine?"
- "who closes this channel?"

Good. That's senior-level thinking.

Once this is merged, the next tasks will cover:

- multiple workers
- backpressure
- retries
- failure states
- clean abstractions
- testability
