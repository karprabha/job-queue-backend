# Understanding Encapsulation: Returning Copies

## Table of Contents

1. [What is Encapsulation?](#what-is-encapsulation)
2. [The Problem: Returning Pointers](#the-problem-returning-pointers)
3. [The Solution: Returning Copies](#the-solution-returning-copies)
4. [How Copying Works](#how-copying-works)
5. [Memory Implications](#memory-implications)
6. [When to Copy vs When to Share](#when-to-copy-vs-when-to-share)
7. [Common Mistakes](#common-mistakes)

---

## What is Encapsulation?

### Definition

**Encapsulation** = Hiding internal implementation details and preventing external code from directly accessing or modifying internal state.

### The Goal

**Internal state should be:**

- Protected from external modification
- Only accessible through controlled methods
- Not directly exposed

### Analogy

Think of a bank vault:

- **Internal state** = Money in vault
- **Public methods** = Withdraw, deposit (controlled access)
- **Direct access** = Breaking into vault (shouldn't be possible)

**Encapsulation prevents "breaking into the vault."**

---

## The Problem: Returning Pointers

### The Issue

**❌ Bad: Returns pointer to internal state**

```go
type InMemoryMetricStore struct {
    mu      sync.RWMutex
    metrics *domain.Metric  // Internal state
}

func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    return s.metrics  // Returns pointer to internal state!
}
```

### What Can Go Wrong

**External code can mutate internal state:**

```go
metrics, _ := store.GetMetrics(ctx)
metrics.TotalJobsCreated = 999999  // Mutated internal state!

// Now store's internal metrics are corrupted!
```

**Problems:**

1. **Breaks encapsulation** - External code can modify internal state
2. **Bypasses mutex** - Changes made without lock protection
3. **Race conditions** - Concurrent modifications possible
4. **Inconsistent state** - Internal state can be corrupted

### Real-World Example

```go
// Get metrics
metrics, _ := store.GetMetrics(ctx)
fmt.Println(metrics.TotalJobsCreated)  // Prints: 100

// Mutate (by accident or maliciously)
metrics.TotalJobsCreated = 0

// Get metrics again
metrics2, _ := store.GetMetrics(ctx)
fmt.Println(metrics2.TotalJobsCreated)  // Prints: 0 (corrupted!)
```

**Internal state was corrupted by external code.**

---

## The Solution: Returning Copies

### The Fix

**✅ Good: Returns copy**

```go
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Return a copy to prevent external mutation
    m := *s.metrics  // Copy the struct
    return &m, nil   // Return pointer to copy
}
```

### How It Works

**Step 1: Copy the struct**

```go
m := *s.metrics  // Dereference pointer, copy value
```

**What happens:**

- `s.metrics` is a pointer to `domain.Metric`
- `*s.metrics` dereferences it, getting the struct value
- `m := *s.metrics` creates a copy of that struct value

**Step 2: Return pointer to copy**

```go
return &m, nil  // Return pointer to the copy
```

**What happens:**

- `&m` takes address of the copy
- Returns pointer to the copy (not original)

### Protection

**Now external code can't mutate internal state:**

```go
metrics, _ := store.GetMetrics(ctx)
metrics.TotalJobsCreated = 999999  // Only mutates the copy!

// Internal state unchanged ✅
metrics2, _ := store.GetMetrics(ctx)
fmt.Println(metrics2.TotalJobsCreated)  // Still correct value!
```

---

## How Copying Works

### Struct Copying in Go

**When you copy a struct:**

```go
original := domain.Metric{
    TotalJobsCreated: 100,
    JobsCompleted:    90,
}

copy := original  // Copy the struct
copy.TotalJobsCreated = 0  // Only modifies copy

fmt.Println(original.TotalJobsCreated)  // Still 100
fmt.Println(copy.TotalJobsCreated)       // Now 0
```

**Key point:** Copying creates a new struct with same values.

### Pointer Dereferencing

**Our case:**

```go
s.metrics  // Type: *domain.Metric (pointer)
*s.metrics  // Type: domain.Metric (dereferenced value)
```

**Dereferencing gets the value the pointer points to.**

### The Full Process

```go
// Original (internal state)
s.metrics = &domain.Metric{
    TotalJobsCreated: 100,
}

// Step 1: Dereference and copy
m := *s.metrics  // m is a copy, not a pointer

// Step 2: Return pointer to copy
return &m  // Returns pointer to copy, not original
```

**Result:**

- `s.metrics` points to original (protected)
- Returned pointer points to copy (safe to mutate)

---

## Memory Implications

### Cost of Copying

**Our Metric struct:**

```go
type Metric struct {
    TotalJobsCreated int  // 8 bytes (64-bit int)
    JobsCompleted    int  // 8 bytes
    JobsFailed       int  // 8 bytes
    JobsRetried      int  // 8 bytes
    JobsInProgress   int  // 8 bytes
}
// Total: 40 bytes
```

**Copying cost:**

- **40 bytes** per copy
- **Very cheap** for small structs
- **Negligible** compared to network I/O

### When Copying is Expensive

**Large structs:**

```go
type LargeStruct struct {
    Data [1000000]byte  // 1MB
}
```

**Copying 1MB is expensive!**

**For large structs, consider:**

- Returning pointer but documenting it's read-only
- Using interfaces to prevent mutation
- Copying only when necessary

### Our Case: Copying is Fine

**40 bytes is negligible:**

- Modern CPUs copy 40 bytes in nanoseconds
- Much faster than network I/O
- Worth it for safety

---

## When to Copy vs When to Share

### When to Copy (Our Case)

**Copy when:**

- Struct is small (< 1KB)
- External code might mutate
- Safety is more important than performance
- Internal state must be protected

**Our case:** ✅ Copy (40 bytes, safety critical)

### When to Share (Advanced)

**Share when:**

- Struct is very large (> 1MB)
- Performance is critical
- External code is trusted
- Using read-only interfaces

**Example (advanced):**

```go
// Read-only interface
type ReadOnlyMetric interface {
    TotalJobsCreated() int
    JobsCompleted() int
}

// Return interface, not struct
func (s *InMemoryMetricStore) GetMetrics() ReadOnlyMetric {
    return s.metrics  // Safe: interface prevents mutation
}
```

**For now, copying is simpler and safer.**

---

## Common Mistakes

### Mistake 1: Returning Pointer to Internal State

```go
// ❌ BAD: Returns pointer to internal state
func (s *InMemoryMetricStore) GetMetrics() *domain.Metric {
    return s.metrics  // External code can mutate!
}
```

**Fix:** Return copy

```go
// ✅ GOOD: Returns copy
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    m := *s.metrics  // Copy
    return &m, nil
}
```

### Mistake 2: Forgetting to Lock Before Copying

```go
// ❌ BAD: No lock protection
func (s *InMemoryMetricStore) GetMetrics() *domain.Metric {
    m := *s.metrics  // Race condition! Another goroutine might modify
    return &m
}
```

**Fix:** Lock before copying

```go
// ✅ GOOD: Lock before copying
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    m := *s.metrics  // Safe: protected by lock
    return &m, nil
}
```

### Mistake 3: Copying Slices or Maps

```go
// ❌ BAD: Copying slice doesn't copy elements
type Metric struct {
    JobIDs []string  // Slice
}

m := *s.metrics  // Copy struct, but slice still points to same underlying array!
m.JobIDs[0] = "modified"  // Modifies original!
```

**Fix:** Deep copy slices/maps if needed

```go
// ✅ GOOD: Deep copy slice
m := *s.metrics
m.JobIDs = make([]string, len(s.metrics.JobIDs))
copy(m.JobIDs, s.metrics.JobIDs)  // Copy slice elements
```

**Our case:** No slices/maps, so simple copy works.

### Mistake 4: Not Copying at All

```go
// ❌ BAD: Returns original pointer
func (s *InMemoryMetricStore) GetMetrics() *domain.Metric {
    return s.metrics  // No copy!
}
```

**Fix:** Always copy

```go
// ✅ GOOD: Always copy
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    m := *s.metrics  // Copy
    return &m, nil
}
```

---

## Key Takeaways

1. **Encapsulation** = Protect internal state from external modification
2. **Returning pointers** = Allows external mutation (bad)
3. **Returning copies** = Prevents external mutation (good)
4. **Copying is cheap** = For small structs (< 1KB)
5. **Lock before copying** = Ensure consistent snapshot
6. **Safety over performance** = For small structs, copy is worth it

---

## Real-World Example

**Our protected metrics store:**

```go
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    s.mu.RLock()         // Lock for reading
    defer s.mu.RUnlock()
    
    // Return a copy to prevent external mutation of internal state
    m := *s.metrics      // Copy the struct
    return &m, nil       // Return pointer to copy
}
```

**Protection:**

- ✅ Lock protects during read
- ✅ Copy prevents external mutation
- ✅ Internal state always safe

**External code:**

```go
metrics, _ := store.GetMetrics(ctx)
metrics.TotalJobsCreated = 999999  // Only mutates copy, not internal state
```

**Result:** Internal state remains protected.

---

## Next Steps

- Read [Concurrency-Safe Metrics](./04-concurrency-safe-metrics.md) to understand mutex protection
- Read [Metrics Collection and Storage](./02-metrics-collection-storage.md) to see the full metrics implementation
- Read [Dependency Injection for Observability](./03-dependency-injection-observability.md) to see how we wire the store

