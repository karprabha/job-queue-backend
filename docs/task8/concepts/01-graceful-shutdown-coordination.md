# Graceful Shutdown Coordination

## Table of Contents

1. [What is Graceful Shutdown Coordination?](#what-is-graceful-shutdown-coordination)
2. [Why Coordination Matters](#why-coordination-matters)
3. [The Shutdown Challenge](#the-shutdown-challenge)
4. [Our Shutdown Sequence](#our-shutdown-sequence)
5. [Component Lifecycle Management](#component-lifecycle-management)
6. [Context Propagation for Shutdown](#context-propagation-for-shutdown)
7. [Common Mistakes](#common-mistakes)

---

## What is Graceful Shutdown Coordination?

### The Simple Answer

**Graceful shutdown coordination** means orchestrating the shutdown of multiple components in the correct order, ensuring:
- No work is lost
- No resources leak
- No panics occur
- All components stop cleanly

### The Challenge

When you have multiple components running concurrently:
- HTTP server accepting requests
- Workers processing jobs
- Sweeper retrying failed jobs
- Channels carrying data

**Question:** How do you shut them all down safely?

**Answer:** You need a **coordinated shutdown sequence** that respects dependencies and ensures each component finishes its current work before stopping.

---

## Why Coordination Matters

### Problem 1: Jobs Left in Processing State

**Without proper coordination:**

```go
// Worker is processing a job
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

**With proper coordination:**

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    select {
    case <-ctx.Done():
        // Clean up: mark job as failed
        lastError := "Job aborted due to shutdown"
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
        w.metricStore.IncrementJobsFailed(ctx)
        return  // ✅ Job properly cleaned up
    case <-timer.C:
        // Complete job
    }
}
```

**Benefit:** Jobs are never left in an inconsistent state.

### Problem 2: Handlers Accepting Jobs During Shutdown

**Without shutdown state checking:**

```go
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // Server is shutting down, but handler doesn't know
    job := domain.NewJob(...)
    h.jobQueue <- job.ID  // ❌ Job accepted during shutdown!
}
```

**Problem:** New jobs can be accepted even after shutdown starts, leading to:
- Jobs created but never processed
- Inconsistent system state
- Confusing user experience

**With shutdown state checking:**

```go
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // Check shutdown state first
    select {
    case <-h.shutdownCtx.Done():
        ErrorResponse(w, "Server is shutting down", http.StatusServiceUnavailable)
        return  // ✅ Reject new jobs during shutdown
    default:
    }
    
    // Continue with job creation...
}
```

**Benefit:** System cleanly rejects new work during shutdown.

### Problem 3: Channel Closing Race Conditions

**Without proper sequence:**

```go
// Close channel first
close(jobQueue)  // ❌ Workers might still be trying to send!

// Then cancel workers
workerCancel()
wg.Wait()
```

**Problem:** If a handler tries to send to the channel after it's closed, you get a panic: "send on closed channel".

**With proper sequence:**

```go
// 1. Signal shutdown to handlers (they stop accepting)
shutdownCancel()

// 2. Shutdown HTTP server (stops new requests)
srv.Shutdown(ctx)

// 3. Stop sweeper
sweeperCancel()
sweeperWg.Wait()

// 4. Cancel workers and wait for them to finish
workerCancel()
wg.Wait()

// 5. Now safe to close channel
close(jobQueue)  // ✅ No one is using it anymore
```

**Benefit:** No panics, no race conditions.

---

## The Shutdown Challenge

### Components That Need Coordination

1. **HTTP Server** - Accepts requests, must stop accepting new ones
2. **Handlers** - Process requests, must reject new jobs during shutdown
3. **Workers** - Process jobs, must finish current job before stopping
4. **Sweeper** - Retries failed jobs, must stop gracefully
5. **Channels** - Carry job IDs, must be closed safely

### Dependencies

```
HTTP Server
  └─> Handlers (depend on server being up)
       └─> Job Queue (handlers send to it)
            └─> Workers (read from it)
                 └─> Store (workers update it)
```

**Key insight:** You must shut down in **reverse dependency order**:
1. Stop accepting new work (handlers)
2. Finish current work (workers)
3. Close communication channels (queue)

---

## Our Shutdown Sequence

### The Complete Flow

```go
// 1. Signal shutdown to handlers (they will reject new jobs)
shutdownCancel()
logger.Info("Shutdown signal sent to handlers")

// 2. Shutdown HTTP server (stops accepting new requests, waits for in-flight)
serverShutdownCtx, serverShutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer serverShutdownCancel()

if err := srv.Shutdown(serverShutdownCtx); err != nil {
    if err == context.DeadlineExceeded {
        logger.Warn("Server shutdown timeout exceeded, forcing close")
    } else {
        logger.Error("Server shutdown error", "error", err)
    }
}

// 3. Cancel sweeper and wait
sweeperCancel()
sweeperWg.Wait()
logger.Info("Sweeper stopped")

// 4. Cancel workers (stops picking new jobs) and wait for them to finish current jobs
workerCancel()
wg.Wait()
logger.Info("Workers stopped")

// 5. Close the job queue (safe now that workers are done)
close(jobQueue)

logger.Info("Server stopped")
```

### Step-by-Step Breakdown

#### Step 1: Signal Shutdown to Handlers

```go
shutdownCancel()
```

**What happens:**
- `shutdownCtx.Done()` channel closes
- All handlers checking `h.shutdownCtx.Done()` will return `503 Service Unavailable`
- New jobs are rejected immediately

**Why first:** Prevents new work from entering the system.

#### Step 2: Shutdown HTTP Server

```go
serverShutdownCtx, serverShutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer serverShutdownCancel()

if err := srv.Shutdown(serverShutdownCtx); err != nil {
    // Handle timeout or other errors
}
```

**What happens:**
- Server stops accepting new connections
- Waits for in-flight requests to complete
- Times out after 10 seconds if requests take too long

**Why with timeout:** Prevents server from waiting forever if a request hangs.

**Why second:** Allows in-flight requests to complete, but no new ones start.

#### Step 3: Stop Sweeper

```go
sweeperCancel()
sweeperWg.Wait()
```

**What happens:**
- Sweeper's context is canceled
- Sweeper stops its ticker loop
- WaitGroup ensures sweeper goroutine finishes

**Why third:** Sweeper might be trying to enqueue jobs, but we've already stopped accepting new work.

#### Step 4: Stop Workers

```go
workerCancel()
wg.Wait()
```

**What happens:**
- Worker contexts are canceled
- Workers finish their current job (if any)
- Workers stop picking new jobs from queue
- WaitGroup ensures all worker goroutines finish

**Why fourth:** Workers must finish current jobs before we can safely close the channel.

**Critical:** Workers check `ctx.Done()` during job processing and clean up job state if shutdown happens.

#### Step 5: Close Channel

```go
close(jobQueue)
```

**What happens:**
- Channel is closed
- Any remaining reads from channel return `ok = false`
- No more sends possible (would panic)

**Why last:** Only safe to close after all senders (handlers, sweeper) and receivers (workers) have stopped.

---

## Component Lifecycle Management

### Context Creation Pattern

```go
// In main.go - create contexts for each component
sweeperCtx, sweeperCancel := context.WithCancel(context.Background())
defer sweeperCancel()

workerCtx, workerCancel := context.WithCancel(context.Background())
defer workerCancel()

// Create shutdown context for handlers
shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
defer shutdownCancel()
```

**Key points:**
- Each component gets its own context
- `defer cancel()` ensures cleanup even if main panics
- Contexts are independent (can cancel separately)

### WaitGroup Pattern

```go
// For sweeper
var sweeperWg sync.WaitGroup
sweeperWg.Go(func() {
    sweeper.Run(sweeperCtx)
})

// For workers
var wg sync.WaitGroup
for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, ...)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}
```

**Key points:**
- `wg.Go()` (Go 1.21+) automatically handles `Add(1)` and `Done()`
- Each goroutine must call `defer wg.Done()` or use `wg.Go()`
- `wg.Wait()` blocks until all goroutines finish

### Component Shutdown Pattern

```go
// Each component respects context cancellation
func (s *Sweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return  // Stop when context canceled
        case <-ticker.C:
            // Do work
        }
    }
}
```

**Key points:**
- Components check `ctx.Done()` in their main loop
- Return immediately when context is canceled
- Clean up resources (like tickers) with `defer`

---

## Context Propagation for Shutdown

### The Shutdown Context

```go
// Create shutdown context for handlers
shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
defer shutdownCancel()

// Inject into handler
jobHandler := internalhttp.NewJobHandler(..., shutdownCtx)
```

**Purpose:** Handlers can check if shutdown has started.

### Handler Shutdown Check

```go
type JobHandler struct {
    shutdownCtx context.Context  // Injected shutdown context
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // Check shutdown state first
    select {
    case <-h.shutdownCtx.Done():
        ErrorResponse(w, "Server is shutting down", http.StatusServiceUnavailable)
        return
    default:
    }
    
    // Continue with normal processing...
}
```

**How it works:**
- `select` with `default` is non-blocking
- If `shutdownCtx.Done()` is closed, return immediately
- Otherwise, continue with normal processing

**Why this pattern:** Non-blocking check doesn't delay normal requests.

### Worker Context Cancellation

```go
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return  // Stop when canceled
        case jobID, ok := <-w.jobQueue:
            if !ok {
                return  // Channel closed
            }
            // Process job
        }
    }
}
```

**How it works:**
- Worker checks `ctx.Done()` in its main loop
- When context is canceled, worker stops picking new jobs
- Worker finishes current job (if any) before returning

**Why this pattern:** Ensures workers stop gracefully without losing work.

---

## Common Mistakes

### Mistake 1: Closing Channel Before Workers Stop

```go
// ❌ BAD: Channel closed while workers still running
close(jobQueue)
workerCancel()
wg.Wait()
```

**Problem:** If a handler tries to send after channel is closed, you get a panic.

**Fix:**

```go
// ✅ GOOD: Workers stop first, then close channel
workerCancel()
wg.Wait()
close(jobQueue)
```

### Mistake 2: Not Checking Shutdown State in Handlers

```go
// ❌ BAD: Handler accepts jobs during shutdown
func (h *JobHandler) CreateJob(...) {
    job := domain.NewJob(...)
    h.jobQueue <- job.ID  // Accepted even during shutdown!
}
```

**Problem:** Jobs can be created but never processed.

**Fix:**

```go
// ✅ GOOD: Check shutdown state first
func (h *JobHandler) CreateJob(...) {
    select {
    case <-h.shutdownCtx.Done():
        ErrorResponse(w, "Server is shutting down", http.StatusServiceUnavailable)
        return
    default:
    }
    // Continue...
}
```

### Mistake 3: Not Cleaning Up Job State on Shutdown

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

**Problem:** Jobs remain in "processing" state forever.

**Fix:**

```go
// ✅ GOOD: Clean up job state
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    select {
    case <-ctx.Done():
        lastError := "Job aborted due to shutdown"
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
        w.metricStore.IncrementJobsFailed(ctx)
        return
    case <-timer.C:
        // Complete
    }
}
```

### Mistake 4: No Timeout on Server Shutdown

```go
// ❌ BAD: Server might wait forever
srv.Shutdown(context.Background())
```

**Problem:** If a request hangs, server never shuts down.

**Fix:**

```go
// ✅ GOOD: Timeout ensures shutdown completes
serverShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(serverShutdownCtx)
```

### Mistake 5: Wrong Shutdown Order

```go
// ❌ BAD: Wrong order
close(jobQueue)
srv.Shutdown(ctx)
workerCancel()
```

**Problem:** Handlers might try to send to closed channel, causing panic.

**Fix:**

```go
// ✅ GOOD: Correct order
shutdownCancel()  // Handlers reject new work
srv.Shutdown(ctx)  // Server stops accepting
sweeperCancel()    // Sweeper stops
workerCancel()     // Workers stop
wg.Wait()          // Wait for workers
close(jobQueue)    // Safe to close
```

---

## Key Takeaways

1. **Shutdown coordination is critical** - Multiple components must stop in the right order
2. **Context propagation** - Use contexts to signal shutdown to all components
3. **WaitGroups** - Wait for goroutines to finish before closing channels
4. **Shutdown state checking** - Handlers must reject new work during shutdown
5. **Job state cleanup** - Workers must clean up job state when aborted
6. **Proper sequence** - Stop accepting work → Finish current work → Close channels
7. **Timeouts** - Always use timeouts for shutdown operations

---

## Next Steps

- Read about [Backpressure Implementation](./02-backpressure.md)
- Learn about [Channel Closing Strategy](./03-channel-closing-strategy.md)
- Understand [Worker Lifecycle Management](./04-worker-lifecycle-management.md)

