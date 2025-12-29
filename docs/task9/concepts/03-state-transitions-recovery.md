# State Transitions in Recovery

## Table of Contents

1. [Why State Transitions Matter in Recovery](#why-state-transitions-matter-in-recovery)
2. [The Recovery Transition](#the-recovery-transition)
3. [Why We Need processing → pending](#why-we-need-processing--pending)
4. [How We Implement It](#how-we-implement-it)
5. [Common Mistakes](#common-mistakes)

---

## Why State Transitions Matter in Recovery

### The Core Principle

**All state changes must go through the same validation rules, even during recovery.**

### Why This Matters

**Without transition validation:**
```go
// Recovery directly mutates state
job.Status = domain.StatusPending  // Bypasses validation!
s.jobs[jobID] = job
```

**Problems:**
- Bypasses state machine rules
- Could create invalid states
- Breaks encapsulation
- Inconsistent with normal operations

**With transition validation:**
```go
// Recovery uses UpdateStatus
jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
// ✅ Respects state machine rules
```

**Benefits:**
- All state changes validated
- Consistent behavior
- Encapsulation maintained
- Same rules everywhere

---

## The Recovery Transition

### The Transition We Need

**Recovery needs:** `processing → pending`

**Why:**
- Jobs in "processing" state were interrupted during crash
- They need to be moved back to "pending" to be retried
- This is the only way to recover them

### The State Machine

**Normal transitions:**
- `pending → processing` ✅ (when worker claims job)
- `processing → completed` ✅ (when job succeeds)
- `processing → failed` ✅ (when job fails)
- `failed → pending` ✅ (when retrying)

**Recovery transition:**
- `processing → pending` ✅ (when recovering from crash)

### Why This Is Special

**Normal flow:**
- `processing → pending` doesn't happen normally
- Once a job is processing, it either completes or fails
- It doesn't go back to pending

**Recovery flow:**
- `processing → pending` is needed for recovery
- Jobs were interrupted, need to be retried
- This is the exception to normal flow

---

## Why We Need processing → pending

### The Problem

**Scenario: Process crashes during job processing**

1. Worker claims job → Status: `pending → processing`
2. Worker starts processing
3. **Process crashes** → Job stuck in `processing` state
4. On restart: Job is still `processing`
5. **Problem:** How do we recover this job?

### The Solution

**Move job back to `pending`:**
1. Recovery finds job in `processing` state
2. Moves it to `pending` state
3. Re-enqueues it
4. Worker processes it again

### Why Not Other States?

**Why not `failed`?**
- Job didn't fail, it was interrupted
- Different semantics
- Would need different error message

**Why not `completed`?**
- Job didn't complete
- Would be incorrect state
- Data inconsistency

**Why `pending`?**
- Job needs to be processed
- Pending is the correct state for unprocessed jobs
- Can be retried normally

---

## How We Implement It

### Step 1: Update Transition Rules

**Before (Task 6):**
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

**After (Task 9):**
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
    case from == domain.StatusProcessing && to == domain.StatusPending:
        return true // ← ADDED for recovery
    default:
        return false
    }
}
```

**Key change:** Added `processing → pending` transition.

### Step 2: Use UpdateStatus in Recovery

**Recovery code:**
```go
processingJobs, err := jobStore.GetProcessingJobs(ctx)
if err != nil {
    return fmt.Errorf("failed to get processing jobs: %w", err)
}

for _, job := range processingJobs {
    // Use UpdateStatus to respect state transition rules
    err := jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
    if err != nil {
        logger.Error("Failed to recover processing job",
            "event", "recovery_error",
            "job_id", job.ID,
            "error", err)
        continue  // Continue with other jobs
    }
    logger.Info("Recovered processing job",
        "event", "job_recovered",
        "job_id", job.ID)
}
```

**Key points:**
- Uses `UpdateStatus` (not direct mutation)
- Respects state transition rules
- Handles errors gracefully
- Logs recovery events

### Why This Works

**UpdateStatus flow:**
1. Gets job from store
2. Validates transition (`canTransition`)
3. Updates status
4. Saves to store

**Recovery flow:**
1. Gets processing jobs
2. Calls `UpdateStatus` for each
3. Transition validated automatically
4. Job moved to pending

**Result:** All state changes go through same validation.

---

## Common Mistakes

### Mistake 1: Direct Mutation in Recovery

**❌ BAD:**
```go
// Recovery directly mutates state
for _, job := range processingJobs {
    job.Status = domain.StatusPending  // Direct mutation!
    s.jobs[job.ID] = job
}
```

**Problems:**
- Bypasses state machine
- No validation
- Breaks encapsulation
- Inconsistent with normal flow

**✅ GOOD:**
```go
// Recovery uses UpdateStatus
for _, job := range processingJobs {
    jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
}
```

**Benefit:** All state changes validated.

### Mistake 2: Not Adding Transition Rule

**❌ BAD:**
```go
// Recovery tries to use UpdateStatus
jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
// ❌ Fails: "invalid state transition"
```

**Problem:**
- Transition not allowed
- Recovery fails
- Jobs stuck

**✅ GOOD:**
```go
// Add transition rule first
case from == domain.StatusProcessing && to == domain.StatusPending:
    return true // Allow for recovery

// Then recovery works
jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
```

**Benefit:** Recovery works correctly.

### Mistake 3: Allowing Transition in Normal Flow

**❌ BAD:**
```go
// Worker can move processing → pending
func (w *Worker) processJob(job *domain.Job) {
    if something {
        jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
        // ❌ Worker shouldn't do this!
    }
}
```

**Problem:**
- Allows invalid transitions in normal flow
- Breaks state machine semantics
- Confusing behavior

**✅ GOOD:**
```go
// Transition only allowed for recovery
// Workers can only: processing → completed or processing → failed
// Recovery can: processing → pending
```

**Benefit:** Clear semantics.

### Mistake 4: Not Handling Transition Errors

**❌ BAD:**
```go
// Recovery doesn't handle errors
for _, job := range processingJobs {
    jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
    // ❌ What if this fails?
}
```

**Problem:**
- One failure stops entire recovery
- Jobs might not be recovered
- No error handling

**✅ GOOD:**
```go
// Recovery handles errors
for _, job := range processingJobs {
    err := jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
    if err != nil {
        logger.Error("Failed to recover processing job", ...)
        continue  // Continue with other jobs
    }
}
```

**Benefit:** Recovery continues even if one job fails.

---

## Key Takeaways

1. **Always use UpdateStatus** - Never bypass state machine
2. **Add transition rules** - Allow recovery transitions
3. **Handle errors** - Don't let one failure stop recovery
4. **Consistent behavior** - Same rules everywhere
5. **Clear semantics** - Recovery transitions are special cases

---

## Related Concepts

- [Startup Recovery](./01-startup-recovery.md) - Overall recovery process
- [Recovery Backpressure](./02-recovery-backpressure.md) - How recovery handles queue full
- [Source of Truth](./04-source-of-truth.md) - Why store is authoritative

