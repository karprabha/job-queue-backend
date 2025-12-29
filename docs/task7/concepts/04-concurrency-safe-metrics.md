# Understanding Concurrency-Safe Metrics

## Table of Contents

1. [Why Concurrency Safety Matters](#why-concurrency-safety-matters)
2. [The Race Condition Problem](#the-race-condition-problem)
3. [Mutex Protection](#mutex-protection)
4. [Read-Write Locks (RWMutex)](#read-write-locks-rwmutex)
5. [Returning Copies vs Pointers](#returning-copies-vs-pointers)
6. [Context in Metric Operations](#context-in-metric-operations)
7. [Common Mistakes](#common-mistakes)

---

## Why Concurrency Safety Matters

### The Problem: Concurrent Access

**Our system has multiple goroutines:**

- Multiple workers processing jobs
- HTTP handlers creating jobs
- Sweeper retrying jobs
- Metrics endpoint reading metrics

**All accessing metrics simultaneously!**

### The Race Condition

**Without protection:**

```go
type InMemoryMetricStore struct {
    metrics *domain.Metric  // No mutex!
}

func (s *InMemoryMetricStore) IncrementJobsCreated() {
    s.metrics.TotalJobsCreated++  // Race condition!
}
```

**What happens:**

```
Worker 1: Read TotalJobsCreated (value: 5)
Worker 2: Read TotalJobsCreated (value: 5)
Worker 1: Increment to 6
Worker 2: Increment to 6  // Lost update! Should be 7
```

**Result:** Lost updates, incorrect metrics.

---

## The Race Condition Problem

### Example Race Condition

**Scenario:** Two workers complete jobs simultaneously

```go
// Worker 1 goroutine
s.metrics.JobsCompleted++  // Read: 10, Write: 11

// Worker 2 goroutine (at same time)
s.metrics.JobsCompleted++  // Read: 10, Write: 11 (should be 12!)
```

**Timeline:**

```
Time 0: JobsCompleted = 10
Time 1: Worker 1 reads: 10
Time 1: Worker 2 reads: 10  (both read same value)
Time 2: Worker 1 writes: 11
Time 2: Worker 2 writes: 11  (lost update! should be 12)
```

**Result:** Metrics are incorrect.

### Detecting Race Conditions

**Go race detector:**

```bash
go run -race ./cmd/server
```

**Will detect:**

```
WARNING: DATA RACE
Read at 0x... by goroutine 5:
  IncrementJobsCompleted()

Previous write at 0x... by goroutine 3:
  IncrementJobsCompleted()
```

**Always run with `-race` flag during development!**

---

## Mutex Protection

### What is a Mutex?

**Mutex** = Mutual Exclusion Lock

**Purpose:** Only one goroutine can access protected code at a time.

**Analogy:** Like a bathroom lock - only one person can enter at a time.

### Basic Mutex Usage

```go
type InMemoryMetricStore struct {
    mu      sync.Mutex      // Mutex to protect metrics
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()         // Lock: only one goroutine can proceed
    defer s.mu.Unlock() // Unlock: when function exits

    s.metrics.TotalJobsCreated++
    return nil
}
```

**How it works:**

1. **`s.mu.Lock()`** - Acquire lock (wait if another goroutine has it)
2. **Do work** - Safely modify metrics
3. **`defer s.mu.Unlock()`** - Release lock (always, even on error)

### Why defer Unlock?

**Without defer:**

```go
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()

    if someError {
        return err  // Forgot to unlock! Deadlock!
    }

    s.metrics.TotalJobsCreated++
    s.mu.Unlock()  // Only reached if no error
    return nil
}
```

**Problem:** If error occurs, lock never released → deadlock.

**With defer:**

```go
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()  // Always unlocks, even on error

    if someError {
        return err  // Unlock still happens
    }

    s.metrics.TotalJobsCreated++
    return nil
}
```

**Benefit:** Lock always released, even on error or panic.

---

## Read-Write Locks (RWMutex)

### The Problem with Regular Mutex

**Regular mutex:** Only one goroutine can access (read or write)

```go
func (s *InMemoryMetricStore) GetMetrics() *Metric {
    s.mu.Lock()      // Blocks all other goroutines
    defer s.mu.Unlock()
    return s.metrics
}
```

**Problem:** Even reads block other reads (unnecessary).

### The Solution: RWMutex

**RWMutex:** Multiple readers OR one writer

```go
type InMemoryMetricStore struct {
    mu      sync.RWMutex  // Read-write mutex
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()      // Read lock: allows multiple readers
    defer s.mu.RUnlock()

    m := *s.metrics
    return &m, nil
}

func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()       // Write lock: exclusive access
    defer s.mu.Unlock()

    s.metrics.TotalJobsCreated++
    return nil
}
```

**How it works:**

- **`RLock()`** - Multiple goroutines can read simultaneously
- **`Lock()`** - Only one goroutine can write (blocks all readers and writers)

### When to Use RWMutex

**Use RWMutex when:**

- Many reads, few writes
- Reads don't block other reads
- Writes need exclusive access

**Our case:**

- **Reads:** `GetMetrics()` called frequently
- **Writes:** Increment operations less frequent
- **Perfect for RWMutex!**

---

## Returning Copies vs Pointers

### The Problem: Returning Pointer to Internal State

**❌ Bad: Returns pointer to internal state**

```go
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    return s.metrics  // Returns pointer to internal state!
}
```

**Problem:**

```go
metrics, _ := store.GetMetrics(ctx)
metrics.TotalJobsCreated = 999999  // Mutated internal state!
```

**External code can mutate internal state, breaking encapsulation.**

### The Solution: Return a Copy

**✅ Good: Returns copy**

```go
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Return a copy to prevent external mutation
    m := *s.metrics  // Copy the struct
    return &m, nil   // Return pointer to copy
}
```

**How it works:**

1. **`m := *s.metrics`** - Copy the struct value
2. **`return &m`** - Return pointer to the copy

**Now external code can't mutate internal state:**

```go
metrics, _ := store.GetMetrics(ctx)
metrics.TotalJobsCreated = 999999  // Only mutates the copy!
// Internal state unchanged ✅
```

### Memory Implications

**Copying is cheap for small structs:**

```go
type Metric struct {
    TotalJobsCreated int  // 8 bytes
    JobsCompleted    int  // 8 bytes
    JobsFailed       int  // 8 bytes
    JobsRetried      int  // 8 bytes
    JobsInProgress   int  // 8 bytes
}
// Total: 40 bytes - very cheap to copy
```

**For large structs, might want to return pointer, but ensure external code can't mutate.**

---

## Context in Metric Operations

### Why Check Context?

**Our implementation:**

```go
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    s.metrics.TotalJobsCreated++
    return nil
}
```

### Should We Check Context?

**For fast operations (like incrementing):**

- Context check adds overhead
- Operation is very fast (< 1 microsecond)
- Context cancellation unlikely during such short operation

**For slow operations (like database queries):**

- Context check is essential
- Operation might take seconds
- Need to respect cancellation

### Our Approach

**We check context for consistency:**

- All methods have same signature
- Consistent error handling
- Future-proof (if operations become slower)

**In practice:**

- Context check is fast (just channel read)
- Overhead is minimal
- Better to be consistent

---

## Common Mistakes

### Mistake 1: No Mutex Protection

```go
// ❌ BAD: No mutex
type InMemoryMetricStore struct {
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) IncrementJobsCreated() {
    s.metrics.TotalJobsCreated++  // Race condition!
}
```

**Fix:** Add mutex

```go
// ✅ GOOD: Mutex protection
type InMemoryMetricStore struct {
    mu      sync.RWMutex
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.metrics.TotalJobsCreated++
    return nil
}
```

### Mistake 2: Forgetting defer Unlock

```go
// ❌ BAD: No defer
func (s *InMemoryMetricStore) IncrementJobsCreated() error {
    s.mu.Lock()

    if err != nil {
        return err  // Forgot unlock!
    }

    s.metrics.TotalJobsCreated++
    s.mu.Unlock()
    return nil
}
```

**Fix:** Always use defer

```go
// ✅ GOOD: defer unlock
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()  // Always unlocks

    if err != nil {
        return err
    }

    s.metrics.TotalJobsCreated++
    return nil
}
```

### Mistake 3: Returning Pointer to Internal State

```go
// ❌ BAD: Returns pointer to internal state
func (s *InMemoryMetricStore) GetMetrics() *Metric {
    s.mu.RLock()
    defer s.mu.RUnlock()

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

### Mistake 4: Using Mutex Instead of RWMutex

```go
// ❌ BAD: Regular mutex for read-heavy operations
type InMemoryMetricStore struct {
    mu      sync.Mutex  // Blocks all reads
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) GetMetrics() *Metric {
    s.mu.Lock()  // Blocks other readers unnecessarily
    defer s.mu.Unlock()
    return s.metrics
}
```

**Fix:** Use RWMutex

```go
// ✅ GOOD: RWMutex for read-heavy operations
type InMemoryMetricStore struct {
    mu      sync.RWMutex  // Allows concurrent reads
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()  // Allows other readers
    defer s.mu.RUnlock()
    m := *s.metrics
    return &m, nil
}
```

### Mistake 5: Holding Lock Too Long

```go
// ❌ BAD: Holding lock during slow operation
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Slow operation while holding lock!
    time.Sleep(1 * time.Second)  // Blocks all other operations

    s.metrics.TotalJobsCreated++
    return nil
}
```

**Fix:** Do slow work before acquiring lock

```go
// ✅ GOOD: Do slow work first
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    // Do slow work first (before lock)
    if err := validateSomething(); err != nil {
        return err
    }

    // Now acquire lock for fast operation
    s.mu.Lock()
    defer s.mu.Unlock()

    s.metrics.TotalJobsCreated++  // Fast operation
    return nil
}
```

---

## Key Takeaways

1. **Concurrency safety** = Required when multiple goroutines access shared data
2. **Mutex** = Protects critical sections (one goroutine at a time)
3. **RWMutex** = Allows multiple readers OR one writer
4. **defer Unlock()** = Always releases lock, even on error
5. **Return copies** = Prevents external mutation of internal state
6. **Race detector** = Always test with `-race` flag
7. **Hold locks briefly** = Do slow work before acquiring lock

---

## Real-World Example

**Our concurrency-safe metrics store:**

```go
type InMemoryMetricStore struct {
    mu      sync.RWMutex  // Protects concurrent access
    metrics *domain.Metric
}

// Read operation: uses RLock (allows concurrent reads)
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    m := *s.metrics  // Copy to prevent mutation
    return &m, nil
}

// Write operation: uses Lock (exclusive access)
func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.metrics.TotalJobsCreated++
    return nil
}
```

**Benefits:**

- Multiple goroutines can read simultaneously
- Only one goroutine can write at a time
- No race conditions
- External code can't mutate internal state

---

## Next Steps

- Read [Metrics Collection and Storage](./02-metrics-collection-storage.md) to understand what metrics we track
- Read [Encapsulation: Returning Copies](./07-encapsulation-returning-copies.md) for more on returning copies
- Read [Dependency Injection for Observability](./03-dependency-injection-observability.md) to see how we wire the metric store
