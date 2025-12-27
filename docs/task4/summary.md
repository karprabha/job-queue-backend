# Task 4 — Summary of Learnings

## Quick Reference

### Worker Pattern

```go
// 1. Define worker struct with dependencies
type Worker struct {
    jobStore store.JobStore
    jobQueue chan *domain.Job
}

// 2. Constructor accepts dependencies
func NewWorker(jobStore store.JobStore, jobQueue chan *domain.Job) *Worker {
    return &Worker{
        jobStore: jobStore,
        jobQueue: jobQueue,
    }
}

// 3. Start method runs in goroutine
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

### Channel Creation

```go
const jobQueueCapacity = 100
jobQueue := make(chan *domain.Job, jobQueueCapacity)
```

### Worker Startup

```go
workerCtx, workerCancel := context.WithCancel(context.Background())
defer workerCancel()

worker := worker.NewWorker(jobStore, jobQueue)

var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(workerCtx)
}()
```

### Graceful Shutdown

```go
// 1. Cancel worker context
workerCancel()

// 2. Wait for worker to finish
wg.Wait()

// 3. Close channel
close(jobQueue)

// 4. Shutdown HTTP server
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()
srv.Shutdown(shutdownCtx)
```

### Job Enqueueing in Handler

```go
// Handler sends job to queue
select {
case h.jobQueue <- job:
    // Successfully enqueued
case <-time.After(100 * time.Millisecond):
    log.Printf("Warning: Job queue full, job %s may be delayed", job.ID)
case <-r.Context().Done():
    return
}
```

### Atomic Job Claiming

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
        return false, nil
    }

    job.Status = domain.StatusProcessing
    s.jobs[jobID] = job

    return true, nil
}
```

## Key Concepts

### Goroutines for Workers

- **What**: Lightweight threads for background processing
- **Why**: Don't block HTTP handlers, process asynchronously
- **How**: `go worker.Start(ctx)` starts worker in separate goroutine
- **Ownership**: main() owns worker goroutine, controls lifecycle

### Channels for Communication

- **What**: Typed conduit for sending/receiving between goroutines
- **Why**: Thread-safe communication, natural synchronization
- **Buffered vs Unbuffered**: Buffered = async, Unbuffered = sync
- **Our choice**: Buffered (capacity 100) for decoupling and burst handling

### Worker Pattern

- **What**: Background goroutine that processes work from channel
- **Why**: Decouples HTTP from processing, enables async operations
- **Responsibilities**: Receive jobs, claim atomically, process, update status
- **Lifecycle**: Start → Process → Shutdown (via context)

### Channel Buffering

- **Buffered**: Can queue values, sender doesn't block until full
- **Unbuffered**: Sender and receiver must meet
- **Our decision**: Buffered with capacity 100
- **Reasons**: Decouple HTTP handler, handle bursts, natural backpressure

### Graceful Shutdown

- **What**: Controlled shutdown that finishes current work
- **Components**: Context cancellation, WaitGroup, channel closing
- **Order**: Cancel context → Wait for goroutines → Close channels → Shutdown server
- **Why**: Prevents data loss, resource leaks, incomplete operations

### Context in Workers

- **Purpose**: Enable cancellation and shutdown
- **Pattern**: First parameter is context
- **Usage**: Check `ctx.Done()` in loops and long operations
- **Propagation**: Pass context through function calls

### Atomic Operations

- **Problem**: Race conditions when multiple workers access same job
- **Solution**: ClaimJob atomically checks and sets status
- **How**: Mutex protects check-and-set operation
- **Result**: Only one worker can claim a job

### Select Statement

- **What**: Like switch, but for channels
- **Why**: Wait for multiple channels simultaneously
- **Pattern**: `select { case <-ctx.Done(): return; case job := <-queue: process }`
- **Behavior**: Blocks until one case ready, random if multiple ready

## Project Structure

```
internal/
├── domain/
│   └── job.go              # Domain model (added status constants)
├── http/
│   └── job_handler.go      # Handler sends jobs to queue
├── store/
│   └── job_store.go        # Added UpdateJob, ClaimJob methods
└── worker/                 # NEW: Worker package
    └── worker.go           # Worker implementation
```

## Common Patterns

### Worker Startup Pattern

