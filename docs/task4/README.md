# Task 4 — Background Worker & Job Processing Loop

## Overview

This task introduces **asynchronous job processing** using Go's concurrency primitives. The focus is on goroutines, channels, worker lifecycle management, state transitions, and preventing race conditions.

## ✅ Completed Requirements

### Functional Requirements

- ✅ Background worker implemented
- ✅ Worker processes jobs asynchronously
- ✅ Jobs transition: `pending → processing → completed`
- ✅ Only `pending` jobs can be picked up
- ✅ Each job processed exactly once (atomic claiming)
- ✅ Job status updated atomically
- ✅ Worker starts when server starts
- ✅ Worker stops when server stops
- ✅ Worker continuously receives jobs from channel
- ✅ Single worker (as required)

### Technical Requirements

- ✅ Worker package created (`internal/worker/`)
- ✅ Channel-based communication (no polling)
- ✅ Buffered channel (capacity 100)
- ✅ Context for worker cancellation
- ✅ WaitGroup for goroutine coordination
- ✅ Graceful shutdown implemented
- ✅ No goroutine leaks
- ✅ No data races
- ✅ No busy loops
- ✅ Store remains concurrency-safe
- ✅ Worker doesn't depend on HTTP packages
- ✅ No globals

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go              # Server setup with worker
├── internal/
│   ├── domain/
│   │   └── job.go               # Domain model (added status constants)
│   ├── http/
│   │   ├── handler.go           # Health check handler
│   │   ├── job_handler.go       # Job handlers (sends to queue)
│   │   └── response.go          # Error response helper
│   ├── store/
│   │   └── job_store.go         # Store (added UpdateJob, ClaimJob)
│   └── worker/                  # NEW: Worker package
│       └── worker.go           # Worker implementation
├── docs/
│   ├── task4/
│   │   ├── README.md            # This file
│   │   ├── summary.md           # Quick reference
│   │   ├── description.md       # Task requirements
│   │   └── concepts/            # Detailed concept explanations
│   └── learnings.md             # Overall learnings
└── go.mod                       # Go module
```

**Structure improvements:**
- `internal/worker/` - Worker package separated from HTTP
- Channel-based communication - No polling, efficient
- Clear separation: HTTP → Worker → Store → Domain

## 🔑 Key Concepts Learned

### 1. Goroutines for Workers

- **What**: Lightweight threads for background processing
- **Why**: Don't block HTTP handlers, process asynchronously
- **How**: `go worker.Start(ctx)` starts worker in separate goroutine
- **Ownership**: main() owns worker goroutine, controls lifecycle

### 2. Channels for Communication

- **What**: Typed conduit for sending/receiving between goroutines
- **Why**: Thread-safe communication, natural synchronization
- **Buffered vs Unbuffered**: Buffered = async, Unbuffered = sync
- **Our choice**: Buffered (capacity 100) for decoupling

### 3. Worker Pattern

- **What**: Background goroutine that processes work from channel
- **Why**: Decouples HTTP from processing, enables async operations
- **Responsibilities**: Receive jobs, claim atomically, process, update status
- **Lifecycle**: Start → Process → Shutdown (via context)

### 4. Channel Buffering

- **Buffered**: Can queue values, sender doesn't block until full
- **Unbuffered**: Sender and receiver must meet
- **Our decision**: Buffered with capacity 100
- **Reasons**: Decouple HTTP handler, handle bursts, natural backpressure

### 5. Graceful Shutdown

- **What**: Controlled shutdown that finishes current work
- **Components**: Context cancellation, WaitGroup, channel closing
- **Order**: Cancel context → Wait for goroutines → Close channels → Shutdown server
- **Why**: Prevents data loss, resource leaks, incomplete operations

### 6. Context in Workers

- **Purpose**: Enable cancellation and shutdown
- **Pattern**: First parameter is context
- **Usage**: Check `ctx.Done()` in loops and long operations
- **Propagation**: Pass context through function calls

### 7. Atomic Operations

- **Problem**: Race conditions when multiple workers access same job
- **Solution**: ClaimJob atomically checks and sets status
- **How**: Mutex protects check-and-set operation
- **Result**: Only one worker can claim a job

### 8. Select Statement

- **What**: Like switch, but for channels
- **Why**: Wait for multiple channels simultaneously
- **Pattern**: `select { case <-ctx.Done(): return; case job := <-queue: process }`
- **Behavior**: Blocks until one case ready, random if multiple ready

## 📝 Implementation Details

### Server Setup with Worker

```go
func main() {
    // 1. Create dependencies
    jobStore := store.NewInMemoryJobStore()
    
    // 2. Create job queue channel
    const jobQueueCapacity = 100
    jobQueue := make(chan *domain.Job, jobQueueCapacity)
    
    // 3. Create worker context
    workerCtx, workerCancel := context.WithCancel(context.Background())
    defer workerCancel()
    
    // 4. Create worker
    worker := worker.NewWorker(jobStore, jobQueue)
    
    // 5. Start worker in goroutine
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        worker.Start(workerCtx)
    }()
    
    // 6. Create handler with queue
    jobHandler := internalhttp.NewJobHandler(jobStore, jobQueue)
    
    // 7. Setup HTTP server
    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)
    mux.HandleFunc("GET /jobs", jobHandler.GetJobs)
    mux.HandleFunc("POST /jobs", jobHandler.CreateJob)
    
    srv := &http.Server{
        Addr:    ":" + port,
        Handler: mux,
    }
    
    // 8. Start server
    go srv.ListenAndServe()
    
    // 9. Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan
    
    // 10. Graceful shutdown
    log.Println("Shutting down...")
    workerCancel()  // Cancel worker context
    wg.Wait()       // Wait for worker to finish
    close(jobQueue) // Close channel
    
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()
    srv.Shutdown(shutdownCtx)
}
```

**Key points:**
- Channel created with capacity 100
- Worker context for cancellation
- WaitGroup tracks worker goroutine
- Handler receives channel for enqueueing
- Graceful shutdown sequence

### Worker Implementation

```go
type Worker struct {
    jobStore store.JobStore
    jobQueue chan *domain.Job
}

