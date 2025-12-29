# Understanding Dependency Injection for Observability

## Table of Contents

1. [Why Dependency Injection?](#why-dependency-injection)
2. [The Problem with Globals](#the-problem-with-globals)
3. [Constructor Injection Pattern](#constructor-injection-pattern)
4. [Passing Dependencies Through Layers](#passing-dependencies-through-layers)
5. [Our Implementation](#our-implementation)
6. [Testing with Injected Dependencies](#testing-with-injected-dependencies)
7. [Common Mistakes](#common-mistakes)

---

## Why Dependency Injection?

### The Problem: Hidden Dependencies

**Without dependency injection:**

```go
var logger = slog.Default()  // Global

func handler() {
    logger.Info("message")  // Where does logger come from?
}
```

**Problems:**

1. **Hidden dependency** - Not clear function needs logger
2. **Hard to test** - Can't replace logger in tests
3. **Global state** - Violates Go best practices
4. **Tight coupling** - Code depends on global

### The Solution: Dependency Injection

**With dependency injection:**

```go
type Handler struct {
    logger *slog.Logger  // Explicit dependency
}

func NewHandler(logger *slog.Logger) *Handler {
    return &Handler{logger: logger}
}

func (h *Handler) handler() {
    h.logger.Info("message")  // Clear where logger comes from
}
```

**Benefits:**

1. **Explicit dependency** - Clear function needs logger
2. **Easy to test** - Can inject test logger
3. **No global state** - Each component has its own logger
4. **Loose coupling** - Code depends on interface, not implementation

---

## The Problem with Globals

### Global Logger Example

```go
var logger = slog.Default()  // Global state

func CreateJob() {
    logger.Info("Job created")  // Uses global
}
```

### Problems with Globals

**1. Hidden Dependencies**

```go
func CreateJob() {
    logger.Info("Job created")  // Where does logger come from?
}
```

**Problem:** Not clear function needs logger. Must read code to discover dependency.

**2. Hard to Test**

```go
func TestCreateJob(t *testing.T) {
    // How do we test with a test logger?
    // Global logger is hardcoded!
    CreateJob()
}
```

**Problem:** Can't replace logger in tests. Must use global logger.

**3. Global State**

```go
var logger = slog.Default()  // Shared by all code

func handler1() {
    logger.Info("message 1")  // Uses global
}

func handler2() {
    logger.Info("message 2")  // Uses same global
}
```

**Problem:** All code shares same logger. Can't have different loggers for different components.

**4. Tight Coupling**

```go
var logger = slog.Default()  // Concrete type

func handler() {
    logger.Info("message")  // Coupled to slog.Default()
}
```

**Problem:** Code is coupled to specific logger implementation.

---

## Constructor Injection Pattern

### What is Constructor Injection?

**Constructor injection** means passing dependencies when creating an object.

**Pattern:**

```go
type Component struct {
    dependency DependencyType
}

func NewComponent(dependency DependencyType) *Component {
    return &Component{dependency: dependency}
}
```

### Our Logger Injection

**Handler with injected logger:**

```go
type JobHandler struct {
    logger *slog.Logger  // Injected dependency
}

func NewJobHandler(logger *slog.Logger) *JobHandler {
    return &JobHandler{logger: logger}
}

func (h *JobHandler) CreateJob() {
    h.logger.Info("Job created", "event", "job_created")
}
```

**Key points:**

1. **Logger is a field** - Stored in struct
2. **Passed in constructor** - `NewJobHandler(logger)`
3. **Used in methods** - `h.logger.Info(...)`

### Multiple Dependencies

**Handler with multiple dependencies:**

```go
type JobHandler struct {
    store       store.JobStore
    metricStore store.MetricStore
    logger      *slog.Logger
    jobQueue    chan string
}

func NewJobHandler(
    store store.JobStore,
    metricStore store.MetricStore,
    logger *slog.Logger,
    jobQueue chan string,
) *JobHandler {
    return &JobHandler{
        store:       store,
        metricStore: metricStore,
        logger:      logger,
        jobQueue:    jobQueue,
    }
}
```

**Benefits:**

- All dependencies explicit
- Easy to see what handler needs
- Easy to test (inject mocks)

---

## Passing Dependencies Through Layers

### The Dependency Flow

**In main.go:**

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
metricStore := store.NewInMemoryMetricStore()

// Pass to handlers
jobHandler := internalhttp.NewJobHandler(jobStore, metricStore, logger, jobQueue)

// Pass to workers
worker := worker.NewWorker(workerID, jobStore, metricStore, logger, jobQueue)

// Pass to sweeper
sweeper := store.NewInMemorySweeper(jobStore, metricStore, logger, interval, jobQueue)
```

**Flow:**

```
main()
  ├─> Creates logger
  ├─> Creates metricStore
  │
  ├─> Passes to JobHandler
  │   └─> Handler uses logger and metricStore
  │
  ├─> Passes to Worker
  │   └─> Worker uses logger and metricStore
  │
  └─> Passes to Sweeper
      └─> Sweeper uses logger and metricStore
```

### Why This Pattern?

**1. Single Source of Truth**

```go
// In main.go - create once
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
```

**All components use same logger instance.**

**2. Easy to Configure**

```go
// In main.go - configure once
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

**All components automatically use configured logger.**

**3. Easy to Test**

```go
// In test - inject test logger
testLogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
handler := NewJobHandler(store, metricStore, testLogger, jobQueue)
```

**Can inject different logger for tests.**

---

## Our Implementation

### Main Function

```go
func main() {
    // Create dependencies once
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    metricStore := store.NewInMemoryMetricStore()
    
    // Inject into handlers
    jobHandler := internalhttp.NewJobHandler(jobStore, metricStore, logger, jobQueue)
    metricHandler := internalhttp.NewMetricHandler(metricStore, logger)
    
    // Inject into workers
    for i := 0; i < config.WorkerCount; i++ {
        worker := worker.NewWorker(i, jobStore, metricStore, logger, jobQueue)
        // ...
    }
    
    // Inject into sweeper
    sweeper := store.NewInMemorySweeper(jobStore, metricStore, logger, interval, jobQueue)
}
```

**Key points:**

- Dependencies created in `main()`
- Passed to all components
- Single logger instance shared

### Handler Implementation

```go
type JobHandler struct {
    store       store.JobStore
    metricStore store.MetricStore
    logger      *slog.Logger
    jobQueue    chan string
}

func NewJobHandler(
    store store.JobStore,
    metricStore store.MetricStore,
    logger *slog.Logger,
    jobQueue chan string,
) *JobHandler {
    return &JobHandler{
        store:       store,
        metricStore: metricStore,
        logger:      logger,
        jobQueue:    jobQueue,
    }
}

func (h *JobHandler) CreateJob(...) {
    h.logger.Info("Job created", "event", "job_created", "job_id", jobID)
    h.metricStore.IncrementJobsCreated(ctx)
}
```

**Key points:**

- Dependencies stored as fields
- Passed in constructor
- Used in methods

### Worker Implementation

```go
type Worker struct {
    id          int
    jobStore    store.JobStore
    metricStore store.MetricStore
    logger      *slog.Logger
    jobQueue    chan string
}

func NewWorker(
    id int,
    jobStore store.JobStore,
    metricStore store.MetricStore,
    logger *slog.Logger,
    jobQueue chan string,
) *Worker {
    return &Worker{
        id:          id,
        jobStore:    jobStore,
        metricStore: metricStore,
        logger:      logger,
        jobQueue:    jobQueue,
    }
}

func (w *Worker) processJob(...) {
    w.logger.Info("Job started", "event", "job_started", "worker_id", w.id, "job_id", jobID)
    w.metricStore.IncrementJobsInProgress(ctx)
}
```

**Key points:**

- Same pattern as handler
- Dependencies injected
- Used throughout worker

---

## Testing with Injected Dependencies

### The Problem with Globals in Tests

**With global logger:**

```go
var logger = slog.Default()  // Global

func TestCreateJob(t *testing.T) {
    // Can't replace logger!
    // Must use global logger
    handler := NewJobHandler(store, jobQueue)
    handler.CreateJob()
}
```

**Problem:** Can't control logger in tests.

### The Solution: Injected Dependencies

**With injected logger:**

```go
func TestCreateJob(t *testing.T) {
    // Create test logger
    testLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,  // Can configure for tests
    }))
    
    // Inject test logger
    handler := NewJobHandler(store, metricStore, testLogger, jobQueue)
    handler.CreateJob()
    
    // Can verify logs if needed
}
```

**Benefits:**

- Can use different logger in tests
- Can configure logger for tests
- Can capture logs for verification

### Mock Dependencies

**With interfaces, can inject mocks:**

```go
type MockMetricStore struct {
    CreatedCount int
}

func (m *MockMetricStore) IncrementJobsCreated(ctx context.Context) error {
    m.CreatedCount++
    return nil
}

func TestCreateJob(t *testing.T) {
    mockMetricStore := &MockMetricStore{}
    handler := NewJobHandler(store, mockMetricStore, logger, jobQueue)
    
    handler.CreateJob()
    
    if mockMetricStore.CreatedCount != 1 {
        t.Errorf("Expected 1, got %d", mockMetricStore.CreatedCount)
    }
}
```

**Benefits:**

- Can verify metric updates
- Can test error cases
- No need for real metric store

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

func NewHandler(logger *slog.Logger) *Handler {
    return &Handler{logger: logger}
}
```

### Mistake 2: Creating Logger in Component

```go
// ❌ BAD: Component creates its own logger
type Handler struct {
    logger *slog.Logger
}

func NewHandler() *Handler {
    return &Handler{
        logger: slog.Default(),  // Creates logger inside
    }
}
```

**Problems:**

- Can't configure logger
- Can't use different logger in tests
- Not flexible

**Fix:** Pass logger as parameter

```go
// ✅ GOOD: Logger passed as parameter
func NewHandler(logger *slog.Logger) *Handler {
    return &Handler{logger: logger}
}
```

### Mistake 3: Not Passing Logger to All Components

```go
// ❌ BAD: Some components don't have logger
type Handler struct {
    store store.JobStore
    // Missing logger!
}

func (h *Handler) CreateJob() {
    log.Printf("Job created")  // Uses global log
}
```

**Fix:** Pass logger to all components

```go
// ✅ GOOD: Logger passed to all components
type Handler struct {
    store  store.JobStore
    logger *slog.Logger
}

func (h *Handler) CreateJob() {
    h.logger.Info("Job created", "event", "job_created")
}
```

### Mistake 4: Passing nil Logger

```go
// ❌ BAD: Passing nil
handler := NewJobHandler(store, metricStore, nil, jobQueue)
```

**Problem:** Will panic when trying to use logger.

**Fix:** Always pass valid logger

```go
// ✅ GOOD: Pass valid logger
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
handler := NewJobHandler(store, metricStore, logger, jobQueue)
```

### Mistake 5: Not Using Injected Logger

```go
// ❌ BAD: Injected logger but not using it
type Handler struct {
    logger *slog.Logger
}

func (h *Handler) CreateJob() {
    log.Printf("Job created")  // Not using h.logger!
}
```

**Fix:** Use injected logger

```go
// ✅ GOOD: Use injected logger
func (h *Handler) CreateJob() {
    h.logger.Info("Job created", "event", "job_created")
}
```

---

## Key Takeaways

1. **Dependency injection** = Pass dependencies as parameters
2. **No globals** = All dependencies injected
3. **Constructor pattern** = Dependencies passed in `New*` function
4. **Explicit dependencies** = Clear what component needs
5. **Easy to test** = Can inject mocks or test implementations
6. **Single source** = Create dependencies once in `main()`
7. **Pass through layers** = Dependencies flow from `main()` to components

---

## Real-World Example

**Our dependency flow:**

```
main()
  │
  ├─> logger := slog.New(...)
  ├─> metricStore := store.NewInMemoryMetricStore()
  │
  ├─> jobHandler := NewJobHandler(store, metricStore, logger, jobQueue)
  │   └─> Uses logger and metricStore
  │
  ├─> worker := NewWorker(id, store, metricStore, logger, jobQueue)
  │   └─> Uses logger and metricStore
  │
  └─> sweeper := NewInMemorySweeper(store, metricStore, logger, interval, jobQueue)
      └─> Uses logger and metricStore
```

**Benefits:**

- All components use same logger instance
- All components use same metric store instance
- Easy to configure (change in one place)
- Easy to test (inject test dependencies)

---

## Next Steps

- Read [Concurrency-Safe Metrics](./04-concurrency-safe-metrics.md) to understand thread safety
- Read [Structured Logging with slog](./01-structured-logging-slog.md) to see how we use the injected logger
- Read [Metrics Collection and Storage](./02-metrics-collection-storage.md) to see how we use the injected metric store

