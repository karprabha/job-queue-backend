# Task 5 — Multiple Workers & Controlled Concurrency

## Overview

This task introduces **worker pools** to scale job processing by running multiple workers concurrently. The focus is on fan-out concurrency patterns, preventing duplicate processing, proper shutdown order, and configuration management.

## ✅ Completed Requirements

### Functional Requirements

- ✅ Multiple workers implemented (configurable, default 10)
- ✅ All workers listen to the same job channel (fan-out pattern)
- ✅ Each job processed exactly once (ClaimJob prevents duplicates)
- ✅ Jobs transition correctly: `pending → processing → completed`
- ✅ Worker count configurable via environment variable
- ✅ Workers terminate cleanly on shutdown
- ✅ No duplicate processing possible
- ✅ No jobs lost

### Technical Requirements

- ✅ Worker pool created in `main`
- ✅ Fan-out via channels (one channel, multiple workers)
- ✅ No per-worker queues
- ✅ No locks in worker layer (store handles concurrency)
- ✅ Buffered channel (configurable capacity, default 100)
- ✅ Configuration package (`internal/config`)
- ✅ Environment variable support (PORT, WORKER_COUNT, JOB_QUEUE_CAPACITY)
- ✅ Proper shutdown order (server → channel → workers)
- ✅ Modern WaitGroup pattern (`wg.Go()` - Go 1.21+)
- ✅ Worker IDs for logging and debugging
- ✅ No goroutine leaks
- ✅ No data races
- ✅ No globals

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go              # Worker pool setup, proper shutdown
├── internal/
│   ├── config/                  # NEW: Configuration package
│   │   └── config.go           # Environment variable parsing
│   ├── domain/
│   │   └── job.go              # Domain model
│   ├── http/
│   │   ├── handler.go           # Health check handler
│   │   ├── job_handler.go       # Job handlers
│   │   └── response.go         # Error response helper
│   ├── store/
│   │   └── job_store.go        # Store with ClaimJob
│   └── worker/
│       └── worker.go           # Worker with ID tracking
├── docs/
│   ├── task5/
│   │   ├── README.md           # This file
│   │   ├── summary.md           # Quick reference
│   │   ├── description.md      # Task requirements
│   │   └── concepts/           # Detailed concept explanations
│   └── learnings.md            # Overall learnings
└── go.mod                      # Go module
```

**Structure improvements:**
- `internal/config/` - Configuration management separated
- Worker pool in `main.go` - Multiple workers created in loop
- Worker IDs - Each worker has unique identifier

## 🔑 Key Concepts Learned

### 1. Worker Pools

- **What**: Multiple workers processing jobs concurrently
- **Why**: Scale throughput (N workers = N jobs/second)
- **How**: Multiple goroutines listening to same channel (fan-out)
- **Pattern**: One channel, multiple workers, automatic load balancing

### 2. Fan-Out Pattern

- **What**: Distributing work from one source to multiple consumers
- **Why**: Automatic load balancing, simple code
- **How**: One channel, multiple workers all listening
- **Benefit**: First available worker gets the job

### 3. Preventing Duplicate Processing

- **Problem**: Multiple workers could process same job
- **Solution**: ClaimJob = atomic check-and-set operation
- **How**: Mutex-protected check and status update
- **Result**: Only one worker can claim a job

### 4. Configuration Management

- **What**: External values controlling program behavior
- **Why**: Flexibility without recompilation, environment-specific values
- **How**: Environment variables with sensible defaults
- **Package**: Separate `config` package for organization

### 5. Proper Shutdown Order

- **Problem**: Wrong order causes "send on closed channel" panics
- **Solution**: Shutdown in reverse dependency order
- **Order**: HTTP server → Channel → Workers
- **Why**: Prevents handlers from sending to closed channel

### 6. WaitGroup with Multiple Goroutines (Modern Pattern)

- **Modern (Go 1.21+)**: `wg.Go()` automatically handles Add/Done
- **Traditional**: Manual `Add(1)` and `defer Done()`
- **Pattern**: `wg.Go()` inside loop for multiple workers
- **Benefit**: Cleaner, less error-prone code

### 7. Closure Variable Capture

- **Problem**: Loop variables captured by closure see final value
- **Solution**: Create worker before closure, or pass as parameter
- **Pattern**: `worker := worker.NewWorker(i, ...)` before `wg.Go()`
- **Critical**: Always be aware of what closure captures

## 📝 Implementation Details

### Worker Pool Creation

```go
func main() {
    config := config.NewConfig()
    
    jobStore := store.NewInMemoryJobStore()
    jobQueue := make(chan *domain.Job, config.JobQueueCapacity)
    
    workerCtx, workerCancel := context.WithCancel(context.Background())
    defer workerCancel()
    
    var wg sync.WaitGroup
    
    // Create multiple workers (fan-out pattern)
    for i := 0; i < config.WorkerCount; i++ {
        worker := worker.NewWorker(i, jobStore, jobQueue)
        wg.Go(func() {
            worker.Start(workerCtx)
        })
    }
    
    // ... HTTP server setup ...
    
    // Proper shutdown order
    srv.Shutdown(shutdownCtx)  // 1. Stop accepting requests
    close(jobQueue)            // 2. Close channel
    workerCancel()             // 3. Cancel workers
    wg.Wait()                  // 4. Wait for all workers
}
```

**Key points:**
- Modern `wg.Go()` pattern (Go 1.21+)
- Worker created before closure (captures correct instance)
- All workers share same channel (fan-out)
- Proper shutdown order prevents panics

### Configuration Package

```go
package config

