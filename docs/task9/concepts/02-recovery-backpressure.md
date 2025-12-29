# Recovery Backpressure

## Table of Contents

1. [What is Recovery Backpressure?](#what-is-recovery-backpressure)
2. [Why Backpressure During Recovery?](#why-backpressure-during-recovery)
3. [The Problem](#the-problem)
4. [Our Solution: Exponential Backoff](#our-solution-exponential-backoff)
5. [Implementation Details](#implementation-details)
6. [Common Mistakes](#common-mistakes)

---

## What is Recovery Backpressure?

### The Simple Answer

**Recovery backpressure** is the mechanism that prevents recovery from dropping jobs when the queue is full. Instead of failing immediately, recovery waits and retries with exponential backoff until the queue has space.

### The Challenge

During recovery:
- We need to re-enqueue all pending jobs
- The queue might be full (if it has limited capacity)
- We cannot drop jobs (they must be processed)
- We cannot block indefinitely (recovery must complete)

**Question:** How do we handle a full queue during recovery?

**Answer:** Use exponential backoff with retries - wait a bit, try again, wait longer, try again, until success or max attempts.

---

## Why Backpressure During Recovery?

### The Requirement

**Task requirement:** "Re-enqueueing must respect backpressure. If queue is full: recovery must pause and retry. No jobs may be dropped."

### Why This Matters

**Without backpressure handling:**
```go
// Recovery tries to enqueue
select {
case jobQueue <- job.ID:
    // Success
default:
    return fmt.Errorf("queue full")  // ❌ Job dropped!
}
```

**Problems:**
- Jobs are lost if queue is full
- Recovery fails immediately
- No retry mechanism
- System loses work

**With backpressure handling:**
```go
// Recovery retries with backoff
reEnqueueWithBackpressure(ctx, job.ID, jobQueue, logger)
// ✅ Job eventually enqueued or fails after max attempts
```

**Benefits:**
- No jobs dropped
- Recovery handles temporary queue full
- System eventually processes all jobs

---

## The Problem

### Scenario: Queue Full During Recovery

**Situation:**
- Queue capacity: 100
- Pending jobs to recover: 150
- Queue already has 50 jobs (from before crash)

**What happens:**
1. Recovery starts re-enqueuing
2. First 50 jobs enqueue successfully (queue now full)
3. Next job: queue is full
4. **Problem:** What do we do?

**Options:**
1. **Drop job** ❌ - Violates requirement
2. **Block forever** ❌ - Recovery never completes
3. **Fail immediately** ❌ - Jobs lost
4. **Retry with backoff** ✅ - Wait and try again

### Why Queue Might Be Full

**Possible reasons:**
- Queue has limited capacity (configurable)
- Previous jobs still in queue
- System under load
- Workers processing slowly

**Key insight:** Queue full is temporary. If we wait, space will become available as workers process jobs.

---

## Our Solution: Exponential Backoff

### The Strategy

**Exponential backoff:**
1. Try to enqueue immediately
2. If full, wait a short time (50ms)
3. Try again
4. If still full, wait longer (75ms)
5. Try again
6. Continue with increasing wait times (up to max)
7. After max attempts, fail

### Why Exponential?

**Linear backoff:**
- Wait 50ms, 50ms, 50ms, ...
- Predictable but might be too slow

**Exponential backoff:**
- Wait 50ms, 75ms, 112ms, 168ms, ...
- Starts fast, increases gradually
- Balances between quick recovery and not overwhelming system

**Formula:**
```go
backoff = backoff * 1.5  // 50% increase each time
if backoff > maxBackoff {
    backoff = maxBackoff  // Cap at maximum
}
```

### Implementation

```go
func reEnqueueWithBackpressure(
    ctx context.Context,
    jobID string,
    jobQueue chan string,
    logger *slog.Logger,
) error {
    backoff := 50 * time.Millisecond
    maxBackoff := 5 * time.Second
    maxAttempts := 10

    for attempt := 0; attempt < maxAttempts; attempt++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case jobQueue <- jobID:
            if attempt > 0 {
                logger.Info("Job re-enqueued after backoff",
                    "event", "job_re_enqueued",
                    "job_id", jobID,
                    "attempt", attempt+1)
            }
            return nil // Success!
        default:
            if attempt < maxAttempts-1 {
                logger.Info("Queue full during recovery, backing off",
                    "event", "recovery_backpressure",
                    "job_id", jobID,
                    "attempt", attempt+1,
                    "backoff_ms", backoff.Milliseconds())

                select {
                case <-ctx.Done():
                    return ctx.Err()
                case <-time.After(backoff):
                    // Exponential backoff with cap
                    backoff = time.Duration(float64(backoff) * 1.5)
                    if backoff > maxBackoff {
                        backoff = maxBackoff
                    }
                }
            }
        }
    }

    return fmt.Errorf("failed to enqueue job %s after %d attempts: queue persistently full", jobID, maxAttempts)
}
```

---

## Implementation Details

### Breaking Down the Function

**1. Initial Setup**
```go
backoff := 50 * time.Millisecond
maxBackoff := 5 * time.Second
maxAttempts := 10
```
- Start with 50ms backoff
- Cap at 5 seconds
- Try up to 10 times

**2. Retry Loop**
```go
for attempt := 0; attempt < maxAttempts; attempt++ {
    // Try to enqueue
}
```
- Try up to 10 times
- Each attempt tries to enqueue

**3. Try to Enqueue**
```go
select {
case <-ctx.Done():
    return ctx.Err()
case jobQueue <- jobID:
    return nil // Success!
default:
    // Queue is full
}
```
- Non-blocking send
- Success if queue has space
- Default case if queue is full

**4. Backoff on Failure**
```go
default:
    if attempt < maxAttempts-1 {
        logger.Info("Queue full during recovery, backing off", ...)
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
            // Wait, then increase backoff
            backoff = time.Duration(float64(backoff) * 1.5)
            if backoff > maxBackoff {
                backoff = maxBackoff
            }
        }
    }
```
- Log the backoff
- Wait for backoff duration
- Increase backoff for next attempt
- Cap at maxBackoff

**5. Context Cancellation**
```go
select {
case <-ctx.Done():
    return ctx.Err()
case jobQueue <- job.ID:
    // ...
}
```
- Always check context
- Return immediately if canceled
- Allows recovery to be canceled

### Why These Values?

**Initial backoff: 50ms**
- Short enough to be responsive
- Long enough to allow queue to drain
- Not too aggressive

**Max backoff: 5 seconds**
- Long enough to wait for workers
- Short enough that recovery completes
- Reasonable maximum wait

**Max attempts: 10**
- Enough retries to handle temporary full
- Not infinite (prevents hanging)
- Reasonable limit

**Backoff multiplier: 1.5**
- 50ms → 75ms → 112ms → 168ms → ...
- Gradual increase
- Not too aggressive

### Example Timeline

**Attempt 1:** Try immediately → Queue full
**Attempt 2:** Wait 50ms → Try → Queue full
**Attempt 3:** Wait 75ms → Try → Queue full
**Attempt 4:** Wait 112ms → Try → Queue full
**Attempt 5:** Wait 168ms → Try → **Success!**

**Total time:** ~405ms

---

## Common Mistakes

### Mistake 1: Dropping Jobs on Full Queue

**❌ BAD:**
```go
select {
case jobQueue <- job.ID:
    // Success
default:
    continue  // Skip job!
}
```

**Problem:**
- Jobs are silently dropped
- Violates requirement
- Jobs are lost

**✅ GOOD:**
```go
if err := reEnqueueWithBackpressure(ctx, job.ID, jobQueue, logger); err != nil {
    return err  // Fail if can't enqueue after retries
}
```

**Benefit:** No jobs dropped.

### Mistake 2: Blocking Forever

**❌ BAD:**
```go
// Blocking send - waits forever
jobQueue <- job.ID  // Blocks if queue full!
```

**Problem:**
- Recovery hangs if queue is full
- No progress
- System stuck

**✅ GOOD:**
```go
// Non-blocking with retry
select {
case jobQueue <- job.ID:
    return nil
default:
    // Retry with backoff
}
```

**Benefit:** Recovery makes progress.

### Mistake 3: No Retry Limit

**❌ BAD:**
```go
for {
    select {
    case jobQueue <- job.ID:
        return nil
    default:
        time.Sleep(100 * time.Millisecond)
        // Infinite loop!
    }
}
```

**Problem:**
- Could retry forever
- Recovery never completes
- System hangs

**✅ GOOD:**
```go
for attempt := 0; attempt < maxAttempts; attempt++ {
    // Try with limit
}
```

**Benefit:** Recovery completes or fails clearly.

### Mistake 4: Fixed Backoff

**❌ BAD:**
```go
for attempt := 0; attempt < maxAttempts; attempt++ {
    select {
    case jobQueue <- job.ID:
        return nil
    default:
        time.Sleep(100 * time.Millisecond)  // Fixed wait
    }
}
```

**Problem:**
- Always waits same time
- Might be too short or too long
- Not adaptive

**✅ GOOD:**
```go
backoff := 50 * time.Millisecond
for attempt := 0; attempt < maxAttempts; attempt++ {
    // ...
    backoff = time.Duration(float64(backoff) * 1.5)  // Exponential
}
```

**Benefit:** Adaptive, efficient.

### Mistake 5: Not Checking Context

**❌ BAD:**
```go
for attempt := 0; attempt < maxAttempts; attempt++ {
    select {
    case jobQueue <- job.ID:
        return nil
    default:
        time.Sleep(backoff)  // No context check!
    }
}
```

**Problem:**
- Can't cancel recovery
- Blocks even if shutdown requested
- No responsiveness

**✅ GOOD:**
```go
select {
case <-ctx.Done():
    return ctx.Err()
case <-time.After(backoff):
    // Continue
}
```

**Benefit:** Recovery can be canceled.

---

## Key Takeaways

1. **Never drop jobs** - Always retry with backoff
2. **Non-blocking operations** - Use select with default
3. **Exponential backoff** - Start fast, increase gradually
4. **Retry limits** - Don't retry forever
5. **Context cancellation** - Allow recovery to be canceled
6. **Logging** - Log backoff attempts for observability

---

## Related Concepts

- [Startup Recovery](./01-startup-recovery.md) - Overall recovery process
- [State Transitions](./03-state-transitions-recovery.md) - How recovery respects state machine
- [Source of Truth](./04-source-of-truth.md) - Why store is authoritative

