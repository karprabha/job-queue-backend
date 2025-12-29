# Understanding Retry Logic and Attempt Tracking

## Table of Contents

1. [Why Retry Limits Matter](#why-retry-limits-matter)
2. [Attempts vs MaxRetries](#attempts-vs-maxretries)
3. [Preventing Infinite Retries](#preventing-infinite-retries)
4. [When to Retry vs Permanent Failure](#when-to-retry-vs-permanent-failure)
5. [Atomic Attempt Increment](#atomic-attempt-increment)
6. [Common Mistakes](#common-mistakes)

---

## Why Retry Limits Matter

### The Problem Without Retry Limits

**Scenario:** A job fails due to a permanent issue (e.g., invalid email address).

**Without retry limits:**

```go
// ❌ BAD: Infinite retry loop
for {
    if jobFails {
        retry()  // Retry forever!
    }
}
```

**Problems:**

- Job retries forever (infinite loop)
- Wastes resources (CPU, memory, network)
- Blocks queue (job never completes)
- System becomes unresponsive
- Can't distinguish temporary vs permanent failures

### The Solution: Retry Limits

**With retry limits:**

```go
// ✅ GOOD: Limited retries
if job.Attempts < job.MaxRetries {
    retry()  // Retry only if under limit
} else {
    // Permanent failure - stop retrying
}
```

**Benefits:**

- Prevents infinite loops
- Saves resources
- Distinguishes temporary vs permanent failures
- System stays responsive
- Failed jobs are observable

### Real-World Analogy

Think of calling someone on the phone:

- **Without limit:** Keep calling forever (annoying, wastes time)
- **With limit:** Call 3 times, then give up (reasonable, efficient)

A job queue is similar - retry a few times, then accept permanent failure.

---

## Attempts vs MaxRetries

### The Fields

```go
type Job struct {
    ID         string
    Type       string
    Status     JobStatus
    Payload    json.RawMessage
    MaxRetries int  // Maximum number of retries allowed
    Attempts   int  // Current number of attempts
    LastError  *string
    CreatedAt  time.Time
}
```

### What is MaxRetries?

**MaxRetries** = Maximum number of times a job can be attempted.

**Example:**

```go
MaxRetries = 3  // Job can be attempted up to 3 times
```

**Meaning:**

- Attempt 1: First try
- Attempt 2: First retry
- Attempt 3: Second retry
- Attempt 4: Would exceed limit (permanent failure)

### What is Attempts?

**Attempts** = Current number of times the job has been attempted.

**Example:**

```go
Attempts = 2  // Job has been attempted 2 times so far
```

**Meaning:**

- When job is created: `Attempts = 0`
- After first claim: `Attempts = 1`
- After second claim: `Attempts = 2`
- After third claim: `Attempts = 3`

### The Relationship

**Retry condition:**

```go
if job.Attempts < job.MaxRetries {
    // Can retry - attempts haven't reached limit
    retry()
} else {
    // Cannot retry - attempts reached limit
    // Permanent failure
}
```

**Example with MaxRetries = 3:**

| Attempts | Can Retry? | Reason                  |
| -------- | ---------- | ----------------------- |
| 0        | ✅ Yes     | 0 < 3                   |
| 1        | ✅ Yes     | 1 < 3                   |
| 2        | ✅ Yes     | 2 < 3                   |
| 3        | ❌ No      | 3 >= 3 (limit reached)  |
| 4        | ❌ No      | 4 >= 3 (limit exceeded) |

---

## Preventing Infinite Retries

### The Check

```go
func (s *InMemoryJobStore) RetryFailedJobs(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    for jobID, job := range s.jobs {
        if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
            // Only retry if attempts haven't reached limit
            job.Status = domain.StatusPending
            s.jobs[jobID] = job
        }
        // If attempts >= MaxRetries, job stays in Failed (permanent failure)
    }

    return nil
}
```

### How It Works

**Step 1: Check Status**

```go
if job.Status == domain.StatusFailed {
    // Job is failed - might be retryable
}
```

**Step 2: Check Attempts**

```go
if job.Attempts < job.MaxRetries {
    // Can retry - attempts under limit
}
```

**Step 3: Retry or Keep Failed**

```go
if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
    job.Status = domain.StatusPending  // Retry
} else {
    // Stay in Failed (permanent failure)
}
```

### Example Flow

**Job with MaxRetries = 3:**

```
Attempt 1: Pending → Processing → Failed
           Attempts = 1, Can retry? Yes (1 < 3)
           ↓
Attempt 2: Failed → Pending → Processing → Failed
           Attempts = 2, Can retry? Yes (2 < 3)
           ↓
Attempt 3: Failed → Pending → Processing → Failed
           Attempts = 3, Can retry? No (3 >= 3)
           ↓
Permanent Failure: Stays in Failed
```

### Why This Prevents Infinite Loops

**Without check:**

```go
// ❌ BAD: Always retries
if job.Status == domain.StatusFailed {
    job.Status = domain.StatusPending  // Retry forever!
}
```

**With check:**

```go
// ✅ GOOD: Retry only if under limit
if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
    job.Status = domain.StatusPending  // Retry only if allowed
}
// If attempts >= MaxRetries, job stays Failed (no retry)
```

---

## When to Retry vs Permanent Failure

### Temporary Failures (Retry)

**Characteristics:**

- Issue might resolve itself
- Network timeout
- Service temporarily unavailable
- Rate limiting

**Example:**

```go
// Attempt 1: Network timeout
err := sendEmail(job)
if err != nil && isTemporary(err) {
    // Temporary - retry
    markAsFailed(job, err)
    // Will retry if attempts < maxRetries
}
```

### Permanent Failures (Don't Retry)

**Characteristics:**

- Issue won't resolve itself
- Invalid input data
- Authentication failure
- Resource doesn't exist

**Example:**

```go
// Attempt 1: Invalid email address
err := sendEmail(job)
if err != nil && isPermanent(err) {
    // Permanent - don't retry
    markAsFailed(job, err)
    // Won't retry (or set MaxRetries = 0)
}
```

### Our Implementation

**Current approach:** All failures are retried (up to limit).

```go
// All failed jobs are retried (if attempts < maxRetries)
if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
    job.Status = domain.StatusPending  // Retry
}
```

**Future improvement:** Distinguish temporary vs permanent.

```go
// Could add field:
type Job struct {
    // ...
    IsPermanentFailure bool  // If true, never retry
}

// Then check:
if job.Status == domain.StatusFailed &&
   !job.IsPermanentFailure &&
   job.Attempts < job.MaxRetries {
    job.Status = domain.StatusPending  // Retry
}
```

### The Decision

**When to retry:**

- Temporary issues (network, timeouts)
- Service unavailable
- Rate limiting
- Attempts < MaxRetries

**When not to retry:**

- Invalid input
- Authentication failure
- Resource doesn't exist
- Attempts >= MaxRetries

---

## Atomic Attempt Increment

### Where Attempts Are Incremented

```go
func (s *InMemoryJobStore) ClaimJob(ctx context.Context, jobID string) (*domain.Job, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    job, ok := s.jobs[jobID]
    if !ok || job.Status != domain.StatusPending {
        return nil, nil
    }

    job.Status = domain.StatusProcessing
    job.Attempts++  // Increment attempts atomically
    s.jobs[jobID] = job

    return &jobCopy, nil
}
```

### Why Increment in ClaimJob?

**Timing:**

- Attempts increment when job is **claimed** (not when it fails)
- This counts each processing attempt
- Even if job fails immediately, attempt is counted

**Example:**

```
Job created: Attempts = 0
ClaimJob: Attempts = 1 (first attempt)
Processing fails: Still Attempts = 1
Retry → ClaimJob: Attempts = 2 (second attempt)
Processing fails: Still Attempts = 2
Retry → ClaimJob: Attempts = 3 (third attempt)
Processing fails: Still Attempts = 3
Retry? No (3 >= 3)
```

### Why Atomic?

**Without mutex (race condition):**

```go
// ❌ BAD: Race condition
job.Attempts++  // Two workers could increment simultaneously!
// Worker 1 reads: Attempts = 1
// Worker 2 reads: Attempts = 1
// Worker 1 writes: Attempts = 2
// Worker 2 writes: Attempts = 2 (should be 3!)
```

**With mutex (atomic):**

```go
// ✅ GOOD: Atomic increment
s.mu.Lock()
job.Attempts++  // Protected by mutex
s.jobs[jobID] = job
s.mu.Unlock()
// Only one worker can increment at a time
```

### The Attempt Lifecycle

**Initial State:**

```go
job := domain.NewJob("email", payload)
// Attempts = 0
// MaxRetries = 3
```

**First Claim:**

```go
job, _ := store.ClaimJob(ctx, jobID)
// Attempts = 1 (incremented in ClaimJob)
```

**First Failure:**

```go
store.UpdateStatus(ctx, jobID, domain.StatusFailed, &errMsg)
// Attempts = 1 (unchanged)
// Can retry? Yes (1 < 3)
```

**Second Claim (Retry):**

```go
job, _ := store.ClaimJob(ctx, jobID)
// Attempts = 2 (incremented again)
```

**Second Failure:**

```go
store.UpdateStatus(ctx, jobID, domain.StatusFailed, &errMsg)
// Attempts = 2 (unchanged)
// Can retry? Yes (2 < 3)
```

**Third Claim (Retry):**

```go
job, _ := store.ClaimJob(ctx, jobID)
// Attempts = 3 (incremented again)
```

**Third Failure:**

```go
store.UpdateStatus(ctx, jobID, domain.StatusFailed, &errMsg)
// Attempts = 3 (unchanged)
// Can retry? No (3 >= 3)
// Permanent failure
```

---

## Common Mistakes

### Mistake 1: No Retry Limit Check

```go
// ❌ BAD: Infinite retries
func retryFailedJobs() {
    for jobID, job := range jobs {
        if job.Status == domain.StatusFailed {
            job.Status = domain.StatusPending  // Always retries!
        }
    }
}
```

**Fix:** Check attempts.

```go
// ✅ GOOD: Check limit
func retryFailedJobs() {
    for jobID, job := range jobs {
        if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
            job.Status = domain.StatusPending  // Retry only if allowed
        }
    }
}
```

### Mistake 2: Incrementing Attempts in Wrong Place

```go
// ❌ BAD: Increment when failing
func markAsFailed(job *domain.Job) {
    job.Status = domain.StatusFailed
    job.Attempts++  // Wrong! Should increment when claiming
}
```

**Fix:** Increment when claiming.

```go
// ✅ GOOD: Increment when claiming
func claimJob(jobID string) {
    job.Attempts++  // Increment here
    job.Status = domain.StatusProcessing
}
```

### Mistake 3: Not Checking Attempts Before Retry

```go
// ❌ BAD: No check
if job.Status == domain.StatusFailed {
    job.Status = domain.StatusPending  // Retry without checking attempts
}
```

**Fix:** Always check attempts.

```go
// ✅ GOOD: Check attempts
if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
    job.Status = domain.StatusPending  // Retry only if allowed
}
```

### Mistake 4: Race Condition on Attempts

```go
// ❌ BAD: Not atomic
job.Attempts++  // Two workers could increment simultaneously!
```

**Fix:** Use mutex.

```go
// ✅ GOOD: Atomic
s.mu.Lock()
job.Attempts++
s.jobs[jobID] = job
s.mu.Unlock()
```

### Mistake 5: MaxRetries = 0 Meaning Unclear

```go
// ❌ BAD: Unclear meaning
MaxRetries = 0  // Does this mean no retries? Or infinite retries?
```

**Fix:** Use clear semantics.

```go
// ✅ GOOD: Clear meaning
MaxRetries = 3  // Can retry up to 3 times (4 total attempts)
// Or use separate field:
AllowRetries = false  // Explicit: no retries allowed
```

---

## Key Takeaways

1. **Retry limits** prevent infinite loops
2. **Attempts** track how many times job has been tried
3. **MaxRetries** sets the maximum allowed retries
4. **Check attempts < MaxRetries** before retrying
5. **Increment attempts** when claiming (not when failing)
6. **Atomic increment** prevents race conditions
7. **Permanent failures** stay in Failed state

---

## Real-World Analogy

Think of calling a customer service line:

- **Attempt 1:** Call, busy signal → retry
- **Attempt 2:** Call, still busy → retry
- **Attempt 3:** Call, still busy → retry
- **Attempt 4:** Call, still busy → give up (permanent failure)

A job queue is similar - retry a few times, then accept permanent failure.

---

## Next Steps

- Read [The Sweeper Pattern](./04-sweeper-pattern.md) to see how retries are implemented
- Read [Atomic State Updates](./05-atomic-state-updates.md) to understand how attempts are tracked safely
- Read [State Machine](./01-state-machine-transitions.md) to see how retries fit into state transitions
