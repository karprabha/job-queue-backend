# Understanding State Machines and Transitions

## Table of Contents

1. [Why Explicit State Machines?](#why-explicit-state-machines)
2. [What is a State Machine?](#what-is-a-state-machine)
3. [Our Job State Machine](#our-job-state-machine)
4. [The canTransition Function](#the-cantransition-function)
5. [Why Store Enforces Transitions](#why-store-enforces-transitions)
6. [Invalid Transition Handling](#invalid-transition-handling)
7. [Common Mistakes](#common-mistakes)

---

## Why Explicit State Machines?

### The Problem Without State Machines

**Before Task 6:** Jobs could transition between any states without validation.

```go
// ❌ BAD: No validation, any transition allowed
job.Status = domain.StatusCompleted  // Even if job was never processed!
job.Status = domain.StatusPending    // Even if job is already completed!
```

**Problems:**
- Jobs could skip states (pending → completed, skipping processing)
- Jobs could go backwards (completed → pending)
- No way to prevent invalid transitions
- Bugs are hard to catch
- State becomes inconsistent

### The Solution: Explicit State Machine

**With Task 6:** Only valid transitions are allowed.

```go
// ✅ GOOD: Validation prevents invalid transitions
if !canTransition(job.Status, domain.StatusCompleted) {
    return errors.New("invalid state transition")
}
job.Status = domain.StatusCompleted
```

**Benefits:**
- Invalid transitions are rejected
- State is always consistent
- Bugs are caught early
- Clear state flow
- Easier to reason about

### Real-World Analogy

Think of a traffic light:

- **Without state machine:** Light could go from RED → GREEN → RED → YELLOW (invalid!)
- **With state machine:** Light must follow RED → YELLOW → GREEN → YELLOW → RED (valid cycle)

A job queue is similar - jobs must follow a valid lifecycle.

---

## What is a State Machine?

### The Simple Answer

A **state machine** is a model that defines:
1. **States** - What conditions something can be in
2. **Transitions** - How to move between states
3. **Rules** - Which transitions are allowed

### The Detailed Answer

A state machine ensures that:
- Only valid state changes are allowed
- Invalid transitions are rejected
- State is always consistent
- The system behavior is predictable

### Visual Representation

```
Our Job State Machine:

    [Pending]
        │
        │ ClaimJob()
        ↓
    [Processing]
        │
        ├─→ UpdateStatus(Completed) ──→ [Completed] ✅
        │
        └─→ UpdateStatus(Failed) ──→ [Failed]
                                        │
                                        │ RetryFailedJobs()
                                        ↓
                                    [Pending] (if attempts < maxRetries)
```

**Key Points:**
- Jobs start in `Pending`
- Jobs move to `Processing` when claimed
- Jobs can complete or fail from `Processing`
- Failed jobs can retry (move back to `Pending`)
- Completed jobs never change state

---

## Our Job State Machine

### The States

```go
const (
    StatusPending    JobStatus = "pending"
    StatusProcessing JobStatus = "processing"
    StatusCompleted  JobStatus = "completed"
    StatusFailed     JobStatus = "failed"
)
```

**State Meanings:**
- `Pending` - Job is waiting to be processed
- `Processing` - Job is currently being processed by a worker
- `Completed` - Job finished successfully
- `Failed` - Job failed during processing

### The Valid Transitions

```go
func canTransition(from, to domain.JobStatus) bool {
    switch {
    case from == domain.StatusPending && to == domain.StatusProcessing:
        return true  // Claiming a job
    case from == domain.StatusProcessing && to == domain.StatusCompleted:
        return true  // Job succeeded
    case from == domain.StatusProcessing && to == domain.StatusFailed:
        return true  // Job failed
    case from == domain.StatusFailed && to == domain.StatusPending:
        return true  // Retrying a failed job
    default:
        return false  // All other transitions are invalid
    }
}
```

**What This Means:**
- `Pending → Processing` ✅ (worker claims job)
- `Processing → Completed` ✅ (job succeeds)
- `Processing → Failed` ✅ (job fails)
- `Failed → Pending` ✅ (retry)
- Everything else ❌ (invalid)

### Invalid Transitions (Rejected)

```go
// ❌ These are all rejected:
Pending → Completed        // Can't skip processing
Completed → Pending        // Can't undo completion
Completed → Failed         // Can't fail after completion
Failed → Completed         // Can't complete from failed (must retry first)
Processing → Pending       // Can't go backwards
```

**Why These Are Invalid:**
- `Pending → Completed`: Job was never processed
- `Completed → Pending`: Completed jobs are final
- `Processing → Pending`: Job is already being processed
- `Failed → Completed`: Must retry first (go through Pending → Processing)

---

## The canTransition Function

### The Function

```go
func canTransition(from, to domain.JobStatus) bool {
    switch {
    case from == domain.StatusPending && to == domain.StatusProcessing:
        return true
    case from == domain.StatusProcessing && to == domain.StatusCompleted:
        return true
    case from == domain.StatusProcessing && to == domain.StatusFailed:
        return true
    case from == domain.StatusFailed && to == domain.StatusPending:
        return true
    default:
        return false
    }
}
```

### How It Works

**Step 1: Check Current State**
- `from` is the current state (e.g., `StatusProcessing`)

**Step 2: Check Target State**
- `to` is the desired state (e.g., `StatusCompleted`)

**Step 3: Match Against Valid Transitions**
- Switch statement checks if `from → to` is valid
- Returns `true` if valid, `false` if invalid

### Example Usage

```go
// Valid transition
if canTransition(domain.StatusPending, domain.StatusProcessing) {
    // true - this is allowed
}

// Invalid transition
if canTransition(domain.StatusCompleted, domain.StatusPending) {
    // false - this is rejected
}
```

### Why This Design?

**Benefits:**
- **Centralized** - All transition rules in one place
- **Explicit** - Clear what transitions are allowed
- **Testable** - Easy to test all combinations
- **Maintainable** - Easy to add new states/transitions

---

## Why Store Enforces Transitions

### The Critical Design Decision

**Rule:** The **store** validates transitions, not workers.

### Why Not Workers?

**❌ BAD: Workers validate transitions**

```go
// Worker code
func (w *Worker) processJob(job *domain.Job) {
    if canTransition(job.Status, domain.StatusCompleted) {
        job.Status = domain.StatusCompleted  // Worker decides
    }
}
```

**Problems:**
- Multiple workers could have different logic
- Validation logic scattered across codebase
- Hard to ensure consistency
- Easy to bypass validation

### Why Store?

**✅ GOOD: Store validates transitions**

```go
// Store code
func (s *InMemoryJobStore) UpdateStatus(ctx context.Context, jobID string, status domain.JobStatus, lastError *string) error {
    job, ok := s.jobs[jobID]
    if !ok {
        return errors.New("job not found")
    }
    
    // Store validates
    if !canTransition(job.Status, status) {
        return errors.New("invalid state transition")
    }
    
    job.Status = status
    s.jobs[jobID] = job
    return nil
}
```

**Benefits:**
- **Single source of truth** - Store is authoritative
- **Consistent** - All updates go through same validation
- **Secure** - Can't bypass validation
- **Centralized** - Easy to maintain

### The Pattern

```
Worker: "I want to mark this job as completed"
    ↓
Store: "Let me check if that's valid..."
    ↓
Store: "✅ Valid transition (Processing → Completed), updating"
    ↓
Store: "❌ Invalid transition (Pending → Completed), rejecting"
```

**Key Point:** Workers **request** state changes, store **validates and enforces** them.

---

## Invalid Transition Handling

### What Happens When Transition is Invalid?

```go
func (s *InMemoryJobStore) UpdateStatus(ctx context.Context, jobID string, status domain.JobStatus, lastError *string) error {
    // ... get job ...
    
    // Validate transition
    if !canTransition(job.Status, status) {
        return errors.New("invalid state transition")  // Return error
    }
    
    // ... update job ...
}
```

### Error Handling in Workers

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Try to mark as completed
    err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
    if err != nil {
        // Invalid transition - log and return
        log.Printf("Worker %d error updating job: %s: %v", w.id, job.ID, err)
        return
    }
}
```

### Why Return Error Instead of Panic?

**Error (current approach):**
- Invalid transition is a **recoverable** error
- Worker can log and continue
- System keeps running
- Can investigate why transition failed

**Panic (bad approach):**
- Invalid transition would crash worker
- System becomes unstable
- Hard to debug
- Overkill for validation errors

### Common Invalid Transition Scenarios

**Scenario 1: Race Condition**
```
Worker 1: Claims job (Pending → Processing)
Worker 2: Tries to complete job (Processing → Completed) ✅
Worker 1: Tries to complete job (Processing → Completed) ✅
```

**What happens:** Both workers can complete if they both claim successfully. But `ClaimJob` prevents this - only one worker can claim.

**Scenario 2: Double Completion**
```
Worker: Completes job (Processing → Completed) ✅
Worker: Tries to complete again (Completed → Completed) ❌
```

**What happens:** Second attempt returns error "invalid state transition" - job is already completed.

**Scenario 3: Retry After Completion**
```
Worker: Completes job (Processing → Completed) ✅
Sweeper: Tries to retry (Completed → Pending) ❌
```

**What happens:** Sweeper's retry is rejected - completed jobs never retry.

---

## Common Mistakes

### Mistake 1: Workers Validate Transitions

```go
// ❌ BAD: Validation in worker
func (w *Worker) processJob(job *domain.Job) {
    if canTransition(job.Status, domain.StatusCompleted) {
        job.Status = domain.StatusCompleted  // Worker decides
        w.jobStore.UpdateJob(job)  // No validation in store
    }
}
```

**Problem:** Validation logic scattered, easy to bypass.

**Fix:** Store validates, worker just requests.

```go
// ✅ GOOD: Store validates
func (w *Worker) processJob(job *domain.Job) {
    err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
    // Store validates transition internally
}
```

### Mistake 2: Missing Transition Validation

```go
// ❌ BAD: No validation
func (s *InMemoryJobStore) UpdateStatus(jobID string, status domain.JobStatus) {
    job := s.jobs[jobID]
    job.Status = status  // No check!
    s.jobs[jobID] = job
}
```

**Problem:** Any transition allowed, state can become inconsistent.

**Fix:** Always validate.

```go
// ✅ GOOD: Always validate
func (s *InMemoryJobStore) UpdateStatus(jobID string, status domain.JobStatus) error {
    job := s.jobs[jobID]
    if !canTransition(job.Status, status) {
        return errors.New("invalid transition")
    }
    job.Status = status
    s.jobs[jobID] = job
    return nil
}
```

### Mistake 3: Silent Failure on Invalid Transition

```go
// ❌ BAD: Silently ignores invalid transition
func (s *InMemoryJobStore) UpdateStatus(jobID string, status domain.JobStatus) {
    job := s.jobs[jobID]
    if canTransition(job.Status, status) {
        job.Status = status
        s.jobs[jobID] = job
    }
    // No error if invalid - silent failure!
}
```

**Problem:** Caller doesn't know transition failed.

**Fix:** Return error.

```go
// ✅ GOOD: Return error
func (s *InMemoryJobStore) UpdateStatus(jobID string, status domain.JobStatus) error {
    job := s.jobs[jobID]
    if !canTransition(job.Status, status) {
        return errors.New("invalid transition")  // Explicit error
    }
    job.Status = status
    s.jobs[jobID] = job
    return nil
}
```

### Mistake 4: Inconsistent State Machine

```go
// ❌ BAD: Different validation in different places
func canTransition1(from, to JobStatus) bool {
    // One set of rules
}

func canTransition2(from, to JobStatus) bool {
    // Different set of rules (inconsistent!)
}
```

**Problem:** State machine rules differ, hard to maintain.

**Fix:** Single source of truth.

```go
// ✅ GOOD: One function, used everywhere
func canTransition(from, to JobStatus) bool {
    // Single set of rules
}
```

---

## Key Takeaways

1. **State machines** prevent invalid state changes
2. **canTransition** validates transitions explicitly
3. **Store enforces** state rules (not workers)
4. **Invalid transitions** return errors (not panics)
5. **Single source of truth** for state rules
6. **Explicit is better** than implicit

---

## Real-World Analogy

Think of a vending machine:

- **States:** Idle, Processing Payment, Dispensing, Out of Stock
- **Valid transitions:**
  - Idle → Processing Payment (customer inserts money)
  - Processing Payment → Dispensing (payment accepted)
  - Dispensing → Idle (product dispensed)
- **Invalid transitions:**
  - Idle → Dispensing (can't skip payment)
  - Dispensing → Processing Payment (can't go backwards)

A job queue is similar - jobs must follow valid state transitions.

---

## Next Steps

- Read [Failure Handling](./02-failure-handling.md) to understand how failures fit into the state machine
- Read [Retry Logic](./03-retry-logic-attempts.md) to see how retries use state transitions
- Read [Atomic State Updates](./05-atomic-state-updates.md) to understand how transitions are applied safely

