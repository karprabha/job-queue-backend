# Understanding Metrics Collection and Storage

## Table of Contents

1. [Why Metrics?](#why-metrics)
2. [Metrics vs Logs](#metrics-vs-logs)
3. [What Metrics to Track](#what-metrics-to-track)
4. [Counter vs Gauge Metrics](#counter-vs-gauge-metrics)
5. [In-Memory Metrics Store](#in-memory-metrics-store)
6. [Metrics Domain Model](#metrics-domain-model)
7. [Incrementing Metrics](#incrementing-metrics)
8. [Common Mistakes](#common-mistakes)

---

## Why Metrics?

### The Problem Without Metrics

**Scenario:** Your job queue is running, but you don't know:

- How many jobs have been created?
- How many jobs completed successfully?
- How many jobs failed?
- How many jobs are currently processing?
- Is the system healthy?

**Without metrics, you're blind.**

### The Solution: Metrics

**Metrics** are numerical measurements of system behavior over time.

**With metrics, you can answer:**

- ✅ How many jobs created? → `jobs_created: 120`
- ✅ How many completed? → `jobs_completed: 110`
- ✅ How many failed? → `jobs_failed: 5`
- ✅ How many in progress? → `jobs_in_progress: 2`
- ✅ Success rate? → `110 / 120 = 91.7%`

### Real-World Analogy

Think of metrics like a car dashboard:

- **Speedometer** = How fast are you going? (jobs_in_progress)
- **Odometer** = Total distance traveled (jobs_created)
- **Fuel gauge** = Current fuel level (jobs_in_progress)
- **Warning lights** = Problems detected (jobs_failed)

**Metrics tell you the health and performance of your system.**

---

## Metrics vs Logs

### Logs: What Happened

**Logs** answer: **"What happened?"**

**Example:**
```
time=2024-01-15T10:30:45Z level=INFO event=job_created job_id=abc123
time=2024-01-15T10:30:46Z level=INFO event=job_started worker_id=3 job_id=abc123
time=2024-01-15T10:30:47Z level=INFO event=job_completed worker_id=3 job_id=abc123
```

**Characteristics:**

- Individual events
- Detailed information
- High volume (many log entries)
- Good for debugging specific issues

### Metrics: How Many / How Much

**Metrics** answer: **"How many?" or "How much?"**

**Example:**
```json
{
  "jobs_created": 120,
  "jobs_completed": 110,
  "jobs_failed": 5,
  "jobs_retried": 10,
  "jobs_in_progress": 2
}
```

**Characteristics:**

- Aggregated counts
- Summary information
- Low volume (one metric set)
- Good for monitoring system health

### When to Use Logs vs Metrics

**Use logs when:**

- Debugging a specific issue
- Need detailed context
- Tracking individual events
- Investigating errors

**Use metrics when:**

- Monitoring system health
- Tracking trends over time
- Alerting on thresholds
- Performance analysis

### Example: Job Processing

**Log (individual event):**
```
time=2024-01-15T10:30:45Z level=INFO event=job_completed worker_id=3 job_id=abc123
```

**Metric (aggregated count):**
```
jobs_completed: 110
```

**Together:**

- **Logs** tell you: "Job abc123 completed by worker 3"
- **Metrics** tell you: "110 jobs have completed total"

---

## What Metrics to Track

### Our Metrics

**Job lifecycle metrics:**

1. **`jobs_created`** - Total jobs created (counter)
2. **`jobs_completed`** - Total jobs completed successfully (counter)
3. **`jobs_failed`** - Total jobs failed (counter)
4. **`jobs_retried`** - Total jobs retried (counter)
5. **`jobs_in_progress`** - Currently processing jobs (gauge)

### Why These Metrics?

**`jobs_created`** - Tracks total workload

```go
metricStore.IncrementJobsCreated(ctx)
```

**Why:** Know how many jobs the system has received.

**`jobs_completed`** - Tracks successful processing

```go
metricStore.IncrementJobsCompleted(ctx)
```

**Why:** Know how many jobs succeeded.

**`jobs_failed`** - Tracks failures

```go
metricStore.IncrementJobsFailed(ctx)
```

**Why:** Know how many jobs failed (for alerting).

**`jobs_retried`** - Tracks retry attempts

```go
metricStore.IncrementJobsRetried(ctx)
```

**Why:** Know how many retries happened.

**`jobs_in_progress`** - Current load

```go
metricStore.IncrementJobsInProgress(ctx)  // When job starts
// ... later ...
metricStore.IncrementJobsCompleted(ctx)   // Decrements in_progress
```

**Why:** Know current system load (for scaling decisions).

### Metrics We Could Add Later

- **`job_processing_duration`** - How long jobs take
- **`queue_size`** - Current queue length
- **`worker_idle_time`** - How long workers wait
- **`retry_rate`** - Percentage of jobs retried

---

## Counter vs Gauge Metrics

### Counter Metrics

**Counters** only go up (they're cumulative).

**Examples:**

- `jobs_created` - Starts at 0, only increases
- `jobs_completed` - Starts at 0, only increases
- `jobs_failed` - Starts at 0, only increases

**Characteristics:**

- Monotonically increasing
- Never decreases
- Resets only on restart
- Good for totals

**Example:**

```
Time 0:  jobs_created = 0
Time 1:  jobs_created = 5   (created 5 jobs)
Time 2:  jobs_created = 12  (created 7 more, total 12)
Time 3:  jobs_created = 20  (created 8 more, total 20)
```

**Note:** Counters never go down, only up.

### Gauge Metrics

**Gauges** can go up or down (they're current values).

**Examples:**

- `jobs_in_progress` - Can be 0, 5, 10, etc.
- `queue_size` - Can increase or decrease
- `memory_usage` - Can increase or decrease

**Characteristics:**

- Can increase or decrease
- Represents current state
- Can be reset
- Good for current values

**Example:**

```
Time 0:  jobs_in_progress = 0
Time 1:  jobs_in_progress = 3  (3 jobs started)
Time 2:  jobs_in_progress = 5  (2 more started)
Time 3:  jobs_in_progress = 2  (3 jobs completed, 2 still processing)
```

**Note:** Gauges can go up and down.

### Our Implementation

**Counters (only increment):**

```go
s.metrics.TotalJobsCreated++  // Counter: only goes up
s.metrics.JobsCompleted++      // Counter: only goes up
s.metrics.JobsFailed++         // Counter: only goes up
```

**Gauge (increment and decrement):**

```go
// When job starts
s.metrics.JobsInProgress++

// When job completes
s.metrics.JobsCompleted++
s.metrics.JobsInProgress--  // Decrement gauge
```

---

## In-Memory Metrics Store

### Why In-Memory?

**For now, we store metrics in memory:**

- Simple to implement
- Fast (no I/O)
- Good for learning
- Sufficient for single-instance systems

**Limitations:**

- Lost on restart
- Not shared across instances
- No historical data

**Later, we'll add:**

- Persistence (database)
- Distributed metrics (Prometheus)
- Historical tracking

### Store Interface

```go
type MetricStore interface {
    GetMetrics(ctx context.Context) (*domain.Metric, error)
    IncrementJobsCreated(ctx context.Context) error
    IncrementJobsCompleted(ctx context.Context) error
    IncrementJobsFailed(ctx context.Context) error
    IncrementJobsRetried(ctx context.Context) error
    IncrementJobsInProgress(ctx context.Context) error
}
```

**Why an interface?**

- Easy to test (mock implementation)
- Easy to swap implementations
- Clear contract

### Implementation

```go
type InMemoryMetricStore struct {
    mu      sync.RWMutex
    metrics *domain.Metric
}

func NewInMemoryMetricStore() *InMemoryMetricStore {
    return &InMemoryMetricStore{
        metrics: domain.NewMetric(),
    }
}
```

**Key components:**

1. **`mu sync.RWMutex`** - Protects concurrent access
2. **`metrics *domain.Metric`** - The actual metrics data
3. **Constructor** - Initializes with zero values

---

## Metrics Domain Model

### The Metric Struct

```go
type Metric struct {
    TotalJobsCreated int
    JobsCompleted    int
    JobsFailed       int
    JobsRetried      int
    JobsInProgress   int
}
```

**Why these fields?**

- **`TotalJobsCreated`** - Counter: total jobs created
- **`JobsCompleted`** - Counter: total jobs completed
- **`JobsFailed`** - Counter: total jobs failed
- **`JobsRetried`** - Counter: total jobs retried
- **`JobsInProgress`** - Gauge: current jobs processing

### Initialization

```go
func NewMetric() *Metric {
    return &Metric{
        TotalJobsCreated: 0,
        JobsCompleted:    0,
        JobsFailed:       0,
        JobsRetried:      0,
        JobsInProgress:   0,
    }
}
```

**Why explicit initialization?**

- Clear starting values
- Self-documenting
- Prevents nil pointer issues

---

## Incrementing Metrics

### Where Metrics Are Updated

**Job creation (handler):**

```go
func (h *JobHandler) CreateJob(...) {
    // ... create job ...
    h.metricStore.IncrementJobsCreated(ctx)
}
```

**Job processing (worker):**

```go
func (w *Worker) processJob(...) {
    // Job starts
    w.metricStore.IncrementJobsInProgress(ctx)
    
    // ... process ...
    
    if failed {
        w.metricStore.IncrementJobsFailed(ctx)
        // JobsInProgress decremented inside
    } else {
        w.metricStore.IncrementJobsCompleted(ctx)
        // JobsInProgress decremented inside
    }
}
```

**Job retry (sweeper):**

```go
func (s *InMemoryJobStore) RetryFailedJobs(...) {
    // ... retry job ...
    metricStore.IncrementJobsRetried(ctx)
}
```

### Increment Methods

**Simple increment (counter):**

```go
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.metrics.TotalJobsCreated++
    return nil
}
```

**Increment with decrement (gauge):**

```go
func (s *InMemoryMetricStore) IncrementJobsCompleted(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.metrics.JobsCompleted++
    s.metrics.JobsInProgress--  // Decrement gauge
    return nil
}
```

**Key points:**

- Mutex protects concurrent access
- Counters only increment
- Gauges increment and decrement
- Context checked for cancellation

---

## Common Mistakes

### Mistake 1: Updating Metrics from Handlers Directly

```go
// ❌ BAD: Handler directly updates metrics
type JobHandler struct {
    metrics *domain.Metric
}

func (h *JobHandler) CreateJob() {
    h.metrics.TotalJobsCreated++  // Direct mutation!
}
```

**Problems:**

- No concurrency protection
- Violates separation of concerns
- Hard to test

**Fix:** Use metric store

```go
// ✅ GOOD: Handler calls metric store
type JobHandler struct {
    metricStore store.MetricStore
}

func (h *JobHandler) CreateJob() {
    h.metricStore.IncrementJobsCreated(ctx)  // Store handles it
}
```

### Mistake 2: Not Protecting Concurrent Access

```go
// ❌ BAD: No mutex protection
func (s *InMemoryMetricStore) IncrementJobsCreated() {
    s.metrics.TotalJobsCreated++  // Race condition!
}
```

**Fix:** Use mutex

```go
// ✅ GOOD: Mutex protection
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.metrics.TotalJobsCreated++
    return nil
}
```

### Mistake 3: Forgetting to Decrement Gauge

```go
// ❌ BAD: Forgot to decrement
func (s *InMemoryMetricStore) IncrementJobsCompleted() {
    s.metrics.JobsCompleted++
    // Forgot: s.metrics.JobsInProgress--
}
```

**Fix:** Decrement gauge

```go
// ✅ GOOD: Decrement gauge
func (s *InMemoryMetricStore) IncrementJobsCompleted(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.metrics.JobsCompleted++
    s.metrics.JobsInProgress--  // Decrement gauge
    return nil
}
```

### Mistake 4: Returning Pointer to Internal State

```go
// ❌ BAD: Returns pointer to internal state
func (s *InMemoryMetricStore) GetMetrics() *domain.Metric {
    return s.metrics  // External code can mutate!
}
```

**Fix:** Return copy

```go
// ✅ GOOD: Returns copy
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    m := *s.metrics  // Copy
    return &m, nil
}
```

### Mistake 5: Mixing Counter and Gauge Logic

```go
// ❌ BAD: Counter that goes down
func (s *InMemoryMetricStore) IncrementJobsCreated() {
    s.metrics.TotalJobsCreated++
}

func (s *InMemoryMetricStore) DecrementJobsCreated() {
    s.metrics.TotalJobsCreated--  // Counters shouldn't go down!
}
```

**Fix:** Counters only increment

```go
// ✅ GOOD: Counter only increments
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.metrics.TotalJobsCreated++  // Only up
    return nil
}
```

---

## Key Takeaways

1. **Metrics** = Numerical measurements of system behavior
2. **Logs** = What happened (individual events)
3. **Metrics** = How many (aggregated counts)
4. **Counters** = Only go up (cumulative)
5. **Gauges** = Can go up or down (current value)
6. **In-memory store** = Simple, fast, but lost on restart
7. **Metric store** = Handles all metric updates (not handlers)
8. **Mutex protection** = Required for concurrent access
9. **Return copies** = Prevent external mutation

---

## Real-World Example

**Our metrics flow:**

1. **Job created** → `IncrementJobsCreated()`
2. **Job started** → `IncrementJobsInProgress()`
3. **Job completed** → `IncrementJobsCompleted()` + `JobsInProgress--`
4. **Job failed** → `IncrementJobsFailed()` + `JobsInProgress--`
5. **Job retried** → `IncrementJobsRetried()`

**Result:**

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

---

## Next Steps

- Read [Dependency Injection for Observability](./03-dependency-injection-observability.md) to see how we wire metrics
- Read [Concurrency-Safe Metrics](./04-concurrency-safe-metrics.md) to understand thread safety
- Read [Metrics Endpoint Design](./06-metrics-endpoint-design.md) to see how we expose metrics