type Config struct {
    Port             string
    JobQueueCapacity int
    WorkerCount      int
}

func NewConfig() *Config {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    workerCount := os.Getenv("WORKER_COUNT")
    if workerCount == "" {
        workerCount = "10"
    }
    
    workerCountInt, err := strconv.Atoi(workerCount)
    if err != nil {
        workerCountInt = 10  // Default on error
    }
    
    // ... similar for JobQueueCapacity ...
    
    return &Config{
        Port:             port,
        JobQueueCapacity: jobQueueCapacityInt,
        WorkerCount:      workerCountInt,
    }
}
```

**Key points:**
- Environment variable support
- Sensible defaults
- Error handling with fallbacks
- Type conversion (string → int)

### Worker with ID

```go
type Worker struct {
    id       int
    jobStore store.JobStore
    jobQueue chan *domain.Job
}

func NewWorker(id int, jobStore store.JobStore, jobQueue chan *domain.Job) *Worker {
    return &Worker{
        id:       id,
        jobStore: jobStore,
        jobQueue: jobQueue,
    }
}

func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            log.Printf("Worker %d shutting down", w.id)
            return
        case job, ok := <-w.jobQueue:
            if !ok {
                log.Printf("Worker %d shutting down because job queue is closed", w.id)
                return
            }
            claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
            if err != nil {
                log.Printf("Worker %d error claiming job: %s: %v", w.id, job.ID, err)
                continue
            }
            if !claimed {
                log.Printf("Worker %d job %s not claimed", w.id, job.ID)
                continue
            }
            log.Printf("Worker %d processing job %s", w.id, job.ID)
            w.processJob(ctx, job)
        }
    }
}
```

**Key points:**
- Worker ID for logging
- ClaimJob before processing
- Handles claim failures gracefully
- Respects context cancellation

### Proper Shutdown Order

```go
// 1. Shutdown HTTP server first (stops accepting new requests, waits for in-flight)
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()

if err := srv.Shutdown(shutdownCtx); err != nil {
    log.Printf("Server shutdown error: %v", err)
}

// 2. NOW close the job queue (no more requests can enqueue)
close(jobQueue)

// 3. Cancel workers and wait
workerCancel()
wg.Wait()
```

**Key points:**
- Server shutdown first (no new requests)
- Channel closed after server (no handlers running)
- Workers canceled and waited for
- Prevents "send on closed channel" panics

## 🎓 Learning Resources

Detailed explanations of all concepts are available in the [`concepts/`](./concepts/) directory:

1. **[Worker Pools](./concepts/01-worker-pools.md)** - Multiple workers, fan-out pattern
2. **[Preventing Duplicate Processing](./concepts/02-preventing-duplicate-processing.md)** - ClaimJob pattern
3. **[Configuration Management](./concepts/03-configuration-management.md)** - Environment variables, defaults
4. **[Proper Shutdown Order](./concepts/04-proper-shutdown-order.md)** - Correct shutdown sequence
5. **[WaitGroup with Multiple Goroutines](./concepts/05-waitgroup-multiple-goroutines.md)** - Modern `wg.Go()` pattern

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

### Test Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Create multiple jobs (observe concurrent processing)
for i in {1..10}; do
  curl -X POST http://localhost:8080/jobs \
    -H "Content-Type: application/json" \
    -d "{\"type\": \"job-$i\", \"payload\": {}}"
done

# List all jobs (check they're being processed)
curl http://localhost:8080/jobs
```

