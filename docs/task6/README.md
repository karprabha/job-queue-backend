# Task 6 — Failure Handling, Retries & Job States

## Overview

This task introduces **failure handling and retry logic** to the job queue system. The focus is on explicit state machines, retry limits, atomic state updates, and the sweeper pattern for periodic retries.

## ✅ Completed Requirements

### Functional Requirements

- ✅ Failure state (`StatusFailed`) implemented
- ✅ Retry logic with attempt tracking
- ✅ Retry limits enforced (MaxRetries)
- ✅ LastError tracking for failed jobs
- ✅ Deterministic failure simulation (email jobs fail)
- ✅ Failed jobs retry correctly (if attempts < maxRetries)
- ✅ Permanent failures stay in Failed state (if attempts >= maxRetries)
- ✅ Completed jobs never retry

### Technical Requirements

- ✅ Explicit state machine with `canTransition()` function
- ✅ Store enforces state rules (not workers)
- ✅ Invalid transitions rejected with errors
- ✅ Atomic state updates with `UpdateStatus()` method
- ✅ Mutex protection for all state changes
- ✅ Sweeper pattern for periodic retries
- ✅ Channel type changed from `*domain.Job` to `string` (job IDs)
- ✅ Store is single source of truth
- ✅ Workers signal failure, store updates state
- ✅ No infinite retry loops
- ✅ No race conditions
- ✅ No lost jobs
- ✅ No duplicate retries

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go              # Sweeper setup, updated shutdown
├── internal/
│   ├── config/
│   │   └── config.go           # Added SweeperInterval
│   ├── domain/
│   │   └── job.go              # Added Attempts, MaxRetries, LastError, StatusFailed
│   ├── http/
│   │   ├── handler.go           # Health check handler
│   │   ├── job_handler.go       # Channel type changed to string
│   │   └── response.go         # Error response helper
│   ├── store/
│   │   ├── job_store.go        # State machine, UpdateStatus, retry methods
│   │   └── sweeper.go          # NEW: Sweeper implementation
│   └── worker/
│       └── worker.go           # Simplified, signals failures
├── docs/
│   ├── task6/
│   │   ├── README.md           # This file
│   │   ├── summary.md           # Quick reference
│   │   ├── description.md      # Task requirements
│   │   └── concepts/           # Detailed concept explanations
│   └── learnings.md            # Overall learnings
└── go.mod                      # Go module
```

**Structure improvements:**
- `internal/store/sweeper.go` - Sweeper pattern separated
- State machine in `job_store.go` - Centralized transition validation
- Channel type simplified - Job IDs instead of full objects

## 🔑 Key Concepts Learned

### 1. State Machines

- **What**: Explicit rules for state transitions
- **Why**: Prevents invalid state changes, ensures consistency
- **How**: `canTransition()` function validates all transitions
- **Pattern**: Store enforces rules, workers just signal events

### 2. Failure Handling

- **What**: Failure as a first-class state
- **Why**: Makes failures observable and recoverable
- **How**: `StatusFailed` state with `LastError` tracking
- **Pattern**: Workers signal failure, store updates state

### 3. Retry Logic

- **What**: Automatic retry of failed jobs (up to limit)
- **Why**: Handles temporary failures gracefully
- **How**: `Attempts` and `MaxRetries` fields, sweeper pattern
- **Pattern**: Check `attempts < maxRetries` before retrying

### 4. Sweeper Pattern

- **What**: Periodic background process for retries
- **Why**: Separates retry logic from worker processing
- **How**: Ticker-based periodic execution
- **Pattern**: Sweeper finds failed jobs, moves to pending, enqueues

### 5. Atomic State Updates

- **What**: All-or-nothing state changes
- **Why**: Prevents race conditions and inconsistent state
- **How**: Mutex protection, validate then update
- **Pattern**: `UpdateStatus()` validates and updates atomically

### 6. Channel Simplification

- **What**: Send job IDs instead of full objects
- **Why**: Less memory, always fresh data, store as source of truth
- **How**: Change channel type from `*domain.Job` to `string`
- **Pattern**: Channel is notification, store is data source

## 📝 Implementation Details

### State Machine

```go
func canTransition(from, to domain.JobStatus) bool {
    switch {
    case from == domain.StatusPending && to == domain.StatusProcessing:
        return true  // Claiming a job
    case from == domain.StatusProcessing && to == domain.StatusCompleted:
        return true  // Job succeeded
    case from == domain.StatusProcessing && to == domain.StatusFailed:
        return true  // Job failed
    case from == domain.StatusFailed && to == domain.StatusPending:
        return true  // Retrying a failed job
    default:
        return false  // All other transitions are invalid
    }
}
```

**Key points:**
- Explicit validation of all transitions
- Invalid transitions return false
- Used by `UpdateStatus()` before updating

### UpdateStatus Method

```go
func (s *InMemoryJobStore) UpdateStatus(
    ctx context.Context,
    jobID string,
    status domain.JobStatus,
    lastError *string,
) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    job, ok := s.jobs[jobID]
    if !ok {
        return errors.New("job not found")
    }
    
    // Validate transition
    if !canTransition(job.Status, status) {
        return errors.New("invalid state transition")
    }
    
    // Update all fields atomically
    job.Status = status
    if lastError != nil {
        job.LastError = lastError
    }
    s.jobs[jobID] = job  // Save after all updates
    
    return nil
}
```

**Key points:**
- Validates transition before updating
- Updates status and error atomically
- Mutex-protected for concurrency safety
- Single method for all state changes

### Sweeper Implementation

```go
func (s *InMemorySweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Retry failed jobs
            s.jobStore.RetryFailedJobs(ctx)
            
            // Get pending jobs and enqueue them
            jobs, _ := s.jobStore.GetPendingJobs(ctx)
            for _, job := range jobs {
                select {
                case <-ctx.Done():
                    return
                case s.jobQueue <- job.ID:
                    log.Printf("Sweeper: job %s added to queue", job.ID)
                default:
                    log.Printf("Sweeper: job queue is full, job %s not added", job.ID)
                }
            }
        }
    }
}
```

**Key points:**
- Periodic execution (configurable interval)
- Retries failed jobs (if attempts < maxRetries)
- Enqueues pending jobs
- Respects context cancellation

### Worker Failure Handling

```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    // Simulate processing
    timer := time.NewTimer(1 * time.Second)
    defer timer.Stop()
    
    select {
    case <-timer.C:
        // Processing complete
    case <-ctx.Done():
        return
    }
    
    // Deterministic failure: email jobs fail
    if job.Type == "email" {
        lastError := "Email sending failed"
        err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
        if err != nil {
            log.Printf("Error marking job as failed: %v", err)
            return
        }
        return
    }
    
    // Success - mark as completed
    err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
    // ...
}
```

**Key points:**
- Worker signals failure (doesn't mutate state directly)
- Store updates state atomically
- Deterministic failure simulation
- Error messages tracked

### Retry Logic

```go
func (s *InMemoryJobStore) RetryFailedJobs(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    for jobID, job := range s.jobs {
        if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
            // Can retry - move back to pending
            job.Status = domain.StatusPending
            s.jobs[jobID] = job
        }
        // If attempts >= MaxRetries, job stays in Failed (permanent failure)
    }
    
    return nil
}
```

**Key points:**
- Checks attempts < maxRetries before retrying
- Prevents infinite retry loops
- Permanent failures stay in Failed state
- Mutex-protected for concurrency safety

## 🎓 Learning Resources

Detailed explanations of all concepts are available in the [`concepts/`](./concepts/) directory:

1. **[State Machine and Transitions](./concepts/01-state-machine-transitions.md)** - Explicit state transitions
2. **[Failure Handling](./concepts/02-failure-handling.md)** - Failure as first-class state
3. **[Retry Logic and Attempt Tracking](./concepts/03-retry-logic-attempts.md)** - Retry limits and attempts
4. **[The Sweeper Pattern](./concepts/04-sweeper-pattern.md)** - Periodic retry mechanism
5. **[Atomic State Updates](./concepts/05-atomic-state-updates.md)** - Thread-safe state changes
6. **[Channel Type Simplification](./concepts/06-channel-type-simplification.md)** - Job IDs vs full objects

## 🚀 Running the Service

### Build

```bash
go build -o bin/server ./cmd/server
```

### Run

```bash
# Default settings (port 8080, 10 workers, queue capacity 100, sweeper interval 10s)
go run ./cmd/server

