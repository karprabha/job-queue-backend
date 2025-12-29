# Go Concepts Explained - Task 7

This directory contains detailed explanations of Go concepts used in Task 7, written for beginners learning observability (structured logging and metrics) in Go.

## 📚 Concepts Covered

### 1. [Structured Logging with slog](./01-structured-logging-slog.md)

- Why structured logging?
- What is slog?
- Structured vs unstructured logs
- Logger injection (no globals)
- Event-based logging
- Consistent field naming
- Common mistakes

### 2. [Metrics Collection and Storage](./02-metrics-collection-storage.md)

- Why metrics matter
- What metrics to track
- In-memory metrics store
- Metrics vs logs
- Gauge vs counter metrics
- Common mistakes

### 3. [Dependency Injection for Observability](./03-dependency-injection-observability.md)

- Why inject logger and metrics?
- No global state principle
- Constructor injection pattern
- Passing dependencies through layers
- Testing with injected dependencies
- Common mistakes

### 4. [Concurrency-Safe Metrics](./04-concurrency-safe-metrics.md)

- Why metrics need to be thread-safe
- Mutex protection for metrics
- Read-write locks (RWMutex)
- Atomic operations considerations
- Returning copies vs pointers
- Common mistakes

### 5. [Event-Based Logging Pattern](./05-event-based-logging.md)

- Why event names matter
- Event field in logs
- Consistent event naming
- What to log vs what not to log
- Log levels (Info, Error, Debug)
- Common mistakes

### 6. [Metrics Endpoint Design](./06-metrics-endpoint-design.md)

- Why expose metrics via HTTP?
- JSON response format
- Handler separation from metrics logic
- Error handling in metrics endpoint
- Common mistakes

### 7. [Encapsulation: Returning Copies](./07-encapsulation-returning-copies.md)

- Why return copies instead of pointers?
- Preventing external mutation
- Memory implications
- When to copy vs when to share
- Common mistakes

## 🎯 How to Use This

These documents are designed to be read **in order** if you're new to observability in Go. Each concept builds on previous ones.

**Recommended reading order:**

1. Start with [Structured Logging with slog](./01-structured-logging-slog.md) - Foundation for all logging
2. Then [Metrics Collection and Storage](./02-metrics-collection-storage.md) - How we track metrics
3. Then [Dependency Injection for Observability](./03-dependency-injection-observability.md) - How we wire dependencies
4. Then [Concurrency-Safe Metrics](./04-concurrency-safe-metrics.md) - How we ensure thread safety
5. Then [Event-Based Logging Pattern](./05-event-based-logging.md) - How we structure logs
6. Then [Metrics Endpoint Design](./06-metrics-endpoint-design.md) - How we expose metrics
7. Finally [Encapsulation: Returning Copies](./07-encapsulation-returning-copies.md) - Design best practice

Or read them as you encounter concepts in the code!

## 💡 Learning Approach

Each document:

- Explains **why** things exist (not just what they do)
- Breaks down code **line by line**
- Uses **analogies** and **mental models**
- Shows **common mistakes** to avoid
- Provides **real examples** from our codebase
- Explains **design decisions** and trade-offs

## 🔗 Related Resources

### Task 6 Concepts

- [State Machine Transitions](../task6/concepts/01-state-machine-transitions.md)
- [Failure Handling](../task6/concepts/02-failure-handling.md)
- [Retry Logic](../task6/concepts/03-retry-logic-attempts.md)

### Task 5 Concepts

- [Worker Pools](../task5/concepts/01-worker-pools.md)
- [Preventing Duplicate Processing](../task5/concepts/02-preventing-duplicate-processing.md)

### Task 3 Concepts

- [Concurrency Safety](../task3/concepts/04-concurrency-safety.md)
- [RWMutex vs Mutex](../task3/concepts/05-rwmutex-vs-mutex.md)

### External Resources

- [Go slog Package Documentation](https://pkg.go.dev/log/slog)
- [Go Official Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [The Twelve-Factor App: Logs](https://12factor.net/logs)

## 📝 Key Concepts Summary

### Structured Logging

- Use `log/slog` for structured logs
- Always inject logger (no globals)
- Include event names in logs
- Use consistent field naming (snake_case)
- Log what matters, not everything

### Metrics

- Track counters and gauges
- Store metrics in memory (for now)
- Make metrics concurrency-safe
- Separate metrics logic from handlers
- Expose metrics via HTTP endpoint

### Dependency Injection

- Pass logger and metrics as constructor parameters
- No global state
- Easy to test
- Clear dependencies

### Concurrency Safety

- Use mutexes to protect metrics
- Return copies to prevent external mutation
- RWMutex for read-heavy operations

### Event-Based Logging

- Every log has an event name
- Consistent event naming
- Makes logs searchable and filterable

## 🎓 What You'll Learn

After reading these documents, you'll understand:

- ✅ How to use structured logging with slog
- ✅ How to collect and store metrics
- ✅ How to inject dependencies properly
- ✅ How to make metrics thread-safe
- ✅ How to design event-based logs
- ✅ How to expose metrics via HTTP
- ✅ Why encapsulation matters (returning copies)
- ✅ What to log vs what not to log

## 🚀 Next Steps

After Task 7, you'll be ready for:

- Prometheus metrics format
- OpenTelemetry integration
- Distributed tracing
- Log aggregation (ELK, Loki)
- Metrics persistence
- Advanced observability patterns

## ⚠️ Critical Bugs to Avoid

### 1. Global Logger

```go
// ❌ BAD: Global logger
var logger = slog.Default()

func handler() {
    logger.Info("message")  // Global state!
}

// ✅ GOOD: Injected logger
type Handler struct {
    logger *slog.Logger
}

func (h *Handler) handler() {
    h.logger.Info("message")  // Injected dependency
}
```

### 2. Unstructured Logs

```go
// ❌ BAD: Unstructured string
log.Printf("Job %s created by worker %d", jobID, workerID)

// ✅ GOOD: Structured with fields
logger.Info("Job created", "event", "job_created", "job_id", jobID, "worker_id", workerID)
```

### 3. Returning Pointer to Internal State

```go
// ❌ BAD: External code can mutate internal state
func (s *MetricStore) GetMetrics() *Metric {
    return s.metrics  // Returns pointer to internal state!
}

// ✅ GOOD: Return a copy
func (s *MetricStore) GetMetrics() *Metric {
    s.mu.RLock()
    defer s.mu.RUnlock()
    m := *s.metrics  // Copy
    return &m
}
```

### 4. Updating Metrics from Handlers

```go
// ❌ BAD: Handler directly updates metrics
func (h *Handler) CreateJob() {
    h.metrics.JobsCreated++  // Handler updates metrics!
}

// ✅ GOOD: Handler calls metric store
func (h *Handler) CreateJob() {
    h.metricStore.IncrementJobsCreated(ctx)  // Store updates metrics
}
```

### 5. Inconsistent Field Naming

```go
// ❌ BAD: Mixed naming
logger.Info("Job created", "jobId", id, "worker_id", wid)  // Mixed camelCase and snake_case

// ✅ GOOD: Consistent snake_case
logger.Info("Job created", "event", "job_created", "job_id", id, "worker_id", wid)
```

## 📝 Contributing

If you find something unclear or want to add explanations, feel free to update these documents!

