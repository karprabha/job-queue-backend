# Task 3 — In-Memory Job Store & List Jobs API

## Overview

This task introduces **state** into the service by storing jobs in memory and exposing a read API. The focus is on in-memory persistence, concurrency safety, separating storage from HTTP, and preparing the codebase for future async workers.

## ✅ Completed Requirements

### Functional Requirements

- ✅ In-memory job store implemented
- ✅ Jobs created via `POST /jobs` are stored in memory
- ✅ `GET /jobs` endpoint implemented
- ✅ Returns `200 OK` status
- ✅ Returns JSON array of all jobs
- ✅ Empty list returns `[]` (not `null`)
- ✅ Jobs sorted by creation time
- ✅ `POST /jobs` continues to work and stores jobs
- ✅ `GET /health` continues to work

### Technical Requirements

- ✅ Storage separated from HTTP layer (`internal/store/`)
- ✅ `JobStore` interface defined
- ✅ `InMemoryJobStore` implementation
- ✅ Concurrency-safe (using `sync.RWMutex`)
- ✅ Context support in store methods
- ✅ Dependency injection (store injected into handlers)
- ✅ Handler refactored to struct pattern
- ✅ No global variables
- ✅ No race conditions
- ✅ Proper error handling

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go              # Server setup with DI
├── internal/
│   ├── domain/
│   │   └── job.go               # Domain model (unchanged)
│   ├── http/
│   │   ├── handler.go           # Health check handler
│   │   ├── job_handler.go       # Job handlers (struct-based)
│   │   └── response.go          # Error response helper
│   └── store/                   # NEW: Storage layer
│       └── job_store.go         # JobStore interface + InMemoryJobStore
├── docs/
│   ├── task3/
│   │   ├── README.md            # This file
│   │   ├── summary.md           # Quick reference
│   │   ├── description.md       # Task requirements
│   │   └── concepts/            # Detailed concept explanations
│   └── learnings.md             # Overall learnings
└── go.mod                       # Go module
```

**Structure improvements:**
- `internal/store/` - Storage layer separated from HTTP
- Handler struct pattern - Enables dependency injection
- Clear dependency direction: HTTP → Store → Domain

## 🔑 Key Concepts Learned

### 1. Dependency Injection

- **What**: Dependencies come from outside, not created inside
- **Why**: Enables testing, flexibility, loose coupling
- **Pattern**: Constructor functions accept dependencies
- **Example**: `NewJobHandler(store) → *JobHandler`

### 2. Handler Struct Pattern

- **What**: Methods on structs that hold dependencies
- **Why**: Enables dependency injection, shared state
- **Refactoring**: Function handlers → Struct handlers
- **Benefits**: Can inject dependencies, multiple related methods

### 3. In-Memory Storage

- **What**: Data stored in RAM using maps
- **Why**: Fast, simple, no database needed
- **Implementation**: `map[string]Job` for key-value storage
- **Limitations**: Temporary, limited by RAM

### 4. Concurrency Safety

- **Problem**: Multiple goroutines accessing shared data
- **Solution**: Mutexes prevent race conditions
- **Pattern**: `Lock() → work → Unlock()`
- **Best practice**: Always use `defer Unlock()`

### 5. RWMutex vs Mutex

- **RWMutex**: Allows concurrent reads, exclusive writes
- **Mutex**: Exclusive access for all operations
- **Our choice**: RWMutex (read-heavy workload)
- **Benefits**: Better performance for concurrent reads

### 6. Context in Storage Layer

- **Why**: Cancellation propagation, timeout support
- **When**: Check before acquiring lock
- **Pattern**: `select { case <-ctx.Done(): return ctx.Err() }`
- **Benefit**: Don't waste resources on canceled requests

### 7. Interface Design

- **What**: Contract defining required methods
- **Why**: Flexibility, testability, loose coupling
- **Pattern**: Accept interfaces, return structs
- **Benefits**: Can swap implementations, easy to test

## 📝 Implementation Details

### Server Setup with Dependency Injection

```go
func main() {
    // 1. Create dependencies first
    jobStore := store.NewInMemoryJobStore()
    
    // 2. Inject dependencies into handlers
    jobHandler := internalhttp.NewJobHandler(jobStore)
    
    // 3. Register handler methods
    mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)
    mux.HandleFunc("GET /jobs", jobHandler.GetJobs)
    mux.HandleFunc("POST /jobs", jobHandler.CreateJob)
    
    // 4. Start server
    srv := &http.Server{
        Addr:    ":" + port,
        Handler: mux,
    }
    // ... server startup
}
```

**Key points:**
- Store created first
- Handler created with store injected
- Handler methods registered
- No global state

### Handler Refactoring: Function → Struct

**Before (Task 2):**
```go
// Function handler - no dependencies
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    job := domain.NewJob(...)
    // No storage - just return response
}
```

**After (Task 3):**
```go
// Struct handler - has dependencies
type JobHandler struct {
    store store.JobStore  // Dependency
}

func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{store: store}
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    job := domain.NewJob(...)
    h.store.CreateJob(r.Context(), job)  // Store job
    // Return response
}

