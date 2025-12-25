# Concurrency Safety with Mutexes in Go

## Table of Contents

1. [What is Concurrency?](#what-is-concurrency)
2. [The Problem: Race Conditions](#the-problem-race-conditions)
3. [What is a Mutex?](#what-is-a-mutex)
4. [How Mutexes Work](#how-mutexes-work)
5. [Using Mutexes in Our Store](#using-mutexes-in-our-store)
6. [Lock and Unlock Patterns](#lock-and-unlock-patterns)
7. [Common Mistakes](#common-mistakes)
8. [Best Practices](#best-practices)

---

## What is Concurrency?

### The Concept

**Concurrency** means multiple operations happening at the same time.

**In our web server:**

- Multiple HTTP requests arrive simultaneously
- Each request runs in its own goroutine
- Multiple goroutines can access the store at the same time

### Example Scenario

```
Time 0ms: Request 1 arrives → CreateJob() starts
Time 1ms: Request 2 arrives → GetJobs() starts
Time 2ms: Request 3 arrives → CreateJob() starts
```

**All three requests are running concurrently!**

### The Problem

If multiple goroutines access the same map **without protection**, bad things happen:

```
Goroutine 1: Read map
Goroutine 2: Write to map  ← At the same time!
Goroutine 3: Read map
```

**Result:** Race condition → Data corruption → Crashes

---

## The Problem: Race Conditions

### What is a Race Condition?

A **race condition** occurs when the outcome depends on the timing of operations.

### Example: Unsafe Map Access

```go
// ❌ BAD: No mutex protection
type InMemoryJobStore struct {
    jobs map[string]Job  // No mutex!
}

func (s *InMemoryJobStore) CreateJob(job Job) {
    s.jobs[job.ID] = job  // Race condition!
}

func (s *InMemoryJobStore) GetJobs() []Job {
    jobs := []Job{}
    for _, job := range s.jobs {  // Race condition!
        jobs = append(jobs, job)
    }
    return jobs
}
```

### What Can Go Wrong?

**Scenario 1: Concurrent Writes**

```
Goroutine 1: s.jobs["id1"] = job1
Goroutine 2: s.jobs["id2"] = job2
             ↑
        Happening at the same time!
```

**Result:** Map internal structure can get corrupted → Crash

**Scenario 2: Read During Write**

```
Goroutine 1: Writing to map (modifying internal structure)
Goroutine 2: Reading from map (iterating)
             ↑
        Happening at the same time!
```

**Result:** Iterator can see inconsistent state → Wrong data or crash

**Scenario 3: Concurrent Reads (Usually Safe)**

```
Goroutine 1: Reading from map
Goroutine 2: Reading from map
             ↑
        Usually safe, but not guaranteed
```

**Result:** Usually works, but not guaranteed by Go spec

### The Solution: Mutex

A **mutex** ensures only one goroutine accesses the map at a time.

---

## What is a Mutex?

### The Concept

**Mutex** = **Mut**ual **Ex**clusion

**Simple analogy:**

- Think of a mutex as a **key to a room**
- Only one person can have the key at a time
- Others must wait until the key is returned

### How It Works

```go
// Goroutine 1
mu.Lock()        // Get the key
// Do work        // In the room
mu.Unlock()      // Return the key

// Goroutine 2 (waiting)
mu.Lock()        // Wait for key...
// (waits until Goroutine 1 unlocks)
// Do work        // Now in the room
mu.Unlock()      // Return the key
```

### The sync.Mutex Type

```go
import "sync"

type InMemoryJobStore struct {
    jobs map[string]Job
    mu   sync.Mutex  // The mutex
}
```

**Properties:**

- Zero value is usable (no initialization needed)
- Lock() blocks until available
- Unlock() releases the lock
- Must be unlocked (can't forget!)

---

## How Mutexes Work

### The Lock Operation

```go
mu.Lock()
```

**What happens:**

1. Check if mutex is available
2. If available → acquire lock, continue
3. If not available → **block** (wait) until available

**Blocking means:**

- Goroutine stops executing
- Waits until mutex is unlocked
- Then acquires lock and continues

### The Unlock Operation

```go
mu.Unlock()
```

**What happens:**

1. Release the lock
2. If other goroutines are waiting → wake one up
3. That goroutine can now acquire the lock

### Visual Example

```
Time 0ms: Goroutine 1 calls mu.Lock()
          → Lock acquired ✅
          → Goroutine 1 proceeds

Time 1ms: Goroutine 2 calls mu.Lock()
          → Lock not available ❌
          → Goroutine 2 blocks (waits) ⏸️

Time 2ms: Goroutine 1 calls mu.Unlock()
          → Lock released
          → Goroutine 2 wakes up ✅
          → Goroutine 2 acquires lock

Time 3ms: Goroutine 2 proceeds
```

---

## Using Mutexes in Our Store

### The Store with Mutex

```go
type InMemoryJobStore struct {
    jobs map[string]domain.Job
    mu   sync.RWMutex  // Read-write mutex (we'll explain this)
}
```

### Creating a Job (Write Operation)

```go
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    s.mu.Lock()         // Acquire lock
    defer s.mu.Unlock() // Release lock when function exits

    s.jobs[job.ID] = *job  // Safe to write now
    return nil
}
```

**Step by step:**

1. `s.mu.Lock()` - Acquire exclusive lock
2. `defer s.mu.Unlock()` - Schedule unlock when function exits
3. `s.jobs[job.ID] = *job` - Write to map (protected)
4. Function exits → `defer` runs → `Unlock()` called

### Getting All Jobs (Read Operation)

```go
func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    s.mu.RLock()         // Acquire read lock
    defer s.mu.RUnlock() // Release read lock

    jobs := make([]domain.Job, 0, len(s.jobs))
    for _, job := range s.jobs {  // Safe to read now
        jobs = append(jobs, job)
    }
    return jobs, nil
}
```

**Note:** We use `RLock()` (read lock) instead of `Lock()`. We'll explain why in the next document.

### Why defer?

**Without defer:**

```go
func (s *InMemoryJobStore) CreateJob(...) error {
    s.mu.Lock()

    if someCondition {
        return errors.New("error")  // Forgot to unlock! 💥
    }

    s.jobs[job.ID] = *job
    s.mu.Unlock()  // Only reached if no error
    return nil
}
```

**Problem:** If function returns early, mutex is never unlocked → **deadlock!**

**With defer:**

```go
func (s *InMemoryJobStore) CreateJob(...) error {
    s.mu.Lock()
    defer s.mu.Unlock()  // Always unlocks, no matter how function exits

    if someCondition {
        return errors.New("error")  // defer runs → unlocks ✅
    }

    s.jobs[job.ID] = *job
    return nil  // defer runs → unlocks ✅
}
```

**Benefit:** `defer` guarantees unlock, even on early return or panic.

---

## Lock and Unlock Patterns

### Pattern 1: defer Unlock (Recommended)

```go
func (s *Store) Operation() {
    s.mu.Lock()
    defer s.mu.Unlock()  // Always unlocks

    // Do work
}
```

**When to use:** Almost always (safest pattern)

### Pattern 2: Manual Unlock (Rare)

```go
func (s *Store) Operation() {
    s.mu.Lock()

    // Do work

    s.mu.Unlock()  // Manual unlock
}
```

**When to use:** Only if you need to unlock before function exits (rare)

### Pattern 3: Multiple Locks (Advanced)

```go
func (s *Store) ComplexOperation() {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Do work that needs lock

    // Call another method that also needs lock
    s.helperMethod()  // This will deadlock! 💥
}

func (s *Store) helperMethod() {
    s.mu.Lock()  // Waiting for lock...
    defer s.mu.Unlock()
    // But lock is held by ComplexOperation!
}
```

**Problem:** Same goroutine can't acquire lock twice → **deadlock**

**Solution:** Restructure code to avoid nested locks

---

## Common Mistakes

### Mistake 1: Forgetting to Unlock

```go
// ❌ BAD: Lock but no unlock
func (s *Store) CreateJob(...) {
    s.mu.Lock()
    s.jobs[id] = job
    // Forgot to unlock! 💥
}
```

**Fix:** Always use defer

```go
// ✅ GOOD: defer unlock
func (s *Store) CreateJob(...) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.jobs[id] = job
}
```

### Mistake 2: Unlocking Without Locking

```go
// ❌ BAD: Unlock without lock
func (s *Store) CreateJob(...) {
    s.mu.Unlock()  // Panic! Not locked
}
```

**Fix:** Only unlock if you locked

```go
// ✅ GOOD: Lock before unlock
func (s *Store) CreateJob(...) {
    s.mu.Lock()
    defer s.mu.Unlock()
}
```

### Mistake 3: Not Protecting All Accesses

```go
// ❌ BAD: Some accesses not protected
func (s *Store) CreateJob(...) {
    s.mu.Lock()
    s.jobs[id] = job
    s.mu.Unlock()

    // Later, unprotected access
    count := len(s.jobs)  // Race condition! 💥
}
```

**Fix:** Protect all accesses

```go
// ✅ GOOD: All accesses protected
func (s *Store) CreateJob(...) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.jobs[id] = job
    count := len(s.jobs)  // Protected
}
```

### Mistake 4: Copying Mutex

```go
// ❌ BAD: Copying struct with mutex
store1 := InMemoryJobStore{...}
store2 := store1  // Mutex copied! 💥
```

**Fix:** Always use pointers

```go
// ✅ GOOD: Use pointers
store1 := &InMemoryJobStore{...}
store2 := store1  // Same mutex (pointer)
```

### Mistake 5: Holding Lock Too Long

```go
// ❌ BAD: Lock held during slow operation
func (s *Store) CreateJob(...) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.jobs[id] = job

    // Slow network call while holding lock!
    http.Get("https://api.example.com")  // Blocks other goroutines
}
```

**Fix:** Release lock before slow operations

```go
// ✅ GOOD: Release lock before slow operations
func (s *Store) CreateJob(...) {
    s.mu.Lock()
    s.jobs[id] = job
    s.mu.Unlock()  // Release lock

    // Slow operation without lock
    http.Get("https://api.example.com")
}
```

---

## Best Practices

### 1. Always Use defer

```go
s.mu.Lock()
defer s.mu.Unlock()
```

**Why:** Guarantees unlock, even on panic or early return

### 2. Keep Critical Sections Small

```go
// ✅ GOOD: Small critical section
s.mu.Lock()
s.jobs[id] = job  // Quick operation
s.mu.Unlock()

// Slow operation outside lock
processJob(job)
```

**Why:** Reduces contention (other goroutines wait less)

### 3. Protect All Accesses

```go
// ✅ GOOD: All map accesses protected
s.mu.Lock()
defer s.mu.Unlock()

job := s.jobs[id]      // Protected
s.jobs[id] = newJob    // Protected
delete(s.jobs, id)     // Protected
```

**Why:** Prevents race conditions

### 4. Use RWMutex for Read-Heavy Workloads

```go
// Reads (many)
s.mu.RLock()
// read operation
s.mu.RUnlock()

// Writes (few)
s.mu.Lock()
// write operation
s.mu.Unlock()
```

**Why:** Allows concurrent reads (better performance)

---

## Key Takeaways

1. **Mutex** = Mutual exclusion (only one goroutine at a time)
2. **Lock()** = Acquire exclusive access
3. **Unlock()** = Release access
4. **defer Unlock()** = Always unlock (best practice)
5. **Race conditions** = Multiple goroutines accessing shared data
6. **Mutex protects** = Prevents race conditions
7. **Keep critical sections small** = Better performance

---

## The Go Philosophy

Go provides **simple, powerful concurrency primitives**:

- ✅ Mutex is simple (Lock/Unlock)
- ✅ Built into standard library
- ✅ No magic, just clear synchronization
- ✅ Explicit is better than implicit

**Go's approach:**

- Simple primitives
- Clear semantics
- Explicit synchronization
- "Don't communicate by sharing memory; share memory by communicating" (but sometimes you need mutexes!)

---

## Next Steps

- Read [RWMutex vs Mutex](./05-rwmutex-vs-mutex.md) to understand why we use RWMutex
- Read [Context in Storage Layer](./06-context-in-storage.md) to see how context works with mutexes
- Read [In-Memory Storage](./03-in-memory-storage.md) to see the full store implementation