### Observing Worker Pool

1. Create 10 jobs quickly
2. Check logs - you'll see multiple workers processing concurrently
3. Each job should be processed by a different worker (or same worker if fast)
4. All jobs should complete (10x faster than single worker)

## 📋 Quick Reference Checklist

### Worker Pool Implementation

- ✅ Multiple workers created in loop
- ✅ All workers share same channel (fan-out)
- ✅ Worker IDs for logging
- ✅ Modern `wg.Go()` pattern (Go 1.21+)
- ✅ Workers claim jobs before processing
- ✅ Proper error handling

### Configuration

- ✅ Configuration package created
- ✅ Environment variable support
- ✅ Sensible defaults
- ✅ Error handling for invalid values
- ✅ Type conversion (string → int)

### Shutdown

- ✅ HTTP server shutdown first
- ✅ Channel closed after server
- ✅ Workers canceled and waited for
- ✅ No "send on closed channel" panics
- ✅ Clean shutdown order

### Duplicate Prevention

- ✅ ClaimJob called before processing
- ✅ Claim result checked
- ✅ Store is source of truth
- ✅ Atomic check-and-set operation

## 🔄 Performance Comparison

| Scenario | Single Worker | 10 Workers |
|----------|--------------|------------|
| 100 jobs, 1s each | 100 seconds | 10 seconds |
| Throughput | 1 job/sec | 10 jobs/sec |
| Scalability | Limited | Scales with worker count |

**10x improvement** with 10 workers!

## 🎯 Design Decisions

### Why Fan-Out (Shared Queue)?

- **Automatic load balancing**: First available worker gets job
- **Simple code**: No manual job distribution
- **Easy to scale**: Just increase worker count
- **No uneven distribution**: Channel handles it automatically

### Why ClaimJob?

- **Prevents duplicates**: Atomic operation ensures only one worker claims
- **Store as source of truth**: Channel is just notification
- **Handles race conditions**: Mutex protection
- **Idempotent**: Safe to call multiple times

### Why Configuration Package?

- **Separation of concerns**: Config logic isolated
- **Reusable**: Can be used by multiple packages
- **Testable**: Easy to test config parsing
- **Maintainable**: Changes don't affect other code

### Why This Shutdown Order?

- **Prevents panics**: No handlers can send to closed channel
- **Ensures clean shutdown**: All components stop in order
- **No lost work**: Workers finish current jobs
- **Predictable behavior**: Always same shutdown sequence

### Why Modern `wg.Go()` Pattern?

- **Cleaner code**: No manual Add/Done
- **Less error-prone**: Can't forget Done()
- **More readable**: Intent is clear
- **Go 1.21+ feature**: Modern best practice

## 🔄 Future Improvements

Potential enhancements for future tasks:

- Worker health monitoring
- Dynamic worker scaling
- Job prioritization
- Retry logic with exponential backoff
- Failure states and error handling
- Dead-letter queue for failed jobs
- Rate limiting per worker
- Metrics and observability (worker utilization)
- Database persistence
- Job cancellation API
- Worker pool metrics (jobs processed, errors, etc.)

## 📚 Additional Notes

- **Go version**: 1.21+ (for `wg.Go()` support)
- **Dependencies**: Standard library only
- **Project structure**: Follows Go best practices
- **Code style**: Idiomatic Go patterns
- **Concurrency**: Safe for concurrent access
- **Storage**: In-memory (temporary, lost on restart)

## ⚠️ Critical Bugs Avoided

### 1. Closure Variable Capture
- **Bug**: Loop variable `i` captured by closure
- **Fix**: Create worker before closure, or pass as parameter
- **Impact**: All workers would get same ID

### 2. Send on Closed Channel
- **Bug**: Channel closed before server shutdown
- **Fix**: Shutdown server first, then close channel
- **Impact**: Panic when handler tries to send

### 3. WaitGroup Add Outside Loop
- **Bug**: Only one Add() for multiple workers
- **Fix**: Add() inside loop, or use `wg.Go()`
- **Impact**: Wait() returns before all workers finish

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

