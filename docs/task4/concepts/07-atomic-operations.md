# Understanding Atomic Operations and Race Conditions

## Table of Contents

1. [The Race Condition Problem](#the-race-condition-problem)
2. [What is a Race Condition?](#what-is-a-race-condition)
3. [Our ClaimJob Solution](#our-claimjob-solution)
4. [How ClaimJob Works](#how-claimjob-works)
5. [Why Atomic Operations Matter](#why-atomic-operations-matter)
6. [Mutex vs Atomic Operations](#mutex-vs-atomic-operations)
7. [Common Mistakes](#common-mistakes)

---

## The Race Condition Problem

### The Scenario

Imagine you have multiple workers (or could have in the future):

```go
// Worker 1
job := <-jobQueue
processJob(job)  // Takes 5 seconds

// Worker 2 (at the same time)
job := <-jobQueue  // Gets same job!
processJob(job)  // Processes it again!
```

**Problem:** Both workers might process the same job!

### Why This Happens

**Timeline:**
```
Time 0: Job created, status = "pending"
Time 1: Worker 1 receives job from queue
Time 2: Worker 2 receives same job from queue (race!)
Time 3: Worker 1 checks status, sees "pending"
Time 4: Worker 2 checks status, sees "pending" (still!)
Time 5: Worker 1 sets status to "processing"
Time 6: Worker 2 sets status to "processing" (overwrites!)
```

**Result:** Job processed twice, status might be wrong.

---

## What is a Race Condition?

### The Definition

A **race condition** occurs when the outcome of a program depends on the timing of events that are not controlled.

### Visual Example

```
Thread 1:          Thread 2:
Read status        Read status
(pending)          (pending)
                   ↓
Set processing     Set processing
                   ↓
Result: Both think they own the job!
```

**Problem:** Both threads see the same state and act on it.

### In Our Code

**Without protection:**
```go
// Worker 1
job := <-jobQueue
if job.Status == "pending" {  // Check
    job.Status = "processing"  // Set (not atomic!)
    processJob(job)
}

// Worker 2 (concurrent)
job := <-jobQueue  // Same job!
if job.Status == "pending" {  // Still sees "pending"!
    job.Status = "processing"  // Overwrites Worker 1!
    processJob(job)  // Processes again!
}
```

**Problem:** Check and set are not atomic (not done together).

---

## Our ClaimJob Solution

### The Atomic Operation

We created `ClaimJob` method that does check-and-set **atomically**:

```go
func (s *InMemoryJobStore) ClaimJob(ctx context.Context, jobID string) (bool, error) {
    select {
    case <-ctx.Done():
        return false, ctx.Err()
    default:
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    job, ok := s.jobs[jobID]
    if !ok || job.Status != domain.StatusPending {
        return false, nil  // Job doesn't exist or not pending
    }

    job.Status = domain.StatusProcessing
    s.jobs[jobID] = job

    return true, nil  // Successfully claimed
}
```

### What "Atomic" Means

**Atomic** = Operation happens as a single, indivisible unit.

**Non-atomic (bad):**
```go
// Step 1: Check
if job.Status == "pending" {
    // Step 2: Set (can be interrupted here!)
    job.Status = "processing"
}
```

**Atomic (good):**
```go
// All done together, can't be interrupted
s.mu.Lock()
// Check and set in one operation
if job.Status == "pending" {
    job.Status = "processing"
}
s.mu.Unlock()
```

---

## How ClaimJob Works

### Step-by-Step Breakdown

**Step 1: Context Check**
```go
select {
case <-ctx.Done():
    return false, ctx.Err()
default:
}
```
- Check if context canceled
- Return early if canceled
- Don't acquire lock if canceled (efficient)

**Step 2: Acquire Lock**
```go
s.mu.Lock()
defer s.mu.Unlock()
```
- Acquire mutex (exclusive access)
- `defer` ensures unlock happens
- Only one goroutine can hold lock at a time

**Step 3: Check Job Status**
```go
job, ok := s.jobs[jobID]
if !ok || job.Status != domain.StatusPending {
    return false, nil
}
```
- Get job from map
- Check if job exists
- Check if status is "pending"
- Return `false` if not claimable

**Step 4: Claim Job**
```go
job.Status = domain.StatusProcessing
s.jobs[jobID] = job
```
- Update status to "processing"
- Save back to map
- **This is atomic** (protected by lock)

**Step 5: Return Success**
```go
return true, nil
```
- Signal successful claim
- Lock released by `defer`

### Why This Prevents Race Conditions

**Timeline with ClaimJob:**
```
Time 0: Job created, status = "pending"
Time 1: Worker 1 calls ClaimJob(jobID)
Time 2: Worker 1 acquires lock
Time 3: Worker 1 checks status, sees "pending"
Time 4: Worker 1 sets status to "processing"
Time 5: Worker 1 releases lock
Time 6: Worker 2 calls ClaimJob(jobID)
Time 7: Worker 2 acquires lock
Time 8: Worker 2 checks status, sees "processing" (not pending!)
Time 9: Worker 2 returns false (not claimed)
```

**Result:** Only Worker 1 claims the job!

---

## Why Atomic Operations Matter

### Problem 1: Duplicate Processing

**Without atomic claim:**
- Multiple workers might process same job
- Wasted resources
- Incorrect results

**With atomic claim:**
- Only one worker can claim job
- No duplicate processing
- Efficient resource usage

### Problem 2: Status Corruption

**Without atomic claim:**
```go
// Worker 1
job.Status = "processing"  // Sets to processing

// Worker 2 (concurrent)
job.Status = "processing"  // Also sets to processing
// But what if Worker 1 already set it?
// Status might be inconsistent
```

**With atomic claim:**
- Status updated atomically
- Consistent state
- No corruption

### Problem 3: Lost Updates

**Without atomic claim:**
```go
// Worker 1
job.Status = "processing"
store.Update(job)  // Saves

// Worker 2 (concurrent)
job.Status = "processing"  // Reads old value
store.Update(job)  // Overwrites Worker 1's update!
```

**With atomic claim:**
- Updates are serialized (one at a time)
- No lost updates
- Consistent state

---

## Mutex vs Atomic Operations

### What We Use: Mutex

```go
s.mu.Lock()
// Critical section
s.mu.Unlock()
```

**Characteristics:**
- Protects a section of code
- Can protect multiple operations
- More flexible
- Slightly more overhead

### Alternative: Atomic Operations

Go also has `sync/atomic` package:

```go
var status int32
atomic.StoreInt32(&status, 1)  // Atomic store
value := atomic.LoadInt32(&status)  // Atomic load
```

**Characteristics:**
- Very fast
- Only works on specific types (int32, int64, etc.)
- Less flexible
- Lower overhead

### Why We Use Mutex

**Our use case:**
- Need to check status AND update it
- Need to update map (not just a single value)
- Multiple operations need to be atomic together

**Mutex is better because:**
- Can protect multiple operations
- Works with any data structure
- More readable
- Standard pattern for this use case

**Atomic would be:**
- Only works for single values
- Can't protect map operations
- Less readable for complex operations

---

## Common Mistakes

### Mistake 1: Check-Then-Set (Not Atomic)

```go
// ❌ BAD: Race condition!
func (w *Worker) processJob(job *Job) {
    if job.Status == "pending" {  // Check
        job.Status = "processing"  // Set (not atomic!)
        processJob(job)
    }
}
```

**Problem:** Check and set are separate operations, race condition possible.

**Fix:** Use atomic operation
```go
// ✅ GOOD: Atomic claim
claimed, _ := store.ClaimJob(ctx, job.ID)
if !claimed {
    return
}
processJob(job)
```

### Mistake 2: Not Using Lock

```go
// ❌ BAD: No protection
func (s *Store) ClaimJob(jobID string) bool {
    job := s.jobs[jobID]  // Not protected!
    if job.Status == "pending" {
        job.Status = "processing"  // Race condition!
        s.jobs[jobID] = job
    }
}
```

**Problem:** Multiple goroutines can access map concurrently.

**Fix:** Use mutex
```go
// ✅ GOOD: Protected by mutex
func (s *Store) ClaimJob(jobID string) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    // Now protected
}
```

### Mistake 3: Forgetting to Check Status

```go
// ❌ BAD: Claims any job
func (s *Store) ClaimJob(jobID string) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    job := s.jobs[jobID]
    job.Status = "processing"  // Doesn't check if pending!
    s.jobs[jobID] = job
    return true
}
```

**Problem:** Might claim already-processing or completed jobs.

**Fix:** Check status first
```go
// ✅ GOOD: Checks status
if job.Status != domain.StatusPending {
    return false  // Not claimable
}
job.Status = domain.StatusProcessing
```

### Mistake 4: Not Handling Errors

```go
// ❌ BAD: Ignores errors
claimed := store.ClaimJob(ctx, job.ID)
if !claimed {
    continue
}
```

**Problem:** Doesn't know why claim failed (error? not pending?).

**Fix:** Check error
```go
// ✅ GOOD: Handles errors
claimed, err := store.ClaimJob(ctx, job.ID)
if err != nil {
    log.Printf("Error: %v", err)
    continue
}
if !claimed {
    continue  // Job already claimed or not pending
}
```

### Mistake 5: Re-checking After Claim

```go
// ❌ BAD: Unnecessary check
claimed, _ := store.ClaimJob(ctx, job.ID)
if !claimed {
    continue
}
if job.Status != "pending" {  // Already checked in ClaimJob!
    continue
}
```

**Problem:** Redundant check, job status already changed by ClaimJob.

**Fix:** Trust ClaimJob
```go
// ✅ GOOD: Trust the atomic operation
claimed, _ := store.ClaimJob(ctx, job.ID)
if !claimed {
    continue  // ClaimJob already checked everything
}
// Job is guaranteed to be "processing" now
```

---

## Key Takeaways

1. **Race conditions** = Timing-dependent bugs in concurrent code
2. **Atomic operations** = Operations that can't be interrupted
3. **ClaimJob** = Atomic check-and-set to prevent duplicates
4. **Mutex** = Protects critical sections from race conditions
5. **Always use locks** = When multiple goroutines access shared data
6. **Check then set** = Must be atomic to prevent races

---

## Real-World Analogy

Think of atomic operations like a **locked door**:

- **Without lock:** Multiple people can enter, chaos
- **With lock:** Only one person can enter at a time
- **ClaimJob:** Like taking a ticket - only one person can take ticket #42

**Race condition** = Two people both think they have ticket #42
**Atomic operation** = Only one person can actually get ticket #42

---

## Next Steps

- Read [Worker Pattern](./03-worker-pattern.md) to see how ClaimJob is used
- Read [Concurrency Safety](../task3/concepts/04-concurrency-safety.md) for mutex basics
- Read [RWMutex vs Mutex](../task3/concepts/05-rwmutex-vs-mutex.md) for mutex types

