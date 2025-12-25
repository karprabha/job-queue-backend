# Task 3 — Summary of Learnings

## Quick Reference

### Dependency Injection Pattern

```go
// 1. Define handler struct with dependency
type JobHandler struct {
    store store.JobStore  // Interface, not concrete type
}

// 2. Constructor accepts dependency
func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{
        store: store,  // Dependency injected
    }
}

// 3. Methods use injected dependency
func (h *JobHandler) CreateJob(...) {
    h.store.CreateJob(...)  // Use injected store
}
```

### Store Interface

```go
type JobStore interface {
    CreateJob(ctx context.Context, job *domain.Job) error
    GetJobs(ctx context.Context) ([]domain.Job, error)
}
```

### In-Memory Store Implementation

```go
type InMemoryJobStore struct {
    jobs map[string]domain.Job  // Map: ID → Job
    mu   sync.RWMutex           // For concurrency safety
}

func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // Check context before lock
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    s.mu.Lock()         // Write lock (exclusive)
    defer s.mu.Unlock()
    
    s.jobs[job.ID] = *job
    return nil
}

func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    // Check context before lock
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    s.mu.RLock()         // Read lock (allows concurrent reads)
    defer s.mu.RUnlock()
    
    // Convert map to slice
    jobs := make([]domain.Job, 0, len(s.jobs))
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

### Server Setup with Dependency Injection

```go
func main() {
    // 1. Create dependencies first
    jobStore := store.NewInMemoryJobStore()
    
    // 2. Inject dependencies into handlers
    jobHandler := internalhttp.NewJobHandler(jobStore)
    
    // 3. Register handler methods
    mux.HandleFunc("GET /jobs", jobHandler.GetJobs)
    mux.HandleFunc("POST /jobs", jobHandler.CreateJob)
}
```

## Key Concepts

### Dependency Injection

- **What**: Dependencies come from outside, not created inside
- **Why**: Enables testing, flexibility, loose coupling
- **How**: Constructor functions accept dependencies as parameters
- **Pattern**: `NewHandler(dependency) → *Handler`

### Handler Struct Pattern

- **What**: Methods on structs that hold dependencies
- **Why**: Enables dependency injection, shared state
- **When**: Use when handlers need dependencies or state
- **Pattern**: `func (h *Handler) Method(...)`

### In-Memory Storage

- **What**: Data stored in RAM using maps
- **Why**: Fast, simple, no database needed
- **How**: `map[string]Job` for key-value storage
- **Limitations**: Temporary (lost on restart), limited by RAM

### Concurrency Safety

- **Problem**: Multiple goroutines accessing shared data
- **Solution**: Mutexes prevent race conditions
- **Pattern**: `Lock() → do work → Unlock()`
- **Always**: Use `defer Unlock()` to guarantee unlock

### RWMutex vs Mutex

- **RWMutex**: Allows concurrent reads, exclusive writes
- **Mutex**: Exclusive access for all operations
- **When RWMutex**: Read-heavy workloads (like our case)
- **When Mutex**: Write-heavy or simple cases

### Context in Storage

- **Why**: Cancellation propagation, timeout support
- **When**: Check before acquiring lock
- **How**: `select { case <-ctx.Done(): return ctx.Err() }`
- **Benefit**: Don't waste resources on canceled requests

### Interface Design

- **What**: Contract defining required methods
- **Why**: Flexibility, testability, loose coupling
- **How**: Type satisfies interface if it has the methods
- **Pattern**: Accept interfaces, return structs

## Project Structure

```
internal/
├── domain/
│   └── job.go              # Domain model (unchanged)
├── http/
│   ├── handler.go          # Health check handler
│   ├── job_handler.go      # Job handlers (struct-based)
│   └── response.go         # Error response helper
└── store/                  # NEW: Storage layer
    └── job_store.go        # JobStore interface + InMemoryJobStore
```

## Common Patterns

### Dependency Injection Flow

```go
1. Create dependency (store)
   ↓
2. Create handler with dependency (inject)
   ↓
3. Register handler methods
   ↓
4. Handler methods use injected dependency
```

### Store Operation Pattern

```go
1. Check context (before lock)
   ↓
2. Acquire lock (Lock or RLock)
   ↓
3. Do work (protected by lock)
   ↓
4. Release lock (defer Unlock)
```

### Map to Slice Conversion

```go
// 1. Pre-allocate slice with capacity
jobs := make([]Job, 0, len(map))

// 2. Iterate map and append
for _, job := range map {
    jobs = append(jobs, job)
}

// 3. Sort if needed
sort.Slice(jobs, func(i, j int) bool {
    return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
})
```

## Checklist: Dependency Injection

- [ ] Handler struct with dependency field
- [ ] Constructor function accepts dependency
- [ ] Dependency is interface (not concrete type)
- [ ] Dependencies created in main() first
- [ ] Handlers created with dependencies injected
- [ ] No global variables
- [ ] No dependencies created inside handlers

## Checklist: Store Implementation

- [ ] Store interface defined
- [ ] In-memory store implements interface
- [ ] Map for key-value storage
- [ ] Mutex for concurrency safety
- [ ] Context checked before lock
- [ ] Lock acquired before operations
- [ ] defer Unlock() used
- [ ] Map converted to slice for GetJobs
- [ ] Results sorted by creation time

## Checklist: Concurrency Safety

- [ ] Mutex field in store struct
- [ ] Lock() for write operations
- [ ] RLock() for read operations
- [ ] Always use defer Unlock()
- [ ] Context checked before lock
- [ ] All map accesses protected
- [ ] No race conditions

## Checklist: Handler Refactoring

- [ ] Changed from function to struct handler
- [ ] Store dependency injected via constructor
- [ ] Methods use injected store
- [ ] Helper function for response conversion
- [ ] GET /jobs endpoint implemented
- [ ] Empty list returns [] (not null)
- [ ] Response format consistent

## Important Notes

1. **Always inject dependencies** - Don't create them inside handlers
2. **Always use defer Unlock()** - Guarantees unlock even on panic
3. **Check context before lock** - Don't block if request canceled
4. **Use RWMutex for reads** - Allows concurrent reads
5. **Use Lock for writes** - Exclusive access needed
6. **Interfaces for flexibility** - Handler depends on interface, not implementation
7. **Pre-allocate slice capacity** - More efficient map to slice conversion

## Design Decisions

### Why RWMutex?

- Read-heavy workload (GET /jobs frequent)
- Multiple concurrent reads are safe
- Better performance than regular Mutex
- Write operations still exclusive

### Why Interface?

- Can swap implementations (in-memory → database)
- Easy to test (can inject mock)
- Loose coupling (handler doesn't depend on implementation)
- Future-proof for different storage backends

### Why Check Context Before Lock?

- Don't acquire lock if request canceled
- Don't block other goroutines unnecessarily
- Respect client disconnections
- Better resource utilization

### Why Struct Handler?

- Need to hold dependencies (store)
- Multiple related methods (CreateJob, GetJobs)
- Enables dependency injection
- Better than function handlers when state needed

## Next Steps

- Review detailed concepts in [`concepts/`](./concepts/) directory
- Understand dependency injection and interfaces
- Practice concurrency safety patterns
- Learn about database storage implementations

