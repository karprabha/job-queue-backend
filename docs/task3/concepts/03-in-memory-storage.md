# In-Memory Storage with Maps in Go

## Table of Contents

1. [What is In-Memory Storage?](#what-is-in-memory-storage)
2. [Why Use Maps for Storage?](#why-use-maps-for-storage)
3. [Map Basics in Go](#map-basics-in-go)
4. [Our In-Memory Store Implementation](#our-in-memory-store-implementation)
5. [Map Operations Explained](#map-operations-explained)
6. [Converting Map to Slice](#converting-map-to-slice)
7. [Sorting Results](#sorting-results)
8. [Map Limitations](#map-limitations)
9. [Common Mistakes](#common-mistakes)

---

## What is In-Memory Storage?

### The Concept

**In-memory storage** means data is stored in the program's RAM (memory) rather than on disk or in a database.

**Characteristics:**

- ✅ Fast (RAM is very fast)
- ✅ Simple (no database setup)
- ❌ Temporary (lost when program stops)
- ❌ Limited by RAM size

### When to Use In-Memory Storage

**Good for:**

- Development and testing
- Caching
- Temporary data
- Prototyping
- Single-server applications

**Not good for:**

- Production data that must persist
- Large datasets (limited by RAM)
- Multi-server deployments (data not shared)

### Our Use Case

We're using in-memory storage to:

- Store jobs created via `POST /jobs`
- Retrieve jobs via `GET /jobs`
- Learn storage patterns before adding a database

---

## Why Use Maps for Storage?

### The Problem: We Need Key-Value Storage

**Requirements:**

- Store jobs by ID
- Look up jobs by ID quickly
- Iterate over all jobs

### Why Maps?

**Maps are perfect for:**

- ✅ Fast lookups by key (O(1) average)
- ✅ Key-value pairs (ID → Job)
- ✅ Easy to add/remove items
- ✅ Built into Go

**Alternatives and why we didn't use them:**

**1. Slice (Array)**

```go
var jobs []Job
```

- ❌ Slow lookups (must search)
- ❌ No key-value relationship
- ✅ Good for ordered lists

**2. Database**

- ❌ Overkill for learning
- ❌ Requires setup
- ✅ Good for production

**3. Map (Our Choice)**

```go
jobs map[string]Job
```

- ✅ Fast lookups by ID
- ✅ Key-value pairs
- ✅ Simple and built-in

---

## Map Basics in Go

### What is a Map?

A **map** is a built-in data structure that stores key-value pairs.

**Syntax:**

```go
map[KeyType]ValueType
```

**Example:**

```go
// Map from string (ID) to Job
jobs map[string]Job
```

### Creating Maps

**1. Zero Value (nil map)**

```go
var jobs map[string]Job
// jobs is nil - can't use yet!
```

**2. Using make()**

```go
jobs := make(map[string]Job)
// jobs is empty but usable
```

**3. With Initial Capacity**

```go
jobs := make(map[string]Job, 100)
// Pre-allocate space for ~100 items
```

**4. Map Literal**

```go
jobs := map[string]Job{
    "id1": job1,
    "id2": job2,
}
```

### Our Pattern

```go
func NewInMemoryJobStore() *InMemoryJobStore {
    return &InMemoryJobStore{
        jobs: make(map[string]domain.Job),  // Create empty map
    }
}
```

**Why `make()`?**

- Creates an empty, usable map
- Not nil (can add items)
- Standard pattern

---

## Our In-Memory Store Implementation

### The Store Struct

```go
type InMemoryJobStore struct {
    jobs map[string]domain.Job  // Map: ID → Job
    mu   sync.RWMutex           // For concurrency safety
}
```

**Breaking it down:**

- `jobs` - The map storing jobs by ID
- `mu` - Mutex for thread safety (we'll cover this later)

### Creating a Job

```go
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    s.mu.Lock()         // Lock for writing
    defer s.mu.Unlock() // Unlock when done

    s.jobs[job.ID] = *job  // Store job by ID
    return nil
}
```

**What happens:**

1. Lock the mutex (prevent concurrent writes)
2. Store job: `s.jobs[job.ID] = *job`
3. Unlock mutex (allow other operations)

**The assignment:**

```go
s.jobs[job.ID] = *job
```

- `job.ID` is the key (string)
- `*job` is the value (Job struct)
- If ID exists, it's overwritten
- If ID doesn't exist, it's added

### Getting All Jobs

```go
func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    s.mu.RLock()         // Lock for reading
    defer s.mu.RUnlock() // Unlock when done

    // Create slice with capacity
    jobs := make([]domain.Job, 0, len(s.jobs))

    // Iterate map and append to slice
    for _, job := range s.jobs {
        jobs = append(jobs, job)
    }

    // Sort by creation time
    sort.Slice(jobs, func(i, j int) bool {
        return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
    })

    return jobs, nil
}
```

**Step by step:**

1. Lock for reading (RLock allows multiple readers)
2. Create slice with capacity = map size
3. Iterate map, append each job to slice
4. Sort slice by CreatedAt
5. Return sorted slice

---

## Map Operations Explained

### 1. Adding/Updating

```go
s.jobs[job.ID] = *job
```

**What happens:**

- If `job.ID` exists → overwrites existing job
- If `job.ID` doesn't exist → adds new job
- Map automatically grows

### 2. Reading

```go
job := s.jobs[id]
```

**What happens:**

- Returns the job if ID exists
- Returns zero value if ID doesn't exist
- No error returned (use `ok` check)

**Checking if exists:**

```go
job, ok := s.jobs[id]
if !ok {
    // ID doesn't exist
}
```

### 3. Deleting

```go
delete(s.jobs, id)
```

**What happens:**

- Removes key-value pair
- Safe if key doesn't exist (no error)
- Map automatically shrinks

### 4. Iterating

```go
for id, job := range s.jobs {
    // id is the key
    // job is the value
}
```

**Important:** Map iteration order is **random** in Go!

- Not insertion order
- Not sorted order
- Random each time

**Why?** Go randomizes iteration to prevent code from depending on order.

---

## Converting Map to Slice

### The Problem

**Map:**

```go
jobs map[string]Job  // Key-value pairs
```

**Need:**

```go
[]Job  // Slice of jobs
```

### The Solution

```go
// Step 1: Create slice with capacity
jobs := make([]domain.Job, 0, len(s.jobs))
//                    ↑    ↑
//                  size capacity
//                  (0)   (map size)

// Step 2: Iterate map and append
for _, job := range s.jobs {
    jobs = append(jobs, job)  // Add each job
}
```

### Why Pre-allocate Capacity?

**Without capacity:**

```go
jobs := []domain.Job{}  // No capacity
for _, job := range s.jobs {
    jobs = append(jobs, job)  // May reallocate multiple times
}
```

**With capacity:**

```go
jobs := make([]domain.Job, 0, len(s.jobs))  // Pre-allocated
for _, job := range s.jobs {
    jobs = append(jobs, job)  // No reallocation needed
}
```

**Benefits:**

- More efficient (no reallocation)
- Less memory churn
- Better performance

### Why Length 0?

```go
make([]domain.Job, 0, len(s.jobs))
//              ↑
//            length = 0 (empty)
//            capacity = len(s.jobs) (space reserved)
```

- Length = 0 means slice is empty
- Capacity = map size means space is reserved
- `append()` will use reserved space

---

## Sorting Results

### The Problem: Map Order is Random

**Map iteration:**

```go
for _, job := range s.jobs {
    // Order is random!
}
```

**We need:** Consistent order (by creation time)

### The Solution: Sort the Slice

```go
sort.Slice(jobs, func(i, j int) bool {
    return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
})
```

### How sort.Slice Works

**Signature:**

```go
func Slice(slice interface{}, less func(i, j int) bool)
```

**Parameters:**

- `slice` - The slice to sort
- `less` - Function that returns true if `i` should come before `j`

**Our less function:**

```go
func(i, j int) bool {
    return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
}
```

**What it means:**

- `jobs[i]` should come before `jobs[j]` if
- `jobs[i].CreatedAt` is before `jobs[j].CreatedAt`
- Result: Oldest jobs first

### Alternative: Newest First

```go
sort.Slice(jobs, func(i, j int) bool {
    return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
    //      ↑
    //    After instead of Before = newest first
})
```

---

## Map Limitations

### 1. No Ordering

**Maps don't preserve order:**

```go
// Add jobs in order
store.CreateJob(ctx, job1)  // Created first
store.CreateJob(ctx, job2)  // Created second
store.CreateJob(ctx, job3)  // Created third

// Iterate map
for _, job := range store.jobs {
    // Order is random! Not 1, 2, 3
}
```

**Solution:** Sort after converting to slice (which we do)

### 2. No Duplicate Keys

**Maps have unique keys:**

```go
store.CreateJob(ctx, job1)  // ID: "abc"
store.CreateJob(ctx, job2)  // ID: "abc" (same ID)
// Second call overwrites first!
```

**Solution:** Ensure unique IDs (we use UUIDs)

### 3. Memory Limited

**Maps grow in memory:**

- Limited by available RAM
- Not suitable for very large datasets
- No persistence (lost on restart)

**Solution:** For production, use a database

### 4. Not Thread-Safe

**Maps are not safe for concurrent access:**

```go
// ❌ BAD: Concurrent access without mutex
go func() { store.jobs["id1"] = job1 }()
go func() { store.jobs["id2"] = job2 }()
// Race condition! 💥
```

**Solution:** Use mutex (which we do)

---

## Common Mistakes

### Mistake 1: Using Nil Map

```go
// ❌ BAD: Nil map
var jobs map[string]Job
jobs["id"] = job  // Panic! Map is nil
```

**Fix:** Use make()

```go
// ✅ GOOD: Initialize map
jobs := make(map[string]Job)
jobs["id"] = job  // Works!
```

### Mistake 2: Assuming Map Order

```go
// ❌ BAD: Assuming insertion order
for _, job := range s.jobs {
    // Order is random!
}
```

**Fix:** Sort after converting to slice

```go
// ✅ GOOD: Sort explicitly
jobs := make([]Job, 0, len(s.jobs))
for _, job := range s.jobs {
    jobs = append(jobs, job)
}
sort.Slice(jobs, ...)  // Sort explicitly
```

### Mistake 3: Not Checking Existence

```go
// ❌ BAD: Doesn't check if exists
job := s.jobs[id]
if job.ID == "" {  // Wrong check!
    // Zero value might be valid
}
```

**Fix:** Use `ok` check

```go
// ✅ GOOD: Check existence
job, ok := s.jobs[id]
if !ok {
    return fmt.Errorf("job not found")
}
```

### Mistake 4: Concurrent Access Without Mutex

```go
// ❌ BAD: No mutex protection
func (s *InMemoryJobStore) CreateJob(...) {
    s.jobs[id] = job  // Race condition!
}
```

**Fix:** Use mutex

```go
// ✅ GOOD: Mutex protection
func (s *InMemoryJobStore) CreateJob(...) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.jobs[id] = job
}
```

### Mistake 5: Inefficient Slice Creation

```go
// ❌ BAD: No capacity, may reallocate
jobs := []Job{}
for _, job := range s.jobs {
    jobs = append(jobs, job)
}
```

**Fix:** Pre-allocate capacity

```go
// ✅ GOOD: Pre-allocate capacity
jobs := make([]Job, 0, len(s.jobs))
for _, job := range s.jobs {
    jobs = append(jobs, job)
}
```

---

## Key Takeaways

1. **Maps** = Key-value storage, fast lookups
2. **make()** = Creates empty, usable map
3. **Map iteration** = Random order (must sort)
4. **Map to slice** = Iterate and append
5. **Pre-allocate capacity** = More efficient
6. **Sort after conversion** = Consistent order
7. **Mutex required** = Maps not thread-safe

---

## The Go Philosophy

Go provides **simple, powerful primitives**:

- ✅ Maps are built-in (no library needed)
- ✅ Simple syntax and operations
- ✅ Fast and efficient
- ✅ Clear and readable

**Go's approach:**

- Simple data structures
- Explicit operations
- No magic, just clear code

---

## Next Steps

- Read [Concurrency Safety with Mutexes](./04-concurrency-safety.md) to understand why maps need protection
- Read [RWMutex vs Mutex](./05-rwmutex-vs-mutex.md) to see why we use RWMutex
- Read [Interface Design](./07-interface-design.md) to understand the store interface
