# Backpressure Implementation

## Table of Contents

1. [What is Backpressure?](#what-is-backpressure)
2. [Why Backpressure Matters](#why-backpressure-matters)
3. [The Problem We're Solving](#the-problem-were-solving)
4. [Our Backpressure Implementation](#our-backpressure-implementation)
5. [Non-Blocking Channel Operations](#non-blocking-channel-operations)
6. [HTTP Status Code: 429 Too Many Requests](#http-status-code-429-too-many-requests)
7. [Common Mistakes](#common-mistakes)

---

## What is Backpressure?

### The Simple Answer

**Backpressure** is a mechanism that allows a system to **reject new work** when it's **overloaded**, preventing:
- System overload
- Memory exhaustion
- Degraded performance
- Cascading failures

### Real-World Analogy

**Restaurant Analogy:**
- **No backpressure:** Restaurant accepts unlimited reservations → Kitchen gets overwhelmed → Food quality drops → Everyone has a bad experience
- **With backpressure:** Restaurant says "We're full, please come back later" → Kitchen can handle current orders → Food quality maintained → Better experience for everyone

**Key insight:** It's better to **reject work gracefully** than to **accept work you can't handle**.

---

## Why Backpressure Matters

### Problem 1: Memory Exhaustion

**Without backpressure:**

```go
// Unlimited queue
jobQueue := make(chan string)  // Unbuffered, but handlers keep sending

// Handler keeps accepting jobs
func (h *JobHandler) CreateJob(...) {
    job := domain.NewJob(...)
    h.jobQueue <- job.ID  // Blocks if queue is full
    // Handler waits forever if queue never drains
}
```

**Problem:**
- If workers are slow, queue fills up
- Handlers block waiting to send
- HTTP connections stay open
- Memory usage grows
- Eventually: Out of memory crash

**With backpressure:**

```go
// Limited queue capacity
jobQueue := make(chan string, 100)  // Buffer of 100

// Handler rejects when full
func (h *JobHandler) CreateJob(...) {
    select {
    case h.jobQueue <- job.ID:
        // Successfully enqueued
    default:
        // Queue is full - reject immediately
        ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)
        return
    }
}
```

**Benefit:**
- System never accepts more work than it can handle
- Handlers respond immediately (no blocking)
- Memory usage bounded
- System remains responsive

### Problem 2: Degraded Performance

**Without backpressure:**
- System accepts all work
- Queue grows unbounded
- Workers can't keep up
- Response times increase
- System becomes unresponsive

**With backpressure:**
- System rejects excess work
- Queue stays at manageable size
- Workers can process efficiently
- Response times remain stable
- System stays responsive

### Problem 3: Cascading Failures

**Without backpressure:**
- One component gets overloaded
- It affects other components
- Entire system degrades
- Hard to recover

**With backpressure:**
- Overloaded component rejects work
- Other components unaffected
- System remains stable
- Easy to recover

---

## The Problem We're Solving

### The Scenario

1. **Job queue has limited capacity** (e.g., 100 jobs)
2. **Workers process jobs** (takes time)
3. **New jobs arrive faster** than workers can process
4. **Queue fills up**

**Question:** What should happen when the queue is full?

**Answer:** Reject new jobs with `429 Too Many Requests` instead of blocking.

### The Challenge

**Blocking approach (bad):**

```go
// Handler blocks waiting for queue space
h.jobQueue <- job.ID  // Blocks if queue is full
// HTTP connection stays open
// Handler can't respond
// Client waits forever
```

**Problems:**
- HTTP handler blocks indefinitely
- Client connection stays open
- Resources tied up
- Poor user experience

**Non-blocking approach (good):**

```go
// Handler checks if queue has space
select {
case h.jobQueue <- job.ID:
    // Success - job enqueued
case default:
    // Queue full - reject immediately
    ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)
    return
}
```

**Benefits:**
- Handler responds immediately
- Client gets clear error message
- Resources freed quickly
- Better user experience

---

## Our Backpressure Implementation

### Step 1: Limited Queue Capacity

```go
// In main.go
config := config.NewConfig()
jobQueue := make(chan string, config.JobQueueCapacity)  // e.g., 100
```

**Key points:**
- Queue has a **maximum capacity** (configurable)
- Once full, new sends will block (unless we use `select` with `default`)
- Capacity should match system's processing ability

**How to choose capacity:**
- Too small: Reject too many jobs
- Too large: Use too much memory
- Rule of thumb: `capacity = worker_count * expected_job_duration`

### Step 2: Non-Blocking Send in Handler

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

### Step 3: Understanding the Select Statement

```go
select {
case h.jobQueue <- job.ID:
    // Case 1: Successfully sent to channel
    // This case is chosen if channel has space
case <-r.Context().Done():
    // Case 2: Request was canceled
    // This case is chosen if client disconnected
default:
    // Case 3: Neither case 1 nor case 2 is ready
    // This case is chosen if channel is full AND request not canceled
}
```

**How `select` works:**
- Evaluates all cases simultaneously
- Chooses the first case that's ready
- If multiple cases ready, chooses one randomly
- If no cases ready and `default` exists, executes `default` immediately
- If no cases ready and no `default`, blocks until one case is ready

**In our case:**
- If queue has space → Case 1 executes (job enqueued)
- If request canceled → Case 2 executes (return early)
- If queue full AND request not canceled → Case 3 executes (reject job)

### Step 4: Cleanup on Rejection

```go
default:
    // Queue is full - reject job
    h.store.DeleteJob(r.Context(), job.ID)  // Remove job from store
    h.metricStore.DecrementJobsCreated(r.Context())  // Fix metrics
    ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)
    return
```

**Why cleanup:**
- Job was created in store
- Job was counted in metrics
- But job won't be processed
- Must clean up to keep state consistent

**Note:** In a production system, you might want to keep the job for retry later instead of deleting it. This implementation deletes it to keep things simple.

---

## Non-Blocking Channel Operations

### Blocking Send

```go
// Blocks if channel is full
jobQueue <- job.ID
// Execution stops here until space available
```

**When it blocks:**
- Channel buffer is full
- No receiver ready
- Execution waits indefinitely

**Problem:** HTTP handler can't respond while blocked.

### Non-Blocking Send with Select

```go
select {
case jobQueue <- job.ID:
    // Send succeeded
default:
    // Send would block - handle it
}
```

**How it works:**
- `select` checks if send would succeed
- If yes → Case 1 executes immediately
- If no → `default` executes immediately
- Never blocks

**Benefit:** Handler always responds quickly.

### Blocking Receive

```go
// Blocks if channel is empty
jobID := <-jobQueue
// Execution stops here until data available
```

**When it blocks:**
- Channel buffer is empty
- No sender ready
- Execution waits indefinitely

**This is OK for workers** - they should wait for work.

### Non-Blocking Receive with Select

```go
select {
case jobID, ok := <-jobQueue:
    if !ok {
        // Channel closed
        return
    }
    // Process job
case <-ctx.Done():
    // Context canceled
    return
default:
    // No job available - do something else
}
```

**When to use:**
- Worker wants to check for work but not block
- Component needs to do other things while waiting
- Not needed in our worker (workers should block waiting for work)

---

## HTTP Status Code: 429 Too Many Requests

### What is 429?

**HTTP 429 Too Many Requests** is a status code that means:
- "I understand your request"
- "But I'm too busy right now"
- "Please try again later"

### When to Use 429

Use `429` when:
- System is overloaded
- Rate limit exceeded
- Queue is full
- Temporary condition (not a permanent error)

### Our Implementation

```go
ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)
```

**Response body:**
```json
{
  "error": "Job queue is full"
}
```

**Status code:** `429`

**Client behavior:**
- Client knows request was valid
- Client knows it's a temporary condition
- Client can retry later
- Client should use exponential backoff

### Alternative: 503 Service Unavailable

```go
ErrorResponse(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
```

**When to use 503:**
- Entire service is down
- Not just overloaded, but unavailable
- More severe than 429

**In our case:** We use `429` because:
- Service is still running
- Just the queue is full
- Temporary condition
- Client should retry

---

## Common Mistakes

### Mistake 1: Blocking on Full Channel

```go
// ❌ BAD: Handler blocks if queue is full
func (h *JobHandler) CreateJob(...) {
    h.jobQueue <- job.ID  // Blocks here!
    // Handler can't respond
    // Client waits forever
}
```

**Problem:** HTTP handler blocks, client connection stays open.

**Fix:**

```go
// ✅ GOOD: Non-blocking with select
select {
case h.jobQueue <- job.ID:
    // Success
default:
    ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)
    return
}
```

### Mistake 2: No Queue Capacity Limit

```go
// ❌ BAD: Unlimited queue
jobQueue := make(chan string)  // Unbuffered, but can grow
// Or
jobQueue := make(chan string, 1000000)  // Too large
```

**Problem:** Memory can grow unbounded.

**Fix:**

```go
// ✅ GOOD: Reasonable capacity
jobQueue := make(chan string, 100)  // Configurable, reasonable size
```

### Mistake 3: Wrong Status Code

```go
// ❌ BAD: Wrong status code
ErrorResponse(w, "Queue full", http.StatusInternalServerError)  // 500
// Or
ErrorResponse(w, "Queue full", http.StatusBadRequest)  // 400
```

**Problem:** Client doesn't know it's a temporary overload condition.

**Fix:**

```go
// ✅ GOOD: Correct status code
ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)  // 429
```

### Mistake 4: Not Cleaning Up on Rejection

```go
// ❌ BAD: Job created but not cleaned up
select {
case h.jobQueue <- job.ID:
    // Success
default:
    ErrorResponse(w, "Queue full", http.StatusTooManyRequests)
    // Job still in store! Metrics still incremented!
    return
}
```

**Problem:** Inconsistent state - job exists but won't be processed.

**Fix:**

```go
// ✅ GOOD: Clean up on rejection
default:
    h.store.DeleteJob(r.Context(), job.ID)
    h.metricStore.DecrementJobsCreated(r.Context())
    ErrorResponse(w, "Job queue is full", http.StatusTooManyRequests)
    return
```

### Mistake 5: Ignoring Request Cancellation

```go
// ❌ BAD: Doesn't check request cancellation
select {
case h.jobQueue <- job.ID:
    // Success
default:
    // Reject
}
// What if client disconnected?
```

**Problem:** Might try to send to queue even after client disconnected.

**Fix:**

```go
// ✅ GOOD: Check request cancellation
select {
case h.jobQueue <- job.ID:
    // Success
case <-r.Context().Done():
    // Request canceled
    ErrorResponse(w, "Request cancelled", http.StatusRequestTimeout)
    return
default:
    // Reject
}
```

---

## Key Takeaways

1. **Backpressure prevents overload** - Reject work when system can't handle it
2. **Non-blocking operations** - Use `select` with `default` to avoid blocking
3. **HTTP 429 status code** - Use for temporary overload conditions
4. **Queue capacity limits** - Set reasonable limits to bound memory usage
5. **Cleanup on rejection** - Remove jobs and fix metrics when rejecting
6. **Request cancellation** - Always check `r.Context().Done()` in handlers
7. **Better to reject than degrade** - It's better to say "no" than to accept work you can't handle

---

## Next Steps

- Read about [Graceful Shutdown Coordination](./01-graceful-shutdown-coordination.md)
- Learn about [Channel Closing Strategy](./03-channel-closing-strategy.md)
- Understand [Worker Lifecycle Management](./04-worker-lifecycle-management.md)

