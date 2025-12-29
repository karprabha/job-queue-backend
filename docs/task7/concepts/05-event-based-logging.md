# Understanding Event-Based Logging Pattern

## Table of Contents

1. [What is Event-Based Logging?](#what-is-event-based-logging)
2. [Why Event Names Matter](#why-event-names-matter)
3. [Event Naming Convention](#event-naming-convention)
4. [Our Event Catalog](#our-event-catalog)
5. [Searching and Filtering Events](#searching-and-filtering-events)
6. [Common Mistakes](#common-mistakes)

---

## What is Event-Based Logging?

### Definition

**Event-based logging** means every log entry includes an **event name** that describes what happened.

**Pattern:**

```go
logger.Info("Message", "event", "event_name", ...other fields)
```

### Example

**Without event name:**

```go
logger.Info("Job created", "job_id", jobID)
logger.Info("Job started", "job_id", jobID)
logger.Info("Job completed", "job_id", jobID)
```

**With event name:**

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID)
logger.Info("Job started", "event", "job_started", "job_id", jobID)
logger.Info("Job completed", "event", "job_completed", "job_id", jobID)
```

**Key difference:** The `event` field makes logs searchable and filterable.

---

## Why Event Names Matter

### The Problem Without Events

**Logs without event names:**

```
time=2024-01-15T10:30:45Z level=INFO msg="Job created" job_id=abc123
time=2024-01-15T10:30:46Z level=INFO msg="Job started" job_id=abc123
time=2024-01-15T10:30:47Z level=INFO msg="Job completed" job_id=abc123
```

**To find all "job created" events:**

- Must search message text: `msg="Job created"`
- Fragile (message might change)
- Hard to aggregate

### The Solution: Event Names

**Logs with event names:**

```
time=2024-01-15T10:30:45Z level=INFO msg="Job created" event=job_created job_id=abc123
time=2024-01-15T10:30:46Z level=INFO msg="Job started" event=job_started job_id=abc123
time=2024-01-15T10:30:47Z level=INFO msg="Job completed" event=job_completed job_id=abc123
```

**To find all "job created" events:**

- Search event field: `event=job_created`
- Stable (event name doesn't change)
- Easy to aggregate

### Benefits

1. **Searchable** - Filter by event type
2. **Aggregatable** - Count events by type
3. **Stable** - Event name doesn't change even if message does
4. **Consistent** - Same event name used everywhere

---

## Event Naming Convention

### Our Convention

**Format:** `{entity}_{action}`

**Examples:**

- `job_created` - Job entity, created action
- `job_started` - Job entity, started action
- `job_completed` - Job entity, completed action
- `worker_started` - Worker entity, started action
- `worker_stopped` - Worker entity, stopped action

### Rules

1. **Use snake_case** - `job_created`, not `jobCreated` or `JobCreated`
2. **Be descriptive** - `job_created`, not `created`
3. **Be consistent** - Same event name for same action
4. **Use past tense for completed actions** - `job_completed`, not `job_complete`
5. **Use present tense for ongoing actions** - `job_started`, not `job_start`

### Event Name Structure

**Entity (what):**

- `job` - Job-related events
- `worker` - Worker-related events
- `sweeper` - Sweeper-related events

**Action (what happened):**

- `created` - Entity was created
- `started` - Process started
- `completed` - Process completed
- `failed` - Process failed
- `retried` - Process retried
- `stopped` - Process stopped

---

## Our Event Catalog

### Job Lifecycle Events

**`job_created`** - Job was created

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID)
```

**`job_enqueued`** - Job was added to queue

```go
logger.Info("Job enqueued", "event", "job_enqueued", "job_id", jobID)
```

**`job_started`** - Worker started processing job

```go
logger.Info("Job started", "event", "job_started", "worker_id", workerID, "job_id", jobID)
```

**`job_completed`** - Job completed successfully

```go
logger.Info("Job completed", "event", "job_completed", "worker_id", workerID, "job_id", jobID)
```

**`job_failed`** - Job failed

```go
logger.Info("Job failed", "event", "job_failed", "worker_id", workerID, "job_id", jobID)
```

**`job_retried`** - Job was retried

```go
logger.Info("Job retried", "event", "job_retried", "job_id", jobID)
```

### Worker Events

**`worker_started`** - Worker started

```go
logger.Info("Worker started", "event", "worker_started", "worker_id", workerID)
```

**`worker_stopped`** - Worker stopped

```go
logger.Info("Worker shutting down", "event", "worker_stopped", "worker_id", workerID)
```

### System Events

**`sweeper_stopped`** - Sweeper stopped

```go
logger.Info("Sweeper shutting down", "event", "sweeper_stopped")
```

### Error Events

**`job_enqueue_failed`** - Failed to enqueue job

```go
logger.Error("Failed to enqueue job", "event", "job_enqueue_failed", "job_id", jobID, "error", "queue_full")
```

**`job_claim_error`** - Error claiming job

```go
logger.Error("Worker error claiming job", "event", "job_claim_error", "worker_id", workerID, "job_id", jobID, "error", err)
```

**`job_update_error`** - Error updating job

```go
logger.Error("Worker error updating job to failed", "event", "job_update_error", "worker_id", workerID, "job_id", jobID, "error", err)
```

**`metric_error`** - Error updating metrics

```go
logger.Error("Worker error incrementing jobs completed", "event", "metric_error", "worker_id", workerID, "error", err)
```

**`sweeper_error`** - Sweeper error

```go
logger.Error("Sweeper error retrying failed jobs", "event", "sweeper_error", "error", err)
```

---

## Searching and Filtering Events

### Searching by Event

**Find all job created events:**

```bash
grep "event=job_created" logs.txt
```

**Find all job failures:**

```bash
grep "event=job_failed" logs.txt
```

**Find all worker events:**

```bash
grep "event=worker_" logs.txt
```

### Aggregating Events

**Count job created events:**

```bash
grep "event=job_created" logs.txt | wc -l
```

**Count job failures:**

```bash
grep "event=job_failed" logs.txt | wc -l
```

### Filtering with Log Tools

**With structured log tools (like Loki, ELK):**

```json
{
  "query": "event=job_created",
  "timeRange": "1h"
}
```

**Returns all job created events in last hour.**

### Real-World Use Cases

**1. Monitor job creation rate:**

```bash
# Count jobs created per minute
grep "event=job_created" logs.txt | awk '{print $1}' | uniq -c
```

**2. Find failed jobs:**

```bash
# List all failed jobs
grep "event=job_failed" logs.txt | grep -o "job_id=[^ ]*"
```

**3. Track worker activity:**

```bash
# Count jobs processed per worker
grep "event=job_completed" logs.txt | grep -o "worker_id=[0-9]*" | sort | uniq -c
```

---

## Common Mistakes

### Mistake 1: Missing Event Field

```go
// ❌ BAD: No event field
logger.Info("Job created", "job_id", jobID)
```

**Fix:** Include event field

```go
// ✅ GOOD: With event field
logger.Info("Job created", "event", "job_created", "job_id", jobID)
```

### Mistake 2: Inconsistent Event Names

```go
// ❌ BAD: Inconsistent naming
logger.Info("Job created", "event", "job_created", ...)
logger.Info("Job started", "event", "jobStart", ...)  // Different format!
logger.Info("Job completed", "event", "JobCompleted", ...)  // Different format!
```

**Fix:** Use consistent naming

```go
// ✅ GOOD: Consistent snake_case
logger.Info("Job created", "event", "job_created", ...)
logger.Info("Job started", "event", "job_started", ...)
logger.Info("Job completed", "event", "job_completed", ...)
```

### Mistake 3: Event Name in Message Only

```go
// ❌ BAD: Event only in message
logger.Info("job_created", "job_id", jobID)  // No event field
```

**Fix:** Use event field

```go
// ✅ GOOD: Event in field
logger.Info("Job created", "event", "job_created", "job_id", jobID)
```

### Mistake 4: Too Generic Event Names

```go
// ❌ BAD: Too generic
logger.Info("Something happened", "event", "event", ...)
```

**Fix:** Be specific

```go
// ✅ GOOD: Specific event name
logger.Info("Job created", "event", "job_created", ...)
```

### Mistake 5: Event Name Doesn't Match Message

```go
// ❌ BAD: Mismatch
logger.Info("Job started", "event", "job_created", ...)  // Message says started, event says created
```

**Fix:** Keep message and event consistent

```go
// ✅ GOOD: Consistent
logger.Info("Job started", "event", "job_started", ...)
```

---

## Key Takeaways

1. **Event-based logging** = Every log has an event name
2. **Event field** = `"event", "event_name"` in log fields
3. **Naming convention** = `{entity}_{action}` in snake_case
4. **Searchable** = Filter by `event=event_name`
5. **Aggregatable** = Count events by type
6. **Consistent** = Same event name everywhere
7. **Stable** = Event name doesn't change even if message does

---

## Real-World Example

**Our job processing flow with events:**

```go
// Job created
logger.Info("Job created", "event", "job_created", "job_id", jobID)

// Job enqueued
logger.Info("Job enqueued", "event", "job_enqueued", "job_id", jobID)

// Job started
logger.Info("Job started", "event", "job_started", "worker_id", workerID, "job_id", jobID)

// Job completed
logger.Info("Job completed", "event", "job_completed", "worker_id", workerID, "job_id", jobID)
```

**Log output:**

```
time=2024-01-15T10:30:45Z level=INFO msg="Job created" event=job_created job_id=abc123
time=2024-01-15T10:30:45Z level=INFO msg="Job enqueued" event=job_enqueued job_id=abc123
time=2024-01-15T10:30:46Z level=INFO msg="Job started" event=job_started worker_id=3 job_id=abc123
time=2024-01-15T10:30:47Z level=INFO msg="Job completed" event=job_completed worker_id=3 job_id=abc123
```

**Benefits:**

- Can search: `event=job_created`
- Can filter: `event=job_completed`
- Can aggregate: Count `event=job_started`
- Can track: All events for `job_id=abc123`

---

## Next Steps

- Read [Structured Logging with slog](./01-structured-logging-slog.md) to understand structured logging
- Read [Metrics Collection and Storage](./02-metrics-collection-storage.md) to see how metrics complement events
- Read [Dependency Injection for Observability](./03-dependency-injection-observability.md) to see how we wire the logger