func NewWorker(jobStore store.JobStore, jobQueue chan *domain.Job) *Worker {
    return &Worker{
        jobStore: jobStore,
        jobQueue: jobQueue,
    }
}

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

**Key points:**
- Infinite loop with select
- Checks context for cancellation
- Checks channel closed (`ok` value)
- Claims job atomically before processing
- Processes with timeout
- Respects context cancellation during processing

### Handler Enqueueing

```go
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // ... create and store job ...
    
    timer := time.NewTimer(100 * time.Millisecond)
    defer timer.Stop()

    select {
    case h.jobQueue <- job:
        // Successfully enqueued
    case <-timer.C:
        log.Printf("Warning: Job queue full, job %s may be delayed", job.ID)
    case <-r.Context().Done():
        return
    }
    
    // ... return response ...
}
```

**Key points:**
- Sends job to channel
- Timeout if queue full (100ms)
- Respects client cancellation
- Job already stored, so it will be processed eventually

### Atomic Claiming

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

**Key points:**
- Checks context before lock
- Acquires lock (exclusive access)
- Checks job exists and is pending
- Atomically updates status
- Returns success/failure

## 🎓 Learning Resources

Detailed explanations of all concepts are available in the [`concepts/`](./concepts/) directory:

1. **[Goroutines for Workers](./concepts/01-goroutines-for-workers.md)** - Background processing with goroutines
2. **[Channels for Communication](./concepts/02-channels-for-communication.md)** - Channel-based job dispatch
3. **[Worker Pattern](./concepts/03-worker-pattern.md)** - Complete worker design
4. **[Channel Buffering Decisions](./concepts/04-channel-buffering.md)** - Why buffered channels
5. **[Graceful Shutdown](./concepts/05-graceful-shutdown.md)** - How to stop workers properly
6. **[Context in Workers](./concepts/06-context-in-workers.md)** - Context for cancellation
7. **[Atomic Operations](./concepts/07-atomic-operations.md)** - Preventing race conditions
8. **[Select Statement](./concepts/08-select-statement.md)** - Coordinating multiple channels

## 🚀 Running the Service

### Build

```bash
go build -o bin/server ./cmd/server
```

### Run

```bash
# Default port (8080)
go run ./cmd/server

# Custom port
PORT=3000 go run ./cmd/server
```

### Test Endpoints

