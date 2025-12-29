# Task 7 Summary

## What We Built

Task 7 introduced **structured logging and metrics** to the job queue system, making it observable and debuggable under load. This task focuses on signal over noise, consistent log structure, and clear operational insight.

## Key Changes

### 1. Structured Logging with slog

**Before (Task 6):**
- Unstructured logs using `log.Printf()`
- Hard to parse and search
- No consistent format

**After (Task 7):**
- Structured logs using `log/slog`
- Key-value pairs for easy parsing
- Consistent field naming (snake_case)
- Event-based logging with event names

### 2. Metrics Collection and Storage

**New Component:** `InMemoryMetricStore`

- Tracks job lifecycle metrics (created, completed, failed, retried)
- Tracks current system state (jobs in progress)
- Concurrency-safe with mutex protection
- In-memory storage (for now)

### 3. Metrics Endpoint

**New Endpoint:** `GET /metrics`

- Exposes metrics as JSON
- Handler separated from metrics logic
- Standard HTTP response format

### 4. Dependency Injection for Observability

**Logger and metrics injected everywhere:**
- No global loggers
- No global metric stores
- Dependencies passed through constructors
- Easy to test with mocks

### 5. Event-Based Logging

**Every log includes event name:**
- `job_created`, `job_enqueued`, `job_started`
- `job_completed`, `job_failed`, `job_retried`
- `worker_started`, `worker_stopped`
- Makes logs searchable and filterable

## Files Changed

### New Files

- `internal/domain/metric.go` - Metric domain model
- `internal/store/metric_store.go` - Metrics storage implementation
- `internal/http/metric_handler.go` - Metrics HTTP endpoint
- `docs/task7/concepts/` - Learning documents

### Modified Files

- `cmd/server/main.go` - Logger and metric store initialization, dependency injection
- `internal/http/job_handler.go` - Added logging and metrics updates
- `internal/worker/worker.go` - Added logging and metrics updates
- `internal/store/job_store.go` - Added logging for retries
- `internal/store/sweeper.go` - Added structured logging

## Key Concepts Learned

### 1. Structured Logging

- Use `log/slog` for structured logs
- Always inject logger (no globals)
- Include event names in logs
- Use consistent field naming (snake_case)
- Log what matters, not everything

### 2. Metrics Collection

- Track counters (jobs_created, jobs_completed)
- Track gauges (jobs_in_progress)
- Store metrics in memory (for now)
- Make metrics concurrency-safe
- Separate metrics logic from handlers

### 3. Dependency Injection

- Pass logger and metrics as constructor parameters
- No global state
- Easy to test
- Clear dependencies

### 4. Concurrency Safety

- Use mutexes to protect metrics
- Return copies to prevent external mutation
- RWMutex for read-heavy operations

### 5. Event-Based Logging

- Every log has an event name
- Consistent event naming
- Makes logs searchable and filterable

### 6. Encapsulation

- Return copies instead of pointers to internal state
- Prevents external mutation
- Protects internal state

## Critical Bugs Avoided

### 1. Global Logger

```go
// ❌ BAD: Global logger
var logger = slog.Default()

func handler() {
    logger.Info("message")
}

// ✅ GOOD: Injected logger
type Handler struct {
    logger *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
    return &Handler{logger: logger}
}
```

### 2. Returning Pointer to Internal State

```go
// ❌ BAD: Returns pointer to internal state
func (s *MetricStore) GetMetrics() *Metric {
    return s.metrics  // External code can mutate!
}

// ✅ GOOD: Returns copy
func (s *MetricStore) GetMetrics(ctx context.Context) (*Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    m := *s.metrics  // Copy
    return &m, nil
}
```

### 3. Updating Metrics from Handlers

```go
// ❌ BAD: Handler directly updates metrics
func (h *Handler) CreateJob() {
    h.metrics.JobsCreated++  // Direct mutation!
}

// ✅ GOOD: Handler calls metric store
func (h *Handler) CreateJob() {
    h.metricStore.IncrementJobsCreated(ctx)  // Store handles it
}
```

### 4. Closure Variable Capture

```go
// ❌ BAD: Closure captures loop variable
for i := 0; i < 10; i++ {
    wg.Go(func() {
        logger.Info("Worker started", "worker_id", i)  // All get 10!
    })
}

// ✅ GOOD: Capture loop variable
for i := 0; i < 10; i++ {
    workerID := i
    wg.Go(func() {
        logger.Info("Worker started", "worker_id", workerID)
    })
}
```

### 5. Inconsistent Field Naming

```go
// ❌ BAD: Mixed naming
logger.Info("Job created", "jobId", id, "worker_id", wid)

// ✅ GOOD: Consistent snake_case
logger.Info("Job created", "event", "job_created", "job_id", id, "worker_id", wid)
```

## Performance Impact

**Logging:**
- Structured logs are slightly slower than unstructured
- Worth it for observability benefits
- Can switch to JSON handler in production

**Metrics:**
- Minimal overhead (mutex locks are fast)
- In-memory storage is very fast
- No I/O operations

## Design Decisions

### Why Structured Logging?

- Easy to parse and search
- Machine-readable format
- Consistent structure
- Better for log aggregation tools

### Why Metrics Store?

- Separates metrics logic from handlers
- Centralized metric management
- Easy to swap implementations
- Testable with mocks

### Why Event Names?

- Makes logs searchable
- Easy to filter by event type
- Consistent naming
- Better for aggregation

### Why Return Copies?

- Prevents external mutation
- Protects internal state
- Maintains encapsulation
- Small cost for large benefit

### Why Dependency Injection?

- No global state
- Easy to test
- Clear dependencies
- Flexible configuration

## Testing Considerations

When testing observability:

- Test with injected test logger
- Test metrics updates
- Test concurrency safety
- Test error logging
- Test metrics endpoint

## Next Steps

After Task 7, you're ready for:

- Prometheus metrics format
- OpenTelemetry integration
- Distributed tracing
- Log aggregation (ELK, Loki)
- Metrics persistence
- Advanced observability patterns

## Key Takeaways

1. **Structured logging** = Key-value pairs, not free-form text
2. **Metrics** = Numerical measurements of system behavior
3. **Dependency injection** = Pass dependencies, don't use globals
4. **Concurrency safety** = Mutex protection for shared state
5. **Event-based logging** = Every log has an event name
6. **Encapsulation** = Return copies to prevent external mutation
7. **Observability** = Making systems explain themselves

## Learning Resources

See `docs/task7/concepts/` for detailed explanations:

- [Structured Logging with slog](./concepts/01-structured-logging-slog.md)
- [Metrics Collection and Storage](./concepts/02-metrics-collection-storage.md)
- [Dependency Injection for Observability](./concepts/03-dependency-injection-observability.md)
- [Concurrency-Safe Metrics](./concepts/04-concurrency-safe-metrics.md)
- [Event-Based Logging Pattern](./concepts/05-event-based-logging.md)
- [Metrics Endpoint Design](./concepts/06-metrics-endpoint-design.md)
- [Encapsulation: Returning Copies](./concepts/07-encapsulation-returning-copies.md)

