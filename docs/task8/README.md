# Task 8 — Graceful Shutdown, Backpressure & Load Safety

## Overview

This task introduces **graceful shutdown, backpressure, and load safety** to the job queue system. The focus is on production-ready behavior: safe shutdown, predictable rejection of work under load, and ensuring no work is lost.

## ✅ Completed Requirements

### Functional Requirements

- ✅ Graceful shutdown implemented
- ✅ Jobs cleaned up on shutdown (no jobs left in "processing" state)
- ✅ Handlers reject new jobs during shutdown
- ✅ Backpressure with queue capacity limits
- ✅ `POST /jobs` returns `429 Too Many Requests` when queue is full
- ✅ Non-blocking HTTP handlers
- ✅ Workers finish current jobs before stopping
- ✅ Workers don't pick up new jobs after shutdown starts

### Technical Requirements

- ✅ Context propagation for shutdown (no ad-hoc boolean flags)
- ✅ Shutdown coordination in `main()`
- ✅ Channel closing centralized (only in `main()`)
- ✅ No sends on closed channels
- ✅ No goroutine leaks
- ✅ Workers respect context cancellation
- ✅ Handlers check shutdown state
- ✅ Proper shutdown sequence

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go              # Shutdown coordination, shutdown context
├── internal/
│   ├── http/
│   │   └── job_handler.go       # Shutdown state checking, backpressure
│   ├── worker/
│   │   └── worker.go           # Job cleanup on shutdown
│   └── store/
│       ├── job_store.go        # DeleteJob method
│       └── metric_store.go     # DecrementJobsCreated method
├── docs/
│   ├── task8/
│   │   ├── README.md           # This file
│   │   ├── summary.md           # Quick reference
│   │   ├── description.md      # Task requirements
│   │   └── concepts/           # Detailed concept explanations
│   └── learnings.md            # Overall learnings
└── go.mod                      # Go module
```

**Structure improvements:**
- Shutdown coordination in `main()`
- Shutdown context for handlers
- Job cleanup in workers

## 🔑 Key Concepts Learned

### 1. Graceful Shutdown Coordination

- **What**: Orchestrating shutdown of multiple components in correct order
- **Why**: Prevents data loss, resource leaks, and panics
- **How**: Context propagation, WaitGroups, proper sequence
- **Pattern**: Stop accepting work → Finish current work → Close channels

### 2. Backpressure

- **What**: Rejecting work when system is overloaded
- **Why**: Prevents system overload, memory exhaustion, degraded performance
- **How**: Non-blocking channel operations, HTTP 429 status code
- **Pattern**: Check queue capacity, reject immediately if full

### 3. Channel Closing Strategy

- **What**: Safely closing channels when no longer needed
- **Why**: Prevents panics from sends on closed channels
- **How**: Channel ownership, wait for all users to stop
- **Pattern**: Only owner closes, wait for senders and receivers to stop first

### 4. Worker Lifecycle Management

- **What**: Controlling worker start, run, and stop phases
- **Why**: Ensures workers exit cleanly without leaving inconsistent state
- **How**: Context cancellation, job state cleanup, WaitGroups
- **Pattern**: Detect shutdown → Clean up current work → Exit gracefully

## 📝 Implementation Details

### Shutdown Sequence

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

**Key points:**
- Proper order prevents panics
- Timeout ensures shutdown completes
- WaitGroups ensure goroutines finish
- Channel closed only after all users stop

### Shutdown State Checking in Handlers

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

**Key points:**
- Non-blocking check doesn't delay normal requests
- Returns 503 if shutdown in progress
- Prevents new work from entering system

### Backpressure Implementation

```go
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // ... create job ...
    
    // Try to enqueue (non-blocking)
    select {
    case h.jobQueue <- job.ID:
        // Successfully enqueued
        h.logger.Info("Job enqueued", "event", "job_enqueued", "job_id", job.ID)
    case <-r.Context().Done():
        // Request canceled
        ErrorResponse(w, "Request cancelled", http.StatusRequestTimeout)
        return
    default:
        // Queue is full - reject job
        h.store.DeleteJob(r.Context(), job.ID)
        h.metricStore.DecrementJobsCreated(r.Context())
        h.logger.Error("Failed to enqueue job", "event", "job_enqueue_failed", "job_id", job.ID, "error", "queue_full")
        ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)
        return
    }
    
    // ... return success response ...
}
```

**Key points:**
- Non-blocking send prevents handler blocking
- Returns 429 if queue is full
- Cleans up job and metrics on rejection

### Worker Job Cleanup

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // ... setup ...
    
    select {
    case <-timer.C:
        // Processing complete
    case <-ctx.Done():
        // Shutdown requested, abort processing - clean up job state
        w.logger.Info("Worker job processing aborted due to shutdown", "event", "job_aborted", "worker_id", w.id, "job_id", job.ID)
        
        // Mark job as failed due to shutdown
        lastError := "Job aborted due to shutdown"
        if err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError); err != nil {
            w.logger.Error("Worker error updating aborted job to failed", ...)
        } else {
            // IncrementJobsFailed also decrements JobsInProgress
            if err := w.metricStore.IncrementJobsFailed(ctx); err != nil {
                w.logger.Error("Worker error incrementing jobs failed for aborted job", ...)
            }
        }
        
        return
    }
    
    // ... continue with normal processing ...
}
```

**Key points:**
- Detects shutdown during processing
- Marks job as failed with clear error message
- Updates metrics correctly
- Never leaves job in "processing" state

## 🚀 Running the Service

### Build

```bash
go build -o bin/server ./cmd/server
```

### Run

```bash
# Default settings (port 8080, 10 workers, queue capacity 100)
go run ./cmd/server

# Custom configuration
PORT=3000 WORKER_COUNT=20 JOB_QUEUE_CAPACITY=200 go run ./cmd/server
```

