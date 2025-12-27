# Preventing Duplicate Processing

## Table of Contents

1. [The Duplicate Processing Problem](#the-duplicate-processing-problem)
2. [Why Duplicates Happen](#why-duplicates-happen)
3. [The ClaimJob Pattern](#the-claimjob-pattern)
4. [How ClaimJob Works](#how-claimjob-works)
5. [Race Condition Prevention](#race-condition-prevention)
6. [Store as Source of Truth](#store-as-source-of-truth)
7. [Common Mistakes](#common-mistakes)

---

## The Duplicate Processing Problem

### The Scenario

You have **multiple workers** listening to the same job queue:

```
Job Queue: [Job A]
    │
    ├─> Worker 1 (waiting)
    ├─> Worker 2 (waiting)
    └─> Worker 3 (waiting)
```

**Question:** What happens when Job A is sent to the channel?

**Naive Answer:** "Only one worker gets it, channels guarantee that."

**Reality:** While channels guarantee only one worker receives the job, there's a **subtle race condition** between receiving the job and claiming it.

### The Race Condition

**Timeline of Events:**

```
Time 0: Job A sent to channel
Time 1: Worker 1 receives Job A from channel
Time 2: Worker 2 receives Job A from channel (if channel had multiple copies)
Time 3: Worker 1 calls ClaimJob(A) → Success
Time 4: Worker 2 calls ClaimJob(A) → Should fail, but what if it doesn't?
```

**Wait, that's not quite right...**

Actually, channels guarantee only one worker receives a job. But here's the **real problem**:

### The Real Problem: Store vs Channel Mismatch

**Scenario:**
1. Job A is created and stored in the store
2. Job A is sent to the channel
3. Worker 1 receives Job A from channel
4. Worker 2 also receives Job A from channel (if there were duplicates)
5. Both workers try to process Job A

**But channels prevent duplicates, so what's the issue?**

The issue is **not** channel duplicates. The issue is ensuring that when a worker receives a job, it can **atomically claim it** before another worker does.

### The Actual Race Condition

**Without ClaimJob:**

```go
// Worker 1
job := <-jobQueue  // Receives Job A
// ... time passes ...
job.Status = StatusProcessing  // Worker 1 updates status

// Worker 2 (at the same time)
job := <-jobQueue  // Also receives Job A (if channel had it)
job.Status = StatusProcessing  // Worker 2 also updates status
// Both workers process the same job!
```

**Problem:** Multiple workers can receive the same job reference and both try to process it.

**But wait...** Channels only deliver each job once. So this shouldn't happen, right?

**Actually, the real issue is different:**

### The Real Issue: Job State in Store

The real problem is ensuring that when a worker picks up a job, it can **atomically claim it in the store** before another worker does.

**Scenario:**
1. Job A is in store with status `pending`
2. Job A is sent to channel
3. Worker 1 receives Job A from channel
4. Worker 2 receives Job A from channel (if somehow it was sent twice, or there's a bug)
5. Both workers call `ClaimJob(A)` at nearly the same time
6. **Without proper locking:** Both might succeed!

---

## Why Duplicates Happen

### Reason 1: Channel Doesn't Prevent Store Races

Channels prevent **channel-level** duplicates (each job is delivered once from the channel), but they don't prevent **store-level** races.

**The Gap:**
- Channel delivers job to one worker ✅
- Worker must claim job in store ✅
- **But:** Between receiving and claiming, another worker could claim it first

### Reason 2: Multiple Workers, Same Job Reference

If the same job is somehow sent to the channel multiple times (bug in code), multiple workers could receive it.

### Reason 3: Store State Not Atomic

Without `ClaimJob`, updating job status is not atomic:

```go
// Worker 1
job := <-jobQueue
job.Status = StatusProcessing  // Not atomic!
store.UpdateJob(job)

// Worker 2 (at same time)
job := <-jobQueue  // Same job if sent twice
job.Status = StatusProcessing  // Also updates!
store.UpdateJob(job)
```

**Problem:** Both workers think they own the job.

---

## The ClaimJob Pattern

### What is ClaimJob?

`ClaimJob` is an **atomic operation** that:
1. Checks if job exists and is `pending`
2. If yes, atomically changes status to `processing`
3. Returns `true` if claimed, `false` if already claimed

### Our Implementation

```go
func (s *InMemoryJobStore) ClaimJob(ctx context.Context, jobID string) (bool, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    job, ok := s.jobs[jobID]
    if !ok || job.Status != domain.StatusPending {
        return false, nil  // Job doesn't exist or not pending
    }

    job.Status = domain.StatusProcessing
    s.jobs[jobID] = job

    return true, nil  // Successfully claimed
}
```

### How Workers Use It

```go
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case job, ok := <-w.jobQueue:
            if !ok {
                return
            }
            
            // Try to claim the job
            claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
            if err != nil {
                log.Printf("Error claiming job: %v", err)
                continue  // Skip this job
            }

            if !claimed {
                log.Printf("Job %s not claimed (already claimed by another worker)", job.ID)
                continue  // Another worker got it first
            }

            // We successfully claimed it, now process
            w.processJob(ctx, job)
        }
    }
}
```

---

## How ClaimJob Works

### Step-by-Step Execution

**Scenario:** Two workers both receive Job A (hypothetically)

**Worker 1 Timeline:**
```
Time 0: Receives Job A from channel
Time 1: Calls ClaimJob("job-a")
Time 2: Acquires mutex lock
Time 3: Checks: job.Status == "pending" → Yes
Time 4: Updates: job.Status = "processing"
Time 5: Releases mutex lock
Time 6: Returns: claimed = true
Time 7: Processes job
```

**Worker 2 Timeline (at the same time):**
```
Time 0: Receives Job A from channel (if somehow duplicated)
Time 1: Calls ClaimJob("job-a")
Time 2: Tries to acquire mutex lock → BLOCKED (Worker 1 has it)
Time 3: Waits...
Time 4: Mutex lock acquired (Worker 1 released it)
Time 5: Checks: job.Status == "pending" → No! It's "processing"
Time 6: Returns: claimed = false
Time 7: Skips job (continues to next)
```

### The Mutex Protection

```go
s.mu.Lock()  // Only one worker can execute this code at a time
defer s.mu.Unlock()

// This entire block is atomic
job, ok := s.jobs[jobID]
if !ok || job.Status != domain.StatusPending {
    return false, nil
}
job.Status = domain.StatusProcessing
s.jobs[jobID] = job
return true, nil
```

**Key Point:** The mutex ensures that only **one worker** can check and update the job status at a time. This makes the operation **atomic**.

### Why This Prevents Duplicates

1. **Atomic check-and-set:** Check status and update in one atomic operation
2. **Mutex serializes access:** Only one worker can claim at a time
3. **Idempotent:** Calling ClaimJob multiple times is safe (second call returns false)

---

## Race Condition Prevention

### Without ClaimJob (Vulnerable)

```go
// Worker 1
job := <-jobQueue
if job.Status == "pending" {  // Check
    job.Status = "processing"  // Update (not atomic!)
    store.UpdateJob(job)
    processJob(job)
}

// Worker 2 (at same time)
job := <-jobQueue  // Same job if bug
if job.Status == "pending" {  // Also passes!
    job.Status = "processing"  // Also updates!
    store.UpdateJob(job)
    processJob(job)  // Duplicate processing!
}
```

**Problem:** Check and update are **separate operations**, not atomic.

### With ClaimJob (Safe)

```go
// Worker 1
job := <-jobQueue
claimed, _ := store.ClaimJob(job.ID)  // Atomic check-and-set
if claimed {
    processJob(job)  // Only processes if claimed
}

// Worker 2 (at same time)
job := <-jobQueue  // Same job if bug
claimed, _ := store.ClaimJob(job.ID)  // Atomic check-and-set
if claimed {
    processJob(job)  // This won't execute (claimed = false)
}
```

**Solution:** Check and update are **one atomic operation**.

---

## Store as Source of Truth

### The Principle

**The store is the single source of truth for job state.**

- Channel is just a **notification mechanism**
- Store is the **authoritative state**
- Workers must **claim in store** before processing

### Why This Matters

**Channel vs Store:**

- **Channel:** Fast delivery, but not authoritative
- **Store:** Authoritative state, but requires locking

**The Pattern:**
1. Job created → Stored in store (status: `pending`)
2. Job sent to channel → Notification to workers
3. Worker receives from channel → Gets notification
4. Worker claims in store → Atomic check-and-set
5. If claimed → Process job
6. If not claimed → Skip (another worker got it)

### Visual Flow

```
Create Job
    ↓
Store (status: pending) ← Source of Truth
    ↓
Send to Channel (notification)
    ↓
Worker receives notification
    ↓
Claim in Store (atomic) ← Check-and-set
    ↓
If claimed → Process
If not claimed → Skip
```

---

## Common Mistakes

### Mistake 1: Not Using ClaimJob

```go
// ❌ BAD: No atomic claim
job := <-jobQueue
job.Status = StatusProcessing
store.UpdateJob(job)
processJob(job)
```

**Problem:** Race condition possible.

**Fix:** Use ClaimJob
```go
// ✅ GOOD: Atomic claim
job := <-jobQueue
claimed, _ := store.ClaimJob(job.ID)
if claimed {
    processJob(job)
}
```

### Mistake 2: Processing Without Checking Claim Result

```go
// ❌ BAD: Ignores claim result
job := <-jobQueue
store.ClaimJob(job.ID)  // Result ignored!
processJob(job)  // Always processes, even if not claimed!
```

**Fix:** Check the result
```go
// ✅ GOOD: Checks claim result
job := <-jobQueue
claimed, _ := store.ClaimJob(job.ID)
if !claimed {
    return  // Skip if not claimed
}
processJob(job)
```

### Mistake 3: Assuming Channel Prevents All Duplicates

```go
// ❌ BAD: Assumes channel is enough
job := <-jobQueue  // "Channels prevent duplicates, so I'm safe"
processJob(job)  // But what if job was sent twice due to bug?
```

**Fix:** Always claim in store
```go
// ✅ GOOD: Store is source of truth
job := <-jobQueue
claimed, _ := store.ClaimJob(job.ID)
if claimed {
    processJob(job)
}
```

### Mistake 4: Not Handling Claim Errors

```go
// ❌ BAD: Ignores errors
claimed, _ := store.ClaimJob(job.ID)
```

**Fix:** Handle errors
```go
// ✅ GOOD: Handles errors
claimed, err := store.ClaimJob(job.ID)
if err != nil {
    log.Printf("Error claiming job: %v", err)
    continue
}
if !claimed {
    continue
}
```

---

## Key Takeaways

1. **ClaimJob** = Atomic check-and-set operation
2. **Store is source of truth** = Channel is just notification
3. **Mutex protects** = Only one worker can claim at a time
4. **Always check claim result** = Don't process if not claimed
5. **Idempotent** = Safe to call ClaimJob multiple times
6. **Prevents duplicates** = Ensures each job processed exactly once

---

## Real-World Analogy

Think of a ticket system:

- **Channel** = Announcement: "Ticket available!"
- **Store** = Actual ticket inventory
- **ClaimJob** = Actually taking the ticket (atomic)
- **Multiple workers** = Multiple people trying to get the ticket

Even if multiple people hear the announcement, only one can actually take the ticket (atomic operation). The others see it's already taken.

---

## Next Steps

- Read [Configuration Management](./03-configuration-management.md) to see how we make the system configurable
- Read [Proper Shutdown Order](./04-proper-shutdown-order.md) to learn how to stop workers cleanly
- Read [WaitGroup with Multiple Goroutines](./05-waitgroup-multiple-goroutines.md) to understand how we track all workers