# Custom configuration
PORT=3000 WORKER_COUNT=20 JOB_QUEUE_CAPACITY=200 SWEEPER_INTERVAL=5s go run ./cmd/server
```

### Test Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Create email job (will fail and retry)
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "email", "payload": {"to": "user@example.com"}}'

# Create other job (will succeed)
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "notification", "payload": {}}'

# List all jobs (observe states: pending, processing, completed, failed)
curl http://localhost:8080/jobs
```

### Observing Failure Handling

1. Create an email job (will fail)
2. Check logs - job fails, error message logged
3. Wait for sweeper interval (default 10s)
4. Check logs - sweeper retries failed job
5. Job will retry up to MaxRetries (default 3)
6. After 3 attempts, job stays in Failed state (permanent failure)

## 📋 Quick Reference Checklist

### State Machine

- ✅ `canTransition()` function validates all transitions
- ✅ Store enforces state rules (not workers)
- ✅ Invalid transitions rejected with errors
- ✅ Explicit state machine rules

### Failure Handling

- ✅ `StatusFailed` state implemented
- ✅ `LastError` tracks error messages
- ✅ Workers signal failure, store updates state
- ✅ Failed jobs are observable

### Retry Logic

- ✅ `Attempts` and `MaxRetries` fields added
- ✅ Retry only if `attempts < maxRetries`
- ✅ Permanent failures stay in Failed state
- ✅ No infinite retry loops