### Test Graceful Shutdown

```bash
# Start server
go run ./cmd/server

# In another terminal, create some jobs
for i in {1..50}; do
  curl -X POST http://localhost:8080/jobs \
    -H "Content-Type: application/json" \
    -d '{"type": "test", "payload": {}}'
done

# Press Ctrl+C to trigger graceful shutdown
# Observe: Jobs finish processing, no panics, clean exit
```

### Test Backpressure

```bash
# Start server with small queue
JOB_QUEUE_CAPACITY=5 go run ./cmd/server

# In another terminal, create many jobs quickly
for i in {1..20}; do
  curl -X POST http://localhost:8080/jobs \
    -H "Content-Type: application/json" \
    -d '{"type": "test", "payload": {}}'
done

# Observe: Some jobs return 429 Too Many Requests
```

## 📋 Quick Reference Checklist

### Graceful Shutdown

- ✅ Shutdown context created in `main()`
- ✅ Handlers check shutdown state
- ✅ Workers clean up job state on shutdown
- ✅ Proper shutdown sequence (handlers → server → sweeper → workers → channel)
- ✅ Timeout on server shutdown
- ✅ WaitGroups ensure goroutines finish
- ✅ No jobs left in "processing" state

### Backpressure

- ✅ Queue has maximum capacity (configurable)
- ✅ Non-blocking channel operations in handlers
- ✅ HTTP 429 status code when queue is full
- ✅ Job cleanup on rejection
- ✅ Metrics updated correctly

### Channel Management

- ✅ Channel ownership in `main()`
- ✅ Channel closed only after all users stop
- ✅ No sends on closed channels
- ✅ Workers check `ok` flag when receiving

### Worker Lifecycle

- ✅ Workers detect shutdown via context cancellation
- ✅ Workers finish current jobs before stopping
- ✅ Workers don't pick up new jobs after shutdown starts
- ✅ Job state cleaned up on shutdown

## 🔄 Shutdown Flow

### Normal Shutdown Flow

```
1. User presses Ctrl+C
   └─> Signal received

2. Shutdown signal sent to handlers
   └─> Handlers reject new jobs (503 Service Unavailable)

3. HTTP server shutdown
   └─> Stops accepting new connections
   └─> Waits for in-flight requests (max 10 seconds)

4. Sweeper stopped
   └─> Context canceled
   └─> Sweeper exits loop

5. Workers stopped
   └─> Context canceled
   └─> Workers finish current jobs
   └─> Workers exit loops

6. Channel closed
   └─> Safe to close (no users left)

7. Server stopped
   └─> Clean exit
```

### Shutdown During Job Processing

```
1. Worker processing job
   └─> Job status: "processing"

2. Shutdown signal received
   └─> ctx.Done() channel closes

3. Worker detects cancellation
   └─> Select statement sees ctx.Done() ready

4. Worker cleans up job
   └─> Mark job as StatusFailed
   └─> Error: "Job aborted due to shutdown"
   └─> Update metrics

5. Worker exits
   └─> Job never left in "processing" state
```

## 🎯 Design Decisions

### Why Shutdown Context for Handlers?

- **Standard mechanism**: Context is Go's standard cancellation mechanism
- **Non-blocking**: Check doesn't delay normal requests
- **Clear intent**: Handler knows when shutdown starts
- **Separation of concerns**: Shutdown logic separate from business logic

### Why Mark Aborted Jobs as Failed?

- **Accurate state**: Job didn't complete, so it failed
- **Clear error**: Error message explains why
- **Retryable**: Can be retried later (if retry logic allows)
- **Better than pending**: Pending implies not attempted, but job was attempted

### Why Non-Blocking Channel Operations?

- **HTTP handler requirement**: Handlers must respond quickly
- **Better UX**: Client gets immediate response (even if rejection)
- **Resource management**: Doesn't tie up connections
- **Predictable behavior**: System behavior is predictable under load

### Why Wait for Workers Before Closing Channel?

- **Clean shutdown**: Ensures all work finishes before cleanup
- **Prevents edge cases**: No possibility of closing while receiving
- **Clear ownership**: Makes it clear when channel is safe to close
- **Better logging**: Can log when workers actually stop

## 🔄 Future Improvements

Potential enhancements for future tasks:

- Job draining to disk before shutdown
- Configurable shutdown timeout
- Metrics for shutdown duration
- Health check endpoint that reflects shutdown state
- Graceful shutdown for database connections
- Job cancellation API
- Advanced backpressure strategies (exponential backoff for retries)

## 📚 Additional Notes

- **Go version**: 1.21+ (for `wg.Go()` support)
- **Dependencies**: Standard library only
- **Project structure**: Follows Go best practices
- **Code style**: Idiomatic Go patterns
- **Concurrency**: Safe for concurrent access
- **Storage**: In-memory (temporary, lost on restart)

## ⚠️ Critical Bugs Avoided

### 1. Jobs Left in Processing State
- **Bug**: Worker exits without cleaning up job state
- **Fix**: Worker marks job as failed on shutdown
- **Impact**: Jobs stuck in "processing" state forever

### 2. Handlers Accepting Jobs During Shutdown
- **Bug**: No shutdown state check in handlers
- **Fix**: Handlers check shutdown context before accepting
- **Impact**: Jobs created but never processed

### 3. Closing Channel Too Early
- **Bug**: Channel closed while handlers/workers still using it
- **Fix**: Close only after all users stop
- **Impact**: Panic from send on closed channel

### 4. Blocking on Full Channel
- **Bug**: Handler blocks waiting for queue space
- **Fix**: Non-blocking send with rejection
- **Impact**: Handler ties up connection, poor UX

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

