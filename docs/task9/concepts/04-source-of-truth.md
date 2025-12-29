# Source of Truth Design

## Table of Contents

1. [What is Source of Truth?](#what-is-source-of-truth)
2. [Why Store is Source of Truth](#why-store-is-source-of-truth)
3. [Queue as Delivery Mechanism](#queue-as-delivery-mechanism)
4. [Recovery Implications](#recovery-implications)
5. [Common Mistakes](#common-mistakes)

---

## What is Source of Truth?

### The Simple Answer

**Source of truth** is the authoritative location where data lives. It's the single place that defines what the data actually is.

### The Principle

**In our system:**
- **Store is the source of truth** - All job state lives here
- **Queue is a delivery mechanism** - Just a way to notify workers
- **Workers never scan store** - They only process from queue

### Why This Matters

**Without clear source of truth:**
- Data can be inconsistent
- Multiple places claim to be authoritative
- Recovery becomes complex
- Bugs are hard to track

**With clear source of truth:**
- Single authoritative location
- Clear ownership
- Simple recovery
- Predictable behavior

---

## Why Store is Source of Truth

### The Store's Role

**Store contains:**
- All job data (ID, type, status, payload, etc.)
- Job state (pending, processing, completed, failed)
- Job metadata (attempts, errors, timestamps)
- State transition rules

**Store enforces:**
- State machine rules
- Data consistency
- Atomic operations
- Concurrency safety

### Why Not Queue?

**Queue characteristics:**
- Temporary buffer
- Can be empty
- Can be full
- Just job IDs (not full data)
- Lost on restart

**Problems if queue is source of truth:**
- Data lost on restart
- No persistence
- No state validation
- No history

### Why Not Workers?

**Workers characteristics:**
- Process jobs
- Don't store data
- Ephemeral (can restart)
- No persistence

**Problems if workers are source of truth:**
- Data lost on restart
- No central location
- Hard to query
- No consistency

### The Design Decision

**Store as source of truth because:**
1. **Persistence** - Can be persisted (future)
2. **Authority** - Enforces rules
3. **Queryability** - Can query all jobs
4. **Consistency** - Single location
5. **Recovery** - Can recover from store

---

## Queue as Delivery Mechanism

### The Queue's Role

**Queue contains:**
- Job IDs (not full job data)
- Temporary buffer
- Notification mechanism

**Queue purpose:**
- Notify workers about work
- Decouple producers from consumers
- Provide backpressure mechanism
- Enable concurrent processing

### Why Queue is Not Source of Truth

**Queue limitations:**
- Temporary (lost on restart)
- Limited capacity
- Just IDs (not full data)
- No state information
- No validation

**Queue is ephemeral:**
- Created on startup
- Filled with job IDs
- Drained by workers
- Lost on restart

### The Relationship

**Store → Queue:**
- Store has all job data
- Queue has job IDs from store
- Queue is populated from store

**Queue → Workers:**
- Workers receive job IDs from queue
- Workers fetch full data from store
- Workers update store (not queue)

**Workers → Store:**
- Workers claim jobs from store
- Workers update job status in store
- Store is authoritative

---

## Recovery Implications

### Recovery Always Starts from Store

**Recovery process:**
1. **Read from store** - Get processing and pending jobs
2. **Update store** - Move processing → pending
3. **Populate queue** - Re-enqueue job IDs

**Key point:** Recovery never reads from queue (queue is empty on startup anyway).

### Why This Works

**On startup:**
- Store: Contains all job state (if persisted)
- Queue: Empty (fresh channel)
- Workers: Not started yet

**Recovery:**
- Reads processing jobs from store
- Moves them to pending in store
- Reads pending jobs from store
- Re-enqueues their IDs

**Result:** Queue is populated from store, not the other way around.

### The Flow

```
Startup:
  Store (source of truth)
    ↓
  Recovery reads from store
    ↓
  Recovery updates store
    ↓
  Recovery populates queue from store
    ↓
  Workers process from queue
    ↓
  Workers update store
```

**Key insight:** Store is always the starting point.

---

## Common Mistakes

### Mistake 1: Queue as Source of Truth

**❌ BAD:**
```go
// Recovery reads from queue
for jobID := range jobQueue {
    // Recover from queue
}
```

**Problem:**
- Queue is empty on startup
- No data to recover
- Queue is ephemeral

**✅ GOOD:**
```go
// Recovery reads from store
processingJobs, _ := jobStore.GetProcessingJobs(ctx)
pendingJobs, _ := jobStore.GetPendingJobs(ctx)
```

**Benefit:** Store has all the data.

### Mistake 2: Workers Scanning Store

**❌ BAD:**
```go
// Worker scans store for work
func (w *Worker) Start(ctx context.Context) {
    for {
        jobs, _ := jobStore.GetPendingJobs(ctx)
        for _, job := range jobs {
            // Process job
        }
    }
}
```

**Problem:**
- Workers compete for same jobs
- No coordination
- Race conditions
- Inefficient

**✅ GOOD:**
```go
// Worker receives from queue
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case jobID := <-w.jobQueue:
            job, _ := jobStore.ClaimJob(ctx, jobID)
            // Process job
        }
    }
}
```

**Benefit:** Queue coordinates work distribution.

### Mistake 3: Updating Queue Instead of Store

**❌ BAD:**
```go
// Worker updates queue
func (w *Worker) processJob(job *domain.Job) {
    // Process job
    job.Status = domain.StatusCompleted
    w.jobQueue <- job  // ❌ Updating queue!
}
```

**Problem:**
- Queue is not source of truth
- State not persisted
- Lost on restart
- Inconsistent

**✅ GOOD:**
```go
// Worker updates store
func (w *Worker) processJob(job *domain.Job) {
    // Process job
    jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
    // ✅ Store is source of truth
}
```

**Benefit:** State persisted and consistent.

### Mistake 4: Recovery from Queue

**❌ BAD:**
```go
// Recovery tries to recover from queue
func RecoverJobs(jobQueue chan string) {
    // Queue is empty on startup!
    for jobID := range jobQueue {
        // Nothing to recover
    }
}
```

**Problem:**
- Queue is empty on startup
- No data to recover
- Recovery does nothing

**✅ GOOD:**
```go
// Recovery from store
func RecoverJobs(jobStore store.JobStore, jobQueue chan string) {
    // Read from store
    processingJobs, _ := jobStore.GetProcessingJobs(ctx)
    pendingJobs, _ := jobStore.GetPendingJobs(ctx)
    
    // Update store
    // Populate queue from store
}
```

**Benefit:** Recovery actually works.

---

## Key Takeaways

1. **Store is source of truth** - All job state lives here
2. **Queue is delivery mechanism** - Just notifies workers
3. **Workers never scan store** - They process from queue
4. **Recovery starts from store** - Always read from store
5. **Clear ownership** - Store owns data, queue owns delivery

---

## Related Concepts

- [Startup Recovery](./01-startup-recovery.md) - Overall recovery process
- [Recovery Backpressure](./02-recovery-backpressure.md) - How recovery handles queue full
- [State Transitions](./03-state-transitions-recovery.md) - How recovery respects state machine

