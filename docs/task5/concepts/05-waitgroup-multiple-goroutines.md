# WaitGroup with Multiple Goroutines

## Table of Contents

1. [Why WaitGroup for Multiple Goroutines?](#why-waitgroup-for-multiple-goroutines)
2. [WaitGroup Basics Recap](#waitgroup-basics-recap)
3. [Tracking Multiple Workers](#tracking-multiple-workers)
4. [The Add-Done-Wait Pattern](#the-add-done-wait-pattern)
5. [Common Mistakes](#common-mistakes)

---

## Why WaitGroup for Multiple Goroutines?

### The Problem

In Task 4, we had **one worker**:

```go
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(workerCtx)
}()
wg.Wait()  // Wait for 1 worker
```

**Simple:** One goroutine, one `Add(1)`, one `Done()`.

### The Challenge in Task 5

Now we have **multiple workers** (e.g., 10 workers):

```go
for i := 0; i < config.WorkerCount; i++ {
    // Create 10 workers
}
```

**Question:** How do we wait for **all 10 workers** to finish?

**Answer:** WaitGroup tracks multiple goroutines!

---

## WaitGroup Basics Recap

### What is WaitGroup?

`sync.WaitGroup` is a counter that tracks how many goroutines are running.

**Operations (Traditional):**

1. `Add(n)` - Add `n` to the counter
2. `Done()` - Subtract 1 from the counter
3. `Wait()` - Block until counter reaches 0

**Modern Method (Go 1.21+):**

- `Go(func())` - Automatically calls `Add(1)`, starts goroutine, and calls `Done()` when finished

### Single Goroutine Pattern

**Traditional Pattern (Go < 1.21):**

```go
var wg sync.WaitGroup

wg.Add(1)  // "I'm about to start 1 goroutine"
go func() {
    defer wg.Done()  // "I'm finished" (subtracts 1)
    doWork()
}()
wg.Wait()  // "Wait until counter is 0"
```

**Modern Pattern (Go 1.21+):**

```go
var wg sync.WaitGroup

wg.Go(func() {
    doWork()
})
wg.Wait()  // "Wait until counter is 0"
```

**Flow (Modern):**

1. Counter = 0
2. `Go()` → Automatically `Add(1)` → Counter = 1
3. Goroutine starts
4. Goroutine finishes → Automatically `Done()` → Counter = 0
5. `Wait()` unblocks (counter is 0)

**Benefits of `wg.Go()`:**

- Cleaner code (no manual `Add(1)` or `defer Done()`)
- Less error-prone (can't forget `Done()`)
- More readable

---

## Tracking Multiple Workers

### Our Implementation (Modern Pattern - Go 1.21+)

```go
var wg sync.WaitGroup

for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}

// Later, during shutdown:
workerCancel()
wg.Wait()  // Wait for ALL workers to finish
```

**Note:** `wg.Go()` automatically handles `Add(1)` and `Done()`, making the code much cleaner!

### Step-by-Step

**Step 1: Create WaitGroup**

```go
var wg sync.WaitGroup
```

- Counter starts at 0

**Step 2: Loop Through Workers**

```go
for i := 0; i < config.WorkerCount; i++ {
    // Create worker i
}
```

- If `WorkerCount = 10`, loop runs 10 times

**Step 3: Create Worker Instance**

```go
worker := worker.NewWorker(i, jobStore, jobQueue)
```

- Creates worker with ID `i`
- Worker is created before the closure, so it captures the correct instance

**Step 4: Start Worker with WaitGroup (Modern - Go 1.21+)**

```go
wg.Go(func() {
    worker.Start(workerCtx)
})
```

- `wg.Go()` automatically calls `wg.Add(1)` before starting
- Starts the goroutine
- Automatically calls `wg.Done()` when goroutine finishes
- Much cleaner than the old pattern!

**Step 5: Wait for All Workers**

```go
wg.Wait()
```

- Blocks until counter reaches 0
- All 10 workers must call `Done()` before this unblocks

### Visual Timeline

```
Time 0:  Counter = 0
Time 1:  wg.Add(1) → Counter = 1 (Worker 1)
Time 2:  wg.Add(1) → Counter = 2 (Worker 2)
Time 3:  wg.Add(1) → Counter = 3 (Worker 3)
...
Time 10: wg.Add(1) → Counter = 10 (Worker 10)

All workers running...

Time 50: Worker 1 finishes → wg.Done() → Counter = 9
Time 51: Worker 2 finishes → wg.Done() → Counter = 8
...
Time 60: Worker 10 finishes → wg.Done() → Counter = 0
Time 61: wg.Wait() unblocks ✅
```

---

## The Modern Go Pattern (Go 1.21+)

### The Pattern

**Modern Pattern (Recommended):**

```go
var wg sync.WaitGroup

// Use wg.Go() - automatically handles Add(1) and Done()
for i := 0; i < N; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}

// Wait after all started
wg.Wait()
```

**Traditional Pattern (Go < 1.21 or for reference):**

```go
var wg sync.WaitGroup

// Manual Add and Done
for i := 0; i < N; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()  // Always defer Done()
        doWork(id)
    }(i)
}

// Wait after all started
wg.Wait()
```

### Why the Modern Pattern Works

1. **`wg.Go()` handles everything** - Automatically calls `Add(1)` and `Done()`
2. **Less error-prone** - Can't forget `Done()` or call `Add()` incorrectly
3. **Cleaner code** - No need for `defer wg.Done()`
4. **Wait after all started** - All goroutines are tracked

### Critical Rules (Modern Pattern)

**Rule 1: Use wg.Go() for each goroutine**

```go
// ✅ GOOD (Go 1.21+)
for i := 0; i < 10; i++ {
    wg.Go(func() { ... })
}

// ❌ BAD: Mixing patterns
for i := 0; i < 10; i++ {
    wg.Add(1)
    wg.Go(func() { ... })  // Double counting!
}
```

**Rule 2: Wait after all goroutines started**

```go
// ✅ GOOD
for i := 0; i < 10; i++ {
    wg.Go(func() { ... })
}
wg.Wait()  // After loop

// ❌ BAD: Wait inside loop (waits for each one)
for i := 0; i < 10; i++ {
    wg.Go(func() { ... })
    wg.Wait()  // Waits after each worker!
}
```

**Rule 3: Capture loop variables correctly**

```go
// ✅ GOOD: Worker created before closure
for i := 0; i < 10; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)  // Captures worker instance
    })
}

// ❌ BAD: Capturing loop variable directly
for i := 0; i < 10; i++ {
    wg.Go(func() {
        worker := worker.NewWorker(i, jobStore, jobQueue)  // i might be 10!
        worker.Start(workerCtx)
    })
}
```

---

## Common Mistakes

### Mistake 1: Using wg.Go() Outside the Loop

```go
// ❌ BAD: Only tracks 1 worker
wg.Go(func() {
    for i := 0; i < 10; i++ {
        doWork()  // All work in one goroutine!
    }
})
wg.Wait()  // Only waits for 1 goroutine!
```

**Problem:** All work runs in a single goroutine, not concurrently.

**Fix:** Use wg.Go() inside loop

```go
// ✅ GOOD: Each worker gets its own goroutine
for i := 0; i < 10; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}
```

### Mistake 2: Mixing wg.Go() with Manual Add/Done

```go
// ❌ BAD: Double counting!
for i := 0; i < 10; i++ {
    wg.Add(1)  // Manual Add
    wg.Go(func() {
        // wg.Go() also calls Add(1) internally!
        doWork()
    })
}
// Counter is 20, not 10!
```

**Fix:** Use wg.Go() alone (it handles Add/Done automatically)

```go
// ✅ GOOD: wg.Go() handles everything
for i := 0; i < 10; i++ {
    wg.Go(func() {
        doWork()
    })
}
```

### Mistake 3: Waiting Inside Loop

```go
// ❌ BAD: Waits for each worker sequentially
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        doWork()
    }()
    wg.Wait()  // Blocks here, waits for this worker
}
```

**Problem:** Workers run one at a time, not concurrently!

**Fix:** Wait after loop

```go
// ✅ GOOD: All workers run concurrently
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        doWork()
    }()
}
wg.Wait()  // Waits for all after they're all started
```

### Mistake 4: Not Using wg.Go() (Using Old Pattern)

```go
// ❌ BAD: Old pattern, more verbose
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        doWork()
    }()
}
```

**Fix:** Use modern wg.Go() (Go 1.21+)

```go
// ✅ GOOD: Modern pattern, cleaner
for i := 0; i < 10; i++ {
    wg.Go(func() {
        doWork()
    })
}
```

### Mistake 5: Forgetting to Wait

```go
// ❌ BAD: No wait, main() might exit
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        doWork()
    }()
}
// main() continues, might exit before workers finish
```

**Fix:** Always wait

```go
// ✅ GOOD: Waits for all workers
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        doWork()
    }()
}
wg.Wait()  // Wait for all
```

---

## Key Takeaways

1. **WaitGroup tracks multiple goroutines** - Counter-based
2. **Modern pattern (Go 1.21+):** Use `wg.Go()` - Automatically handles `Add(1)` and `Done()`
3. **Traditional pattern:** `Add(1)` before starting, `defer Done()` when finished
4. **Wait() after all started** - Blocks until all done
5. **wg.Go() inside loop** - For multiple workers
6. **Don't mix patterns** - Use `wg.Go()` OR manual `Add/Done`, not both

---

## Real-World Analogy

Think of a group project:

- **Add(1)** = "I'm assigning you a task"
- **Done()** = "I finished my task"
- **Wait()** = "Wait until everyone finishes"

If you have 10 people:

- Assign 10 tasks (10 `Add(1)`)
- Each person finishes (10 `Done()`)
- Wait until all 10 are done (`Wait()`)

---

## Next Steps

- Review [Worker Pools](./01-worker-pools.md) to see how WaitGroup fits into the worker pool pattern
- Review [Proper Shutdown Order](./04-proper-shutdown-order.md) to see how WaitGroup is used during shutdown
