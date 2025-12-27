# Go Concepts Explained - Task 5

This directory contains detailed explanations of Go concepts used in Task 5, written for beginners learning Go concurrency at scale.

## 📚 Concepts Covered

### 1. [Worker Pools](./01-worker-pools.md)

- Why worker pools?
- What is a worker pool?
- Single worker vs worker pool
- Creating a worker pool
- The fan-out pattern
- Worker pool lifecycle
- Common mistakes (especially closure variable capture)

### 2. [Preventing Duplicate Processing](./02-preventing-duplicate-processing.md)

- The duplicate processing problem
- Why duplicates happen
- The ClaimJob pattern
- How ClaimJob works
- Race condition prevention
- Store as source of truth
- Common mistakes

### 3. [Configuration Management](./03-configuration-management.md)

- Why configuration management?
- Hardcoded vs configurable
- Our configuration approach
- Environment variables
- Configuration struct
- Default values
- Error handling in config

### 4. [Proper Shutdown Order](./04-proper-shutdown-order.md)

- Why shutdown order matters
- The shutdown problem
- Our shutdown sequence
- Step-by-step breakdown
- Why this order?
- The "send on closed channel" bug
- Common mistakes

### 5. [WaitGroup with Multiple Goroutines](./05-waitgroup-multiple-goroutines.md)

- Why WaitGroup for multiple goroutines?
- WaitGroup basics recap
- Tracking multiple workers
- The Add-Done-Wait pattern
- Common mistakes

## 🎯 How to Use This

These documents are designed to be read **in order** if you're new to Go concurrency at scale. Each concept builds on previous ones.

**Recommended reading order:**

1. Start with [Worker Pools](./01-worker-pools.md) - Foundation for multiple workers
2. Then [Preventing Duplicate Processing](./02-preventing-duplicate-processing.md) - How we ensure correctness
3. Then [Configuration Management](./03-configuration-management.md) - How we make it configurable
4. Then [Proper Shutdown Order](./04-proper-shutdown-order.md) - How to stop everything cleanly
5. Finally [WaitGroup with Multiple Goroutines](./05-waitgroup-multiple-goroutines.md) - How we track all workers

Or read them as you encounter concepts in the code!

## 💡 Learning Approach

Each document:

- Explains **why** things exist (not just what they do)
- Breaks down code **line by line**
- Uses **analogies** and **mental models**
- Shows **common mistakes** to avoid (especially the closure bug!)
- Provides **real examples** from our codebase
- Explains **design decisions** and trade-offs

## 🔗 Related Resources

### Task 4 Concepts

- [Goroutines for Workers](../task4/concepts/01-goroutines-for-workers.md)
- [Channels for Communication](../task4/concepts/02-channels-for-communication.md)
- [Worker Pattern](../task4/concepts/03-worker-pattern.md)
- [Graceful Shutdown](../task4/concepts/05-graceful-shutdown.md)
- [Atomic Operations](../task4/concepts/07-atomic-operations.md)

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

### Worker Pools

- Multiple workers processing jobs concurrently
- Fan-out pattern: one channel, multiple workers
- Shared queue for automatic load balancing
- More workers = faster processing (up to limits)

### Preventing Duplicates

- ClaimJob = atomic check-and-set operation
- Store is source of truth, channel is just notification
- Mutex protects critical sections
- Always check claim result before processing

### Configuration

- Environment variables for flexibility
- Default values for sensible fallbacks
- Config struct groups related settings
- Error handling for invalid config

### Shutdown Order

- Stop producers first (HTTP server)
- Close channel after server shutdown
- Stop consumers last (workers)
- Wrong order = panics or hangs

### WaitGroup with Multiple Goroutines

- **Modern (Go 1.21+):** Use `wg.Go()` - Automatically handles Add/Done
- **Traditional:** Add(1) for each goroutine, Done() when finished (always defer)
- Wait() after all started
- Don't mix patterns - use `wg.Go()` OR manual Add/Done, not both

## 🎓 What You'll Learn

After reading these documents, you'll understand:

- ✅ How to create and manage worker pools
- ✅ How to prevent duplicate processing with atomic operations
- ✅ How to make systems configurable
- ✅ How to shut down multiple components in the correct order
- ✅ How to track multiple goroutines with WaitGroup
- ✅ How to avoid the closure variable capture bug
- ✅ How to prevent "send on closed channel" panics

## 🚀 Next Steps

After Task 5, you'll be ready for:

- Failure handling and retries
- Idempotency patterns
- Advanced error handling
- Metrics and observability
- Database persistence
- Rate limiting
- Job prioritization

## ⚠️ Critical Bugs to Avoid

### 1. Closure Variable Capture

```go
// ❌ BAD: All workers get same ID
for i := 0; i < 10; i++ {
    go func() {
        worker.Start(workerCtx, i)  // i is 10 for all!
    }()
}

// ✅ GOOD: Pass as parameter
for i := 0; i < 10; i++ {
    go func(workerID int) {
        worker.Start(workerCtx, workerID)
    }(i)
}
```

### 2. Send on Closed Channel

```go
// ❌ BAD: Closes channel before server shuts down
close(jobQueue)
srv.Shutdown(ctx)  // Handler might try to send → panic!

// ✅ GOOD: Shutdown server first
srv.Shutdown(ctx)
close(jobQueue)
```

### 3. WaitGroup Add Outside Loop

```go
// ❌ BAD: Only tracks 1 worker
wg.Add(1)
for i := 0; i < 10; i++ {
    go func() { ... }()
}

// ✅ GOOD: Tracks all workers
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() { ... }()
}
```

## 📝 Contributing

If you find something unclear or want to add explanations, feel free to update these documents!
