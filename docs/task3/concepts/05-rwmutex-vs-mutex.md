# RWMutex vs Mutex: When to Use Which?

## Table of Contents

1. [The Problem: Read vs Write Operations](#the-problem-read-vs-write-operations)
2. [What is RWMutex?](#what-is-rwmutex)
3. [Mutex vs RWMutex Comparison](#mutex-vs-rwmutex-comparison)
4. [How RWMutex Works](#how-rwmutex-works)
5. [Our Implementation: Why RWMutex?](#our-implementation-why-rwmutex)
6. [Performance Considerations](#performance-considerations)
7. [When to Use Which?](#when-to-use-which)
8. [Common Mistakes](#common-mistakes)

---

## The Problem: Read vs Write Operations

### Different Types of Operations

**Read operations:**

- `GetJobs()` - Read all jobs
- `GetJob(id)` - Read one job
- `Count()` - Count jobs

**Write operations:**

- `CreateJob()` - Add new job
- `UpdateJob()` - Modify job
- `DeleteJob()` - Remove job

### The Key Insight

**Multiple reads can happen simultaneously** without problems:

```
Goroutine 1: Reading jobs ✅
Goroutine 2: Reading jobs ✅  ← Can happen at same time!
Goroutine 3: Reading jobs ✅
```

**But writes need exclusive access:**

```
Goroutine 1: Writing job ✅
Goroutine 2: Writing job ❌  ← Must wait!
Goroutine 3: Reading jobs ❌  ← Must wait!
```

### The Question

Can we allow **multiple concurrent reads** while still protecting writes?

**Answer:** Yes! That's what `RWMutex` does.

---

## What is RWMutex?

### RWMutex = Read-Write Mutex

**RWMutex** allows:

- ✅ Multiple concurrent **readers**
- ✅ One exclusive **writer** (blocks all readers and writers)

### The Two Types of Locks

**1. Read Lock (RLock)**

```go
s.mu.RLock()   // Acquire read lock
// Multiple goroutines can hold read lock simultaneously
s.mu.RUnlock() // Release read lock
```

**2. Write Lock (Lock)**

```go
s.mu.Lock()    // Acquire write lock (exclusive)
// Only one goroutine can hold write lock
s.mu.Unlock()  // Release write lock
```

### Visual Comparison

**Regular Mutex:**

```
Goroutine 1: Lock()     → ✅ Proceeds
Goroutine 2: Lock()     → ⏸️ Waits (even though it's just reading!)
Goroutine 3: Lock()     → ⏸️ Waits
```

**RWMutex:**

```
Goroutine 1: RLock()    → ✅ Proceeds (reading)
Goroutine 2: RLock()    → ✅ Proceeds (reading) - concurrent!
Goroutine 3: Lock()     → ⏸️ Waits (writing - needs exclusive)
```

---

## Mutex vs RWMutex Comparison

### sync.Mutex

```go
type InMemoryJobStore struct {
    jobs map[string]Job
    mu   sync.Mutex  // Regular mutex
}

func (s *InMemoryJobStore) GetJobs() []Job {
    s.mu.Lock()         // Exclusive lock
    defer s.mu.Unlock()
    // Read operations
}

func (s *InMemoryJobStore) CreateJob(job Job) {
    s.mu.Lock()         // Exclusive lock
    defer s.mu.Unlock()
    // Write operations
}
```

**Characteristics:**

- ✅ Simple (one type of lock)
- ✅ Works for all operations
- ❌ No concurrent reads (even though safe)
- ❌ Slower for read-heavy workloads

### sync.RWMutex

```go
type InMemoryJobStore struct {
    jobs map[string]Job
    mu   sync.RWMutex  // Read-write mutex
}

func (s *InMemoryJobStore) GetJobs() []Job {
    s.mu.RLock()         // Read lock (allows concurrent reads)
    defer s.mu.RUnlock()
    // Read operations
}

func (s *InMemoryJobStore) CreateJob(job Job) {
    s.mu.Lock()          // Write lock (exclusive)
    defer s.mu.Unlock()
    // Write operations
}
```

**Characteristics:**

- ✅ Allows concurrent reads
- ✅ Faster for read-heavy workloads
- ✅ Still protects writes
- ❌ Slightly more complex (two lock types)
- ❌ Slightly more overhead

---

## How RWMutex Works

### Read Lock (RLock)

**Multiple readers can hold read lock:**

```go
// Goroutine 1
s.mu.RLock()  // Acquire read lock
// Read data
s.mu.RUnlock()

// Goroutine 2 (at the same time)
s.mu.RLock()  // Also acquire read lock ✅
// Read data
s.mu.RUnlock()
```

**Rules:**

- Multiple `RLock()` calls are allowed
- Each `RLock()` must have matching `RUnlock()`
- Readers don't block other readers

### Write Lock (Lock)

**Only one writer, blocks everything:**

```go
// Goroutine 1
s.mu.Lock()  // Acquire write lock (exclusive)
// Write data
s.mu.Unlock()

// Goroutine 2 (trying to read)
s.mu.RLock()  // ⏸️ Waits until write lock released

// Goroutine 3 (trying to write)
s.mu.Lock()   // ⏸️ Waits until write lock released
```

**Rules:**

- Only one `Lock()` allowed at a time
- Blocks all `RLock()` calls
- Blocks all other `Lock()` calls

### The Priority System

**When a writer is waiting:**

- New readers are blocked (writer has priority)
- Prevents writer starvation (waiting forever)

**Example:**

```
Time 0ms: Reader 1: RLock() ✅
Time 1ms: Reader 2: RLock() ✅
Time 2ms: Writer:   Lock()  ⏸️ (waits)
Time 3ms: Reader 3:  RLock() ⏸️ (blocked - writer waiting)
Time 4ms: Reader 1: RUnlock()
Time 5ms: Reader 2: RUnlock()
Time 6ms: Writer:   Proceeds ✅ (all readers done)
```

---

## Our Implementation: Why RWMutex?

### Our Store Operations

**Read operations (frequent):**

```go
func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    s.mu.RLock()         // Read lock
    defer s.mu.RUnlock()

    // Iterate map and create slice
    jobs := make([]domain.Job, 0, len(s.jobs))
    for _, job := range s.jobs {
        jobs = append(jobs, job)
    }
    return jobs, nil
}
```

**Write operations (less frequent):**

```go
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    s.mu.Lock()          // Write lock (exclusive)
    defer s.mu.Unlock()

    s.jobs[job.ID] = *job
    return nil
}
```

### Why RWMutex Makes Sense

**1. Read-Heavy Workload**

- `GET /jobs` will be called frequently
- Multiple users might list jobs simultaneously
- RWMutex allows concurrent reads

**2. Write-Light Workload**

- `POST /jobs` happens less frequently
- When it does, it needs exclusive access
- Write lock provides that

**3. Better Performance**

- Multiple `GET /jobs` requests can run concurrently
- No blocking between readers
- Better throughput

### What If We Used Regular Mutex?

```go
// With regular Mutex
func (s *InMemoryJobStore) GetJobs(...) {
    s.mu.Lock()  // Exclusive lock
    // Even though we're just reading!
    s.mu.Unlock()
}
```

**Problem:**

- Every `GET /jobs` blocks all other requests
- Even though multiple reads are safe
- Unnecessary serialization

---

## Performance Considerations

### Benchmark Scenario

**Setup:**

- 100 concurrent `GET /jobs` requests
- 10 `POST /jobs` requests

### With Regular Mutex

```
Request 1:  Lock() → ✅ Read → Unlock()
Request 2:  Lock() → ⏸️ Wait → ✅ Read → Unlock()
Request 3:  Lock() → ⏸️ Wait → ✅ Read → Unlock()
...
Request 100: Lock() → ⏸️ Wait → ✅ Read → Unlock()

Total time: ~100ms (serialized)
```

### With RWMutex

```
Request 1-50:  RLock() → ✅ Read concurrently → RUnlock()
Request 51-100: RLock() → ✅ Read concurrently → RUnlock()

Total time: ~2ms (parallel reads)
```

**Improvement:** ~50x faster for reads!

### The Trade-off

**RWMutex overhead:**

- Slightly more complex internally
- Slightly more memory
- But: Much better for read-heavy workloads

**When overhead matters:**

- Write-heavy workloads (RWMutex has more overhead)
- Very simple cases (regular Mutex is simpler)

---

## When to Use Which?

### Use Regular Mutex When:

**1. Write-Heavy Workload**

```go
// Mostly writes, few reads
func (s *Store) Update() { s.mu.Lock() ... }
func (s *Store) Delete() { s.mu.Lock() ... }
```

**2. Simple Cases**

```go
// Simple counter, no complex reads
type Counter struct {
    mu    sync.Mutex
    count int
}
```

**3. Mixed Operations**

```go
// Operations that both read and write
func (s *Store) Increment() {
    s.mu.Lock()
    s.value++  // Read and write
    s.mu.Unlock()
}
```

### Use RWMutex When:

**1. Read-Heavy Workload**

```go
// Many reads, few writes
func (s *Store) Get() { s.mu.RLock() ... }  // Frequent
func (s *Store) Set() { s.mu.Lock() ... }   // Rare
```

**2. Clear Read/Write Separation**

```go
// Clear distinction between reads and writes
func (s *Store) Read()  { s.mu.RLock() ... }
func (s *Store) Write() { s.mu.Lock() ... }
```

**3. Performance Matters**

```go
// Need concurrent reads for performance
// Multiple goroutines reading simultaneously
```

### Our Case: RWMutex

**Why RWMutex for our store:**

- ✅ `GetJobs()` is a pure read (frequent)
- ✅ `CreateJob()` is a pure write (less frequent)
- ✅ Clear read/write separation
- ✅ Read-heavy workload expected
- ✅ Performance benefit from concurrent reads

---

## Common Mistakes

### Mistake 1: Using RLock for Writes

```go
// ❌ BAD: RLock for write operation
func (s *Store) CreateJob(job Job) {
    s.mu.RLock()  // Wrong! This is a write
    s.jobs[id] = job
    s.mu.RUnlock()
}
```

**Problem:** Multiple goroutines can write simultaneously → Race condition!

**Fix:** Use Lock() for writes

```go
// ✅ GOOD: Lock for write
func (s *Store) CreateJob(job Job) {
    s.mu.Lock()  // Correct for writes
    s.jobs[id] = job
    s.mu.Unlock()
}
```

### Mistake 2: Using Lock for Reads

```go
// ❌ BAD: Lock for read (unnecessary)
func (s *Store) GetJobs() []Job {
    s.mu.Lock()  // Unnecessary - blocks other readers
    defer s.mu.Unlock()
    // Read operations
}
```

**Problem:** Blocks concurrent reads unnecessarily

**Fix:** Use RLock() for reads

```go
// ✅ GOOD: RLock for read
func (s *Store) GetJobs() []Job {
    s.mu.RLock()  // Allows concurrent reads
    defer s.mu.RUnlock()
    // Read operations
}
```

### Mistake 3: Mixing Lock Types Incorrectly

```go
// ❌ BAD: RLock then Lock (deadlock!)
func (s *Store) ComplexOperation() {
    s.mu.RLock()
    // Do read
    s.mu.Lock()  // Deadlock! Can't upgrade read to write
    // Do write
    s.mu.Unlock()
    s.mu.RUnlock()
}
```

**Problem:** Can't upgrade read lock to write lock

**Fix:** Release read lock first

```go
// ✅ GOOD: Release read, then acquire write
func (s *Store) ComplexOperation() {
    s.mu.RLock()
    // Do read
    s.mu.RUnlock()  // Release read first

    s.mu.Lock()     // Then acquire write
    // Do write
    s.mu.Unlock()
}
```

### Mistake 4: Forgetting RUnlock

```go
// ❌ BAD: RLock but no RUnlock
func (s *Store) GetJobs() []Job {
    s.mu.RLock()
    // Read operations
    // Forgot RUnlock! 💥
}
```

**Fix:** Always use defer

```go
// ✅ GOOD: defer RUnlock
func (s *Store) GetJobs() []Job {
    s.mu.RLock()
    defer s.mu.RUnlock()
    // Read operations
}
```

---

## Key Takeaways

1. **RWMutex** = Read-Write Mutex (allows concurrent reads)
2. **RLock()** = Read lock (multiple readers allowed)
3. **Lock()** = Write lock (exclusive, blocks everything)
4. **Use RWMutex** = When you have read-heavy workloads
5. **Use Mutex** = When writes are common or operations are mixed
6. **RLock for reads** = Allows concurrent reads
7. **Lock for writes** = Ensures exclusive access

---

## The Go Philosophy

Go provides **flexible concurrency primitives**:

- ✅ RWMutex for read-heavy workloads
- ✅ Mutex for simpler cases
- ✅ Clear semantics (RLock vs Lock)
- ✅ No magic, just clear synchronization

**Go's approach:**

- Right tool for the job
- Performance when it matters
- Simplicity when it doesn't

---

## Next Steps

- Read [Concurrency Safety](./04-concurrency-safety.md) to understand the basics of mutexes
- Read [Context in Storage Layer](./06-context-in-storage.md) to see how context works with mutexes
- Read [Interface Design](./07-interface-design.md) to understand the store interface
