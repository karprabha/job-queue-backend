# Understanding Graceful Shutdown

## Table of Contents

1. [What is Graceful Shutdown?](#what-is-graceful-shutdown)
2. [Why Graceful Shutdown Matters](#why-graceful-shutdown-matters)
3. [The Shutdown Challenge](#the-shutdown-challenge)
4. [Our Shutdown Implementation](#our-shutdown-implementation)
5. [WaitGroup Explained](#waitgroup-explained)
6. [Shutdown Sequence](#shutdown-sequence)
7. [Common Mistakes](#common-mistakes)

---

## What is Graceful Shutdown?

### The Simple Answer

**Graceful shutdown** means stopping your application in a controlled way:
- Finish current work
- Clean up resources
- Close connections properly
- No data loss

### The Opposite: Ungraceful Shutdown

**Ungraceful shutdown** (crash, kill signal):
- Work in progress is lost
- Resources not cleaned up
- Connections not closed
- Data corruption possible

### Real-World Analogy

**Graceful:** Like closing a restaurant
- Finish serving current customers
- Clean up the kitchen
- Lock the doors
- Turn off lights

**Ungraceful:** Like a power outage
- Customers left mid-meal
- Kitchen left messy
- Doors left open
- Everything stops immediately

---

## Why Graceful Shutdown Matters

### Problem 1: Lost Work

**Without graceful shutdown:**
```go
// Worker processing job
func processJob() {
    updateStatus("processing")
    doWork()  // Takes 5 seconds
    updateStatus("completed")  // Never reached if killed!
}
```

**Problem:** If process is killed, job stuck in "processing" state forever.

**With graceful shutdown:**
```go
// Worker respects shutdown signal
func processJob(ctx context.Context) {
    updateStatus("processing")
    select {
    case <-ctx.Done():
        updateStatus("failed")  // Mark as failed on shutdown
        return
    case <-time.After(5 * time.Second):
        updateStatus("completed")
    }
}
```

**Benefit:** Job status updated even on shutdown.

### Problem 2: Resource Leaks

**Without graceful shutdown:**
```go
// Goroutine running
go worker.Start()

// Process killed
// Goroutine never stops = LEAK!
```

**Problem:** Goroutines keep running, consuming resources.

**With graceful shutdown:**
```go
// Wait for goroutine to finish
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(ctx)
}()

// On shutdown
wg.Wait()  // Wait for goroutine to finish
```

**Benefit:** All goroutines finish before program exits.

### Problem 3: Incomplete Operations

**Without graceful shutdown:**
- Database transactions not committed
- Files not closed
- Network connections not closed
- Jobs half-processed

**With graceful shutdown:**
- Complete current operations
- Commit transactions
- Close files and connections
- Mark jobs appropriately

---

## The Shutdown Challenge

### The Problem

When shutting down, you need to coordinate multiple components:

1. **HTTP Server** - Stop accepting new requests
2. **Worker** - Finish current job, stop processing
3. **Channels** - Close properly
4. **Goroutines** - Wait for all to finish

**Challenge:** How do you coordinate all of this?

### The Solution: Multiple Mechanisms

We use several Go primitives:
1. **Context** - Signal cancellation
2. **WaitGroup** - Wait for goroutines
3. **Channel closing** - Signal no more work
4. **Server.Shutdown()** - Graceful HTTP shutdown

---

## Our Shutdown Implementation

### The Complete Shutdown Code

```go
// Wait for shutdown signal
<-sigChan
log.Println("Shutting down...")

// Cancel the context to stop the worker
workerCancel()
wg.Wait()
close(jobQueue)

// Graceful shutdown with timeout
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()

if err := srv.Shutdown(shutdownCtx); err != nil {
    log.Printf("Server shutdown error: %v", err)
}

log.Println("Server stopped")
```

### Breaking It Down

**Step 1: Wait for Signal**
```go
<-sigChan
```
- Blocks until SIGINT or SIGTERM received
- User presses Ctrl+C or system sends signal

**Step 2: Cancel Worker Context**
```go
workerCancel()
```
- Cancels worker's context
- Worker receives cancellation signal
- Worker stops processing new jobs

**Step 3: Wait for Worker**
```go
wg.Wait()
```
- Waits for worker goroutine to finish
- Worker exits its loop
- Ensures worker completes current job (if any)

**Step 4: Close Channel**
```go
close(jobQueue)
```
- Closes the job queue channel
- Signals no more jobs will be sent
- Safe because:
  - HTTP server stopped (no new jobs)
  - Worker stopped (not receiving)

**Step 5: Shutdown HTTP Server**
```go
srv.Shutdown(shutdownCtx)
```
- Stops accepting new connections
- Waits for existing requests to finish
- Times out after 10 seconds

---

## WaitGroup Explained

### What is WaitGroup?

A **WaitGroup** waits for a collection of goroutines to finish.

### How It Works

```go
var wg sync.WaitGroup

// Add number of goroutines to wait for
wg.Add(1)

// Start goroutine
go func() {
    defer wg.Done()  // Signal this goroutine is done
    doWork()
}()

// Wait for all goroutines to finish
wg.Wait()  // Blocks until wg.Done() called
```

### The Three Operations

**1. Add(n)**
```go
wg.Add(1)  // Expect 1 goroutine
```
- Tells WaitGroup how many goroutines to wait for
- Must be called before starting goroutines

**2. Done()**
```go
defer wg.Done()  // Signal finished
```
- Signals one goroutine finished
- Should use `defer` to ensure it's called

**3. Wait()**
```go
wg.Wait()  // Block until all done
```
- Blocks until all goroutines call `Done()`
- Only returns when count reaches 0

### Our Usage

```go
var wg sync.WaitGroup
wg.Add(1)  // We're starting 1 worker goroutine

go func() {
    defer wg.Done()  // Signal worker finished
    worker.Start(workerCtx)
}()

// Later, on shutdown:
wg.Wait()  // Wait for worker to finish
```

**What happens:**
1. `wg.Add(1)` - Expect 1 goroutine
2. Worker starts
3. On shutdown: `wg.Wait()` - Blocks until worker calls `Done()`
4. Worker exits loop, `defer wg.Done()` runs
5. `wg.Wait()` unblocks, shutdown continues

### Why Use WaitGroup?

**Without WaitGroup:**
```go
// ❌ BAD: No way to know when worker finishes
go worker.Start(ctx)
// main() might exit before worker finishes!
```

**Problem:** Program might exit while worker is still running.

**With WaitGroup:**
```go
// ✅ GOOD: Wait for worker to finish
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(ctx)
}()
// Later...
wg.Wait()  // Ensures worker finishes
```

**Benefit:** Program waits for worker to finish before exiting.

---

## Shutdown Sequence

### The Complete Flow

```
1. Signal Received (SIGINT/SIGTERM)
   ↓
2. Cancel Worker Context
   ├─> Worker receives cancellation
   ├─> Worker stops processing new jobs
   └─> Worker finishes current job (if any)
   ↓
3. Wait for Worker (WaitGroup)
   ├─> Blocks until worker exits
   └─> Worker calls wg.Done()
   ↓
4. Close Job Queue Channel
   ├─> No more jobs will be sent
   └─> Channel closed safely
   ↓
5. Shutdown HTTP Server
   ├─> Stop accepting new connections
   ├─> Wait for existing requests (max 10s)
   └─> Server closed
   ↓
6. Program Exits
```

### Why This Order?

**Order matters!** Let's see why:

**Wrong order:**
```go
// ❌ BAD: Closes channel while worker might be using it
close(jobQueue)  // Closes first
workerCancel()   // Stops worker
```

**Problem:** Worker might try to receive from closed channel.

**Correct order:**
```go
// ✅ GOOD: Stops worker first, then closes channel
workerCancel()   // Stops worker
wg.Wait()        // Wait for worker to finish
close(jobQueue)  // Safe to close now
```

**Why:**
1. Stop worker first (no longer receiving)
2. Wait for worker to finish (ensures it's done)
3. Close channel (safe, no one is using it)

### Timing Considerations

**Question:** What if worker is processing a long job?

**Answer:** We wait for it, but HTTP server shutdown has a timeout:

```go
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
srv.Shutdown(shutdownCtx)  // Max 10 seconds
```

**What happens:**
- Worker finishes current job (might take time)
- HTTP server waits max 10 seconds for requests
- If timeout, server closes anyway (some requests might be interrupted)

**Trade-off:** Balance between graceful shutdown and shutdown time.

---

## Common Mistakes

### Mistake 1: Not Waiting for Goroutines

```go
// ❌ BAD: Program exits before worker finishes
go worker.Start(ctx)
// main() continues and exits
// Worker might still be running!
```

**Problem:** Worker goroutine might not finish.

**Fix:** Use WaitGroup
```go
// ✅ GOOD: Wait for worker
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(ctx)
}()
wg.Wait()  // Wait before exiting
```

### Mistake 2: Closing Channel Too Early

```go
// ❌ BAD: Closes channel while worker using it
close(jobQueue)
workerCancel()
```

**Problem:** Worker might panic receiving from closed channel.

**Fix:** Stop worker first
```go
// ✅ GOOD: Stop worker, then close
workerCancel()
wg.Wait()
close(jobQueue)
```

### Mistake 3: Not Canceling Context

```go
// ❌ BAD: Worker never stops
go worker.Start(ctx)
// ctx never canceled, worker runs forever
```

**Problem:** Worker never receives shutdown signal.

**Fix:** Cancel context on shutdown
```go
// ✅ GOOD: Cancel on shutdown
workerCancel()  // Cancels context
```

### Mistake 4: Forgetting defer wg.Done()

```go
// ❌ BAD: wg.Done() might not be called
go func() {
    worker.Start(ctx)
    wg.Done()  // Only called if no panic/return
}()
```

**Problem:** If worker panics, `wg.Done()` never called, `wg.Wait()` blocks forever.

**Fix:** Use defer
```go
// ✅ GOOD: Always called
go func() {
    defer wg.Done()  // Always called, even on panic
    worker.Start(ctx)
}()
```

### Mistake 5: No Shutdown Timeout

```go
// ❌ BAD: Might wait forever
srv.Shutdown(context.Background())  // No timeout!
```

**Problem:** If requests never finish, shutdown hangs forever.

**Fix:** Use timeout
```go
// ✅ GOOD: Timeout prevents hanging
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

---

## Key Takeaways

1. **Graceful shutdown** = Controlled, clean shutdown
2. **WaitGroup** = Wait for goroutines to finish
3. **Context cancellation** = Signal workers to stop
4. **Shutdown order matters** = Stop workers → Wait → Close channels → Shutdown server
5. **Timeouts** = Prevent hanging shutdown
6. **Always use defer** = Ensures cleanup happens

---

## Real-World Analogy

Think of graceful shutdown like **closing a restaurant**:

1. **Stop accepting customers** (HTTP server stops)
2. **Tell kitchen to finish current orders** (Cancel worker context)
3. **Wait for kitchen to finish** (WaitGroup)
4. **Close order system** (Close channel)
5. **Lock the doors** (Server shutdown)
6. **Turn off lights** (Program exits)

**Ungraceful shutdown** = Power outage (everything stops immediately, chaos!)

---

## Next Steps

- Read [Context in Workers](./06-context-in-workers.md) to understand how context enables shutdown
- Read [Worker Pattern](./03-worker-pattern.md) to see how workers handle shutdown
- Read [Goroutines for Workers](./01-goroutines-for-workers.md) for goroutine lifecycle

