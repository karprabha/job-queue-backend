# Understanding the Worker Pattern

## Table of Contents

1. [What is the Worker Pattern?](#what-is-the-worker-pattern)
2. [Why Use a Worker Pattern?](#why-use-a-worker-pattern)
3. [Our Worker Implementation](#our-worker-implementation)
4. [Worker Lifecycle](#worker-lifecycle)
5. [Worker Responsibilities](#worker-responsibilities)
6. [Worker Design Decisions](#worker-design-decisions)
7. [Common Patterns](#common-patterns)
8. [Common Mistakes](#common-mistakes)

---

## What is the Worker Pattern?

### The Core Idea

The **worker pattern** is a concurrency design where:
1. One or more **worker goroutines** process work items
2. Work items are sent via a **channel** (queue)
3. Workers continuously receive and process items
4. Workers run independently of request handlers

### Visual Representation

```
HTTP Handlers (Producers)
    ↓
[Job Queue Channel]
    ↓
Worker (Consumer)
    ↓
Job Store (Updates)
```

**Flow:**
1. HTTP handler creates job → sends to queue
2. Worker receives job from queue
3. Worker processes job
4. Worker updates job status in store

---

## Why Use a Worker Pattern?

### Problem 1: Blocking HTTP Handlers

**Without workers:**
```go
func CreateJobHandler(...) {
    job := createJob()
    processJob(job)  // Takes 5 seconds!
    // HTTP request blocked for 5 seconds ❌
}
```

**Problems:**
- HTTP handler blocked during processing
- Slow response times
- Can't handle concurrent requests well

**With workers:**
```go
func CreateJobHandler(...) {
    job := createJob()
    jobQueue <- job  // Send to queue (instant)
    // HTTP request returns immediately ✅
}

// Worker (separate goroutine)
func worker() {
    job := <-jobQueue  // Receive from queue
    processJob(job)  // Takes 5 seconds (doesn't block HTTP)
}
```

**Benefits:**
- HTTP handler returns immediately
- Fast response times
- Worker processes in background

### Problem 2: Resource Management

**Without workers:**
- Each HTTP request might spawn processing
- No control over concurrent processing
- Can overwhelm system

**With workers:**
- Fixed number of workers
- Controlled concurrency
- Predictable resource usage

### Problem 3: Decoupling

**Without workers:**
- HTTP layer tightly coupled to processing logic
- Can't change processing without changing HTTP
- Hard to test processing independently

**With workers:**
- HTTP layer only creates and enqueues
- Processing logic in separate package
- Easy to test independently

---

## Our Worker Implementation

### The Worker Struct

```go
type Worker struct {
    jobStore store.JobStore
    jobQueue chan *domain.Job
}
```

**Fields:**
- `jobStore` - Interface to update job status
- `jobQueue` - Channel to receive jobs

**Why these fields?**
- `jobStore` - Worker needs to update job status
- `jobQueue` - Worker receives jobs from this channel

### The Constructor

```go
func NewWorker(jobStore store.JobStore, jobQueue chan *domain.Job) *Worker {
    return &Worker{
        jobStore: jobStore,
        jobQueue: jobQueue,
    }
}
```

**Pattern:** Dependency injection
- Dependencies passed in (not created inside)
- Enables testing (can inject mocks)
- Clear dependencies

### The Start Method

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
            claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
            if err != nil {
                log.Printf("Error claiming job: %s: %v", job.ID, err)
                continue
            }
            if !claimed {
                continue
            }
            w.processJob(ctx, job)
        }
    }
}
```

**Breaking it down:**

**1. Infinite Loop**
```go
for {
    // Keep processing until shutdown
}
```
- Worker runs continuously
- Processes jobs one after another
- Only stops when context canceled or channel closed

**2. Select Statement**
```go
select {
case <-ctx.Done():
    return  // Shutdown signal
case job, ok := <-w.jobQueue:
    // Job received
}
```
- Waits for either shutdown or job
- Blocks until one happens
- Non-busy waiting (efficient)

**3. Channel Closed Check**
```go
if !ok {
    return  // Channel closed, exit
}
```
- Detects when channel is closed
- Exits gracefully

**4. Claim Job**
```go
claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
```
- Atomically claims job (prevents duplicate processing)
- Returns `false` if job already claimed or not pending
- We'll explain this in detail in atomic operations

**5. Process Job**
```go
w.processJob(ctx, job)
```
- Processes the claimed job
- Updates status to processing → completed

---

## Worker Lifecycle

### Stage 1: Creation

```go
worker := worker.NewWorker(jobStore, jobQueue)
```

- Worker struct created
- Dependencies injected
- Not running yet (just a struct)

### Stage 2: Starting

```go
go worker.Start(workerCtx)
```

- Goroutine created
- `Start()` method begins executing
- Enters the `for` loop
- Blocks on `select` (waiting for jobs)

### Stage 3: Running

```go
// Worker receives job
job := <-w.jobQueue

// Worker processes job
w.processJob(ctx, job)

// Returns to select (waits for next job)
```

- Worker continuously processes jobs
- Blocks when no jobs available
- Unblocks when job arrives

### Stage 4: Shutdown

```go
// Context canceled
workerCancel()

// Worker receives cancellation
case <-ctx.Done():
    return  // Exits loop
```

- Context canceled
- Worker receives cancellation signal
- Exits gracefully
- Goroutine terminates

---

## Worker Responsibilities

### What the Worker Does

1. **Receives jobs** from channel
2. **Claims jobs** atomically (prevents duplicates)
3. **Processes jobs** (simulated with sleep)
4. **Updates job status** (pending → processing → completed)
5. **Handles errors** (logs, continues processing)
6. **Respects shutdown** (stops on context cancellation)

### What the Worker Does NOT Do

1. **Doesn't create jobs** (HTTP handler does this)
2. **Doesn't handle HTTP** (separate concern)
3. **Doesn't manage channel** (main() creates and closes)
4. **Doesn't decide what to process** (just processes what it receives)

### Separation of Concerns

```
HTTP Handler:
  - Creates job
  - Stores job
  - Sends to queue
  - Returns response

Worker:
  - Receives from queue
  - Claims job
  - Processes job
  - Updates status

Store:
  - Stores jobs
  - Updates jobs
  - Provides atomic operations
```

**Key Point:** Each component has a single, clear responsibility.

---

## Worker Design Decisions

### Decision 1: Single Worker

**Choice:** One worker goroutine (for now)

**Why?**
- Task requirement (single worker)
- Simpler to understand
- Easier to debug
- Can scale later (add more workers)

**Future:** Can easily add more workers:
```go
for i := 0; i < numWorkers; i++ {
    go worker.Start(ctx)
}
```

### Decision 2: Channel-Based Communication

**Choice:** Use channel instead of polling store

**Why?**
- More efficient (no busy polling)
- Natural backpressure (channel blocks when full)
- Clear ownership (channel controls flow)
- Go idiom (channels are the way to communicate)

**Alternative (not chosen):**
```go
// ❌ Polling approach
for {
    jobs := store.GetPendingJobs()
    for _, job := range jobs {
        process(job)
    }
    time.Sleep(1 * time.Second)  // Busy polling
}
```

**Problems:**
- Wastes CPU (polls even when no jobs)
- Race conditions (multiple workers might get same job)
- No coordination

### Decision 3: Claim Before Process

**Choice:** Claim job atomically before processing

**Why?**
- Prevents duplicate processing
- Ensures only pending jobs are processed
- Atomic operation (thread-safe)

**Implementation:**
```go
claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
if !claimed {
    continue  // Job already claimed or not pending
}
w.processJob(ctx, job)  // Only process if claimed
```

### Decision 4: Context for Shutdown

**Choice:** Use context.Context for cancellation

**Why?**
- Standard Go pattern
- Can cancel from anywhere
- Propagates through call chain
- Works with timeouts

**Usage:**
```go
func (w *Worker) Start(ctx context.Context) {
    // Respects context cancellation
    select {
    case <-ctx.Done():
        return
    }
}
```

---

## Common Patterns

### Pattern 1: Worker Pool

```go
// Multiple workers processing same queue
for i := 0; i < numWorkers; i++ {
    go worker.Start(ctx)
}
```

**Use case:** Process more jobs concurrently

### Pattern 2: Worker with Timeout

```go
func (w *Worker) processJob(ctx context.Context, job *Job) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Process with timeout
    doWork(ctx)
}
```

**Use case:** Prevent jobs from running too long

### Pattern 3: Worker with Retry

```go
func (w *Worker) processJob(ctx context.Context, job *Job) {
    for attempts := 0; attempts < maxRetries; attempts++ {
        if err := doWork(); err == nil {
            return  // Success
        }
        time.Sleep(backoff(attempts))
    }
}
```

**Use case:** Handle transient failures

### Pattern 4: Worker with Metrics

```go
func (w *Worker) processJob(ctx context.Context, job *Job) {
    start := time.Now()
    defer func() {
        metrics.RecordDuration(time.Since(start))
    }()
    
    doWork()
}
```

**Use case:** Monitor worker performance

---

## Common Mistakes

### Mistake 1: Processing Without Claiming

```go
// ❌ BAD: Race condition!
func (w *Worker) Start(ctx context.Context) {
    for {
        job := <-w.jobQueue
        w.processJob(ctx, job)  // Multiple workers might process same job!
    }
}
```

**Problem:** If multiple workers exist, same job might be processed twice.

**Fix:** Claim before processing
```go
// ✅ GOOD: Atomic claim
claimed, _ := w.jobStore.ClaimJob(ctx, job.ID)
if !claimed {
    continue
}
w.processJob(ctx, job)
```

### Mistake 2: Not Checking Channel Closed

```go
// ❌ BAD: Infinite loop on closed channel
func (w *Worker) Start(ctx context.Context) {
    for {
        job := <-w.jobQueue  // Receives zero value forever
        w.processJob(ctx, job)
    }
}
```

**Fix:** Check `ok` value
```go
// ✅ GOOD: Checks if closed
job, ok := <-w.jobQueue
if !ok {
    return
}
```

### Mistake 3: Ignoring Context

```go
// ❌ BAD: Can't stop worker
func (w *Worker) Start(ctx context.Context) {
    for {
        job := <-w.jobQueue
        w.processJob(ctx, job)  // Never checks ctx.Done()
    }
}
```

**Fix:** Check context in select
```go
// ✅ GOOD: Respects context
select {
case <-ctx.Done():
    return
case job := <-w.jobQueue:
    w.processJob(ctx, job)
}
```

### Mistake 4: Processing in HTTP Handler

```go
// ❌ BAD: Blocks HTTP handler
func CreateJobHandler(...) {
    job := createJob()
    processJob(job)  // Don't do this!
}
```

**Fix:** Send to worker
```go
// ✅ GOOD: Sends to worker
func CreateJobHandler(...) {
    job := createJob()
    jobQueue <- job  // Worker processes it
}
```

### Mistake 5: No Error Handling

```go
// ❌ BAD: Panics on error
func (w *Worker) processJob(ctx context.Context, job *Job) {
    w.jobStore.UpdateJob(ctx, job)  // Might fail, no handling
}
```

**Fix:** Handle errors gracefully
```go
// ✅ GOOD: Handles errors
func (w *Worker) processJob(ctx context.Context, job *Job) {
    if err := w.jobStore.UpdateJob(ctx, job); err != nil {
        log.Printf("Error: %v", err)
        return
    }
}
```

---

## Key Takeaways

1. **Worker pattern** = Background processing with channel communication
2. **Single responsibility** = Worker only processes, doesn't create
3. **Channel-based** = Efficient, no polling
4. **Claim before process** = Prevents duplicates
5. **Context for shutdown** = Graceful termination
6. **Error handling** = Log and continue, don't crash
7. **Separation of concerns** = HTTP, worker, store are separate

---

## Real-World Analogy

Think of a restaurant:

- **HTTP Handler** = Waiter (takes order, gives to kitchen)
- **Job Queue** = Order ticket system (waiter puts ticket, kitchen gets it)
- **Worker** = Chef (processes orders from ticket system)
- **Store** = Order tracking system (updates order status)

The waiter doesn't cook - they just take orders and give them to the kitchen. The chef (worker) processes orders from the ticket system (channel) independently.

---

## Next Steps

- Read [Atomic Operations](./07-atomic-operations.md) to understand ClaimJob
- Read [Graceful Shutdown](./05-graceful-shutdown.md) to learn how workers stop
- Read [Context in Workers](./06-context-in-workers.md) for context usage