```bash
# Health check
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Create job
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {"to": "user@example.com"}
  }'
# Expected: 201 Created with job details (status: "pending")

# List all jobs (check status transitions)
curl http://localhost:8080/jobs
# Expected: 200 OK with array of jobs
# Jobs should transition: pending → processing → completed
```

### Observing Job Processing

1. Create a job via `POST /jobs`
2. Immediately check `GET /jobs` - status should be "pending"
3. Wait 1 second (processing time)
4. Check `GET /jobs` again - status should be "completed"

## 📋 Quick Reference Checklist

### Worker Implementation

- ✅ Worker package created
- ✅ Worker struct with dependencies
- ✅ Constructor accepts dependencies
- ✅ Start method accepts context
- ✅ Select statement for cancellation
- ✅ Channel closed check (`ok` value)
- ✅ Atomic job claiming
- ✅ Job processing with timeout
- ✅ Status updates (processing → completed)
- ✅ Context cancellation handling

### Channel Setup

- ✅ Buffered channel (capacity 100)
- ✅ Channel passed to worker and handler
- ✅ Handler sends jobs to channel
- ✅ Worker receives from channel
- ✅ Channel closed on shutdown

### Graceful Shutdown

- ✅ Worker context with cancel
- ✅ WaitGroup for tracking
- ✅ Worker started with defer wg.Done()
- ✅ Context canceled on shutdown
- ✅ WaitGroup.Wait() before closing channel
- ✅ Channel closed after worker stops
- ✅ HTTP server shutdown with timeout
- ✅ Correct shutdown order

### Atomic Operations

- ✅ ClaimJob method in store interface
- ✅ ClaimJob checks context before lock
- ✅ ClaimJob acquires lock
- ✅ ClaimJob checks job exists and is pending
- ✅ ClaimJob atomically updates status
- ✅ ClaimJob releases lock (defer)
- ✅ Worker calls ClaimJob before processing

## 🔄 Job Status Lifecycle

```
pending → processing → completed
```

**Transitions:**
1. **pending**: Job created, waiting to be processed
2. **processing**: Worker claimed job, currently processing
3. **completed**: Job processing finished successfully

**Rules:**
- Only `pending` jobs can be claimed
- Claiming is atomic (prevents duplicates)
- Status updated atomically in store

## 🎯 Design Decisions

### Why Buffered Channel (Capacity 100)?

- **Decouples HTTP handler from worker**: Handler doesn't wait for worker
- **Handles traffic bursts**: Can queue multiple jobs quickly
- **Natural backpressure**: Blocks when full (prevents unbounded growth)
- **Better HTTP response times**: Handler returns immediately
- **Balance**: Throughput vs memory usage

### Why ClaimJob Instead of Check-Then-Set?

- **Prevents race conditions**: Atomic operation
- **Only one worker can claim**: Mutex protection
- **Prevents duplicate processing**: Check and set together
- **Thread-safe**: Protected by mutex

### Why Context for Shutdown?

- **Standard Go pattern**: Widely used
- **Can cancel from anywhere**: Propagates through calls
- **Works with timeouts**: Can add deadlines
- **Enables graceful shutdown**: Workers can exit cleanly

### Why WaitGroup?

- **Tracks goroutine completion**: Know when worker finished
- **Prevents goroutine leaks**: Ensures cleanup
- **Standard pattern**: Go idiom for coordination
- **Ensures proper shutdown**: Wait before closing channels

### Why Single Worker?

- **Task requirement**: Simplicity for learning
- **Easier to understand**: Less complexity
- **Foundation for scaling**: Can add more workers later
- **Clear ownership**: One goroutine to manage

## 🔄 Future Improvements

Potential enhancements for future tasks:

- Multiple workers (worker pool)
- Worker health monitoring
- Job retry logic
- Failure states and error handling
- Dead-letter queue for failed jobs
- Job priority queues
- Rate limiting
- Metrics and observability
- Database persistence
- Job cancellation API

## 📚 Additional Notes

- **Go version**: 1.25+
- **Dependencies**: Standard library only (sync, context, time)
- **Project structure**: Follows Go best practices with worker separation
- **Code style**: Idiomatic Go patterns
- **Concurrency**: Safe for concurrent access
- **Storage**: In-memory (temporary, lost on restart)

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