```go
1. Create context with cancel
   ↓
2. Create worker with dependencies
   ↓
3. Create WaitGroup
   ↓
4. Start worker in goroutine
   ↓
5. Worker runs until context canceled
```

### Job Processing Flow

```go
1. HTTP handler creates job
   ↓
2. Handler stores job
   ↓
3. Handler sends job to queue (channel)
   ↓
4. Handler returns response (fast!)
   ↓
5. Worker receives job from queue
   ↓
6. Worker claims job atomically
   ↓
7. Worker processes job
   ↓
8. Worker updates status (processing → completed)
```

### Shutdown Pattern

```go
1. Signal received (SIGINT/SIGTERM)
   ↓
2. Cancel worker context
   ↓
3. Wait for worker (WaitGroup)
   ↓
4. Close job queue channel
   ↓
5. Shutdown HTTP server (with timeout)
```

### Atomic Claiming Pattern

```go
1. Acquire lock
   ↓
2. Check job exists and is pending
   ↓
3. Set status to processing
   ↓
4. Save to store
   ↓
5. Release lock
   ↓
6. Return success/failure
```

## Checklist: Worker Implementation

- [ ] Worker package created (`internal/worker/`)
- [ ] Worker struct with dependencies (store, channel)
- [ ] Constructor function accepts dependencies
- [ ] Start method accepts context
- [ ] Start method uses select for cancellation
- [ ] Worker checks channel closed (`ok` value)
- [ ] Worker claims job before processing
- [ ] Worker processes job with timeout
- [ ] Worker updates job status
- [ ] Worker respects context cancellation

## Checklist: Channel Setup

- [ ] Channel created with appropriate capacity
- [ ] Channel type matches job type
- [ ] Channel passed to worker and handler
- [ ] Handler sends jobs to channel
- [ ] Worker receives from channel
- [ ] Channel closed on shutdown (after worker stops)

## Checklist: Graceful Shutdown

- [ ] Worker context created with cancel
- [ ] WaitGroup created and used
- [ ] Worker started in goroutine with defer wg.Done()
- [ ] Context canceled on shutdown
- [ ] WaitGroup.Wait() called before closing channel
- [ ] Channel closed after worker stops
- [ ] HTTP server shutdown with timeout
- [ ] Shutdown order is correct

## Checklist: Atomic Operations

- [ ] ClaimJob method added to store interface
- [ ] ClaimJob checks context before lock
- [ ] ClaimJob acquires lock
- [ ] ClaimJob checks job exists and is pending
- [ ] ClaimJob atomically updates status
- [ ] ClaimJob releases lock (defer)
- [ ] Worker calls ClaimJob before processing
- [ ] Worker handles claim failure (not claimed)

## Important Notes

1. **Always use buffered channels** for producer/consumer patterns (unless you need tight coupling)
2. **Always check context** in worker loops and long operations
3. **Always claim before process** to prevent duplicate processing
4. **Always use WaitGroup** to wait for goroutines to finish
5. **Always close channels** after workers stop (not before)
6. **Always check `ok`** when receiving from channels (detect closed)
7. **Shutdown order matters** - Stop workers → Wait → Close channels → Shutdown server

## Design Decisions

### Why Buffered Channel (Capacity 100)?

- Decouples HTTP handler from worker
- Handles traffic bursts
- Natural backpressure when full
- Better HTTP response times
- Balance between throughput and memory

### Why ClaimJob Instead of Check-Then-Set?

- Prevents race conditions
- Atomic operation (protected by mutex)
- Only one worker can claim a job
- Prevents duplicate processing

### Why Context for Shutdown?

- Standard Go pattern
- Can cancel from anywhere
- Propagates through call chain
- Works with timeouts
- Enables graceful shutdown

### Why WaitGroup?

- Tracks goroutine completion
- Ensures worker finishes before exit
- Prevents goroutine leaks
- Standard pattern for coordination

### Why Single Worker?

- Task requirement (simplicity)
- Easier to understand and debug
- Can scale later (add more workers)
- Foundation for worker pools

## Next Steps

- Review detailed concepts in [`concepts/`](./concepts/) directory
- Understand goroutines and channels deeply
- Learn about worker pools (multiple workers)
- Explore backpressure handling
- Study advanced concurrency patterns

