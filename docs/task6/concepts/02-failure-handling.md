# Understanding Failure Handling

## Table of Contents

1. [Why Failure is a First-Class Concept](#why-failure-is-a-first-class-concept)
2. [The StatusFailed State](#the-statusfailed-state)
3. [LastError Tracking](#lasterror-tracking)
4. [Worker Signals Failure, Store Updates State](#worker-signals-failure-store-updates-state)
5. [Failure Simulation](#failure-simulation)
6. [Common Mistakes](#common-mistakes)

---

## Why Failure is a First-Class Concept

### The Problem Without Failure Handling

**Before Task 6:** Jobs could only succeed or... what?

```go
// ❌ BAD: No failure state
if processingSucceeds {
    job.Status = domain.StatusCompleted
} else {
    // What happens here? Job stuck in Processing forever?
    // Or silently ignored?
}
```

**Problems:**
- Failed jobs get stuck in `Processing` state
- No way to track why jobs failed
- No way to retry failed jobs
- No observability into failures
- System becomes unreliable

### The Solution: Explicit Failure State

**With Task 6:** Failure is a first-class state.

```go
// ✅ GOOD: Explicit failure handling
if processingFails {
    errMsg := "Processing failed: connection timeout"
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
} else {
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
}
```

**Benefits:**
- Failed jobs have explicit state
- Error messages are tracked
- Failed jobs can be retried
- System is observable
- Failures are recoverable

### Real-World Analogy

Think of a package delivery service:

- **Without failure handling:** Package gets lost, no record, customer never knows
- **With failure handling:** Package marked as "failed delivery" with reason "address not found", can be retried

A job queue is similar - failures must be tracked and recoverable.

---

## The StatusFailed State

### What is StatusFailed?

```go
const (
    StatusPending    JobStatus = "pending"
    StatusProcessing JobStatus = "processing"
    StatusCompleted  JobStatus = "completed"
    StatusFailed     JobStatus = "failed"  // New state!
)
```

**StatusFailed means:**
- Job was processed but failed
- Job is not stuck in Processing
- Job can potentially be retried
- Failure reason is tracked

### When Does a Job Become Failed?

**Scenario 1: Processing Error**
```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Try to process
    err := processEmail(job)
    if err != nil {
        // Processing failed - mark as failed
        errMsg := fmt.Sprintf("Email sending failed: %v", err)
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
        return
    }
    
    // Success - mark as completed
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
}
```

**Scenario 2: Deterministic Failure (Our Implementation)**
```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Simulate failure for email jobs
    if job.Type == "email" {
        errMsg := "Email sending failed"
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
        return
    }
    
    // Other jobs succeed
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
}
```

### State Transition: Processing → Failed

```go
// Valid transition
canTransition(domain.StatusProcessing, domain.StatusFailed)  // true ✅
```

**What happens:**
1. Job is in `Processing` state
2. Worker detects failure
3. Worker calls `UpdateStatus(StatusFailed)`
4. Store validates transition (Processing → Failed is valid)
5. Store updates state to `Failed`
6. Store saves error message

### Failed Jobs Are Observable

```go
// Can query failed jobs
failedJobs, err := jobStore.GetFailedJobs(ctx)
for _, job := range failedJobs {
    log.Printf("Job %s failed: %s", job.ID, *job.LastError)
}
```

**Benefits:**
- See all failed jobs
- Understand failure patterns
- Debug issues
- Monitor system health

---

## LastError Tracking

### Why Track Errors?

**Without error tracking:**
```go
// ❌ BAD: No error message
job.Status = domain.StatusFailed
// Why did it fail? No idea!
```

**With error tracking:**
```go
// ✅ GOOD: Error message stored
errMsg := "Email sending failed: connection timeout"
job.Status = domain.StatusFailed
job.LastError = &errMsg
// Now we know why it failed!
```

### The LastError Field

```go
type Job struct {
    ID         string
    Type       string
    Status     JobStatus
    Payload    json.RawMessage
    MaxRetries int
    Attempts   int
    LastError  *string  // Pointer to string (optional)
    CreatedAt  time.Time
}
```

**Why `*string` (pointer)?**
- `nil` = no error (job hasn't failed yet, or error cleared)
- `&string` = error message exists

**Alternative (not used):**
```go
LastError string  // Empty string = no error? Unclear!
```

**With pointer:**
```go
LastError *string  // nil = no error, clear and explicit
```

### Setting LastError

```go
func (s *InMemoryJobStore) UpdateStatus(ctx context.Context, jobID string, status domain.JobStatus, lastError *string) error {
    // ... validation ...
    
    job.Status = status
    if lastError != nil {
        job.LastError = lastError  // Set error message
    }
    s.jobs[jobID] = job  // Save after all updates
    
    return nil
}
```

**Key Point:** Error is set atomically with status update - both happen together.

### When LastError is Set

**Scenario 1: Job Fails**
```go
errMsg := "Email sending failed"
w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
// LastError = "Email sending failed"
```

**Scenario 2: Job Completes**
```go
w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
// LastError = nil (no error)
```

**Scenario 3: Job Retries**
```go
// When retrying, LastError might be cleared or kept
// Our implementation keeps it for debugging
```

### Reading LastError

```go
if job.Status == domain.StatusFailed && job.LastError != nil {
    log.Printf("Job %s failed: %s", job.ID, *job.LastError)
}
```

**Important:** Always check `LastError != nil` before dereferencing!

---

## Worker Signals Failure, Store Updates State

### The Critical Pattern

**Rule:** Workers **signal** failure, store **updates** state.

### Why This Separation?

**❌ BAD: Worker directly mutates state**

```go
func (w *Worker) processJob(job *domain.Job) {
    if processingFails {
        job.Status = domain.StatusFailed  // Worker mutates directly
        job.LastError = &errMsg
        // No validation, no atomic update
    }
}
```

**Problems:**
- No validation
- Not atomic
- Worker owns state (wrong!)
- Can bypass state machine

**✅ GOOD: Worker signals, store updates**

```go
// Worker code
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    if processingFails {
        errMsg := "Processing failed"
        // Worker signals failure
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
        // Store validates and updates atomically
    }
}

// Store code
func (s *InMemoryJobStore) UpdateStatus(ctx context.Context, jobID string, status domain.JobStatus, lastError *string) error {
    // Store validates transition
    if !canTransition(job.Status, status) {
        return errors.New("invalid transition")
    }
    
    // Store updates atomically
    job.Status = status
    if lastError != nil {
        job.LastError = lastError
    }
    s.jobs[jobID] = job
    
    return nil
}
```

**Benefits:**
- Store validates transition
- Update is atomic
- Store owns state (correct!)
- Can't bypass state machine

### The Flow

```
1. Worker detects failure
   ↓
2. Worker calls UpdateStatus(StatusFailed, errorMsg)
   ↓
3. Store validates: Processing → Failed is valid? ✅
   ↓
4. Store updates: Status = Failed, LastError = errorMsg
   ↓
5. Store saves atomically
   ↓
6. Worker continues (or exits)
```

**Key Point:** Worker doesn't know about state machine rules - it just signals what happened. Store enforces the rules.

---

## Failure Simulation

### Why Simulate Failures?

**For testing and learning:**
- Need to test retry logic
- Need to see failure handling in action
- Need deterministic behavior (not random)

### Our Implementation

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Simulate processing time
    timer := time.NewTimer(1 * time.Second)
    defer timer.Stop()
    
    select {
    case <-timer.C:
        // Processing complete
    case <-ctx.Done():
        return
    }
    
    // Deterministic failure: email jobs always fail
    if job.Type == "email" {
        lastError := "Email sending failed"
        err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
        if err != nil {
            log.Printf("Error marking job as failed: %v", err)
            return
        }
        log.Printf("Worker %d job %s failed", w.id, job.ID)
        return
    }
    
    // Other jobs succeed
    err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
    // ...
}
```

### Why Deterministic?

**❌ BAD: Random failures**
```go
if rand.Float64() < 0.5 {
    // Sometimes fails, sometimes succeeds - unpredictable!
}
```

**Problems:**
- Hard to reproduce bugs
- Tests are flaky
- Can't verify retry logic
- Unpredictable behavior

**✅ GOOD: Deterministic failures**
```go
if job.Type == "email" {
    // Always fails - predictable, testable
}
```

**Benefits:**
- Reproducible
- Tests are reliable
- Can verify retry logic
- Predictable behavior

### Other Failure Simulation Strategies

**Strategy 1: Fail by Type (Our Approach)**
```go
if job.Type == "email" {
    // Email jobs fail
}
```

**Strategy 2: Fail Every Nth Job**
```go
if job.Attempts % 3 == 0 {
    // Every 3rd attempt fails
}
```

**Strategy 3: Fail Based on Payload**
```go
if strings.Contains(string(job.Payload), "error") {
    // Jobs with "error" in payload fail
}
```

**All are valid** - choose based on what you need to test.

---

## Common Mistakes

### Mistake 1: Worker Directly Mutates State

```go
// ❌ BAD: Worker owns state
func (w *Worker) processJob(job *domain.Job) {
    job.Status = domain.StatusFailed
    job.LastError = &errMsg
    // No validation, not atomic
}
```

**Fix:** Worker signals, store updates.

```go
// ✅ GOOD: Store owns state
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    errMsg := "Processing failed"
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
}
```

### Mistake 2: Not Setting LastError

```go
// ❌ BAD: No error message
w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, nil)
// Why did it fail? No idea!
```

**Fix:** Always provide error message.

```go
// ✅ GOOD: Error message provided
errMsg := "Email sending failed: connection timeout"
w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
```

### Mistake 3: Not Saving After Setting LastError

```go
// ❌ BAD: LastError not saved
job.Status = status
s.jobs[jobID] = job  // Save here
if lastError != nil {
    job.LastError = lastError  // Update but never save!
}
```

**Fix:** Update all fields, then save once.

```go
// ✅ GOOD: Save after all updates
job.Status = status
if lastError != nil {
    job.LastError = lastError
}
s.jobs[jobID] = job  // Save after all updates
```

### Mistake 4: Dereferencing nil LastError

```go
// ❌ BAD: Panic if LastError is nil
log.Printf("Error: %s", *job.LastError)  // Panic!
```

**Fix:** Check for nil first.

```go
// ✅ GOOD: Check for nil
if job.LastError != nil {
    log.Printf("Error: %s", *job.LastError)
}
```

### Mistake 5: Random Failure Simulation

```go
// ❌ BAD: Unpredictable
if rand.Float64() < 0.5 {
    // Sometimes fails
}
```

**Fix:** Use deterministic simulation.

```go
// ✅ GOOD: Deterministic
if job.Type == "email" {
    // Always fails for email jobs
}
```

---

## Key Takeaways

1. **Failure is a first-class state** - not an exception
2. **StatusFailed** represents failed jobs explicitly
3. **LastError** tracks why jobs failed
4. **Workers signal** failure, **store updates** state
5. **Failure simulation** should be deterministic
6. **Store owns state** - workers just signal events

---

## Real-World Analogy

Think of a restaurant order:

- **Without failure handling:** Order gets lost, no record, customer never knows
- **With failure handling:** Order marked as "failed" with reason "kitchen equipment broken", can be retried when equipment fixed

A job queue is similar - failures must be tracked, explained, and recoverable.

---

## Next Steps

- Read [Retry Logic](./03-retry-logic-attempts.md) to understand how failed jobs are retried
- Read [The Sweeper Pattern](./04-sweeper-pattern.md) to see how retries are implemented
- Read [Atomic State Updates](./05-atomic-state-updates.md) to understand how failures are recorded safely

