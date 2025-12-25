# Context in Storage Layer

## Table of Contents

1. [Why Context in Storage?](#why-context-in-storage)
2. [Context for Cancellation](#context-for-cancellation)
3. [Our Implementation](#our-implementation)
4. [Context Check Before Lock](#context-check-before-lock)
5. [Context Check After Lock](#context-check-after-lock)
6. [When Context Cancellation Matters](#when-context-cancellation-matters)
7. [Best Practices](#best-practices)
8. [Common Mistakes](#common-mistakes)

---

## Why Context in Storage?

### The Question

**Why do storage methods accept `context.Context`?**

```go
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error
func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error)
```

### The Reasons

**1. Cancellation Propagation**

- Client disconnects → Request context canceled
- Storage operations should respect cancellation
- Don't do work if client is gone

**2. Timeout Support**

- Future: Database operations might timeout
- Context can carry timeout information
- Storage can respect timeouts

**3. Consistency**

- All layers use context (HTTP → Domain → Storage)
- Consistent API design
- Future-proof for async operations

**4. Request Tracing (Future)**

- Context can carry request IDs
- Storage can log with request context
- Better debugging and tracing

---

## Context for Cancellation

### The Scenario

**What happens when a client disconnects?**

```
Time 0ms:  Client sends POST /jobs
Time 1ms:  Handler starts processing
Time 2ms:  Handler calls store.CreateJob()
Time 3ms:  Store acquires lock
Time 4ms:  Client disconnects (closes connection)
Time 5ms:  Request context is canceled
Time 6ms:  Store is still holding lock, doing work
```

**Problem:** Store continues working even though client is gone!

### The Solution: Check Context

```go
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // Check if context is canceled BEFORE doing work
    select {
    case <-ctx.Done():
        return ctx.Err()  // Client disconnected, stop immediately
    default:
        // Context is still valid, continue
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    s.jobs[job.ID] = *job
    return nil
}
```

**Benefit:** If client disconnects, we stop immediately (don't waste resources)

---

## Our Implementation

### CreateJob with Context

```go
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // Check context BEFORE acquiring lock
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    s.jobs[job.ID] = *job
    return nil
}
```

### GetJobs with Context

```go
func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    // Check context BEFORE acquiring lock
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    s.mu.RLock()
    defer s.mu.RUnlock()

    jobs := make([]domain.Job, 0, len(s.jobs))
    for _, job := range s.jobs {
        jobs = append(jobs, job)
    }

    sort.Slice(jobs, func(i, j int) bool {
        return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
    })

    return jobs, nil
}
```

### The select Statement

**What is `select`?**

- `select` waits for one of multiple channel operations
- `case <-ctx.Done()`: If context is canceled, this channel is closed
- `default`: If context is not canceled, continue immediately

**How it works:**

```go
select {
case <-ctx.Done():
    // Context was canceled → return error
    return ctx.Err()
default:
    // Context is still valid → continue
}
```

**Non-blocking:** `default` case makes this non-blocking (doesn't wait)

---

## Context Check Before Lock

### Why Check Before Lock?

**The Problem:**

```go
// ❌ BAD: Check context AFTER acquiring lock
func (s *Store) CreateJob(ctx context.Context, job Job) error {
    s.mu.Lock()  // Acquire lock first

    select {
    case <-ctx.Done():
        s.mu.Unlock()  // Must unlock!
        return ctx.Err()
    default:
    }

    // Do work
    s.mu.Unlock()
}
```

**Problems:**

1. Lock acquired even if context is canceled
2. Other goroutines blocked unnecessarily
3. Must remember to unlock before returning

**The Solution:**

```go
// ✅ GOOD: Check context BEFORE acquiring lock
func (s *Store) CreateJob(ctx context.Context, job Job) error {
    // Check first
    select {
    case <-ctx.Done():
        return ctx.Err()  // Return immediately, no lock needed
    default:
    }

    // Then acquire lock
    s.mu.Lock()
    defer s.mu.Unlock()

    // Do work
}
```

**Benefits:**

1. Don't acquire lock if canceled
2. Don't block other goroutines
3. Simpler code (no unlock before return)

### Our Pattern

```go
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // 1. Check context first (before lock)
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // 2. Then acquire lock
    s.mu.Lock()
    defer s.mu.Unlock()

    // 3. Do work (quick operation)
    s.jobs[job.ID] = *job
    return nil
}
```

**Why this works:**

- Operations are quick (just map assignment)
- No need to check context again after lock
- Lock is held for minimal time

---

## Context Check After Lock

### When to Check Again?

**For long operations:**

```go
func (s *Store) LongOperation(ctx context.Context) error {
    // Check before lock
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    // Check again after lock (for long operations)
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Long operation here
    for i := 0; i < 1000000; i++ {
        // Do work
        // Check context periodically
        if i%1000 == 0 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }
        }
    }

    return nil
}
```

**When to use:**

- Operations that take time
- Iterations or loops
- Network calls (future)

**Our case:**

- Operations are very quick (map operations)
- No need for additional checks
- Lock is held for microseconds

---

## When Context Cancellation Matters

### Scenario 1: Client Disconnects

```
Client → HTTP Request → Handler → Store.CreateJob()
                                    ↓
                            Client disconnects
                                    ↓
                            Context canceled
                                    ↓
                            Store checks context
                                    ↓
                            Returns immediately
```

**Benefit:** Don't waste resources on disconnected clients

### Scenario 2: Request Timeout

```go
// Future: HTTP server might set timeout
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()

store.CreateJob(ctx, job)  // Will respect 5-second timeout
```

**Benefit:** Operations don't hang forever

### Scenario 3: Server Shutdown

```go
// During graceful shutdown
ctx, cancel := context.WithCancel(context.Background())
cancel()  // Signal shutdown

// All operations check context
store.CreateJob(ctx, job)  // Returns immediately
```

**Benefit:** Fast shutdown, don't start new work

---

## Best Practices

### 1. Always Accept Context

```go
// ✅ GOOD: Context as first parameter
func (s *Store) CreateJob(ctx context.Context, job Job) error
```

**Convention:** Context is always the first parameter

### 2. Check Before Expensive Operations

```go
// ✅ GOOD: Check before lock
select {
case <-ctx.Done():
    return ctx.Err()
default:
}
s.mu.Lock()
```

### 3. Check Periodically in Loops

```go
// ✅ GOOD: Check in loops
for i := 0; i < 1000000; i++ {
    if i%1000 == 0 {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
    }
    // Do work
}
```

### 4. Return Context Errors

```go
// ✅ GOOD: Return ctx.Err()
select {
case <-ctx.Done():
    return ctx.Err()  // Preserve cancellation reason
default:
}
```

### 5. Don't Ignore Context

```go
// ❌ BAD: Ignoring context
func (s *Store) CreateJob(ctx context.Context, job Job) error {
    // No context check!
    s.mu.Lock()
    // ...
}
```

---

## Common Mistakes

### Mistake 1: Ignoring Context

```go
// ❌ BAD: Context parameter but not used
func (s *Store) CreateJob(ctx context.Context, job Job) error {
    // No context check!
    s.mu.Lock()
    s.jobs[id] = job
    s.mu.Unlock()
    return nil
}
```

**Fix:** Check context

```go
// ✅ GOOD: Check context
func (s *Store) CreateJob(ctx context.Context, job Job) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // ...
}
```

### Mistake 2: Checking After Lock

```go
// ❌ BAD: Check after lock (blocks others)
func (s *Store) CreateJob(ctx context.Context, job Job) error {
    s.mu.Lock()
    select {
    case <-ctx.Done():
        s.mu.Unlock()  // Must unlock!
        return ctx.Err()
    default:
    }
    // ...
}
```

**Fix:** Check before lock

```go
// ✅ GOOD: Check before lock
func (s *Store) CreateJob(ctx context.Context, job Job) error {
    select {
    case <-ctx.Done():
        return ctx.Err()  // No lock needed
    default:
    }
    s.mu.Lock()
    // ...
}
```

### Mistake 3: Not Returning Context Error

```go
// ❌ BAD: Generic error instead of ctx.Err()
select {
case <-ctx.Done():
    return errors.New("cancelled")  // Loses context information
default:
}
```

**Fix:** Return ctx.Err()

```go
// ✅ GOOD: Preserve context error
select {
case <-ctx.Done():
    return ctx.Err()  // context.Canceled or context.DeadlineExceeded
default:
}
```

### Mistake 4: Blocking on Context Check

```go
// ❌ BAD: Blocking select (no default)
select {
case <-ctx.Done():
    return ctx.Err()
// Missing default! Blocks forever if context not canceled
}
```

**Fix:** Add default case

```go
// ✅ GOOD: Non-blocking with default
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue immediately
}
```

---

## Key Takeaways

1. **Context in storage** = Allows cancellation propagation
2. **Check before lock** = Don't acquire lock if canceled
3. **select statement** = Non-blocking context check
4. **ctx.Err()** = Preserves cancellation reason
5. **Always accept context** = Future-proof and consistent
6. **Check periodically** = For long operations
7. **Respect cancellation** = Don't waste resources

---

## The Go Philosophy

Go's context package provides **cancellation propagation**:

- ✅ Context flows through all layers
- ✅ Cancellation is cooperative (code must check)
- ✅ Simple pattern (select with default)
- ✅ No magic, just clear cancellation

**Go's approach:**

- Explicit cancellation checks
- Cooperative cancellation (not forced)
- Context as first parameter (convention)
- Simple and clear

---

## Next Steps

- Read [Context in Handlers](../task1/concepts/08-context-in-handlers.md) to see how context flows from HTTP
- Read [Concurrency Safety](./04-concurrency-safety.md) to understand mutexes
- Read [Interface Design](./07-interface-design.md) to see the full store interface
