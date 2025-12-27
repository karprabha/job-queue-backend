# Understanding Channels for Communication

## Table of Contents

1. [Why Channels?](#why-channels)
2. [What is a Channel?](#what-is-a-channel)
3. [Channel Operations](#channel-operations)
4. [Buffered vs Unbuffered Channels](#buffered-vs-unbuffered-channels)
5. [Our Job Queue Channel](#our-job-queue-channel)
6. [Channel Closing](#channel-closing)
7. [Select Statement with Channels](#select-statement-with-channels)
8. [Common Mistakes](#common-mistakes)

---

## Why Channels?

### The Problem: Sharing Data Between Goroutines

When you have multiple goroutines, you need a way to:
1. **Send data** from one goroutine to another
2. **Synchronize** goroutines (coordinate their execution)
3. **Avoid race conditions** (safe data sharing)

### The Wrong Way: Shared Variables

```go
// ❌ BAD: Race condition!
var jobs []Job

// Goroutine 1
go func() {
    jobs = append(jobs, newJob)  // Writing
}()

// Goroutine 2
go func() {
    for _, job := range jobs {  // Reading
        process(job)
    }
}()
```

**Problems:**
- Race conditions (both reading and writing)
- Need mutexes (complex, error-prone)
- No coordination (when is data ready?)

### The Right Way: Channels

```go
// ✅ GOOD: Safe communication
jobQueue := make(chan Job)

// Goroutine 1 (HTTP Handler)
go func() {
    jobQueue <- newJob  // Send job
}()

// Goroutine 2 (Worker)
go func() {
    job := <-jobQueue  // Receive job (safe!)
    process(job)
}()
```

**Benefits:**
- Thread-safe (built-in synchronization)
- Coordinates goroutines (blocks until data available)
- Clear ownership (sender → receiver)

---

## What is a Channel?

### The Simple Answer

A **channel** is a typed conduit (pipe) for sending and receiving values between goroutines.

### The Detailed Answer

A channel is:
- A **typed** communication mechanism
- **Thread-safe** by design
- **Blocking** by default (synchronizes goroutines)
- **First-in-first-out** (FIFO) queue

### Visual Analogy

Think of a channel like a **pipe**:

```
Sender (Goroutine 1)  →  [Channel]  →  Receiver (Goroutine 2)
   jobQueue <- job                    job := <-jobQueue
```

- Sender puts data into the pipe
- Receiver takes data from the pipe
- If pipe is empty, receiver waits
- If pipe is full, sender waits

### Creating a Channel

```go
// Unbuffered channel
ch := make(chan int)

// Buffered channel (capacity 10)
ch := make(chan int, 10)
```

**Syntax:**
- `make(chan Type)` - Unbuffered channel
- `make(chan Type, capacity)` - Buffered channel

---

## Channel Operations

### Sending (Putting Data In)

```go
ch <- value
```

**What happens:**
- Value is sent to the channel
- If channel is full (buffered) or no receiver (unbuffered), **blocks** until space available
- Operation completes when value is received

**Example:**
```go
jobQueue <- job  // Send job to channel
```

### Receiving (Getting Data Out)

```go
value := <-ch
```

**What happens:**
- Value is received from the channel
- If channel is empty, **blocks** until value available
- Operation completes when value is received

**Example:**
```go
job := <-jobQueue  // Receive job from channel
```

### Two-Value Receive (Check if Channel Closed)

```go
value, ok := <-ch
```

**What happens:**
- `value` = the received value
- `ok` = `true` if value received, `false` if channel closed

**Example:**
```go
job, ok := <-jobQueue
if !ok {
    // Channel is closed, no more jobs
    return
}
```

### Closing a Channel

```go
close(ch)
```

**What happens:**
- Channel is marked as closed
- No more values can be sent
- Receivers can still receive remaining values
- After all values received, receives return zero value and `ok = false`

**Important:** Only the sender should close the channel!

---

## Buffered vs Unbuffered Channels

### Unbuffered Channel

```go
ch := make(chan int)  // No capacity specified
```

**Characteristics:**
- Capacity = 0
- Sender **blocks** until receiver is ready
- Receiver **blocks** until sender is ready
- **Synchronous** - sender and receiver must meet

**Example:**
```go
ch := make(chan int)

go func() {
    ch <- 42  // Blocks until receiver ready
    fmt.Println("Sent")
}()

value := <-ch  // Blocks until sender ready
fmt.Println("Received:", value)
```

**Output:**
```
Received: 42
Sent
```

**What happened:**
1. Sender tries to send, blocks (no receiver yet)
2. Receiver receives, unblocks sender
3. Both continue

### Buffered Channel

```go
ch := make(chan int, 10)  // Capacity = 10
```

**Characteristics:**
- Has capacity (buffer)
- Sender **only blocks** if buffer is full
- Receiver **only blocks** if buffer is empty
- **Asynchronous** - sender and receiver don't need to meet

**Example:**
```go
ch := make(chan int, 2)  // Buffer size 2

ch <- 1  // Doesn't block (buffer has space)
ch <- 2  // Doesn't block (buffer has space)
ch <- 3  // Blocks! (buffer is full)

value := <-ch  // Receives 1, unblocks sender
```

### Visual Comparison

**Unbuffered (Synchronous):**
```
Sender  →  [ ]  →  Receiver
         (must meet)
```

**Buffered (Asynchronous):**
```
Sender  →  [1][2][3]  →  Receiver
         (can queue values)
```

---

## Our Job Queue Channel

### Our Implementation

```go
const jobQueueCapacity = 100
jobQueue := make(chan *domain.Job, jobQueueCapacity)
```

### Why Buffered?

**Decision:** We chose a **buffered channel** with capacity 100.

**Reasons:**

1. **Decouple HTTP handler from worker**
   - HTTP handler can send jobs quickly
   - Doesn't block if worker is busy
   - Better HTTP response times

2. **Handle bursts**
   - Multiple requests can create jobs quickly
   - Buffer absorbs temporary spikes
   - Worker processes at its own pace

3. **Backpressure control**
   - If buffer is full, sender blocks
   - Prevents unbounded memory growth
   - Natural rate limiting

### What Happens in Our Code

**Scenario 1: Normal Operation**
```go
// HTTP Handler
jobQueue <- job  // Sends immediately (buffer has space)
// Returns response quickly ✅

// Worker
job := <-jobQueue  // Receives when ready
// Processes job
```

**Scenario 2: Buffer Full**
```go
// HTTP Handler
jobQueue <- job  // Blocks! (buffer is full, worker busy)
// Waits until worker processes a job
// Then sends and returns ✅
```

**Scenario 3: Buffer Empty**
```go
// Worker
job := <-jobQueue  // Blocks! (no jobs in buffer)
// Waits until HTTP handler sends a job
// Then receives and processes ✅
```

### Our Handler's Timeout Logic

In `job_handler.go`, we have:

```go
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
```

**What this does:**
- Tries to send job to queue
- If queue is full, waits up to 100ms
- If still full after 100ms, logs warning but job is still created in store
- If client disconnects, returns immediately

**Note:** This is a design decision. We could also:
- Block indefinitely (simpler, but might block HTTP handler)
- Return error if queue full (client can retry)
- Use larger buffer (more memory, less blocking)

---

## Channel Closing

### When to Close

**Rule:** Only close channels when you're **sure** no more values will be sent.

**In our code:**
```go
// On shutdown
close(jobQueue)  // No more jobs will be sent
```

### How Worker Handles Closed Channel

```go
case job, ok := <-w.jobQueue:
    if !ok {
        return  // Channel closed, exit worker
    }
    // Process job...
```

**What happens:**
1. Channel is closed
2. Worker receives `ok = false`
3. Worker exits gracefully

### Important Rules

1. **Only sender closes** - Never close from receiver
2. **Close once** - Closing an already-closed channel panics
3. **Check before sending** - Don't send to closed channel (panics)

### Our Shutdown Sequence

```go
// 1. Stop accepting new HTTP requests
srv.Shutdown(shutdownCtx)

// 2. Cancel worker context (stops processing new jobs)
workerCancel()

// 3. Wait for worker to finish current job
wg.Wait()

// 4. Close channel (safe now, no one is sending)
close(jobQueue)
```

**Why this order?**
1. Stop new jobs from being created
2. Stop worker from starting new jobs
3. Let worker finish current job
4. Close channel (no one will send to it)

---

## Select Statement with Channels

### What is Select?

`select` is like a `switch` statement, but for channels. It waits for one of multiple channel operations to proceed.

### Basic Syntax

```go
select {
case value := <-ch1:
    // Received from ch1
case ch2 <- data:
    // Sent to ch2
case <-ctx.Done():
    // Context canceled
default:
    // No channel ready (non-blocking)
}
```

### Our Worker's Select

```go
select {
case <-ctx.Done():
    return  // Shutdown signal
case job, ok := <-w.jobQueue:
    if !ok {
        return  // Channel closed
    }
    // Process job...
}
```

**What this does:**
- Waits for either:
  - Context cancellation (shutdown)
  - Job from queue
- Blocks until one of these happens
- Processes whichever happens first

### Why Select is Powerful

**Without select:**
```go
// ❌ Can only wait for one channel
job := <-jobQueue  // Blocks, can't check context
```

**With select:**
```go
// ✅ Can wait for multiple channels
select {
case <-ctx.Done():
    return  // Can respond to shutdown
case job := <-jobQueue:
    process(job)  // Can process jobs
}
```

### Non-Blocking with Default

```go
select {
case job := <-jobQueue:
    process(job)  // Process if available
default:
    // No job available, don't block
    doOtherWork()
}
```

**Use case:** Check if channel has data without blocking.

---

## Common Mistakes

### Mistake 1: Sending to Closed Channel

```go
// ❌ BAD: Panics!
close(ch)
ch <- value  // Panic: send on closed channel
```

**Fix:** Don't send after closing, or check if closed first.

### Mistake 2: Closing from Receiver

```go
// ❌ BAD: Receiver shouldn't close
go func() {
    value := <-ch
    close(ch)  // Wrong! Only sender should close
}()
```

**Fix:** Only sender closes the channel.

### Mistake 3: Unbuffered Channel Deadlock

```go
// ❌ BAD: Deadlock!
ch := make(chan int)
ch <- 42  // Blocks forever (no receiver)
value := <-ch  // Never reached
```

**Fix:** Use goroutines or buffered channel.

### Mistake 4: Not Checking Channel Closed

```go
// ❌ BAD: Infinite loop on closed channel
for {
    job := <-jobQueue  // Receives zero value forever
    process(job)  // Processes empty jobs!
}
```

**Fix:** Check `ok` value
```go
// ✅ GOOD: Checks if channel closed
for {
    job, ok := <-jobQueue
    if !ok {
        return  // Channel closed, exit
    }
    process(job)
}
```

### Mistake 5: Forgetting to Close

```go
// ❌ BAD: Worker never knows when to stop
go worker.Start(ctx)
// Channel never closed, worker waits forever
```

**Fix:** Close channel on shutdown
```go
// ✅ GOOD: Closes channel on shutdown
defer close(jobQueue)
go worker.Start(ctx)
```

---

## Key Takeaways

1. **Channels** = Safe communication between goroutines
2. **Buffered channels** = Asynchronous, can queue values
3. **Unbuffered channels** = Synchronous, sender and receiver meet
4. **Select statement** = Wait for multiple channels
5. **Close channels** = Signal no more values (only sender closes)
6. **Check `ok`** = Detect closed channels
7. **Blocking behavior** = Channels synchronize goroutines naturally

---

## Real-World Analogy

Think of channels like a **mailbox**:

- **Unbuffered** = Hand-to-hand delivery (must meet)
- **Buffered** = Mailbox with slots (can queue letters)
- **Sending** = Putting letter in mailbox
- **Receiving** = Taking letter from mailbox
- **Closing** = Mailbox is full, no more letters accepted
- **Select** = Check multiple mailboxes at once

---

## Next Steps

- Read [Worker Pattern](./03-worker-pattern.md) to see how channels are used in workers
- Read [Channel Buffering Decisions](./04-channel-buffering.md) for deeper analysis
- Read [Select Statement](./08-select-statement.md) for advanced channel patterns

