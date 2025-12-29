# Understanding Structured Logging with slog

## Table of Contents

1. [Why Structured Logging?](#why-structured-logging)
2. [What is slog?](#what-is-slog)
3. [Structured vs Unstructured Logs](#structured-vs-unstructured-logs)
4. [Creating a Logger](#creating-a-logger)
5. [Logging with Fields](#logging-with-fields)
6. [Event-Based Logging](#event-based-logging)
7. [Logger Injection (No Globals)](#logger-injection-no-globals)
8. [Consistent Field Naming](#consistent-field-naming)
9. [What to Log vs What Not to Log](#what-to-log-vs-what-not-to-log)
10. [Common Mistakes](#common-mistakes)

---

## Why Structured Logging?

### The Problem with Unstructured Logs

**Before structured logging:**

```go
log.Printf("Job %s created by worker %d at %v", jobID, workerID, time.Now())
```

**Output:**
```
Job abc123 created by worker 3 at 2024-01-15 10:30:45
```

**Problems:**

1. **Hard to parse** - Can't easily extract `jobID` or `workerID`
2. **Hard to search** - Can't filter by job ID or worker ID
3. **Hard to aggregate** - Can't count how many jobs were created
4. **Format inconsistencies** - Different developers format differently
5. **No machine-readable structure** - Log aggregation tools struggle

### The Solution: Structured Logging

**With structured logging:**

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID, "worker_id", workerID)
```

**Output (text format):**
```
time=2024-01-15T10:30:45Z level=INFO msg="Job created" event=job_created job_id=abc123 worker_id=3
```

**Output (JSON format):**
```json
{
  "time": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "msg": "Job created",
  "event": "job_created",
  "job_id": "abc123",
  "worker_id": 3
}
```

**Benefits:**

1. **Easy to parse** - Each field is separate
2. **Easy to search** - Filter by `job_id=abc123`
3. **Easy to aggregate** - Count events by `event=job_created`
4. **Consistent format** - All logs follow same structure
5. **Machine-readable** - Log tools can process automatically

### Real-World Analogy

Think of logs like a database:

- **Unstructured logs** = Free-form text notes (hard to query)
- **Structured logs** = Database rows with columns (easy to query)

---

## What is slog?

### Introduction

`slog` (Structured Logging) is Go's standard library package for structured logging, introduced in Go 1.21.

**Package:** `log/slog`

**Key Features:**

1. **Structured fields** - Key-value pairs
2. **Log levels** - Debug, Info, Warn, Error
3. **Multiple handlers** - Text, JSON, custom
4. **Context support** - Can attach context to logs
5. **Performance** - Fast, zero-allocation in many cases

### Why slog Instead of log?

**Old way (`log` package):**

```go
import "log"

log.Printf("Job %s created", jobID)  // Unstructured
```

**New way (`slog` package):**

```go
import "log/slog"

logger.Info("Job created", "job_id", jobID)  // Structured
```

**Key Difference:** `slog` supports structured fields, `log` doesn't.

---

## Structured vs Unstructured Logs

### Unstructured Log (Old Way)

```go
log.Printf("Worker %d processing job %s", workerID, jobID)
```

**Output:**
```
Worker 3 processing job abc123
```

**Problems:**

- Can't extract `workerID` programmatically
- Can't filter by worker ID
- Can't count jobs per worker
- Format varies between developers

### Structured Log (New Way)

```go
logger.Info("Job started", "event", "job_started", "worker_id", workerID, "job_id", jobID)
```

**Output (text):**
```
time=2024-01-15T10:30:45Z level=INFO msg="Job started" event=job_started worker_id=3 job_id=abc123
```

**Output (JSON):**
```json
{
  "time": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "msg": "Job started",
  "event": "job_started",
  "worker_id": 3,
  "job_id": "abc123"
}
```

**Benefits:**

- Can extract `worker_id` programmatically
- Can filter: `worker_id=3`
- Can count: `event=job_started`
- Consistent format

### Visual Comparison

**Unstructured:**
```
[2024-01-15 10:30:45] Worker 3 processing job abc123
[2024-01-15 10:30:46] Worker 3 processing job abc123
[2024-01-15 10:30:47] Worker 2 processing job def456
```

**Structured:**
```
time=2024-01-15T10:30:45Z level=INFO event=job_started worker_id=3 job_id=abc123
time=2024-01-15T10:30:46Z level=INFO event=job_started worker_id=3 job_id=abc123
time=2024-01-15T10:30:47Z level=INFO event=job_started worker_id=2 job_id=def456
```

**With structured logs, you can:**
- Count: `grep "event=job_started" | wc -l`
- Filter: `grep "worker_id=3"`
- Aggregate: `grep "job_id=abc123"`

---

## Creating a Logger

### Basic Logger Creation

```go
import (
    "log/slog"
    "os"
)

logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
```

**Breaking this down:**

1. **`slog.New()`** - Creates a new logger
2. **`slog.NewTextHandler()`** - Creates a text format handler
3. **`os.Stdout`** - Writes to standard output
4. **`nil`** - Default options (we'll customize later)

### Text Handler vs JSON Handler

**Text Handler (human-readable):**

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
logger.Info("Job created", "job_id", "abc123")
```

**Output:**
```
time=2024-01-15T10:30:45Z level=INFO msg="Job created" job_id=abc123
```

**JSON Handler (machine-readable):**

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("Job created", "job_id", "abc123")
```

**Output:**
```json
{"time":"2024-01-15T10:30:45Z","level":"INFO","msg":"Job created","job_id":"abc123"}
```

### Our Implementation

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
```

**Why text handler?**

- Easier to read during development
- Can switch to JSON in production
- Good for learning

**In production, you might use:**

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

---

## Logging with Fields

### Basic Field Logging

```go
logger.Info("Job created", "job_id", job.ID)
```

**Syntax:**
```go
logger.Level(message, key1, value1, key2, value2, ...)
```

**Key Points:**

- Fields are key-value pairs
- Always even number of arguments (key, value, key, value, ...)
- Keys should be strings
- Values can be any type

### Multiple Fields

```go
logger.Info("Job started", 
    "event", "job_started",
    "worker_id", workerID,
    "job_id", jobID,
)
```

**Output:**
```
time=2024-01-15T10:30:45Z level=INFO msg="Job started" event=job_started worker_id=3 job_id=abc123
```

### Log Levels

**Available levels:**

1. **Debug** - Detailed information for debugging
2. **Info** - General informational messages
3. **Warn** - Warning messages
4. **Error** - Error messages

**Example:**

```go
logger.Debug("Processing details", "step", "validation")  // Debug level
logger.Info("Job created", "job_id", jobID)              // Info level
logger.Warn("Queue nearly full", "queue_size", 95)        // Warn level
logger.Error("Failed to process", "error", err)          // Error level
```

**Our usage:**

- **Info** - Normal events (job created, job started, job completed)
- **Error** - Errors (failed to create job, failed to process)

---

## Event-Based Logging

### What is Event-Based Logging?

Every log entry includes an **event name** that describes what happened.

**Example:**

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID)
logger.Info("Job started", "event", "job_started", "worker_id", workerID, "job_id", jobID)
logger.Info("Job completed", "event", "job_completed", "worker_id", workerID, "job_id", jobID)
```

### Why Event Names Matter

**Without event names:**

```go
logger.Info("Job created", "job_id", jobID)
logger.Info("Job started", "job_id", jobID)
logger.Info("Job completed", "job_id", jobID)
```

**Problem:** Can't easily filter by event type.

**With event names:**

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID)
logger.Info("Job started", "event", "job_started", "job_id", jobID)
logger.Info("Job completed", "event", "job_completed", "job_id", jobID)
```

**Benefit:** Can filter: `event=job_created` or `event=job_completed`

### Our Event Names

**Job lifecycle events:**

- `job_created` - Job was created
- `job_enqueued` - Job was added to queue
- `job_started` - Worker started processing job
- `job_completed` - Job completed successfully
- `job_failed` - Job failed
- `job_retried` - Job was retried

**Worker events:**

- `worker_started` - Worker started
- `worker_stopped` - Worker stopped

**System events:**

- `sweeper_stopped` - Sweeper stopped
- `job_enqueue_failed` - Failed to enqueue job
- `job_claim_error` - Error claiming job

### Event Naming Convention

**Format:** `{entity}_{action}`

- `job_created` - Job entity, created action
- `job_started` - Job entity, started action
- `worker_stopped` - Worker entity, stopped action

**Benefits:**

- Consistent naming
- Easy to search
- Clear what happened

---

## Logger Injection (No Globals)

### The Problem with Global Loggers

**❌ Bad: Global logger**

```go
var logger = slog.Default()

func handler() {
    logger.Info("message")  // Uses global
}
```

**Problems:**

1. **Hard to test** - Can't replace logger in tests
2. **Hidden dependency** - Not clear function needs logger
3. **Global state** - Violates Go best practices
4. **Can't have different loggers** - All code uses same logger

### The Solution: Dependency Injection

**✅ Good: Injected logger**

```go
type JobHandler struct {
    logger *slog.Logger
}

func NewJobHandler(logger *slog.Logger) *JobHandler {
    return &JobHandler{logger: logger}
}

func (h *JobHandler) CreateJob() {
    h.logger.Info("Job created", "event", "job_created")
}
```

**Benefits:**

1. **Easy to test** - Can inject test logger
2. **Explicit dependency** - Clear function needs logger
3. **No global state** - Each component has its own logger
4. **Flexible** - Can use different loggers for different components

### Our Implementation

**In main.go:**

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

jobHandler := internalhttp.NewJobHandler(jobStore, metricStore, logger, jobQueue)
worker := worker.NewWorker(workerID, jobStore, metricStore, logger, jobQueue)
```

**In handlers:**

```go
type JobHandler struct {
    logger *slog.Logger
}

func (h *JobHandler) CreateJob() {
    h.logger.Info("Job created", "event", "job_created", "job_id", jobID)
}
```

**Key Point:** Logger is created once in `main()` and passed to all components.

---

## Consistent Field Naming

### Why Consistent Naming Matters

**Inconsistent naming:**

```go
logger.Info("Job created", "jobId", id)        // camelCase
logger.Info("Job started", "job_id", id)      // snake_case
logger.Info("Job completed", "JobID", id)     // PascalCase
```

**Problems:**

- Hard to search (need to know which format)
- Inconsistent in logs
- Confusing

### Our Convention: snake_case

**Consistent naming:**

```go
logger.Info("Job created", "event", "job_created", "job_id", id)
logger.Info("Job started", "event", "job_started", "job_id", id, "worker_id", wid)
logger.Info("Job completed", "event", "job_completed", "job_id", id, "worker_id", wid)
```

**Field names:**

- `event` - Event name
- `job_id` - Job identifier
- `worker_id` - Worker identifier
- `error` - Error message

**Benefits:**

- Easy to search: `job_id=abc123`
- Consistent across all logs
- Clear and readable

### Field Naming Rules

1. **Use snake_case** - `job_id`, not `jobId` or `JobID`
2. **Be descriptive** - `worker_id`, not `wid`
3. **Be consistent** - Always use same name for same concept
4. **Use standard names** - `error` for errors, `event` for events

---

## What to Log vs What Not to Log

### What to Log (Signal)

**✅ Log important events:**

- Job lifecycle (created, started, completed, failed)
- Worker lifecycle (started, stopped)
- Errors (with context)
- System events (shutdown, startup)

**Example:**

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID)
logger.Info("Job started", "event", "job_started", "worker_id", workerID, "job_id", jobID)
logger.Error("Failed to create job", "event", "job_create_error", "error", err)
```

### What Not to Log (Noise)

**❌ Don't log:**

- Inside tight loops (too many logs)
- Every iteration of a loop
- Debug information in production
- Sensitive data (passwords, tokens)

**Example of log spam:**

```go
// ❌ BAD: Logging in tight loop
for i := 0; i < 1000000; i++ {
    logger.Info("Processing", "iteration", i)  // 1 million logs!
}
```

**Better:**

```go
// ✅ GOOD: Log summary
logger.Info("Processing started", "total", 1000000)
// ... process ...
logger.Info("Processing completed", "total", 1000000)
```

### Log Levels Strategy

**Debug** - Detailed debugging info (usually disabled in production)

```go
logger.Debug("Validating payload", "payload_size", len(payload))
```

**Info** - Normal operational events

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID)
```

**Warn** - Warning conditions

```go
logger.Warn("Queue nearly full", "queue_size", 95, "capacity", 100)
```

**Error** - Error conditions

```go
logger.Error("Failed to process job", "event", "job_process_error", "error", err)
```

---

## Common Mistakes

### Mistake 1: Using Global Logger

```go
// ❌ BAD: Global logger
var logger = slog.Default()

func handler() {
    logger.Info("message")
}
```

**Fix:** Inject logger

```go
// ✅ GOOD: Injected logger
type Handler struct {
    logger *slog.Logger
}

func (h *Handler) handler() {
    h.logger.Info("message")
}
```

### Mistake 2: Unstructured Logs

```go
// ❌ BAD: Unstructured
log.Printf("Job %s created", jobID)
```

**Fix:** Use structured logging

```go
// ✅ GOOD: Structured
logger.Info("Job created", "event", "job_created", "job_id", jobID)
```

### Mistake 3: Inconsistent Field Names

```go
// ❌ BAD: Mixed naming
logger.Info("Job created", "jobId", id)
logger.Info("Job started", "job_id", id)
```

**Fix:** Use consistent naming

```go
// ✅ GOOD: Consistent snake_case
logger.Info("Job created", "event", "job_created", "job_id", id)
logger.Info("Job started", "event", "job_started", "job_id", id)
```

### Mistake 4: Missing Event Names

```go
// ❌ BAD: No event name
logger.Info("Job created", "job_id", jobID)
```

**Fix:** Include event name

```go
// ✅ GOOD: With event name
logger.Info("Job created", "event", "job_created", "job_id", jobID)
```

### Mistake 5: Log Spam

```go
// ❌ BAD: Logging in tight loop
for i := 0; i < 1000; i++ {
    logger.Info("Processing", "iteration", i)
}
```

**Fix:** Log summary or use Debug level

```go
// ✅ GOOD: Log summary
logger.Info("Processing started", "total", 1000)
// ... process ...
logger.Info("Processing completed", "total", 1000)
```

### Mistake 6: Logging Sensitive Data

```go
// ❌ BAD: Logging sensitive data
logger.Info("User logged in", "password", password, "token", token)
```

**Fix:** Don't log sensitive data

```go
// ✅ GOOD: Log non-sensitive data
logger.Info("User logged in", "event", "user_login", "user_id", userID)
```

---

## Key Takeaways

1. **Structured logging** = Key-value pairs, not free-form text
2. **slog** = Go's standard structured logging package
3. **Event names** = Every log should have an event field
4. **Logger injection** = Pass logger as parameter, not global
5. **Consistent naming** = Use snake_case for field names
6. **Log signal, not noise** = Log important events, not everything
7. **Structured logs** = Easy to parse, search, and aggregate

---

## Real-World Example

**Our job creation logging:**

```go
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // ... create job ...
    
    h.logger.Info("Job created", "event", "job_created", "job_id", job.ID)
    
    // ... enqueue job ...
    
    h.logger.Info("Job enqueued", "event", "job_enqueued", "job_id", job.ID)
}
```

**Output:**
```
time=2024-01-15T10:30:45Z level=INFO msg="Job created" event=job_created job_id=abc123
time=2024-01-15T10:30:45Z level=INFO msg="Job enqueued" event=job_enqueued job_id=abc123
```

**Benefits:**

- Can search: `event=job_created`
- Can filter: `job_id=abc123`
- Can count: How many jobs created?
- Machine-readable for log aggregation tools

---

## Next Steps

- Read [Metrics Collection and Storage](./02-metrics-collection-storage.md) to understand how we track metrics
- Read [Dependency Injection for Observability](./03-dependency-injection-observability.md) to see how we wire dependencies
- Read [Event-Based Logging Pattern](./05-event-based-logging.md) for more on event naming

