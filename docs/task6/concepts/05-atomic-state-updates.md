# Understanding Atomic State Updates

## Table of Contents

1. [Why Atomic Updates Matter](#why-atomic-updates-matter)
2. [UpdateStatus Method Design](#updatestatus-method-design)
3. [Mutex Protection](#mutex-protection)
4. [Transition Validation](#transition-validation)
5. [Error Message Tracking](#error-message-tracking)
6. [Common Mistakes](#common-mistakes)

---

## Why Atomic Updates Matter

### The Problem Without Atomic Updates

**Scenario:** Two workers try to update the same job simultaneously.

**Without mutex (race condition):**
```go
// ❌ BAD: Race condition
func updateJob(jobID string, newStatus JobStatus) {
    job := jobs[jobID]
    job.Status = newStatus  // Worker 1 reads: Status = Processing
    // Worker 2 reads: Status = Processing (same value)
    job.LastError = &errMsg
    jobs[jobID] = job  // Worker 1 writes: Status = Failed
    // Worker 2 writes: Status = Completed (overwrites Worker 1!)
    // Result: Status = Completed, but LastError might be from Worker 1
    // Inconsistent state!
}
```

**Problems:**
- **Lost updates** - One worker's update overwrites another's
- **Inconsistent state** - Status and LastError might not match
- **Data corruption** - Job state becomes invalid
- **Hard to debug** - Race conditions are non-deterministic

### The Solution: Atomic Updates

**With mutex (atomic):**
```go
// ✅ GOOD: Atomic update
func updateJob(jobID string, newStatus JobStatus) {
    s.mu.Lock()  // Only one worker can enter
    defer s.mu.Unlock()
    
    job := jobs[jobID]
    job.Status = newStatus
    job.LastError = &errMsg
    jobs[jobID] = job  // All updates happen together
    // Next worker waits until this one finishes
}
```

**Benefits:**
- **No lost updates** - Updates are serialized
- **Consistent state** - Status and LastError always match
- **Data integrity** - Job state is always valid
- **Predictable** - No race conditions

### Real-World Analogy

Think of a bank account:

- **Without atomic updates:** Two people withdraw $100 simultaneously, both read balance = $1000, both write $900 (should be $800!)
- **With atomic updates:** One person withdraws, then the other (serialized, correct balance)

A job queue is similar - state updates must be atomic to prevent corruption.

---

## UpdateStatus Method Design

### The Method Signature

```go
func (s *InMemoryJobStore) UpdateStatus(
    ctx context.Context,
    jobID string,
    status domain.JobStatus,
    lastError *string,
) error
```

**Parameters:**
- `ctx` - Context for cancellation
- `jobID` - Which job to update
- `status` - New status
- `lastError` - Error message (optional, nil if no error)

**Returns:**
- `error` - Error if update fails (job not found, invalid transition, etc.)

### The Implementation

```go
func (s *InMemoryJobStore) UpdateStatus(
    ctx context.Context,
    jobID string,
    status domain.JobStatus,
    lastError *string,
) error {
    // 1. Check context
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // 2. Acquire lock
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // 3. Get job
    job, ok := s.jobs[jobID]
    if !ok {
        return errors.New("job not found in store")
    }
    
    // 4. Validate transition
    if !canTransition(job.Status, status) {
        return errors.New("invalid state transition")
    }
    
    // 5. Update all fields atomically
    job.Status = status
    if lastError != nil {
        job.LastError = lastError
    }
    
    // 6. Save atomically
    s.jobs[jobID] = job
    
    return nil
}
```

### Why This Design?

**Benefits:**
- **Single method** - All state updates go through one place
- **Atomic** - All fields updated together
- **Validated** - Transitions are checked
- **Safe** - Mutex prevents race conditions
- **Explicit** - Error messages are optional but tracked

---

## Mutex Protection

### The Mutex

```go
type InMemoryJobStore struct {
    jobs map[string]domain.Job
    mu   sync.RWMutex  // Protects jobs map
}
```

**What is a Mutex?**
- **Mutual Exclusion** - Only one goroutine can hold the lock at a time
- **Protects shared data** - Prevents concurrent access
- **Serializes operations** - Ensures atomicity

### How It Works

**Acquiring Lock:**
```go
s.mu.Lock()  // Blocks until lock is available
```

**What happens:**
- If lock is free: Acquire it, continue
- If lock is held: Wait until it's released

**Releasing Lock:**
```go
defer s.mu.Unlock()  // Releases lock when function exits
```

**What happens:**
- Lock is released
- Next waiting goroutine can acquire it

### The Protection

```go
func (s *InMemoryJobStore) UpdateStatus(...) error {
    s.mu.Lock()      // Acquire lock
    defer s.mu.Unlock()  // Release lock (always, even on error)
    
    // Critical section - only one goroutine can be here
    job := s.jobs[jobID]
    job.Status = status
    s.jobs[jobID] = job
    // End of critical section
}
```

**What this ensures:**
- Only one goroutine can update jobs at a time
- Updates are serialized (one after another)
- No race conditions
- State is always consistent

### Why defer?

**Without defer:**
```go
// ❌ BAD: Might forget to unlock
s.mu.Lock()
if err != nil {
    return err  // Forgot to unlock! Deadlock!
}
s.mu.Unlock()
```

**With defer:**
```go
// ✅ GOOD: Always unlocks
s.mu.Lock()
defer s.mu.Unlock()  // Always runs, even on early return
if err != nil {
    return err  // Unlock still runs!
}
```

**Key Point:** `defer` ensures unlock happens even if function returns early.

---

## Transition Validation

### The Validation

```go
// Validate transition
if !canTransition(job.Status, status) {
    return errors.New("invalid state transition")
}
```

**What this does:**
- Checks if transition from current status to new status is valid
- Rejects invalid transitions
- Returns error if invalid

### Why Validate in UpdateStatus?

**Benefits:**
- **Centralized** - All updates go through same validation
- **Consistent** - Same rules for all callers
- **Safe** - Can't bypass validation
- **Explicit** - Invalid transitions return errors

### Example: Valid Transition

```go
// Job is in Processing state
job.Status = domain.StatusProcessing

// Try to mark as Completed
err := store.UpdateStatus(ctx, jobID, domain.StatusCompleted, nil)
// canTransition(Processing, Completed) = true ✅
// Update succeeds
```

### Example: Invalid Transition

```go
// Job is in Completed state
job.Status = domain.StatusCompleted

// Try to mark as Pending
err := store.UpdateStatus(ctx, jobID, domain.StatusPending, nil)
// canTransition(Completed, Pending) = false ❌
// Update fails with error: "invalid state transition"
```

---

## Error Message Tracking

### Setting Error Message

```go
job.Status = status
if lastError != nil {
    job.LastError = lastError  // Set error message
}
s.jobs[jobID] = job  // Save after all updates
```

**Key Point:** Error is set atomically with status update.

### When Error is Set

**Scenario 1: Job Fails**
```go
errMsg := "Email sending failed"
store.UpdateStatus(ctx, jobID, domain.StatusFailed, &errMsg)
// Status = Failed, LastError = "Email sending failed"
```

**Scenario 2: Job Completes**
```go
store.UpdateStatus(ctx, jobID, domain.StatusCompleted, nil)
// Status = Completed, LastError = nil (unchanged or cleared)
```

### Why Update All Fields Before Saving?

**❌ BAD: Save before updating all fields**
```go
job.Status = status
s.jobs[jobID] = job  // Save here
if lastError != nil {
    job.LastError = lastError  // Update but never save!
}
```

**Problem:** LastError is updated but never saved to map.

**✅ GOOD: Update all fields, then save once**
```go
job.Status = status
if lastError != nil {
    job.LastError = lastError
}
s.jobs[jobID] = job  // Save after all updates
```

**Benefit:** All fields are updated atomically, then saved together.

---

## Common Mistakes

### Mistake 1: Not Using Mutex

```go
// ❌ BAD: Race condition
func (s *InMemoryJobStore) UpdateStatus(jobID string, status JobStatus) {
    job := s.jobs[jobID]  // Not protected!
    job.Status = status
    s.jobs[jobID] = job
}
```

**Fix:** Always use mutex.

```go
// ✅ GOOD: Protected by mutex
func (s *InMemoryJobStore) UpdateStatus(jobID string, status JobStatus) {
    s.mu.Lock()
    defer s.mu.Unlock()
    job := s.jobs[jobID]
    job.Status = status
    s.jobs[jobID] = job
}
```

### Mistake 2: Forgetting defer Unlock

```go
// ❌ BAD: Might forget to unlock
func (s *InMemoryJobStore) UpdateStatus(...) error {
    s.mu.Lock()
    if err != nil {
        return err  // Forgot to unlock!
    }
    s.mu.Unlock()
}
```

**Fix:** Always use defer.

```go
// ✅ GOOD: Always unlocks
func (s *InMemoryJobStore) UpdateStatus(...) error {
    s.mu.Lock()
    defer s.mu.Unlock()  // Always unlocks
    if err != nil {
        return err  // Unlock still runs
    }
}
```

### Mistake 3: Not Saving After Updating Fields

```go
// ❌ BAD: LastError not saved
job.Status = status
s.jobs[jobID] = job  // Save here
if lastError != nil {
    job.LastError = lastError  // Update but never save!
}
```

**Fix:** Update all fields, then save.

```go
// ✅ GOOD: Save after all updates
job.Status = status
if lastError != nil {
    job.LastError = lastError
}
s.jobs[jobID] = job  // Save after all updates
```

### Mistake 4: Missing Transition Validation

```go
// ❌ BAD: No validation
func (s *InMemoryJobStore) UpdateStatus(jobID string, status JobStatus) {
    job := s.jobs[jobID]
    job.Status = status  // No check!
    s.jobs[jobID] = job
}
```

**Fix:** Always validate.

```go
// ✅ GOOD: Always validate
func (s *InMemoryJobStore) UpdateStatus(jobID string, status JobStatus) error {
    job := s.jobs[jobID]
    if !canTransition(job.Status, status) {
        return errors.New("invalid transition")
    }
    job.Status = status
    s.jobs[jobID] = job
    return nil
}
```

### Mistake 5: Reading Without Lock

```go
// ❌ BAD: Read without lock
func (s *InMemoryJobStore) GetJob(jobID string) *Job {
    return s.jobs[jobID]  // Not protected!
}
```

**Fix:** Use RLock for reads.

```go
// ✅ GOOD: Protected read
func (s *InMemoryJobStore) GetJob(jobID string) *Job {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.jobs[jobID]
}
```

---

## Key Takeaways

1. **Atomic updates** prevent race conditions
2. **Mutex** protects shared data
3. **Always defer unlock** to prevent deadlocks
4. **Validate transitions** before updating
5. **Update all fields** before saving
6. **Single method** for all state updates

---

## Real-World Analogy

Think of a shared document:

- **Without atomic updates:** Two people edit simultaneously, changes conflict, document becomes inconsistent
- **With atomic updates:** One person edits, saves, then the other edits (serialized, consistent)

A job queue is similar - state updates must be atomic to prevent corruption.

---

## Next Steps

- Read [State Machine](./01-state-machine-transitions.md) to understand transition validation
- Read [Failure Handling](./02-failure-handling.md) to see how errors are tracked
- Read [Concurrency Safety](../task3/concepts/04-concurrency-safety.md) for more on mutex usage

