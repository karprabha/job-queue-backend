# Go Concepts Explained - Task 6

This directory contains detailed explanations of Go concepts used in Task 6, written for beginners learning failure handling and retry logic in Go.

## 📚 Concepts Covered

### 1. [State Machine and Transitions](./01-state-machine-transitions.md)

- Why explicit state machines?
- What is a state machine?
- Our job state machine
- The canTransition function
- Why store enforces transitions (not workers)
- Invalid transition handling
- Common mistakes

### 2. [Failure Handling](./02-failure-handling.md)

- Why failure is a first-class concept
- The StatusFailed state
- LastError tracking
- Worker signals failure, store updates state
- Failure simulation
- Common mistakes

### 3. [Retry Logic and Attempt Tracking](./03-retry-logic-attempts.md)

- Why retry limits matter
- Attempts vs MaxRetries
- Preventing infinite retries
- When to retry vs permanent failure
- Atomic attempt increment
- Common mistakes

### 4. [The Sweeper Pattern](./04-sweeper-pattern.md)

- Why a sweeper?
- What is a sweeper?
- Periodic retry mechanism
- Sweeper vs worker responsibilities
- Sweeper lifecycle
- Common mistakes

### 5. [Atomic State Updates](./05-atomic-state-updates.md)

- Why atomic updates matter
- UpdateStatus method design
- Mutex protection
- Transition validation
- Error message tracking
- Common mistakes

### 6. [Channel Type Simplification](./06-channel-type-simplification.md)

- Why change from `*domain.Job` to `string`?
- Job ID vs full job object
- Store as source of truth
- Reduced memory usage
- Better separation of concerns
- Common mistakes

## 🎯 How to Use This

These documents are designed to be read **in order** if you're new to failure handling in Go. Each concept builds on previous ones.

**Recommended reading order:**

1. Start with [State Machine and Transitions](./01-state-machine-transitions.md) - Foundation for all state changes
2. Then [Failure Handling](./02-failure-handling.md) - How we handle failures
3. Then [Retry Logic and Attempt Tracking](./03-retry-logic-attempts.md) - How we prevent infinite retries
4. Then [The Sweeper Pattern](./04-sweeper-pattern.md) - How we implement retries
5. Then [Atomic State Updates](./05-atomic-state-updates.md) - How we ensure correctness
6. Finally [Channel Type Simplification](./06-channel-type-simplification.md) - Design improvement

Or read them as you encounter concepts in the code!

## 💡 Learning Approach

Each document:

- Explains **why** things exist (not just what they do)
- Breaks down code **line by line**
- Uses **analogies** and **mental models**
- Shows **common mistakes** to avoid
- Provides **real examples** from our codebase
- Explains **design decisions** and trade-offs

## 🔗 Related Resources

### Task 5 Concepts

- [Worker Pools](../task5/concepts/01-worker-pools.md)
- [Preventing Duplicate Processing](../task5/concepts/02-preventing-duplicate-processing.md)
- [Proper Shutdown Order](../task5/concepts/04-proper-shutdown-order.md)

### Task 4 Concepts

- [Goroutines for Workers](../task4/concepts/01-goroutines-for-workers.md)
- [Channels for Communication](../task4/concepts/02-channels-for-communication.md)
- [Worker Pattern](../task4/concepts/03-worker-pattern.md)

### Task 3 Concepts

- [Concurrency Safety](../task3/concepts/04-concurrency-safety.md)
- [RWMutex vs Mutex](../task3/concepts/05-rwmutex-vs-mutex.md)

### External Resources

- [Go Official Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Go Memory Model](https://go.dev/ref/mem)

## 📝 Key Concepts Summary

### State Machine

- Explicit state transitions prevent bugs
- Store enforces state rules (not workers)
- Invalid transitions are rejected
- State machine is the source of truth

### Failure Handling

- Failure is a first-class state
- LastError tracks why jobs failed
- Workers signal failure, store updates state
- Failed jobs are observable

### Retry Logic

- Attempts are tracked per job
- MaxRetries prevents infinite loops
- Retries only happen if attempts < maxRetries
- Permanent failure when limit reached

### Sweeper Pattern

- Periodic background process
- Moves failed jobs back to pending
- Enqueues pending jobs
- Separate from worker processing

### Atomic Updates

- UpdateStatus validates transitions
- Mutex protects state changes
- Error messages stored atomically
- All-or-nothing updates

### Channel Simplification

- Channels carry job IDs (strings), not full objects
- Store is source of truth
- Reduces memory usage
- Better separation of concerns

## 🎓 What You'll Learn

After reading these documents, you'll understand:

- ✅ How to design explicit state machines
- ✅ How to handle failures gracefully
- ✅ How to implement retry logic with limits
- ✅ How to use the sweeper pattern
- ✅ How to ensure atomic state updates
- ✅ How to simplify channel types
- ✅ Why store should enforce state rules
- ✅ How to prevent infinite retries

## 🚀 Next Steps

After Task 6, you'll be ready for:

- Exponential backoff strategies
- Dead-letter queues
- Metrics and observability
- Database persistence
- Advanced error handling
- Job prioritization
- Rate limiting

## ⚠️ Critical Bugs to Avoid

### 1. Workers Directly Mutating State

```go
// ❌ BAD: Worker mutates state directly
func (w *Worker) processJob(job *domain.Job) {
    job.Status = domain.StatusFailed  // Direct mutation!
}

// ✅ GOOD: Worker signals, store updates
func (w *Worker) processJob(job *domain.Job) {
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
}
```

### 2. Infinite Retries

```go
// ❌ BAD: No retry limit check
if job.Status == domain.StatusFailed {
    job.Status = domain.StatusPending  // Retry forever!
}

// ✅ GOOD: Check retry limit
if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
    job.Status = domain.StatusPending
}
```

### 3. Missing Transition Validation

```go
// ❌ BAD: No validation
job.Status = newStatus  // Could be invalid!

// ✅ GOOD: Validate transition
if !canTransition(job.Status, newStatus) {
    return errors.New("invalid transition")
}
job.Status = newStatus
```

### 4. Not Saving After Updating Fields

```go
// ❌ BAD: LastError not saved
job.Status = status
s.jobs[jobID] = job  // Save here
if lastError != nil {
    job.LastError = lastError  // Update but never save!
}

// ✅ GOOD: Update all fields, then save once
job.Status = status
if lastError != nil {
    job.LastError = lastError
}
s.jobs[jobID] = job  // Save after all updates
```

## 📝 Contributing

If you find something unclear or want to add explanations, feel free to update these documents!

