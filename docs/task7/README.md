# Task 7 — Observability: Structured Logging & Metrics

## Overview

This task introduces **structured logging and basic metrics** to the job queue system, making it observable and debuggable under load. The focus is on signal over noise, consistent log structure, and clear operational insight.

## ✅ Completed Requirements

### Functional Requirements

- ✅ Structured logging with `log/slog`
- ✅ Event-based logging (job_created, job_started, job_completed, etc.)
- ✅ Metrics tracking (jobs_created, jobs_completed, jobs_failed, jobs_retried, jobs_in_progress)
- ✅ Metrics endpoint (`GET /metrics`)
- ✅ JSON response format for metrics
- ✅ All required events logged

### Technical Requirements

- ✅ Logger injected (no globals)
- ✅ Metrics store injected (no globals)
- ✅ Concurrency-safe metrics (mutex protection)
- ✅ Metrics logic separated from HTTP handlers
- ✅ Consistent field naming (snake_case)
- ✅ Event names in all logs
- ✅ Returning copies to prevent external mutation
- ✅ No race conditions

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go              # Logger and metric store initialization
├── internal/
│   ├── domain/
│   │   ├── job.go              # Job domain model
│   │   └── metric.go           # NEW: Metric domain model
│   ├── http/
│   │   ├── handler.go          # Health check handler
│   │   ├── job_handler.go      # Added logging and metrics
│   │   ├── metric_handler.go   # NEW: Metrics endpoint
│   │   └── response.go         # Error response helper
│   ├── store/
│   │   ├── job_store.go        # Added logging for retries
│   │   ├── metric_store.go     # NEW: Metrics storage
│   │   └── sweeper.go          # Added structured logging
│   └── worker/
│       └── worker.go           # Added logging and metrics
├── docs/
│   ├── task7/
│   │   ├── README.md           # This file
│   │   ├── summary.md           # Quick reference
│   │   ├── description.md      # Task requirements
│   │   └── concepts/           # Detailed concept explanations
│   └── learnings.md            # Overall learnings
└── go.mod                      # Go module
```

**Structure improvements:**
- `internal/domain/metric.go` - Metric domain model
- `internal/store/metric_store.go` - Metrics storage separated
- `internal/http/metric_handler.go` - Metrics endpoint handler
- Logger and metrics injected throughout

## 🔑 Key Concepts Learned

### 1. Structured Logging

- **What**: Key-value pairs instead of free-form text
- **Why**: Easy to parse, search, and aggregate
- **How**: `log/slog` package with structured fields
- **Pattern**: Always include event name, use consistent field naming

### 2. Metrics Collection

- **What**: Numerical measurements of system behavior
- **Why**: Monitor system health and performance
- **How**: In-memory metric store with mutex protection
- **Pattern**: Counters (only increment) and gauges (increment/decrement)

### 3. Dependency Injection

- **What**: Pass dependencies as constructor parameters
- **Why**: No global state, easy to test, clear dependencies
- **How**: Logger and metrics passed to all components
- **Pattern**: Create once in `main()`, pass everywhere

### 4. Concurrency Safety

- **What**: Mutex protection for shared state
- **Why**: Prevent race conditions with multiple goroutines
- **How**: RWMutex for reads, Mutex for writes
- **Pattern**: Lock before access, defer unlock

### 5. Event-Based Logging

- **What**: Every log includes an event name
- **Why**: Makes logs searchable and filterable
- **How**: `"event", "event_name"` in log fields
- **Pattern**: `{entity}_{action}` naming convention

### 6. Encapsulation

- **What**: Return copies instead of pointers to internal state
- **Why**: Prevent external mutation, maintain encapsulation
- **How**: Copy struct before returning
- **Pattern**: Small cost for large safety benefit

## 📝 Implementation Details

### Logger Initialization

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
```

**Why text handler?**
- Easier to read during development
- Can switch to JSON in production
- Good for learning

### Metric Store

```go
type InMemoryMetricStore struct {
    mu      sync.RWMutex
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Return a copy to prevent external mutation
    m := *s.metrics
    return &m, nil
}
```