func (h *JobHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
    jobs, _ := h.store.GetJobs(r.Context())  // Get all jobs
    // Return response
}
```

**Benefits:**
- Can inject dependencies
- Multiple related methods
- Shared state (store)
- Better testability

### Store Interface

```go
type JobStore interface {
    CreateJob(ctx context.Context, job *domain.Job) error
    GetJobs(ctx context.Context) ([]domain.Job, error)
}
```

**Design decisions:**
- Interface in store package
- Context as first parameter
- Error returns for all operations
- Pointer for job in CreateJob (efficiency)

### In-Memory Store Implementation

```go
type InMemoryJobStore struct {
    jobs map[string]domain.Job  // Map: ID → Job
    mu   sync.RWMutex           // For concurrency safety
}

func NewInMemoryJobStore() *InMemoryJobStore {
    return &InMemoryJobStore{
        jobs: make(map[string]domain.Job),
    }
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

**Key points:**
- Map for key-value storage
- RWMutex for concurrency safety
- Context checked before lock
- Map converted to slice for ordered results
- Sorted by creation time

## 🎓 Learning Resources

Detailed explanations of all concepts are available in the [`concepts/`](./concepts/) directory:

1. **[Dependency Injection](./concepts/01-dependency-injection.md)** - How to inject dependencies
2. **[Handler Struct Pattern](./concepts/02-handler-struct-pattern.md)** - Struct handlers vs function handlers
3. **[In-Memory Storage](./concepts/03-in-memory-storage.md)** - Maps for storage
4. **[Concurrency Safety](./concepts/04-concurrency-safety.md)** - Mutexes and race conditions
5. **[RWMutex vs Mutex](./concepts/05-rwmutex-vs-mutex.md)** - When to use which
6. **[Context in Storage](./concepts/06-context-in-storage.md)** - Context in storage layer
7. **[Interface Design](./concepts/07-interface-design.md)** - Storage interfaces

## 🚀 Running the Service

### Build

```bash
go build -o bin/server ./cmd/server
```

### Run

```bash
# Default port (8080)
go run ./cmd/server

# Custom port
PORT=3000 go run ./cmd/server
```

### Test Endpoints

```bash
# Health check
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Create job
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {"to": "user@example.com"}
  }'
# Expected: 201 Created with job details

# List all jobs
curl http://localhost:8080/jobs
# Expected: 200 OK with array of jobs

# Empty list
curl http://localhost:8080/jobs
# Expected: 200 OK with [] (empty array, not null)
```

## 📋 Quick Reference Checklist

### Dependency Injection

- ✅ Handler struct with dependency field
- ✅ Constructor function accepts dependency
- ✅ Dependency is interface (not concrete type)
- ✅ Dependencies created in main() first
- ✅ Handlers created with dependencies injected
- ✅ No global variables
- ✅ No dependencies created inside handlers

### Store Implementation

- ✅ Store interface defined
- ✅ In-memory store implements interface
- ✅ Map for key-value storage
- ✅ Mutex for concurrency safety
- ✅ Context checked before lock
- ✅ Lock acquired before operations
- ✅ defer Unlock() used
- ✅ Map converted to slice for GetJobs
- ✅ Results sorted by creation time

### Handler Refactoring

- ✅ Changed from function to struct handler
- ✅ Store dependency injected via constructor
- ✅ Methods use injected store
- ✅ Helper function for response conversion
- ✅ GET /jobs endpoint implemented
- ✅ Empty list returns [] (not null)
- ✅ Response format consistent

## 🔄 Refactoring: Function Handler → Struct Handler

### Before (Task 2)

```go
// Function handler - no dependencies
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // Parse request
    // Create job
    // Return response (no storage)
}
```

**Limitations:**
- Can't store jobs
- No dependencies
- Can't share state

### After (Task 3)

```go
// Struct handler - has dependencies
type JobHandler struct {
    store store.JobStore
}

func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{store: store}
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // Parse request
    // Create job
    h.store.CreateJob(r.Context(), job)  // Store job
    // Return response
}

func (h *JobHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
    jobs, _ := h.store.GetJobs(r.Context())  // Get all jobs
    // Return response
}
```

**Benefits:**
- Can store jobs
- Dependencies injected
- Multiple related methods
- Better testability

## 🔄 Future Improvements

Potential enhancements for future tasks:

- Database persistence (replace in-memory store)
- Background job processing
- Job status updates
- Job retrieval by ID (GET /jobs/:id)
- Job deletion (DELETE /jobs/:id)
- Pagination for job listing
- Filtering and sorting options
- Request logging middleware
- Structured logging
- Metrics and monitoring

## 📚 Additional Notes

- **Go version**: 1.25+
- **Dependencies**: Standard library only (sync, context, sort)
- **Project structure**: Follows Go best practices with storage separation
- **Code style**: Idiomatic Go patterns
- **Concurrency**: Safe for concurrent access
- **Storage**: In-memory (temporary, lost on restart)

## 🎯 Design Decisions

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

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

