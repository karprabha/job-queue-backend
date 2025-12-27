# ⚖️ Task 5 — Multiple Workers & Controlled Concurrency

## Objective

Introduce **multiple workers** processing jobs concurrently while ensuring:

- correctness
- no duplicate processing
- predictable behavior under load

This task focuses on:

- worker pools
- fan-out concurrency
- coordination
- avoiding subtle race conditions

This task is about **scaling safely**, not adding features.

---

## Scope

- Support N workers (configurable)
- Ensure each job is processed exactly once
- Preserve correct job state transitions

---

## Functional Requirements

### Worker Pool

- Worker count must be configurable at startup
- Workers must all listen on the same job channel
- Workers must terminate cleanly on shutdown

### Job Processing Guarantees

- A job must never be processed by two workers
- A job must not be lost
- Job state transitions remain:

```
pending → processing → completed
```

---

## Technical Constraints

### Concurrency Model

- Use **fan-out via channels**
- Do NOT create per-worker queues
- Do NOT introduce locks in the worker layer
- Store remains the single source of truth

### Channel Design

- Channel must be buffered
- Buffer size must be configurable or justified
- No busy waits
- No sleeps for synchronization

### Configuration

- Worker count must be configurable via:
  - environment variable OR
  - config struct passed at startup

### Dependency Wiring

- Worker pool must be created in `main`
- Workers receive:
  - job channel
  - store
  - context
- Workers do NOT depend on HTTP or config parsing

---

## Explicit Non-Goals

- Job prioritization
- Retry logic
- Failure states
- Metrics
- Persistence
- Rate limiting

---

## Review Criteria

**PR will be blocked if:**

- Job duplication is possible
- Workers pull jobs by scanning the store
- Channels are misused as mutexes
- Workers manage their own lifecycle independently
- Globals are introduced
- Configuration is hardcoded

**Will be commented on:**

- Worker pool design and coordination
- Channel usage and buffering decisions
- Concurrency safety implementation
- Shutdown handling approach
- Configuration approach

---

## Definition of Done

- `go build ./...` succeeds
- N workers process jobs concurrently
- Each job processed exactly once
- Shutdown stops all workers cleanly
- Existing endpoints still work

---

## Deliverables

1. Feature branch: `feature/worker-pool`
2. Pull request into `main`
3. PR description must include:
   - How you prevented duplicate processing
   - Why the channel buffer size was chosen
   - One scaling concern you see

---

## Notes

This task is deceptively hard.

If your implementation feels simple:

- you probably thought clearly 👍

If it feels complex:

- you may be fighting the model (that's okay too)

We'll refactor after review.

This task separates:

- "I can write Go"
  from
- "I understand concurrent systems"

Don't rush it.

Once this is done, the **next phase** will be:

- failure handling
- retries
- idempotency
- testability