**Key points:**
- RWMutex allows concurrent reads
- Returns copy to prevent mutation
- Context checked for cancellation

### Structured Logging

```go
logger.Info("Job created", "event", "job_created", "job_id", jobID)
logger.Info("Job started", "event", "job_started", "worker_id", workerID, "job_id", jobID)
logger.Info("Job completed", "event", "job_completed", "worker_id", workerID, "job_id", jobID)
```

**Key points:**
- Event name in every log
- Consistent field naming (snake_case)
- Relevant context included

### Metrics Updates

```go
// In handler
h.metricStore.IncrementJobsCreated(ctx)

// In worker
w.metricStore.IncrementJobsInProgress(ctx)
// ... process job ...
w.metricStore.IncrementJobsCompleted(ctx)
// JobsInProgress decremented inside
```

**Key points:**
- Handlers and workers call metric store
- Store handles all updates
- Thread-safe operations

### Metrics Endpoint

```go
func (h *MetricHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    metrics, err := h.metricStore.GetMetrics(r.Context())
    if err != nil {
        ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
        return
    }
    
    response := MetricResponse{
        TotalJobsCreated: metrics.TotalJobsCreated,
        JobsCompleted:    metrics.JobsCompleted,
        JobsFailed:       metrics.JobsFailed,
        JobsRetried:      metrics.JobsRetried,
        JobsInProgress:   metrics.JobsInProgress,
    }
    
    responseBytes, _ := json.Marshal(response)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(responseBytes)
}
```

**Key points:**
- Handler separated from metrics logic
- JSON response format
- Error handling

## 🎓 Learning Resources

Detailed explanations of all concepts are available in the [`concepts/`](./concepts/) directory:

1. **[Structured Logging with slog](./concepts/01-structured-logging-slog.md)** - Structured logging basics
2. **[Metrics Collection and Storage](./concepts/02-metrics-collection-storage.md)** - Metrics tracking
3. **[Dependency Injection for Observability](./concepts/03-dependency-injection-observability.md)** - Dependency injection pattern
4. **[Concurrency-Safe Metrics](./concepts/04-concurrency-safe-metrics.md)** - Thread safety
5. **[Event-Based Logging Pattern](./concepts/05-event-based-logging.md)** - Event naming
6. **[Metrics Endpoint Design](./concepts/06-metrics-endpoint-design.md)** - HTTP endpoint
7. **[Encapsulation: Returning Copies](./concepts/07-encapsulation-returning-copies.md)** - Encapsulation best practices

## 🚀 Running the Service

### Build

```bash
go build -o bin/server ./cmd/server
```

### Run

```bash
# Default settings
go run ./cmd/server

# Custom configuration
PORT=3000 WORKER_COUNT=20 JOB_QUEUE_CAPACITY=200 go run ./cmd/server
```

### Test Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Create a job
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "notification", "payload": {}}'

# List all jobs
curl http://localhost:8080/jobs

# Get metrics
curl http://localhost:8080/metrics
```

### Observing Logs

**Structured log output:**

```
time=2024-01-15T10:30:45Z level=INFO msg="Job created" event=job_created job_id=abc123
time=2024-01-15T10:30:45Z level=INFO msg="Job enqueued" event=job_enqueued job_id=abc123
time=2024-01-15T10:30:46Z level=INFO msg="Job started" event=job_started worker_id=3 job_id=abc123
time=2024-01-15T10:30:47Z level=INFO msg="Job completed" event=job_completed worker_id=3 job_id=abc123
```

**Searching logs:**

```bash
# Find all job created events
grep "event=job_created" logs.txt

# Find all events for a specific job
grep "job_id=abc123" logs.txt

