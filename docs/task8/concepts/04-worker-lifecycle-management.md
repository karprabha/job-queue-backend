# Worker Lifecycle Management

## Table of Contents

1. [What is Worker Lifecycle Management?](#what-is-worker-lifecycle-management)
2. [The Worker Lifecycle](#the-worker-lifecycle)
3. [Context Cancellation in Workers](#context-cancellation-in-workers)
4. [Job State Cleanup on Shutdown](#job-state-cleanup-on-shutdown)
5. [Worker Shutdown Pattern](#worker-shutdown-pattern)
6. [Common Mistakes](#common-mistakes)

---

## What is Worker Lifecycle Management?

### The Simple Answer

**Worker lifecycle management** means controlling when workers:

- **Start** - Begin processing jobs
- **Run** - Process jobs from queue
- **Stop** - Finish current job and exit cleanly

**Key requirement:** Workers must **never leave jobs in an inconsistent state** when stopping.

### The Challenge

When shutdown happens:

- Workers might be **processing a job**
- Job status is **"processing"**
- Worker is **interrupted**

**Question:** What happens to the job?

**Answer:** Worker must **clean up the job state** before exiting.

---

## The Worker Lifecycle

### Lifecycle Stages

```
1. START
   └─> Worker goroutine starts
   └─> Worker enters main loop

2. RUN
   └─> Worker waits for jobs from queue
   └─> Worker claims job
   └─> Worker processes job
   └─> Worker updates job status
   └─> Repeat

3. SHUTDOWN SIGNAL
   └─> Context is canceled
   └─> Worker detects cancellation

4. CLEANUP
   └─> If processing job: Clean up job state
   └─> Update metrics
   └─> Exit gracefully

5. STOP
   └─> Worker goroutine exits
   └─> WaitGroup signals completion
```

### Our Implementation

```go
func (w *Worker) Start(ctx context.Context) {
    w.logger.Info("Worker started", "event", "worker_started", "worker_id", w.id)

    for {
        select {
        case <-ctx.Done():
            // SHUTDOWN: Context canceled
            w.logger.Info("Worker shutting down", "event", "worker_stopped", "worker_id", w.id)
            return  // Exit loop, goroutine ends

        case jobID, ok := <-w.jobQueue:
            if !ok {
                // SHUTDOWN: Channel closed
                w.logger.Info("Worker shutting down because job queue is closed", "event", "worker_stopped", "worker_id", w.id)
                return  // Exit loop, goroutine ends
            }

            // RUN: Process job
            job, err := w.jobStore.ClaimJob(ctx, jobID)
            if err != nil || job == nil {
                continue  // Skip invalid jobs
            }

            w.logger.Info("Job started", "event", "job_started", "worker_id", w.id, "job_id", jobID)
            w.processJob(ctx, job)  // Process job (might be interrupted)
        }
    }
}
```

---

## Context Cancellation in Workers

### How Workers Detect Shutdown

**Two ways workers detect shutdown:**

1. **Context cancellation:**

   ```go
   case <-ctx.Done():
       // Context was canceled
       return
   ```

2. **Channel closed:**
   ```go
   case jobID, ok := <-w.jobQueue:
       if !ok {
           // Channel was closed
           return
       }
   ```

### Context Cancellation Flow

```go
// In main.go
workerCtx, workerCancel := context.WithCancel(context.Background())

// Start workers
for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, ...)
    wg.Go(func() {
        worker.Start(workerCtx)  // Workers watch this context
    })
}

// On shutdown
workerCancel()  // Cancels workerCtx
// ctx.Done() channel closes
// All workers detect cancellation
// Workers exit their loops
wg.Wait()  // Wait for all workers to finish
```

**What happens:**

1. `workerCancel()` is called
2. `workerCtx.Done()` channel closes
3. All workers' `select` statements see `ctx.Done()` is ready
4. Workers exit their loops
5. Workers return (goroutines end)
6. `wg.Wait()` unblocks when all workers done

### Channel Closed Detection

```go
// In main.go
close(jobQueue)  // Close channel

// In worker
case jobID, ok := <-w.jobQueue:
    if !ok {
        // ok = false means channel closed
        return  // Exit
    }
```

**What happens:**

1. Channel is closed
2. Receives from closed channel return immediately
3. `ok = false` indicates channel closed
4. Worker exits loop

**Note:** We close the channel **after** canceling workers, so workers usually exit via context cancellation, not channel close.

---

## Job State Cleanup on Shutdown

### The Problem

**Without cleanup:**

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Job status is "processing"
    select {
    case <-ctx.Done():
        return  // ❌ Job left in "processing" state!
    case <-timer.C:
        // Complete job
    }
}
```

**Problem:** If shutdown happens during processing, job is stuck in "processing" state forever.

### The Solution

**With cleanup:**

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    timer := time.NewTimer(1 * time.Second)
    defer timer.Stop()

    err := w.metricStore.IncrementJobsInProgress(ctx)
    if err != nil {
        w.logger.Error("Worker error incrementing jobs in progress", "event", "metric_error", "worker_id", w.id, "error", err)
        return
    }

    select {
    case <-timer.C:
        // Processing complete - continue to success/failure logic
    case <-ctx.Done():
        // Shutdown requested, abort processing - clean up job state
        w.logger.Info("Worker job processing aborted due to shutdown", "event", "job_aborted", "worker_id", w.id, "job_id", job.ID)

        // Mark job as failed due to shutdown to prevent it from being stuck in processing state
        lastError := "Job aborted due to shutdown"
        if err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError); err != nil {
            w.logger.Error("Worker error updating aborted job to failed", "event", "job_update_error", "worker_id", w.id, "job_id", job.ID, "error", err)
        } else {
            // IncrementJobsFailed also decrements JobsInProgress, so this handles both metrics
            if err := w.metricStore.IncrementJobsFailed(ctx); err != nil {
                w.logger.Error("Worker error incrementing jobs failed for aborted job", "event", "metric_error", "worker_id", w.id, "error", err)
            }
        }

        return
    }

    // ... continue with normal processing ...
}
```

**What happens:**

1. Shutdown signal received during processing
2. Worker detects `ctx.Done()`
3. Worker marks job as `StatusFailed` with error "Job aborted due to shutdown"
4. Worker updates metrics (increment failed, decrement in-progress)
5. Worker returns (exits cleanly)

**Benefit:** Job is never left in "processing" state.

### Why Mark as Failed?

**Options:**

1. **Mark as Failed** ✅ (Our choice)

   - Job didn't complete
   - Clear error message explains why
   - Can be retried later (if retry logic allows)

2. **Mark as Pending** ❌

   - Misleading - job was attempted
   - No indication it was interrupted
   - Might retry immediately (not desired)

3. **Leave as Processing** ❌
   - Job stuck forever
   - No way to recover
   - System inconsistency

**Our choice:** Mark as Failed with clear error message.

---

## Worker Shutdown Pattern

### Complete Pattern

```go
// 1. Create worker context
workerCtx, workerCancel := context.WithCancel(context.Background())
defer workerCancel()

// 2. Start workers with WaitGroup
var wg sync.WaitGroup
for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, metricStore, logger, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)  // Workers watch context
    })
}

// 3. On shutdown: Cancel context
workerCancel()  // Signals all workers to stop

// 4. Wait for workers to finish
wg.Wait()  // Blocks until all workers exit

// 5. Now safe to close channel
close(jobQueue)
```

### Worker Implementation

```go
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            // Shutdown detected
            return  // Exit cleanly

        case jobID, ok := <-w.jobQueue:
            if !ok {
                // Channel closed
                return  // Exit cleanly
            }

            // Process job (handles shutdown during processing)
            job, err := w.jobStore.ClaimJob(ctx, jobID)
            if err != nil || job == nil {
                continue
            }

            w.processJob(ctx, job)  // May be interrupted
        }
    }
}

func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // ... setup ...

    select {
    case <-ctx.Done():
        // Shutdown during processing - clean up
        w.cleanupJobOnShutdown(ctx, job)
        return
    case <-timer.C:
        // Processing complete - continue normally
    }

    // ... normal completion logic ...
}

func (w *Worker) cleanupJobOnShutdown(ctx context.Context, job *domain.Job) {
    lastError := "Job aborted due to shutdown"
    if err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError); err != nil {
        w.logger.Error("Failed to update aborted job", ...)
    } else {
        w.metricStore.IncrementJobsFailed(ctx)
    }
}
```

---

## Common Mistakes

### Mistake 1: Not Cleaning Up Job State

```go
// ❌ BAD: Job left in processing state
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    select {
    case <-ctx.Done():
        return  // Job stuck in "processing"!
    case <-timer.C:
        // Complete
    }
}
```

**Problem:** Job remains in "processing" state forever.

**Fix:**

```go
// ✅ GOOD: Clean up job state
case <-ctx.Done():
    lastError := "Job aborted due to shutdown"
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
    w.metricStore.IncrementJobsFailed(ctx)
    return
```

### Mistake 2: Not Waiting for Workers

```go
// ❌ BAD: Don't wait for workers
workerCancel()
// Workers might still be processing!
close(jobQueue)  // Might close while workers using it
```

**Problem:** Channel might be closed while workers still using it.

**Fix:**

```go
// ✅ GOOD: Wait for workers
workerCancel()
wg.Wait()  // Wait for all workers to finish
close(jobQueue)  // Safe now
```

### Mistake 3: Ignoring Context in processJob

```go
// ❌ BAD: Doesn't check context during processing
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Long operation that doesn't check ctx.Done()
    doLongWork()  // Can't be interrupted!
}
```

**Problem:** Worker can't respond to shutdown during long operations.

**Fix:**

```go
// ✅ GOOD: Check context periodically
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    for {
        select {
        case <-ctx.Done():
            return  // Can interrupt
        default:
            doWork()  // Do a bit of work
        }
    }
}
```

### Mistake 4: Not Updating Metrics on Cleanup

```go
// ❌ BAD: Updates job state but not metrics
case <-ctx.Done():
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &err)
    // Metrics not updated!
    return
```

**Problem:** Metrics become inconsistent (job failed but not counted).

**Fix:**

```go
// ✅ GOOD: Update both state and metrics
case <-ctx.Done():
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &err)
    w.metricStore.IncrementJobsFailed(ctx)  // Update metrics
    return
```

---

## Key Takeaways

1. **Worker lifecycle** - Start → Run → Shutdown → Cleanup → Stop
2. **Context cancellation** - Workers detect shutdown via `ctx.Done()`
3. **Job state cleanup** - Always clean up job state when interrupted
4. **Wait for workers** - Use WaitGroup to ensure workers finish before closing channels
5. **Mark aborted jobs as failed** - Clear error message explains why job failed
6. **Update metrics** - Keep metrics consistent when cleaning up
7. **Graceful exit** - Workers should exit cleanly, never leave system in inconsistent state

---

## Next Steps

- Read about [Graceful Shutdown Coordination](./01-graceful-shutdown-coordination.md)
- Learn about [Backpressure Implementation](./02-backpressure.md)
- Understand [Channel Closing Strategy](./03-channel-closing-strategy.md)
