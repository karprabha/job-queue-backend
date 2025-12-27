# Understanding Context in Workers

## Table of Contents

1. [Why Context in Workers?](#why-context-in-workers)
2. [Context for Cancellation](#context-for-cancellation)
3. [Our Worker's Context Usage](#our-workers-context-usage)
4. [Context Propagation](#context-propagation)
5. [Context in Processing](#context-in-processing)
6. [Common Patterns](#common-patterns)
7. [Common Mistakes](#common-mistakes)

---

## Why Context in Workers?

### The Problem

Workers run in background goroutines. How do you tell them to stop?

**Without context:**
```go
// ❌ BAD: No way to stop worker
func (w *Worker) Start() {
    for {
        job := <-w.jobQueue
        processJob(job)  // Runs forever, can't stop!
    }
}
```

**Problem:** Worker runs forever, no way to stop it gracefully.

### The Solution: Context

Context provides a standard way to signal cancellation:

```go
// ✅ GOOD: Can be stopped
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return  // Stop when context canceled
        case job := <-w.jobQueue:
            processJob(job)
        }
    }
}
```

**Benefit:** Worker can be stopped by canceling the context.

---

## Context for Cancellation

### What is Context Cancellation?

**Context cancellation** is a way to signal "stop what you're doing" across function calls and goroutines.

### How It Works

```go
// Create cancelable context
ctx, cancel := context.WithCancel(context.Background())

// Start worker with context
go worker.Start(ctx)

// Later, cancel it
cancel()  // Signals worker to stop
```

**What happens:**
1. `ctx.Done()` channel is closed
2. Code checking `ctx.Done()` receives signal
3. Code can exit gracefully

### The Done Channel

```go
ctx.Done()  // Returns a channel
```

**Properties:**
- Returns a `<-chan struct{}`
- Channel is **closed** when context is canceled
- Reading from closed channel returns immediately (non-blocking)

**Usage:**
```go
select {
case <-ctx.Done():
    // Context canceled, stop working
    return
default:
    // Context still active, continue
}
```

---

## Our Worker's Context Usage

### Worker Start Method

```go
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-w.jobQueue:
            if !ok {
                return
            }
            // Process job...
        }
    }
}
```

### Breaking It Down

**1. Context Parameter**
```go
func (w *Worker) Start(ctx context.Context)
```
- Worker accepts context
- Context controls worker lifecycle
- Standard Go pattern

**2. Check for Cancellation**
```go
select {
case <-ctx.Done():
    return  // Exit if canceled
}
```
- Checks if context canceled
- Exits immediately if canceled
- Non-blocking check

**3. Process Jobs**
```go
case job, ok := <-w.jobQueue:
    // Process job (respects context)
}
```
- Receives jobs when available
- Processes them
- Still checks context periodically

### How It's Created in main.go

```go
workerCtx, workerCancel := context.WithCancel(context.Background())
defer workerCancel()

worker := worker.NewWorker(jobStore, jobQueue)
go worker.Start(workerCtx)
```

**What happens:**
1. Create cancelable context
2. Get cancel function
3. Pass context to worker
4. On shutdown: call `workerCancel()`

---

## Context Propagation

### Passing Context Through Calls

Context should be passed through function calls:

```go
func (w *Worker) Start(ctx context.Context) {
    // ...
    w.processJob(ctx, job)  // Pass context down
}

func (w *Worker) processJob(ctx context.Context, job *Job) {
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return  // Respect cancellation
    case <-time.After(1 * time.Second):
        // Process complete
    }
}
```

**Why propagate?**
- Allows cancellation at any level
- Consistent cancellation behavior
- Standard Go pattern

### Our Implementation

```go
// Worker receives context
func (w *Worker) Start(ctx context.Context) {
    // ...
    w.processJob(ctx, job)  // Passes to processJob
}

// processJob receives context
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Uses context for cancellation
    select {
    case <-ctx.Done():
        log.Printf("Job %s processing aborted due to shutdown", job.ID)
        w.updateJobStatus(ctx, job, domain.StatusFailed)
        return
    case <-timer.C:
        // Processing complete
    }
}
```

**Flow:**
```
main() creates context
    ↓
worker.Start(ctx) receives it
    ↓
processJob(ctx, job) receives it
    ↓
Checks ctx.Done() for cancellation
```

---

## Context in Processing

### Cancellation During Processing

**Scenario:** Worker is processing a job when shutdown is requested.

**What should happen?**
- Stop processing immediately
- Mark job appropriately (failed/cancelled)
- Exit gracefully

### Our Implementation

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    timer := time.NewTimer(1 * time.Second)
    defer timer.Stop()

    select {
    case <-timer.C:
        // Processing complete
    case <-ctx.Done():
        // Shutdown requested, abort processing
        log.Printf("Job %s processing aborted due to shutdown", job.ID)
        w.updateJobStatus(ctx, job, domain.StatusFailed)
        return
    }

    w.updateJobStatus(ctx, job, domain.StatusCompleted)
}
```

**What happens:**
1. Start processing (1 second timer)
2. If context canceled during processing:
   - Timer canceled
   - Job marked as failed
   - Function returns (stops processing)
3. If processing completes:
   - Job marked as completed
   - Function returns normally

### Why Mark as Failed?

**Question:** Why mark as failed instead of leaving as processing?

**Answer:** 
- Job was interrupted
- Not completed successfully
- Failed is more accurate than processing
- Allows retry later (if we add retry logic)

---

## Common Patterns

### Pattern 1: Context Check in Loop

```go
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case job := <-w.jobQueue:
            processJob(ctx, job)
        }
    }
}
```

**Use case:** Check cancellation between jobs.

### Pattern 2: Context Check During Processing

```go
func processJob(ctx context.Context, job *Job) {
    select {
    case <-ctx.Done():
        return  // Abort processing
    case <-time.After(duration):
        // Continue processing
    }
}
```

**Use case:** Check cancellation during long operations.

### Pattern 3: Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

processJob(ctx, job)  // Max 30 seconds
```

**Use case:** Prevent jobs from running too long.

### Pattern 4: Context in Store Operations

```go
func (s *Store) UpdateJob(ctx context.Context, job *Job) error {
    select {
    case <-ctx.Done():
        return ctx.Err()  // Respect cancellation
    default:
    }
    // Do update...
}
```

**Use case:** Respect cancellation in store operations.

---

## Common Mistakes

### Mistake 1: Not Accepting Context

```go
// ❌ BAD: Can't be canceled
func (w *Worker) Start() {
    for {
        job := <-w.jobQueue
        processJob(job)
    }
}
```

**Problem:** No way to stop worker.

**Fix:** Accept context
```go
// ✅ GOOD: Can be canceled
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case job := <-w.jobQueue:
            processJob(ctx, job)
        }
    }
}
```

### Mistake 2: Not Checking Context

```go
// ❌ BAD: Ignores cancellation
func (w *Worker) Start(ctx context.Context) {
    for {
        job := <-w.jobQueue
        processJob(job)  // Never checks ctx.Done()
    }
}
```

**Problem:** Worker doesn't respond to cancellation.

**Fix:** Check context
```go
// ✅ GOOD: Checks context
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case job := <-w.jobQueue:
            processJob(ctx, job)
        }
    }
}
```

### Mistake 3: Not Propagating Context

```go
// ❌ BAD: Context not passed down
func (w *Worker) Start(ctx context.Context) {
    w.processJob(job)  // Missing ctx parameter
}

func (w *Worker) processJob(job *Job) {
    // Can't check context here
}
```

**Problem:** Lower-level functions can't respect cancellation.

**Fix:** Propagate context
```go
// ✅ GOOD: Context propagated
func (w *Worker) Start(ctx context.Context) {
    w.processJob(ctx, job)  // Passes context
}

func (w *Worker) processJob(ctx context.Context, job *Job) {
    select {
    case <-ctx.Done():
        return
    default:
        // Process
    }
}
```

### Mistake 4: Creating New Context

```go
// ❌ BAD: Creates new context, ignores cancellation
func (w *Worker) processJob(ctx context.Context, job *Job) {
    newCtx := context.Background()  // Wrong!
    doWork(newCtx)  // Won't respect parent cancellation
}
```

**Problem:** New context doesn't respect parent cancellation.

**Fix:** Use parent context or derive from it
```go
// ✅ GOOD: Uses parent context
func (w *Worker) processJob(ctx context.Context, job *Job) {
    doWork(ctx)  // Respects parent cancellation
}

// Or derive with timeout
func (w *Worker) processJob(ctx context.Context, job *Job) {
    timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    doWork(timeoutCtx)  // Respects both parent and timeout
}
```

### Mistake 5: Not Handling Cancellation in Long Operations

```go
// ❌ BAD: Long operation, no cancellation check
func processJob(ctx context.Context, job *Job) {
    time.Sleep(10 * time.Second)  // Blocks, can't cancel
    updateStatus(job)
}
```

**Problem:** Can't cancel during long operation.

**Fix:** Check context during operation
```go
// ✅ GOOD: Checks context during operation
func processJob(ctx context.Context, job *Job) {
    select {
    case <-time.After(10 * time.Second):
        updateStatus(job)
    case <-ctx.Done():
        return  // Can cancel
    }
}
```

---

## Key Takeaways

1. **Context in workers** = Enables graceful shutdown
2. **Context.Done()** = Channel that closes on cancellation
3. **Propagate context** = Pass through function calls
4. **Check context** = In loops and long operations
5. **Respect cancellation** = Exit gracefully when canceled
6. **Standard pattern** = First parameter is usually context

---

## Real-World Analogy

Think of context like a **stop button**:

- **Creating context** = Installing stop button
- **Cancel()** = Pressing stop button
- **ctx.Done()** = Checking if button was pressed
- **Propagating context** = Passing stop button to workers

When you press the stop button, all workers should stop what they're doing.

---

## Next Steps

- Read [Graceful Shutdown](./05-graceful-shutdown.md) to see how context enables shutdown
- Read [Worker Pattern](./03-worker-pattern.md) for complete worker design
- Read [Context in Go](../task1/concepts/01-context.md) for context basics