# Count job completions
grep "event=job_completed" logs.txt | wc -l
```

### Observing Metrics

**Metrics response:**

```json
{
  "total_jobs_created": 120,
  "jobs_completed": 110,
  "jobs_failed": 5,
  "jobs_retried": 10,
  "jobs_in_progress": 2
}
```

**Insights:**
- 120 jobs created
- 110 completed (91.7% success rate)
- 5 failed (4.2% failure rate)
- 10 retried
- 2 currently processing

## 📋 Quick Reference Checklist

### Structured Logging

- ✅ Logger injected (no globals)
- ✅ Event names in all logs
- ✅ Consistent field naming (snake_case)
- ✅ Structured format (key-value pairs)

### Metrics

- ✅ Metrics store injected (no globals)
- ✅ Concurrency-safe (mutex protection)
- ✅ Metrics logic separated from handlers
- ✅ All required metrics tracked

### Dependency Injection

- ✅ Logger created in `main()`
- ✅ Metric store created in `main()`
- ✅ Dependencies passed to all components
- ✅ No global state

### Concurrency Safety

- ✅ Mutex protection for metrics
- ✅ RWMutex for read operations
- ✅ Returning copies to prevent mutation
- ✅ No race conditions

## 🔄 Event Catalog

### Job Lifecycle Events

- `job_created` - Job was created
- `job_enqueued` - Job was added to queue
- `job_started` - Worker started processing job
- `job_completed` - Job completed successfully
- `job_failed` - Job failed
- `job_retried` - Job was retried

### Worker Events

- `worker_started` - Worker started
- `worker_stopped` - Worker stopped

### System Events

- `sweeper_stopped` - Sweeper stopped
- `job_enqueue_failed` - Failed to enqueue job
- `job_claim_error` - Error claiming job
- `job_update_error` - Error updating job
- `metric_error` - Error updating metrics
- `sweeper_error` - Sweeper error

## 🎯 Design Decisions

### Why Structured Logging?

- **Easy to parse**: Key-value pairs are machine-readable
- **Easy to search**: Filter by event type or field
- **Consistent**: All logs follow same structure
- **Better tools**: Works with log aggregation tools

### Why Metrics Store?

- **Separation of concerns**: Metrics logic separate from handlers
- **Centralized**: All metrics in one place
- **Testable**: Easy to mock for tests
- **Flexible**: Easy to swap implementations

### Why Event Names?

- **Searchable**: Filter by event type
- **Aggregatable**: Count events by type
- **Stable**: Event name doesn't change even if message does
- **Consistent**: Same event name used everywhere

### Why Return Copies?

- **Encapsulation**: Prevents external mutation
- **Safety**: Internal state always protected
- **Small cost**: Copying small structs is cheap
- **Best practice**: Maintains encapsulation

### Why Dependency Injection?

- **No globals**: Avoids global state
- **Testable**: Easy to inject test dependencies
- **Clear**: Explicit dependencies
- **Flexible**: Easy to configure

## 🔄 Future Improvements

Potential enhancements for future tasks:

- Prometheus metrics format
- OpenTelemetry integration
- Distributed tracing
- Log aggregation (ELK, Loki)
- Metrics persistence
- Advanced observability patterns
- Log levels configuration
- Metrics histograms
- Performance metrics (latency, throughput)

## 📚 Additional Notes

- **Go version**: 1.21+ (for `slog` support)
- **Dependencies**: Standard library only (`log/slog`)
- **Project structure**: Follows Go best practices
- **Code style**: Idiomatic Go patterns
- **Concurrency**: Safe for concurrent access
- **Storage**: In-memory (temporary, lost on restart)

## ⚠️ Critical Bugs Avoided

### 1. Global Logger
- **Bug**: Using global logger
- **Fix**: Inject logger as dependency
- **Impact**: Hard to test, hidden dependencies

### 2. Returning Pointer to Internal State
- **Bug**: Returns pointer to internal metrics
- **Fix**: Return copy of metrics
- **Impact**: External code can mutate internal state

### 3. Updating Metrics from Handlers
- **Bug**: Handler directly updates metrics
- **Fix**: Handler calls metric store
- **Impact**: No concurrency protection, violates separation

### 4. Closure Variable Capture
- **Bug**: Closure captures loop variable
- **Fix**: Capture loop variable explicitly
- **Impact**: All workers get same ID

### 5. Inconsistent Field Naming
- **Bug**: Mixed camelCase and snake_case
- **Fix**: Consistent snake_case
- **Impact**: Hard to search and filter

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

