# Understanding Worker Pools

## Table of Contents

1. [Why Worker Pools?](#why-worker-pools)
2. [What is a Worker Pool?](#what-is-a-worker-pool)
3. [Single Worker vs Worker Pool](#single-worker-vs-worker-pool)
4. [Creating a Worker Pool](#creating-a-worker-pool)
5. [The Fan-Out Pattern](#the-fan-out-pattern)
6. [Worker Pool Lifecycle](#worker-pool-lifecycle)
7. [Common Mistakes](#common-mistakes)

---

## Why Worker Pools?

### The Problem with Single Worker

In Task 4, we had **one worker** processing jobs:

```go
worker := worker.NewWorker(jobStore, jobQueue)
go worker.Start(workerCtx)
```

**Scenario:** What happens when you have 100 jobs queued and each job takes 1 second to process?

- Single worker processes 1 job per second
- 100 jobs = 100 seconds to process all
- **Problem:** Slow throughput, jobs wait in queue

### The Solution: Multiple Workers

**With 10 workers:**

- 10 workers process 10 jobs simultaneously
- 100 jobs = 10 seconds to process all
- **Result:** 10x faster throughput

### Real-World Analogy

Think of a restaurant:

- **Single worker** = One chef cooking all orders (slow!)
- **Worker pool** = Multiple chefs cooking simultaneously (fast!)

The more chefs (workers), the more orders (jobs) you can process at once.

---

## What is a Worker Pool?

A **worker pool** is a group of multiple workers (goroutines) that all listen to the same job queue (channel) and process jobs concurrently.

### Key Characteristics

1. **Multiple workers** - Not just one
2. **Shared queue** - All workers listen to the same channel
3. **Concurrent processing** - Workers process jobs simultaneously
4. **Load distribution** - Jobs are distributed among workers automatically

### Visual Representation

```
Job Queue (Channel)
    │
    ├─> Worker 1 ──> Processing Job A
    ├─> Worker 2 ──> Processing Job B
    ├─> Worker 3 ──> Processing Job C
    ├─> Worker 4 ──> (waiting for job)
    └─> Worker 5 ──> (waiting for job)
```

**Key Point:** When a job arrives, **one** worker picks it up. The channel automatically distributes jobs to available workers.

---

## Single Worker vs Worker Pool

### Single Worker (Task 4)

```go
// One worker
worker := worker.NewWorker(jobStore, jobQueue)
go worker.Start(workerCtx)
```

**Characteristics:**

- One goroutine processing jobs
- Sequential processing (one at a time)
- Simple but slow under load
- Good for low-volume scenarios

**Throughput:** 1 job per second (if each job takes 1 second)

### Worker Pool (Task 5)

```go
// Multiple workers
for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Add(1)
    go func(workerID int) {
        defer wg.Done()
        worker.Start(workerCtx)
    }(i)
}
```

**Characteristics:**

- Multiple goroutines processing jobs
- Concurrent processing (many at once)
- Scales with worker count
- Good for high-volume scenarios

**Throughput:** N jobs per second (where N = number of workers)

### Performance Comparison

**Scenario:** 100 jobs, each takes 1 second

| Workers | Time to Process All Jobs |
| ------- | ------------------------ |
| 1       | 100 seconds              |
| 5       | 20 seconds               |
| 10      | 10 seconds               |
| 20      | 5 seconds                |

**Key Insight:** More workers = faster processing, but with diminishing returns (CPU/memory limits).

---

## Creating a Worker Pool

### Our Implementation (Modern Pattern - Go 1.21+)

```go
var wg sync.WaitGroup

for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}
```

**Note:** `wg.Go()` is available in Go 1.21+. It automatically:

- Calls `wg.Add(1)` before starting the goroutine
- Calls `wg.Done()` when the goroutine finishes
- Eliminates the need for `defer wg.Done()`

### Step-by-Step Breakdown

**Step 1: Create WaitGroup**

```go
var wg sync.WaitGroup
```

- Tracks multiple goroutines
- We'll wait for all workers to finish

**Step 2: Loop to Create Workers**

```go
for i := 0; i < config.WorkerCount; i++ {
```

- Creates `config.WorkerCount` workers (e.g., 10 workers)
- `i` is the worker ID (0, 1, 2, ..., 9)

**Step 3: Create Worker Instance**

```go
worker := worker.NewWorker(i, jobStore, jobQueue)
```

- Creates a worker struct with ID `i`
- All workers share the same `jobStore` and `jobQueue`
- Each worker has a unique ID for logging

**Step 4: Start Worker with WaitGroup (Modern Pattern - Go 1.21+)**

```go
wg.Go(func() {
    worker.Start(workerCtx)
})
```

**Breaking this down:**

- `wg.Go(func() { ... })` - Modern WaitGroup method (Go 1.21+)
- Automatically calls `wg.Add(1)` before starting goroutine
- Automatically calls `wg.Done()` when goroutine finishes
- Much cleaner than the old `Add(1)` + `go func()` + `defer Done()` pattern
- `worker.Start(workerCtx)` - Runs the worker loop

**Note:** The worker is created with the correct ID `i` before the closure, so the closure safely captures the worker instance (not the loop variable).

### Why Pass `i` as Parameter?

**⚠️ Critical Bug to Avoid:**

```go
// ❌ BAD: Closure captures loop variable
for i := 0; i < 10; i++ {
    go func() {
        fmt.Println(i)  // All print 10!
    }()
}
```

**Problem:** All goroutines see the **final value** of `i` (10), not the value when the goroutine was created.

**Why?** The closure captures the **variable** `i`, not its **value**. By the time goroutines run, the loop has finished and `i = 10`.

**✅ GOOD: Pass as parameter**

```go
// ✅ GOOD: Each goroutine gets its own copy
for i := 0; i < 10; i++ {
    go func(workerID int) {
        fmt.Println(workerID)  // Prints 0, 1, 2, ..., 9
    }(i)  // Pass i as parameter
}
```

**Why this works:** Each goroutine receives a **copy** of `i` as `workerID`. The copy is made when the goroutine is created, so each worker gets the correct ID.

---

## The Fan-Out Pattern

### What is Fan-Out?

**Fan-out** means distributing work from one source (channel) to multiple consumers (workers).

```
Single Source (Channel)
    │
    ├─> Worker 1
    ├─> Worker 2
    ├─> Worker 3
    └─> Worker N
```

### How It Works

1. **One channel** holds all jobs
2. **Multiple workers** all listen to the same channel
3. **Go runtime** automatically distributes jobs to available workers
4. **First available worker** gets the job

### Our Implementation

```go
// One channel shared by all workers
jobQueue := make(chan *domain.Job, config.JobQueueCapacity)

// All workers listen to the same channel
for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    // Each worker's Start() method does:
    // case job := <-w.jobQueue:  // All workers listening here
}
```

### How Jobs Are Distributed

**Scenario:** 5 jobs arrive, 3 workers available

```
Time 0: Job 1 arrives → Worker 1 picks it up
Time 0: Job 2 arrives → Worker 2 picks it up
Time 0: Job 3 arrives → Worker 3 picks it up
Time 1: Job 4 arrives → Worker 1 (finished Job 1) picks it up
Time 1: Job 5 arrives → Worker 2 (finished Job 2) picks it up
```

**Key Point:** The channel automatically distributes jobs. You don't need to manually assign jobs to workers.

### Why Not Per-Worker Queues?

**❌ Anti-Pattern: Per-Worker Queues**

```go
// Don't do this!
worker1Queue := make(chan *domain.Job)
worker2Queue := make(chan *domain.Job)
// ... separate queues for each worker
```

**Problems:**

- Manual load balancing (complex)
- Uneven distribution (some workers idle, others overloaded)
- More code to manage
- Harder to scale

**✅ Our Pattern: Shared Queue**

```go
// One queue for all workers
jobQueue := make(chan *domain.Job, 100)
// All workers listen to the same queue
```

**Benefits:**

- Automatic load balancing
- Even distribution (first available worker gets job)
- Simple code
- Easy to scale (just add more workers)

---

## Worker Pool Lifecycle

### The Lifecycle Stages

```
1. Creation
   ↓
2. Running (all workers active)
   ↓
3. Processing Jobs (concurrent)
   ↓
4. Shutdown Signal
   ↓
5. All Workers Stop
   ↓
6. Cleanup
```

### Stage 1: Creation

```go
for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Add(1)
    go func(workerID int) {
        defer wg.Done()
        worker.Start(workerCtx)
    }(i)
}
```

- All workers created
- All goroutines started
- All workers waiting for jobs

### Stage 2: Running

```go
// All workers are now running
// Each worker is in this loop:
for {
    select {
    case job := <-w.jobQueue:
        // Process job
    }
}
```

- Workers are active
- Waiting for jobs from channel
- Ready to process

### Stage 3: Processing Jobs

```
Job arrives → Channel → Worker picks it up → Processes → Returns to waiting
```

- Jobs distributed automatically
- Multiple jobs processed concurrently
- Workers work independently

### Stage 4: Shutdown Signal

```go
workerCancel()  // Cancel context
```

- Context is canceled
- All workers receive cancellation signal
- Workers finish current job and exit

### Stage 5: All Workers Stop

```go
wg.Wait()  // Wait for all workers to finish
```

- WaitGroup waits for all `wg.Done()` calls
- All workers have exited
- Pool is stopped

### Stage 6: Cleanup

```go
close(jobQueue)  // Close channel
```

- Channel closed
- Resources cleaned up
- Pool shutdown complete

---

## Common Mistakes

### Mistake 1: Closure Variable Capture

```go
// ❌ BAD: All workers get same ID
for i := 0; i < 10; i++ {
    go func() {
        worker.Start(workerCtx, i)  // i is 10 for all!
    }()
}
```

**Fix:** Pass as parameter

```go
// ✅ GOOD: Each worker gets correct ID
for i := 0; i < 10; i++ {
    go func(workerID int) {
        worker.Start(workerCtx, workerID)
    }(i)
}
```

### Mistake 2: Forgetting WaitGroup

```go
// ❌ BAD: No way to wait for workers
for i := 0; i < 10; i++ {
    go worker.Start(workerCtx)
}
// main() might exit before workers finish!
```

**Fix:** Use WaitGroup (Modern Pattern - Go 1.21+)

```go
// ✅ GOOD: Can wait for all workers
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}
wg.Wait()  // Wait for all
```

### Mistake 3: Creating Per-Worker Queues

```go
// ❌ BAD: Separate queues (manual balancing)
workerQueues := make([]chan *domain.Job, 10)
for i := range workerQueues {
    workerQueues[i] = make(chan *domain.Job)
}
```

**Fix:** Shared queue (fan-out)

```go
// ✅ GOOD: One queue, multiple workers
jobQueue := make(chan *domain.Job, 100)
for i := 0; i < 10; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    // All workers share same queue
}
```

### Mistake 4: Not Tracking Worker IDs

```go
// ❌ BAD: Can't identify which worker did what
worker := worker.NewWorker(jobStore, jobQueue)
```

**Fix:** Add worker ID

```go
// ✅ GOOD: Each worker has unique ID
worker := worker.NewWorker(i, jobStore, jobQueue)
// Logs: "Worker 3 processing job abc123"
```

---

## Key Takeaways

1. **Worker pools** = Multiple workers processing jobs concurrently
2. **Fan-out pattern** = One channel, multiple workers
3. **Shared queue** = Automatic load balancing
4. **WaitGroup** = Track multiple goroutines
5. **Closure capture** = Always pass loop variables as parameters
6. **Worker ID** = Helps with debugging and logging
7. **More workers** = Faster processing (up to CPU limits)

---

## Real-World Analogy

Think of a call center:

- **Single worker** = One operator handling all calls (slow!)
- **Worker pool** = Multiple operators, calls distributed automatically (fast!)

When a call comes in, it goes to the first available operator. The more operators you have, the more calls you can handle simultaneously.

---

## Next Steps

- Read [Preventing Duplicate Processing](./02-preventing-duplicate-processing.md) to understand how we ensure each job is processed exactly once
- Read [Configuration Management](./03-configuration-management.md) to see how we make worker count configurable
- Read [Proper Shutdown Order](./04-proper-shutdown-order.md) to learn how to stop worker pools cleanly
