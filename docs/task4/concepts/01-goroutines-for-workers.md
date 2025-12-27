# Understanding Goroutines for Background Workers

## Table of Contents

1. [Why Background Workers?](#why-background-workers)
2. [What is a Goroutine?](#what-is-a-goroutine)
3. [Goroutines vs Threads](#goroutines-vs-threads)
4. [Creating Goroutines for Workers](#creating-goroutines-for-workers)
5. [Our Worker Goroutine](#our-worker-goroutine)
6. [Goroutine Lifecycle](#goroutine-lifecycle)
7. [Goroutine Ownership](#goroutine-ownership)
8. [Common Mistakes](#common-mistakes)

---

## Why Background Workers?

### The Problem

In Task 2 and 3, when a client created a job via `POST /jobs`, the HTTP handler would:
1. Create the job
2. Store it
3. Return immediately

**Question:** What if processing a job takes time? Should the HTTP request wait?

**Answer:** No! That would block the HTTP handler, making it slow and unresponsive.

### The Solution: Background Workers

Instead of processing jobs in the HTTP handler, we:
1. HTTP handler creates the job and stores it
2. HTTP handler sends the job to a **queue** (channel)
3. HTTP handler returns immediately (fast response)
4. A **background worker** (separate goroutine) picks up jobs from the queue
5. Worker processes jobs asynchronously

### The Flow

```
Client Request
    ↓
HTTP Handler (goroutine 1)
    ├─> Create job
    ├─> Store job
    ├─> Send to queue (channel)
    └─> Return response ✅ (fast!)
    
Background Worker (goroutine 2)
    ├─> Receive from queue
    ├─> Process job (takes time)
    └─> Update status
```

**Key Point:** HTTP handler and worker run **concurrently** (at the same time), not sequentially.

---

## What is a Goroutine?

### The Simple Answer

A **goroutine** is a lightweight thread managed by Go's runtime.

### The Detailed Answer

A goroutine is:
- A function that runs **concurrently** with other goroutines
- **Lightweight** - thousands can run simultaneously
- **Managed by Go runtime** - not OS threads
- **Cooperative** - goroutines yield control voluntarily

### Creating a Goroutine

```go
// Regular function call (blocking)
doWork()

// Goroutine (non-blocking)
go doWork()
```

**The `go` keyword:**
- Starts a new goroutine
- Function runs concurrently
- Caller continues immediately (doesn't wait)

### Example: Simple Goroutine

```go
func main() {
    fmt.Println("Start")
    
    go func() {
        time.Sleep(1 * time.Second)
        fmt.Println("Goroutine finished")
    }()
    
    fmt.Println("End")
    time.Sleep(2 * time.Second) // Wait for goroutine
}
```

**Output:**
```
Start
End
Goroutine finished
```

**What happened:**
1. "Start" prints immediately
2. Goroutine starts (but doesn't block)
3. "End" prints immediately (before goroutine finishes)
4. After 1 second, "Goroutine finished" prints

---

## Goroutines vs Threads

### Traditional Threads (Other Languages)

**Characteristics:**
- Heavy (1-2 MB stack per thread)
- Managed by OS
- Limited number (hundreds to thousands)
- Context switching is expensive

**Example (Java):**
```java
Thread thread = new Thread(() -> {
    // Do work
});
thread.start(); // OS thread created
```

### Goroutines (Go)

**Characteristics:**
- Lightweight (2 KB initial stack, grows as needed)
- Managed by Go runtime
- Millions can run simultaneously
- Context switching is cheap

**Example (Go):**
```go
go func() {
    // Do work
}() // Goroutine created (very cheap!)
```

### Why This Matters

**With threads:**
- Can't create thousands of threads
- Each thread uses significant memory
- Context switching is slow

**With goroutines:**
- Can create millions of goroutines
- Each goroutine uses minimal memory
- Context switching is fast

**Real-world example:**
- A web server might handle 10,000 concurrent requests
- With threads: Need 10,000 OS threads (impossible or very expensive)
- With goroutines: 10,000 goroutines (easy and cheap)

---

## Creating Goroutines for Workers

### Our Worker Pattern

In `main.go`, we create a worker goroutine:

```go
worker := worker.NewWorker(jobStore, jobQueue)

var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(workerCtx)
}()
```

### Breaking It Down

**Step 1: Create Worker Instance**
```go
worker := worker.NewWorker(jobStore, jobQueue)
```
- Creates a worker struct (not running yet)
- Just a struct with fields, no goroutine yet

**Step 2: Create WaitGroup**
```go
var wg sync.WaitGroup
wg.Add(1)
```
- `WaitGroup` tracks goroutines (we'll explain this later)
- `Add(1)` means we're about to start 1 goroutine

**Step 3: Start Goroutine**
```go
go func() {
    defer wg.Done()
    worker.Start(workerCtx)
}()
```

**What's happening:**
- `go func() { ... }()` - Starts a new goroutine
- `defer wg.Done()` - Signals goroutine finished (when function exits)
- `worker.Start(workerCtx)` - Runs the worker loop

### Why Anonymous Function?

We could write:
```go
go worker.Start(workerCtx)  // Simpler!
```

But we use an anonymous function to:
1. Add `defer wg.Done()` for tracking
2. Make goroutine lifecycle explicit
3. Ensure cleanup happens

**Both approaches work, but the anonymous function is more explicit.**

---

## Our Worker Goroutine

### The Worker Start Method

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

### What This Does

**The Infinite Loop:**
```go
for {
    // Keep running until context is canceled
}
```

**The Select Statement:**
```go
select {
case <-ctx.Done():
    // Context canceled, stop worker
case job, ok := <-w.jobQueue:
    // Job received, process it
}
```

**Key Point:** The worker runs in a loop, continuously waiting for:
1. Context cancellation (shutdown signal)
2. Jobs from the channel

### How It Works

**Step 1: Worker Starts**
- Goroutine begins running
- Enters the `for` loop
- Blocks on `select` statement (waiting)

**Step 2: Job Arrives**
- HTTP handler sends job to channel
- `select` receives the job
- Worker processes it
- Returns to `select` (waits for next job)

**Step 3: Shutdown**
- Context is canceled
- `select` receives from `ctx.Done()`
- Worker returns (exits loop)
- Goroutine ends

---

## Goroutine Lifecycle

### The Lifecycle Stages

```
1. Creation
   ↓
2. Running
   ↓
3. Waiting (blocked on channel/select)
   ↓
4. Running (unblocked)
   ↓
5. Termination
```

### Stage 1: Creation

```go
go func() {
    worker.Start(workerCtx)
}()
```

- Goroutine is created
- Added to Go runtime scheduler
- Ready to run

### Stage 2: Running

```go
func (w *Worker) Start(ctx context.Context) {
    for {
        // Worker is running
    }
}
```

- Goroutine is executing code
- Using CPU time

### Stage 3: Waiting

```go
select {
case job := <-w.jobQueue:
    // Blocked here, waiting for job
}
```

- Goroutine is blocked
- Waiting for channel to have data
- Not using CPU (efficient!)
- Go runtime can schedule other goroutines

### Stage 4: Running Again

```go
case job, ok := <-w.jobQueue:
    // Job received, unblocked
    // Now running again
```

- Channel has data
- Goroutine unblocks
- Continues execution

### Stage 5: Termination

```go
case <-ctx.Done():
    return  // Goroutine exits
```

- Context canceled
- Function returns
- Goroutine terminates
- Resources cleaned up

---

## Goroutine Ownership

### The Critical Question

**"Who owns this goroutine?"**

This is a **senior-level concern**. You must know:
- Who created the goroutine?
- Who is responsible for stopping it?
- What happens if it never stops?

### Our Worker Ownership

**Owner:** `main()` function

**Responsibilities:**
1. Create the goroutine
2. Provide context for cancellation
3. Wait for it to finish (WaitGroup)
4. Ensure it stops on shutdown

### The Pattern

```go
// 1. Create context (owner controls cancellation)
workerCtx, workerCancel := context.WithCancel(context.Background())

// 2. Create WaitGroup (owner tracks completion)
var wg sync.WaitGroup
wg.Add(1)

// 3. Start goroutine (owner creates it)
go func() {
    defer wg.Done()  // Signal completion
    worker.Start(workerCtx)  // Worker respects context
}()

// 4. On shutdown (owner stops it)
workerCancel()  // Cancel context
wg.Wait()       // Wait for completion
```

**Key Point:** The owner (main) has full control over the goroutine lifecycle.

### Why Ownership Matters

**Without clear ownership:**
- Goroutines might never stop (leaks)
- No way to coordinate shutdown
- Unclear who's responsible

**With clear ownership:**
- Owner can stop goroutine (via context)
- Owner can wait for completion (via WaitGroup)
- Lifecycle is explicit and controlled

---

## Common Mistakes

### Mistake 1: Goroutine Leak

```go
// ❌ BAD: Goroutine never stops
go func() {
    for {
        doWork()  // Infinite loop, no way to stop!
    }
}()
```

**Problem:** Goroutine runs forever, consuming resources.

**Fix:** Use context for cancellation
```go
// ✅ GOOD: Can be stopped
go func() {
    for {
        select {
        case <-ctx.Done():
            return  // Can exit
        default:
            doWork()
        }
    }
}()
```

### Mistake 2: No Way to Wait

```go
// ❌ BAD: No way to know when goroutine finishes
go worker.Start(ctx)
// main() continues, might exit before worker finishes
```

**Problem:** Program might exit before worker completes.

**Fix:** Use WaitGroup
```go
// ✅ GOOD: Can wait for completion
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(ctx)
}()
// Later...
wg.Wait()  // Wait for worker to finish
```

### Mistake 3: Forgetting to Cancel

```go
// ❌ BAD: Context created but never canceled
ctx, cancel := context.WithCancel(context.Background())
go worker.Start(ctx)
// cancel() never called - worker never stops!
```

**Problem:** Worker runs forever.

**Fix:** Always cancel on shutdown
```go
// ✅ GOOD: Cancel on shutdown
ctx, cancel := context.WithCancel(context.Background())
defer cancel()  // Will cancel when function exits
go worker.Start(ctx)
```

### Mistake 4: Creating Goroutines Without Control

```go
// ❌ BAD: Goroutine created in handler, no way to track it
func CreateJobHandler(...) {
    go processJob(job)  // Who owns this? How do we stop it?
}
```

**Problem:** No ownership, no control, potential leaks.

**Fix:** Use a worker pattern with controlled goroutines
```go
// ✅ GOOD: Worker goroutine owned by main()
// Handler just sends to channel
func CreateJobHandler(...) {
    jobQueue <- job  // Send to controlled worker
}
```

---

## Key Takeaways

1. **Goroutines** = Lightweight threads for concurrency
2. **Background workers** = Process work asynchronously
3. **`go` keyword** = Starts a new goroutine
4. **Goroutine lifecycle** = Creation → Running → Waiting → Termination
5. **Ownership matters** = Know who creates and stops goroutines
6. **Use context** = For cancellation and shutdown
7. **Use WaitGroup** = To wait for goroutines to finish

---

## Real-World Analogy

Think of a restaurant:

- **HTTP Handler** = Waiter (takes order, returns quickly)
- **Goroutine** = Kitchen worker (prepares food in background)
- **Channel** = Order ticket system (waiter puts ticket, kitchen gets it)

The waiter doesn't cook the food - they just take the order and give it to the kitchen. The kitchen worker (goroutine) prepares the food asynchronously.

---

## Next Steps

- Read [Channels for Communication](./02-channels-for-communication.md) to understand how workers receive jobs
- Read [Worker Pattern](./03-worker-pattern.md) to see the complete worker design
- Read [Graceful Shutdown](./05-graceful-shutdown.md) to learn how to stop workers properly

