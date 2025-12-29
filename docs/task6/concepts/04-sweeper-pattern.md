# Understanding the Sweeper Pattern

## Table of Contents

1. [Why a Sweeper?](#why-a-sweeper)
2. [What is a Sweeper?](#what-is-a-sweeper)
3. [Periodic Retry Mechanism](#periodic-retry-mechanism)
4. [Sweeper vs Worker Responsibilities](#sweeper-vs-worker-responsibilities)
5. [Sweeper Lifecycle](#sweeper-lifecycle)
6. [Common Mistakes](#common-mistakes)

---

## Why a Sweeper?

### The Problem Without a Sweeper

**Scenario:** A job fails and needs to be retried.

**Option 1: Worker handles retry immediately**
```go
// ❌ BAD: Worker handles retry
func (w *Worker) processJob(job *domain.Job) {
    if jobFails {
        if job.Attempts < job.MaxRetries {
            job.Status = domain.StatusPending
            w.jobQueue <- job.ID  // Worker re-enqueues
        }
    }
}
```

**Problems:**
- Worker is responsible for both processing AND retry logic
- Retry happens immediately (no delay)
- Worker logic becomes complex
- Hard to control retry timing
- Mixes concerns (processing vs retry)

**Option 2: No retry mechanism**
```go
// ❌ BAD: No retry
func (w *Worker) processJob(job *domain.Job) {
    if jobFails {
        job.Status = domain.StatusFailed
        // Job stays failed forever - no retry!
    }
}
```

**Problems:**
- Failed jobs never retry
- No way to recover from temporary failures
- System becomes unreliable

### The Solution: Sweeper Pattern

**With sweeper:**
```go
// ✅ GOOD: Separate sweeper handles retries
func (s *InMemorySweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    for {
        select {
        case <-ticker.C:
            // Periodically retry failed jobs
            s.jobStore.RetryFailedJobs(ctx)
            // Enqueue pending jobs
            jobs, _ := s.jobStore.GetPendingJobs(ctx)
            for _, job := range jobs {
                s.jobQueue <- job.ID
            }
        }
    }
}
```

**Benefits:**
- **Separation of concerns** - Worker processes, sweeper retries
- **Periodic retries** - Can control retry timing
- **Centralized retry logic** - All retries in one place
- **Simple worker logic** - Worker just processes, doesn't retry
- **Flexible** - Can adjust retry interval

### Real-World Analogy

Think of a cleaning service:

- **Worker:** Cleans the room (processes jobs)
- **Sweeper:** Periodically checks for dirty rooms and schedules cleaning (retries failed jobs)

A job queue is similar - workers process jobs, sweeper handles retries.

---

## What is a Sweeper?

### Definition

A **sweeper** is a background process that periodically:
1. Finds failed jobs that can be retried
2. Moves them back to pending state
3. Enqueues them for processing

### Our Implementation

```go
type InMemorySweeper struct {
    jobStore JobStore
    interval time.Duration
    jobQueue chan string
}

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
                s.jobQueue <- job.ID
            }
        }
    }
}
```

### Key Characteristics

1. **Periodic** - Runs on a schedule (every N seconds)
2. **Background** - Runs in separate goroutine
3. **Stateless** - Doesn't maintain state between runs
4. **Idempotent** - Can run multiple times safely

---

## Periodic Retry Mechanism

### How It Works

**Step 1: Create Ticker**
```go
ticker := time.NewTicker(s.interval)  // e.g., 10 seconds
defer ticker.Stop()
```

**What this does:**
- Creates a ticker that fires every `interval` duration
- `ticker.C` is a channel that receives a value every interval

**Step 2: Wait for Tick**
```go
select {
case <-ticker.C:
    // Time to run sweeper logic
}
```

**What this does:**
- Blocks until ticker fires
- When ticker fires, executes sweeper logic

**Step 3: Retry Failed Jobs**
```go
s.jobStore.RetryFailedJobs(ctx)
```

**What this does:**
- Finds all failed jobs
- Checks if they can be retried (attempts < maxRetries)
- Moves retryable jobs to pending state

**Step 4: Enqueue Pending Jobs**
```go
jobs, _ := s.jobStore.GetPendingJobs(ctx)
for _, job := range jobs {
    s.jobQueue <- job.ID
}
```

**What this does:**
- Gets all pending jobs (including newly retried ones)
- Enqueues them for processing

### The Timing

**Example with interval = 10 seconds:**

```
Time 0s:  Sweeper starts, waits for tick
Time 10s: Tick! Retry failed jobs, enqueue pending
Time 20s: Tick! Retry failed jobs, enqueue pending
Time 30s: Tick! Retry failed jobs, enqueue pending
...
```

**Key Point:** Sweeper runs periodically, not continuously.

### Why Periodic Instead of Continuous?

**Continuous (bad):**
```go
// ❌ BAD: Runs continuously, wastes CPU
for {
    retryFailedJobs()
    time.Sleep(1 * time.Millisecond)  // Too frequent!
}
```

**Problems:**
- Wastes CPU (runs constantly)
- No delay between retries
- Can't control retry frequency

**Periodic (good):**
```go
// ✅ GOOD: Runs periodically, efficient
ticker := time.NewTicker(10 * time.Second)
for {
    select {
    case <-ticker.C:
        retryFailedJobs()  // Runs every 10 seconds
    }
}
```

**Benefits:**
- Efficient (runs on schedule)
- Can control retry frequency
- Doesn't waste CPU

---

## Sweeper vs Worker Responsibilities

### Clear Separation

**Worker Responsibilities:**
- Claim jobs from queue
- Process jobs
- Signal success or failure
- **Does NOT handle retries**

**Sweeper Responsibilities:**
- Find failed jobs that can be retried
- Move them to pending state
- Enqueue pending jobs
- **Does NOT process jobs**

### The Flow

```
1. Worker processes job
   ↓
2. Job fails → Status = Failed
   ↓
3. Sweeper (periodically) finds failed job
   ↓
4. Sweeper checks: attempts < maxRetries? Yes
   ↓
5. Sweeper moves job: Failed → Pending
   ↓
6. Sweeper enqueues job
   ↓
7. Worker claims and processes again
```

### Why This Separation?

**Benefits:**
- **Single responsibility** - Each component has one job
- **Simpler code** - Worker doesn't need retry logic
- **Easier to test** - Can test worker and sweeper separately
- **Flexible** - Can change retry logic without touching workers

**Without separation:**
```go
// ❌ BAD: Worker handles everything
func (w *Worker) processJob(job *domain.Job) {
    // Process job
    if fails {
        // Handle retry
        // Enqueue again
        // Track attempts
        // Check limits
        // Complex!
    }
}
```

**With separation:**
```go
// ✅ GOOD: Worker just processes
func (w *Worker) processJob(job *domain.Job) {
    // Process job
    if fails {
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
        // That's it! Sweeper handles retry
    }
}

// ✅ GOOD: Sweeper handles retries
func (s *Sweeper) Run(ctx context.Context) {
    // Retry failed jobs
    // Enqueue pending jobs
    // Simple!
}
```

---

## Sweeper Lifecycle

### Creation

```go
sweeper := store.NewInMemorySweeper(jobStore, config.SweeperInterval, jobQueue)
```

**What happens:**
- Creates sweeper struct
- Sets interval (e.g., 10 seconds)
- Stores references to jobStore and jobQueue

### Starting

```go
sweeperCtx, sweeperCancel := context.WithCancel(context.Background())
defer sweeperCancel()

var sweeperWg sync.WaitGroup
sweeperWg.Go(func() {
    sweeper.Run(sweeperCtx)
})
```

**What happens:**
- Creates context for cancellation
- Starts sweeper in goroutine
- Sweeper begins periodic loop

### Running

```go
func (s *InMemorySweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return  // Shutdown
        case <-ticker.C:
            // Do work
        }
    }
}
```

**What happens:**
- Sweeper runs in loop
- Waits for ticker or context cancellation
- Executes retry logic on each tick

### Shutdown

```go
// In main.go shutdown sequence
sweeperCancel()  // Cancel context
sweeperWg.Wait()  // Wait for sweeper to finish
```

**What happens:**
- Context is canceled
- Sweeper receives cancellation signal
- Sweeper exits loop
- WaitGroup waits for completion

### The Complete Lifecycle

```
1. Creation: NewInMemorySweeper()
   ↓
2. Starting: sweeper.Run(ctx) in goroutine
   ↓
3. Running: Periodic loop (every interval)
   ↓
4. Shutdown: Context canceled
   ↓
5. Cleanup: Goroutine exits, WaitGroup done
```

---

## Common Mistakes

### Mistake 1: Worker Handles Retries

```go
// ❌ BAD: Worker mixes concerns
func (w *Worker) processJob(job *domain.Job) {
    if fails {
        if job.Attempts < job.MaxRetries {
            job.Status = domain.StatusPending
            w.jobQueue <- job.ID  // Worker retries
        }
    }
}
```

**Fix:** Separate sweeper.

```go
// ✅ GOOD: Worker just signals failure
func (w *Worker) processJob(job *domain.Job) {
    if fails {
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
        // Sweeper handles retry
    }
}
```

### Mistake 2: No Interval Control

```go
// ❌ BAD: Hardcoded interval
ticker := time.NewTicker(10 * time.Second)  // Can't change
```

**Fix:** Make it configurable.

```go
// ✅ GOOD: Configurable interval
sweeper := NewInMemorySweeper(jobStore, config.SweeperInterval, jobQueue)
// Can change via environment variable
```

### Mistake 3: Forgetting to Stop Ticker

```go
// ❌ BAD: Ticker leaks
func (s *Sweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    // Missing: defer ticker.Stop()
    // Ticker keeps running even after function exits!
}
```

**Fix:** Always defer stop.

```go
// ✅ GOOD: Always stop ticker
func (s *Sweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()  // Always stops ticker
    // ...
}
```

### Mistake 4: Not Checking Context in Loop

```go
// ❌ BAD: Can't stop sweeper
func (s *Sweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    for {
        <-ticker.C  // No way to exit!
        // Do work
    }
}
```

**Fix:** Check context.

```go
// ✅ GOOD: Can be stopped
func (s *Sweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    for {
        select {
        case <-ctx.Done():
            return  // Can exit
        case <-ticker.C:
            // Do work
        }
    }
}
```

### Mistake 5: Enqueueing All Pending Jobs (Including New Ones)

```go
// ⚠️ POTENTIAL ISSUE: Enqueues all pending jobs
jobs, _ := s.jobStore.GetPendingJobs(ctx)
for _, job := range jobs {
    s.jobQueue <- job.ID  // Might enqueue jobs already in queue
}
```

**Note:** This is acceptable in our implementation because:
- `ClaimJob` prevents duplicate processing
- Jobs already in queue will be claimed by workers
- Jobs already being processed won't be claimed again

**But:** Could be optimized to only enqueue newly retried jobs.

---

## Key Takeaways

1. **Sweeper** handles retries, **worker** processes jobs
2. **Periodic** execution (not continuous)
3. **Separation of concerns** - Each component has one job
4. **Always stop ticker** with defer
5. **Check context** for graceful shutdown
6. **Configurable interval** for flexibility

---

## Real-World Analogy

Think of a janitorial service:

- **Worker:** Cleans the room (processes jobs)
- **Sweeper:** Periodically checks for dirty rooms and schedules cleaning (retries failed jobs)

The sweeper doesn't clean - it just schedules cleaning. The worker does the actual cleaning.

---

## Next Steps

- Read [Retry Logic](./03-retry-logic-attempts.md) to understand how retries work
- Read [Atomic State Updates](./05-atomic-state-updates.md) to see how sweeper updates state safely
- Read [State Machine](./01-state-machine-transitions.md) to understand how retries fit into state transitions