### Sweeper Pattern

- ✅ Sweeper runs periodically (configurable interval)
- ✅ Retries failed jobs (if retryable)
- ✅ Enqueues pending jobs
- ✅ Separate from worker processing

### Atomic Updates

- ✅ `UpdateStatus()` validates and updates atomically
- ✅ Mutex protection for all state changes
- ✅ Error messages stored atomically
- ✅ All fields updated before saving

### Channel Simplification

- ✅ Channel type changed to `string` (job IDs)
- ✅ Store is source of truth
- ✅ Less memory usage
- ✅ Always fresh data

## 🔄 State Transitions

### Valid Transitions

- `Pending → Processing` ✅ (ClaimJob)
- `Processing → Completed` ✅ (Success)
- `Processing → Failed` ✅ (Failure)
- `Failed → Pending` ✅ (Retry, if attempts < maxRetries)

### Invalid Transitions (Rejected)

- `Pending → Completed` ❌ (Can't skip processing)
- `Completed → Pending` ❌ (Can't undo completion)
- `Completed → Failed` ❌ (Can't fail after completion)
- `Failed → Completed` ❌ (Must retry first)
- `Processing → Pending` ❌ (Can't go backwards)

## 🎯 Design Decisions

### Why State Machine?

- **Prevents bugs**: Invalid transitions caught early
- **Explicit rules**: Clear what transitions are allowed
- **Easier to reason**: State flow is predictable
- **Maintainable**: Rules in one place

### Why Sweeper Pattern?

- **Separation of concerns**: Worker processes, sweeper retries
- **Periodic retries**: Can control retry frequency
- **Centralized logic**: All retries in one place
- **Flexible**: Easy to adjust retry timing

### Why Store Enforces Rules?

- **Single source of truth**: Store is authoritative
- **Consistent**: All updates go through same validation
- **Secure**: Can't bypass validation
- **Centralized**: Easy to maintain

### Why Channel Simplification?

- **Less memory**: IDs are much smaller than objects
- **Fresh data**: Always read latest from store
- **Single source**: Store has the data
- **Simpler**: No need to sync objects

### Why Atomic Updates?

- **No race conditions**: Updates are serialized
- **Consistent state**: All fields updated together
- **Thread-safe**: Mutex protection
- **Predictable**: No lost updates

## 🔄 Future Improvements

Potential enhancements for future tasks:

- Exponential backoff strategies
- Dead-letter queues for permanent failures
- Metrics and observability (failure rates, retry counts)
- Database persistence
- Advanced error handling (error types, categorization)
- Job cancellation API
- Retry strategies (immediate, exponential, fixed delay)
- Failure notifications
- Job prioritization
- Rate limiting per job type

## 📚 Additional Notes

- **Go version**: 1.21+ (for `wg.Go()` support)
- **Dependencies**: Standard library only
- **Project structure**: Follows Go best practices
- **Code style**: Idiomatic Go patterns
- **Concurrency**: Safe for concurrent access
- **Storage**: In-memory (temporary, lost on restart)

## ⚠️ Critical Bugs Avoided

### 1. Workers Directly Mutating State
- **Bug**: Worker mutates job state directly
- **Fix**: Worker signals, store updates
- **Impact**: No validation, not atomic, can bypass rules

### 2. Infinite Retries
- **Bug**: No retry limit check
- **Fix**: Check `attempts < maxRetries` before retrying
- **Impact**: Infinite loops, wasted resources

### 3. Missing Transition Validation
- **Bug**: No validation of state transitions
- **Fix**: `canTransition()` validates all transitions
- **Impact**: Invalid state changes, inconsistent state

### 4. Not Saving After Updating Fields
- **Bug**: Update LastError but don't save
- **Fix**: Update all fields, then save once
- **Impact**: LastError not persisted

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

