# Startup Recovery & Persistence Boundary

## Table of Contents

1. [What is Startup Recovery?](#what-is-startup-recovery)
2. [Why Recovery Matters](#why-recovery-matters)
3. [The Problem We're Solving](#the-problem-were-solving)
4. [Source of Truth Design](#source-of-truth-design)
5. [Recovery Rules](#recovery-rules)
6. [Our Recovery Implementation](#our-recovery-implementation)
7. [Startup Order](#startup-order)
8. [Common Mistakes](#common-mistakes)

---

## What is Startup Recovery?

### The Simple Answer

**Startup recovery** is the process of restoring system state when the application restarts after a crash or shutdown. It ensures that:

- Jobs that were in progress are not lost
- Jobs that were pending are re-enqueued
- No jobs are left in an inconsistent state
- The system can resume work from where it left off

### The Challenge

When a process crashes or is killed:

- **In-memory state is lost** - All data in RAM disappears
- **Jobs in "processing" state** - Were being worked on, but never completed
- **Jobs in "pending" state** - Were waiting to be processed
- **Queue state is lost** - Channel contents disappear

**Question:** How do we recover from this?

**Answer:** We need to **reconcile the store state** (if persisted) with the queue state, and ensure all recoverable jobs are re-enqueued.

---

## Why Recovery Matters

### Problem 1: Jobs Stuck in Processing State

**Without recovery:**

```go
// Process crashes while worker is processing job
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Job status: "processing"
    // Process crashes here...
    // Job is stuck in "processing" state forever!
}
```

**Problem:** If the process crashes while a job is being processed, the job remains in "processing" state. When the system restarts, this job will never be processed again because:

- Workers only pick up jobs from the queue
- The queue is empty on startup
- The job is stuck in "processing" state in the store

**With recovery:**

```go
// On startup, recovery moves processing jobs back to pending
processingJobs, _ := jobStore.GetProcessingJobs(ctx)
for _, job := range processingJobs {
    jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
    // Job is now recoverable!
}
```

**Benefit:** Jobs are never permanently stuck.

### Problem 2: Pending Jobs Not Re-enqueued

**Without recovery:**

```go
// Process crashes with pending jobs in store
// On restart:
// - Store has pending jobs
// - Queue is empty
// - Workers are waiting for jobs from queue
// - Jobs never get processed!
```

**Problem:** Pending jobs exist in the store, but they're not in the queue. Workers only process jobs from the queue, so these jobs are never processed.

**With recovery:**

```go
// On startup, recovery re-enqueues all pending jobs
pendingJobs, _ := jobStore.GetPendingJobs(ctx)
for _, job := range pendingJobs {
    jobQueue <- job.ID  // Re-enqueue
}
```

**Benefit:** All pending jobs are processed.

---

## The Problem We're Solving

### Scenario: Process Crash During Operation

**Before crash:**
- 10 jobs in "pending" state
- 3 jobs in "processing" state (being worked on)
- 5 jobs in "completed" state
- 2 jobs in "failed" state

**After crash:**
- Store state: Lost (in-memory)
- Queue state: Lost (channel contents)
- Workers: Stopped

**After restart (without recovery):**
- Store: Empty (fresh start)
- Queue: Empty
- Workers: Waiting for jobs
- **Problem:** All 10 pending jobs and 3 processing jobs are lost!

**After restart (with recovery):**
- Store: Recovered from persistence (if we had it) or empty
- Recovery: Moves processing → pending, re-enqueues pending
- Queue: Contains all recoverable jobs
- Workers: Process recovered jobs
- **Result:** No jobs lost!

### The Reality Check

**In-memory systems:**
- All state is lost on restart
- Recovery can't restore lost jobs
- But recovery ensures **future restarts** work correctly

**With persistence (future):**
- Store state survives restarts
- Recovery restores jobs from persisted store
- No jobs lost

**Key insight:** Even in-memory systems need recovery logic because:
1. It prepares the system for persistence
2. It ensures correct behavior when persistence is added
3. It prevents stuck jobs in edge cases

---

## Source of Truth Design

### The Core Principle

**Store is the source of truth. Queue is a delivery mechanism.**

### What This Means

**Store (Source of Truth):**
- Contains all job state
- Enforces state transition rules
- Is authoritative for job status
- Persists job data (when persistence is added)

**Queue (Delivery Mechanism):**
- Temporary buffer for job IDs
- Not authoritative
- Can be empty or full
- Just a way to notify workers about work

### Why This Design?

**Separation of concerns:**
- Store manages state
- Queue manages delivery
- Workers process work

**Benefits:**
- Store can be persisted independently
- Queue can be recreated from store
- Workers don't need to scan store
- Clear ownership of data

### The Recovery Implication

Since store is the source of truth:

```go
// Recovery reads from store
processingJobs, _ := jobStore.GetProcessingJobs(ctx)
pendingJobs, _ := jobStore.GetPendingJobs(ctx)

// Recovery updates store
jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)

// Recovery populates queue from store
jobQueue <- job.ID
```

**Key point:** Recovery always starts from the store, never from the queue.

---

## Recovery Rules

### Rule 1: Processing → Pending

**What:** Jobs in "processing" state must be moved back to "pending"

**Why:** These jobs were interrupted during crash. They need to be retried.

**How:**
```go
processingJobs, _ := jobStore.GetProcessingJobs(ctx)
for _, job := range processingJobs {
    jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
}
```

**State transition:** `processing → pending` (allowed for recovery)

### Rule 2: Pending → Re-enqueued

**What:** All pending jobs must be re-enqueued

**Why:** Pending jobs need to be processed. They're in the store but not in the queue.

**How:**
```go
pendingJobs, _ := jobStore.GetPendingJobs(ctx)
for _, job := range pendingJobs {
    jobQueue <- job.ID
}
```

**Note:** This includes jobs that were just moved from "processing" to "pending"

### Rule 3: Completed → Untouched

**What:** Jobs in "completed" state remain untouched

**Why:** These jobs are done. No need to reprocess them.

**How:**
```go
// Recovery doesn't touch completed jobs
// They're not in GetProcessingJobs() or GetPendingJobs()
```

### Rule 4: Failed → Untouched

**What:** Jobs in permanent "failed" state remain untouched

**Why:** These jobs have exhausted retries. They're permanently failed.

**How:**
```go
// Recovery doesn't touch failed jobs
// They're not in GetProcessingJobs() or GetPendingJobs()
```

**Note:** Failed jobs that can be retried are handled by the sweeper, not recovery.

---

## Our Recovery Implementation

### The Recovery Function

```go
func RecoverJobs(
    ctx context.Context,
    jobStore store.JobStore,
    jobQueue chan string,
    logger *slog.Logger,
) error {
    logger.Info("Starting recovery", "event", "recovery_started")

    // Step 1: Move processing jobs back to pending
    processingJobs, err := jobStore.GetProcessingJobs(ctx)
    if err != nil {
        return fmt.Errorf("failed to get processing jobs: %w", err)
    }

    processingRecovered := 0
    for _, job := range processingJobs {
        err := jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
        if err != nil {
            logger.Error("Failed to recover processing job",
                "event", "recovery_error",
                "job_id", job.ID,
                "error", err)
            continue  // Continue with other jobs
        }
        processingRecovered++
        logger.Info("Recovered processing job",
            "event", "job_recovered",
            "job_id", job.ID)
    }

    // Step 2: Re-enqueue all pending jobs
    pendingJobs, err := jobStore.GetPendingJobs(ctx)
    if err != nil {
        return fmt.Errorf("failed to get pending jobs: %w", err)
    }

    pendingReEnqueued := 0
    for _, job := range pendingJobs {
        if err := reEnqueueWithBackpressure(ctx, job.ID, jobQueue, logger); err != nil {
            return fmt.Errorf("failed to re-enqueue job %s: %w", job.ID, err)
        }
        pendingReEnqueued++
    }

    logger.Info("Recovery completed",
        "event", "recovery_completed",
        "processing_recovered", processingRecovered,
        "pending_re_enqueued", pendingReEnqueued)

    return nil
}
```

### Breaking It Down

**Step 1: Get Processing Jobs**
```go
processingJobs, err := jobStore.GetProcessingJobs(ctx)
```
- Gets all jobs stuck in "processing" state
- These are jobs that were interrupted during crash

**Step 2: Move Processing → Pending**
```go
for _, job := range processingJobs {
    jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
}
```
- Uses `UpdateStatus` to respect state transition rules
- Moves each job from "processing" to "pending"
- Continues even if one job fails (doesn't fail entire recovery)

**Step 3: Get Pending Jobs**
```go
pendingJobs, err := jobStore.GetPendingJobs(ctx)
```
- Gets all pending jobs (including newly recovered ones)
- These need to be enqueued

**Step 4: Re-enqueue with Backpressure**
```go
for _, job := range pendingJobs {
    reEnqueueWithBackpressure(ctx, job.ID, jobQueue, logger)
}
```
- Re-enqueues each job
- Respects backpressure (waits if queue is full)
- Fails if job can't be enqueued after retries

### Error Handling Strategy

**Individual job failures:**
- Log error but continue
- Don't fail entire recovery
- One bad job shouldn't prevent recovery of others

**Critical failures:**
- Fail if can't get jobs from store
- Fail if can't re-enqueue after retries
- These indicate system problems

---

## Startup Order

### The Critical Sequence

**Correct order:**
1. Initialize store
2. Run recovery logic
3. Initialize queue
4. Start workers
5. Start HTTP server

### Why This Order Matters

**1. Initialize Store First**
```go
jobStore := store.NewInMemoryJobStore()
```
- Store must exist before recovery
- Store is the source of truth

**2. Run Recovery Before Workers**
```go
recovery.RecoverJobs(ctx, jobStore, jobQueue, logger)
```
- Recovery must complete before workers start
- Workers will start processing immediately
- If recovery runs after workers start, workers might process jobs before recovery finishes

**3. Initialize Queue Before Recovery**
```go
jobQueue := make(chan string, config.JobQueueCapacity)
```
- Queue must exist for recovery to enqueue jobs
- But workers shouldn't start yet

**4. Start Workers After Recovery**
```go
for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(...)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}
```
- Workers start processing after recovery completes
- All recoverable jobs are already in queue

**5. Start HTTP Server Last**
```go
go func() {
    srv.ListenAndServe()
}()
```
- Server can start accepting requests
- System is fully initialized

### Our Implementation

```go
func main() {
    // 1. Initialize store
    jobStore := store.NewInMemoryJobStore()
    metricStore := store.NewInMemoryMetricStore()

    // 2. Initialize queue (needed for recovery)
    jobQueue := make(chan string, config.JobQueueCapacity)

    // 3. Run recovery logic
    recoveryCtx := context.Background()
    if err := recovery.RecoverJobs(recoveryCtx, jobStore, jobQueue, logger); err != nil {
        log.Fatalf("Recovery failed: %v", err)
    }

    // 4. Start workers (after recovery)
    for i := 0; i < config.WorkerCount; i++ {
        worker := worker.NewWorker(...)
        wg.Go(func() {
            worker.Start(workerCtx)
        })
    }

    // 5. Start HTTP server
    go func() {
        srv.ListenAndServe()
    }()
}
```

**Key points:**
- Recovery runs synchronously (blocks until complete)
- Workers start after recovery completes
- Server starts after workers start

---

## Common Mistakes

### Mistake 1: Workers Start Before Recovery

**❌ BAD:**
```go
// Start workers first
for i := 0; i < config.WorkerCount; i++ {
    worker.Start(workerCtx)
}

// Then run recovery
recovery.RecoverJobs(ctx, jobStore, jobQueue, logger)
```

**Problem:**
- Workers start processing immediately
- Queue is empty (recovery hasn't run)
- Workers might process new jobs before recovery finishes
- Race condition: recovery and workers competing

**✅ GOOD:**
```go
// Run recovery first
recovery.RecoverJobs(ctx, jobStore, jobQueue, logger)

// Then start workers
for i := 0; i < config.WorkerCount; i++ {
    worker.Start(workerCtx)
}
```

**Benefit:** Recovery completes before workers start.

### Mistake 2: Recovery Bypasses Store Invariants

**❌ BAD:**
```go
// Directly mutate job status
job.Status = domain.StatusPending
s.jobs[jobID] = job  // Bypasses UpdateStatus!
```

**Problem:**
- Bypasses state transition validation
- Could create invalid states
- Breaks encapsulation

**✅ GOOD:**
```go
// Use UpdateStatus to respect rules
jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
```

**Benefit:** All state changes go through validation.

### Mistake 3: Dropping Jobs During Recovery

**❌ BAD:**
```go
// If queue is full, skip job
select {
case jobQueue <- job.ID:
    // Success
default:
    continue  // Skip job!
}
```

**Problem:**
- Jobs are silently dropped
- No retry mechanism
- Jobs are lost

**✅ GOOD:**
```go
// Retry with backpressure
reEnqueueWithBackpressure(ctx, job.ID, jobQueue, logger)
```

**Benefit:** No jobs are dropped.

### Mistake 4: Not Handling Processing Jobs

**❌ BAD:**
```go
// Only re-enqueue pending jobs
pendingJobs, _ := jobStore.GetPendingJobs(ctx)
for _, job := range pendingJobs {
    jobQueue <- job.ID
}
// Processing jobs are ignored!
```

**Problem:**
- Processing jobs remain stuck
- Never get retried
- System has inconsistent state

**✅ GOOD:**
```go
// Move processing to pending first
processingJobs, _ := jobStore.GetProcessingJobs(ctx)
for _, job := range processingJobs {
    jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
}

// Then re-enqueue all pending (including recovered ones)
pendingJobs, _ := jobStore.GetPendingJobs(ctx)
for _, job := range pendingJobs {
    jobQueue <- job.ID
}
```

**Benefit:** All recoverable jobs are processed.

### Mistake 5: Recovery Running After Workers

**❌ BAD:**
```go
// Workers start immediately
go worker.Start(workerCtx)

// Recovery runs in background
go recovery.RecoverJobs(ctx, jobStore, jobQueue, logger)
```

**Problem:**
- Race condition
- Workers might process jobs before recovery
- Unpredictable behavior

**✅ GOOD:**
```go
// Recovery runs synchronously first
if err := recovery.RecoverJobs(ctx, jobStore, jobQueue, logger); err != nil {
    log.Fatalf("Recovery failed: %v", err)
}

// Then workers start
go worker.Start(workerCtx)
```

**Benefit:** Predictable, correct behavior.

---

## Key Takeaways

1. **Recovery is essential** - Even in-memory systems need recovery logic
2. **Store is source of truth** - Queue is just a delivery mechanism
3. **Startup order matters** - Recovery must complete before workers start
4. **Respect state transitions** - Use UpdateStatus, don't bypass
5. **Never drop jobs** - Use backpressure, retry if needed
6. **Handle all states** - Processing jobs need recovery too
7. **Error handling** - Individual failures shouldn't stop recovery

---

## Related Concepts

- [Backpressure During Recovery](./02-recovery-backpressure.md) - How recovery handles queue full
- [State Transition Rules](./03-state-transitions-recovery.md) - How recovery respects state machine
- [Source of Truth Design](./04-source-of-truth.md) - Why store is authoritative

