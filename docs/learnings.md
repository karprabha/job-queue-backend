# Learnings Summary

## Task 1 — Service Skeleton & Health Endpoint

### Quick Setup Commands

- `go mod init github.com/karprabha/job-queue-backend` - Initialize Go module
- `go install golang.org/x/tools/cmd/goimports@latest` - Automatic import formatting
- `brew install postgresql@15` - Install PostgreSQL (for future tasks)
- `go install github.com/pressly/goose/v3/cmd/goose@latest` - Database migrations (for future tasks)

### JSON Response Checklist (Memorize This)

1. Set `Content-Type: application/json` header
2. Marshal data to JSON bytes: `json.Marshal(data)`
3. Check for encoding errors
4. Write JSON bytes to response: `w.Write(jsonBytes)`
5. Handle write errors

### JSON Request Checklist (For Future Tasks)

1. Read request body: `io.ReadAll(r.Body)`
2. Parse JSON into struct: `json.Unmarshal(bodyBytes, &struct)`
3. Validate data
4. Use the parsed data

### Key Patterns Learned

#### Server Setup

```go
// Read port from environment
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

// Create server
srv := &http.Server{Addr: ":" + port}

// Start in goroutine
go srv.ListenAndServe()

// Handle signals
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

// Graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

#### HTTP Handler Pattern

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    // 1. Get context
    ctx := r.Context()

    // 2. Check cancellation
    select {
    case <-ctx.Done():
        return
    default:
    }

    // 3. Validate method
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", 405)
        return
    }

    // 4. Process request
    // 5. Marshal JSON
    // 6. Set headers
    // 7. Write response
}
```

### Important Concepts

- **Context**: Request cancellation, timeouts, graceful shutdown
- **Goroutines**: Concurrency for non-blocking server startup
- **Channels**: Communication for signal handling
- **Error Handling**: Always check errors, distinguish expected vs unexpected
- **JSON Encoding**: Marshal for small data, Encoder for large data

### Project Structure

- `cmd/` - Main applications
- `internal/` - Private code (cannot be imported externally)
- `docs/task1/` - Task 1 documentation and concepts

### Detailed Documentation

For comprehensive explanations, see:

- [Task 1 Summary](./task1/summary.md) - Quick reference
- [Task 1 README](./task1/README.md) - Complete overview
- [Task 1 Concepts Documentation](./task1/concepts/README.md) - Detailed concept explanations

---

## Task 2 — Job Creation Endpoint

### Quick Setup Commands

- `go get github.com/google/uuid` - UUID generation package

### Create Job Request Checklist (Memorize This)

1. Limit body size: `r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)`
2. Read request body: `io.ReadAll(r.Body)`
3. Parse JSON: `json.Unmarshal(bodyBytes, &request)`
4. Validate required fields: `if request.Type == "" { ... }`
5. Create domain object: `job := domain.NewJob(request.Type, request.Payload)`
6. Format response: `CreateJobResponse{...}`
7. Marshal to JSON: `json.Marshal(response)`
8. Set headers: `w.Header().Set("Content-Type", "application/json")`
9. Set status: `w.WriteHeader(http.StatusCreated)`
10. Write response: `w.Write(responseBytes)`

### Key Patterns Learned

#### Server Setup with Enhanced Mux

```go
// Create mux
mux := http.NewServeMux()

// Method-specific routing (Go 1.22+)
mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)
mux.HandleFunc("POST /jobs", internalhttp.CreateJobHandler)

// Create server with mux
srv := &http.Server{
    Addr:    ":" + port,
    Handler: mux,
}
```

#### Create Job Handler Pattern

```go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Limit body size (security)
    r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

    // 2. Read body
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
        return
    }

    // 3. Parse JSON
    var request CreateJobRequest
    if err := json.Unmarshal(bodyBytes, &request); err != nil {
        ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
        return
    }

    // 4. Validate
    if request.Type == "" {
        ErrorResponse(w, "Job type is required", http.StatusBadRequest)
        return
    }

    // 5. Create domain object
    job := domain.NewJob(request.Type, request.Payload)

    // 6. Format response
    response := CreateJobResponse{
        ID:        job.ID,
        Type:      job.Type,
        Status:    string(job.Status),
        CreatedAt: job.CreatedAt.Format(time.RFC3339),
    }

    // 7. Marshal and write
    responseBytes, _ := json.Marshal(response)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    w.Write(responseBytes)
}
```

#### Error Response Pattern

```go
// Centralized error response
ErrorResponse(w, "Clear error message", http.StatusBadRequest)
```

#### Domain Model Pattern

```go
// Typed constants
type JobStatus string
const (
    StatusPending JobStatus = "pending"
)

// Domain struct
type Job struct {
    ID        string
    Type      string
    Status    JobStatus
    Payload   json.RawMessage  // Opaque JSON
    CreatedAt time.Time
}

// Constructor
func NewJob(jobType string, jobPayload json.RawMessage) *Job {
    return &Job{
        ID:        uuid.New().String(),
        Type:      jobType,
        Status:    StatusPending,
        Payload:   jobPayload,
        CreatedAt: time.Now().UTC(),
    }
}
```

### Important Concepts

- **Domain Separation**: Business logic separate from HTTP layer
- **Typed Constants**: `type JobStatus string` for type safety
- **Opaque Payloads**: `json.RawMessage` for flexible JSON storage
- **Request Validation**: Validate at HTTP boundary, fail fast
- **Error Centralization**: Consistent error format with `ErrorResponse()`
- **HTTP Status Codes**: 201 Created, 400 Bad Request, 413 Too Large, 500 Internal Error
- **UUID Generation**: `uuid.New().String()` for unique IDs
- **Time Handling**: Always UTC, RFC3339 format for JSON
- **Enhanced ServeMux**: Method-specific routing (Go 1.22+)

### Project Structure

- `internal/domain/` - Business logic (Job model)
- `internal/http/` - HTTP layer (handlers, responses)
- Clear separation: HTTP → Domain (not the reverse)

### Detailed Documentation

For comprehensive explanations, see:

- [Task 2 Summary](./task2/summary.md) - Quick reference
- [Task 2 README](./task2/README.md) - Complete overview
- [Task 2 Concepts Documentation](./task2/concepts/README.md) - Detailed concept explanations

---

## Task 3 — In-Memory Job Store & List Jobs API

### Quick Setup Commands

- No new dependencies needed (uses standard library: `sync`, `context`, `sort`)

### Dependency Injection Pattern (Memorize This)

```go
// 1. Define handler struct with dependency
type JobHandler struct {
    store store.JobStore  // Interface, not concrete type
}

// 2. Constructor accepts dependency
func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{store: store}
}

// 3. Methods use injected dependency
func (h *JobHandler) CreateJob(...) {
    h.store.CreateJob(r.Context(), job)
}
```

### Store Implementation Pattern

```go
// Store interface
type JobStore interface {
    CreateJob(ctx context.Context, job *domain.Job) error
    GetJobs(ctx context.Context) ([]domain.Job, error)
}

// In-memory implementation
type InMemoryJobStore struct {
    jobs map[string]domain.Job
    mu   sync.RWMutex
}

func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // Check context before lock
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    s.mu.Lock()         // Write lock
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
    store store.JobStore
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

### Important Concepts

- **Dependency Injection**: Dependencies come from outside, not created inside
- **Handler Struct Pattern**: Methods on structs that hold dependencies
- **In-Memory Storage**: Maps for key-value storage (`map[string]Job`)
- **Concurrency Safety**: Mutexes prevent race conditions
- **RWMutex**: Allows concurrent reads, exclusive writes (better for read-heavy workloads)
- **Context in Storage**: Check context before lock, respect cancellation
- **Interface Design**: Accept interfaces, return structs (flexibility, testability)
- **Map to Slice**: Convert map to slice for ordered results, then sort

### Project Structure

- `internal/store/` - Storage layer separated from HTTP
- Handler struct pattern - Enables dependency injection
- Clear dependency direction: HTTP → Store → Domain

### Detailed Documentation

For comprehensive explanations, see:

- [Task 3 Summary](./task3/summary.md) - Quick reference
- [Task 3 README](./task3/README.md) - Complete overview
- [Task 3 Concepts Documentation](./task3/concepts/README.md) - Detailed concept explanations

---

## Task 4 — Background Worker & Job Processing Loop

### Quick Setup Commands

- No new dependencies needed (uses standard library: `sync`, `context`, `time`)

### Worker Pattern (Memorize This)

```go
// 1. Create worker with dependencies
worker := worker.NewWorker(jobStore, jobQueue)

// 2. Create context for cancellation
workerCtx, workerCancel := context.WithCancel(context.Background())
defer workerCancel()

// 3. Start worker in goroutine with WaitGroup
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    worker.Start(workerCtx)
}()

// 4. On shutdown
workerCancel()  // Cancel context
wg.Wait()       // Wait for worker to finish
close(jobQueue) // Close channel
```

### Channel Setup Pattern

```go
// Create buffered channel
const jobQueueCapacity = 100
jobQueue := make(chan *domain.Job, jobQueueCapacity)

// Send job (in handler)
select {
case jobQueue <- job:
    // Successfully enqueued
case <-time.After(100 * time.Millisecond):
    log.Printf("Warning: Job queue full")
case <-r.Context().Done():
    return
}

// Receive job (in worker)
select {
case <-ctx.Done():
    return
case job, ok := <-jobQueue:
    if !ok {
        return  // Channel closed
    }
    // Process job
}
```

### Worker Implementation Pattern

```go
type Worker struct {
    jobStore store.JobStore
    jobQueue chan *domain.Job
}

func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-w.jobQueue:
            if !ok {
                return
            }
            // Claim job atomically
            claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
            if err != nil || !claimed {
                continue
            }
            // Process job
            w.processJob(ctx, job)
        }
    }
}

func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    timer := time.NewTimer(1 * time.Second)
    defer timer.Stop()

    select {
    case <-timer.C:
        // Processing complete
        w.updateJobStatus(ctx, job, domain.StatusCompleted)
    case <-ctx.Done():
        // Shutdown requested
        w.updateJobStatus(ctx, job, domain.StatusFailed)
        return
    }
}
```

### Atomic Claiming Pattern

```go
func (s *InMemoryJobStore) ClaimJob(ctx context.Context, jobID string) (bool, error) {
    // Check context before lock
    select {
    case <-ctx.Done():
        return false, ctx.Err()
    default:
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    job, ok := s.jobs[jobID]
    if !ok || job.Status != domain.StatusPending {
        return false, nil
    }

    // Atomically update status
    job.Status = domain.StatusProcessing
    s.jobs[jobID] = job

    return true, nil
}
```

### Graceful Shutdown Pattern

```go
// 1. Wait for shutdown signal
<-sigChan
log.Println("Shutting down...")

// 2. Cancel worker context
workerCancel()

// 3. Wait for worker to finish
wg.Wait()

// 4. Close channel (safe now)
close(jobQueue)

// 5. Shutdown HTTP server with timeout
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()
srv.Shutdown(shutdownCtx)
```

### Important Concepts

- **Goroutines**: Lightweight threads for background processing
- **Channels**: Thread-safe communication between goroutines
- **Buffered Channels**: Can queue values, decouples sender/receiver
- **Worker Pattern**: Background goroutine processes work from channel
- **Context Cancellation**: Standard way to signal shutdown
- **WaitGroup**: Tracks goroutine completion
- **Atomic Operations**: ClaimJob prevents race conditions
- **Select Statement**: Wait for multiple channels simultaneously
- **Graceful Shutdown**: Finish current work before exiting

### Project Structure

- `internal/worker/` - Worker package separated from HTTP
- Channel-based communication - No polling, efficient
- Clear separation: HTTP → Worker → Store → Domain

### Detailed Documentation

For comprehensive explanations, see:

- [Task 4 Summary](./task4/summary.md) - Quick reference
- [Task 4 README](./task4/README.md) - Complete overview
- [Task 4 Concepts Documentation](./task4/concepts/README.md) - Detailed concept explanations

---

## Task 5 — Multiple Workers & Controlled Concurrency

### Quick Setup Commands

- `WORKER_COUNT=20 go run ./cmd/server` - Run with 20 workers
- `JOB_QUEUE_CAPACITY=200 go run ./cmd/server` - Run with larger queue
- `PORT=3000 WORKER_COUNT=10 go run ./cmd/server` - Custom configuration

### Worker Pool Pattern (Memorize This)

```go
var wg sync.WaitGroup

for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}

// Shutdown
workerCancel()
wg.Wait()
```

### Configuration Pattern

```go
// Environment variables with defaults
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

workerCount := os.Getenv("WORKER_COUNT")
if workerCount == "" {
    workerCount = "10"
}

workerCountInt, err := strconv.Atoi(workerCount)
if err != nil {
    workerCountInt = 10  // Default on error
}
```

### Proper Shutdown Order (Critical!)

```go
// 1. Shutdown HTTP server first (stops accepting new requests)
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()
srv.Shutdown(shutdownCtx)

// 2. Close job queue (no more requests can enqueue)
close(jobQueue)

// 3. Cancel workers and wait
workerCancel()
wg.Wait()
```

**Why this order?** Prevents "send on closed channel" panics!

### Key Patterns Learned

#### Worker Pool Creation

```go
// Modern pattern (Go 1.21+)
var wg sync.WaitGroup

for i := 0; i < config.WorkerCount; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)
    })
}
```

#### ClaimJob Pattern (Prevents Duplicates)

```go
// Worker receives job from channel
job := <-w.jobQueue

// Try to claim atomically
claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
if err != nil {
    continue  // Skip on error
}

if !claimed {
    continue  // Another worker got it first
}

// We successfully claimed it, process it
w.processJob(ctx, job)
```

#### Configuration Package

```go
type Config struct {
    Port             string
    JobQueueCapacity int
    WorkerCount      int
}

func NewConfig() *Config {
    // Read from environment, provide defaults
    // Handle errors gracefully
    return &Config{...}
}
```

### Important Concepts

- **Worker Pools**: Multiple workers processing concurrently
- **Fan-Out Pattern**: One channel, multiple workers (automatic load balancing)
- **ClaimJob**: Atomic check-and-set prevents duplicate processing
- **Configuration Management**: Environment variables with defaults
- **Proper Shutdown Order**: Server → Channel → Workers (prevents panics)
- **Modern WaitGroup**: `wg.Go()` (Go 1.21+) automatically handles Add/Done
- **Closure Variable Capture**: Always be aware of what closures capture
- **Store as Source of Truth**: Channel is notification, store is authoritative

### Project Structure

- `internal/config/` - Configuration management
- Worker pool in `main.go` - Multiple workers created in loop
- Worker IDs - Each worker has unique identifier for logging

### Critical Bugs to Avoid

#### 1. Closure Variable Capture
```go
// ❌ BAD: All workers get same ID
for i := 0; i < 10; i++ {
    wg.Go(func() {
        worker.Start(workerCtx, i)  // i is 10 for all!
    })
}

// ✅ GOOD: Worker created before closure
for i := 0; i < 10; i++ {
    worker := worker.NewWorker(i, jobStore, jobQueue)
    wg.Go(func() {
        worker.Start(workerCtx)  // Captures worker instance
    })
}
```

#### 2. Send on Closed Channel
```go
// ❌ BAD: Channel closed before server shutdown
close(jobQueue)
srv.Shutdown(ctx)  // Handler might panic!

// ✅ GOOD: Server shutdown first
srv.Shutdown(ctx)
close(jobQueue)
```

#### 3. WaitGroup Add Outside Loop (Traditional Pattern)
```go
// ❌ BAD: Only tracks 1 worker
wg.Add(1)
for i := 0; i < 10; i++ {
    go func() { ... }()
}

// ✅ GOOD: Modern pattern (Go 1.21+)
for i := 0; i < 10; i++ {
    wg.Go(func() { ... })  // Automatically handles Add/Done
}
```

### Performance Impact

- **Single Worker**: 100 jobs = 100 seconds
- **10 Workers**: 100 jobs = 10 seconds
- **10x improvement** in throughput!

### Detailed Documentation

For comprehensive explanations, see:

- [Task 5 Summary](./task5/summary.md) - Quick reference
- [Task 5 README](./task5/README.md) - Complete overview
- [Task 5 Concepts Documentation](./task5/concepts/README.md) - Detailed concept explanations

---

## Task 6 — Failure Handling, Retries & Job States

### Quick Setup Commands

- `SWEEPER_INTERVAL=5s go run ./cmd/server` - Run with 5 second sweeper interval
- `PORT=3000 SWEEPER_INTERVAL=10s go run ./cmd/server` - Custom configuration

### State Machine Pattern (Memorize This)

```go
// 1. Define transition validation function
func canTransition(from, to domain.JobStatus) bool {
    switch {
    case from == domain.StatusPending && to == domain.StatusProcessing:
        return true
    case from == domain.StatusProcessing && to == domain.StatusCompleted:
        return true
    case from == domain.StatusProcessing && to == domain.StatusFailed:
        return true
    case from == domain.StatusFailed && to == domain.StatusPending:
        return true
    default:
        return false
    }
}

// 2. Validate before updating
func (s *InMemoryJobStore) UpdateStatus(jobID string, status JobStatus, lastError *string) error {
    job := s.jobs[jobID]
    if !canTransition(job.Status, status) {
        return errors.New("invalid state transition")
    }
    job.Status = status
    if lastError != nil {
        job.LastError = lastError
    }
    s.jobs[jobID] = job
    return nil
}
```

### Failure Handling Pattern

```go
// Worker signals failure
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    if processingFails {
        errMsg := "Processing failed: connection timeout"
        w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &errMsg)
        return
    }
    
    // Success
    w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
}
```

### Retry Logic Pattern

```go
// Retry only if attempts < maxRetries
func (s *InMemoryJobStore) RetryFailedJobs(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    for jobID, job := range s.jobs {
        if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
            job.Status = domain.StatusPending
            s.jobs[jobID] = job
        }
    }
    return nil
}
```

### Sweeper Pattern

```go
// Periodic retry mechanism
func (s *InMemorySweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Retry failed jobs
            s.jobStore.RetryFailedJobs(ctx)
            
            // Enqueue pending jobs
            jobs, _ := s.jobStore.GetPendingJobs(ctx)
            for _, job := range jobs {
                s.jobQueue <- job.ID
            }
        }
    }
}
```

### Channel Simplification Pattern

```go
// Before: Full objects
jobQueue := make(chan *domain.Job, 100)
jobQueue <- job  // Sends entire object

// After: Just IDs
jobQueue := make(chan string, 100)
jobQueue <- job.ID  // Sends just ID

// Worker fetches from store
jobID := <-jobQueue
job, _ := store.ClaimJob(ctx, jobID)  // Get fresh data
```

### Key Patterns Learned

#### UpdateStatus Method

```go
func (s *InMemoryJobStore) UpdateStatus(
    ctx context.Context,
    jobID string,
    status domain.JobStatus,
    lastError *string,
) error {
    // Check context
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Acquire lock
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Get job
    job, ok := s.jobs[jobID]
    if !ok {
        return errors.New("job not found")
    }
    
    // Validate transition
    if !canTransition(job.Status, status) {
        return errors.New("invalid state transition")
    }
    
    // Update all fields atomically
    job.Status = status
    if lastError != nil {
        job.LastError = lastError
    }
    s.jobs[jobID] = job  // Save after all updates
    
    return nil
}
```

#### ClaimJob with Attempt Tracking

```go
func (s *InMemoryJobStore) ClaimJob(ctx context.Context, jobID string) (*domain.Job, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    job, ok := s.jobs[jobID]
    if !ok || job.Status != domain.StatusPending {
        return nil, nil
    }
    
    // Atomically update status and increment attempts
    job.Status = domain.StatusProcessing
    job.Attempts++  // Increment when claiming
    s.jobs[jobID] = job
    
    return &jobCopy, nil
}
```

#### Sweeper Setup

```go
// In main.go
sweeper := store.NewInMemorySweeper(jobStore, config.SweeperInterval, jobQueue)

sweeperCtx, sweeperCancel := context.WithCancel(context.Background())
defer sweeperCancel()

var sweeperWg sync.WaitGroup
sweeperWg.Go(func() {
    sweeper.Run(sweeperCtx)
})

// On shutdown
sweeperCancel()
sweeperWg.Wait()
```

### Important Concepts

- **State Machines**: Explicit rules for state transitions prevent bugs
- **Failure Handling**: Failure is a first-class state, not an exception
- **Retry Logic**: Attempts track retries, MaxRetries prevents infinite loops
- **Sweeper Pattern**: Periodic background process handles retries separately
- **Atomic Updates**: Mutex-protected state changes ensure consistency
- **Channel Simplification**: Send IDs, not objects (less memory, fresh data)
- **Store as Source of Truth**: Store enforces rules, workers just signal events
- **Transition Validation**: All state changes validated before applying

### Project Structure

- `internal/store/sweeper.go` - Sweeper pattern separated
- State machine in `job_store.go` - Centralized transition validation
- Channel type simplified - Job IDs instead of full objects
- Clear separation: Worker processes, Sweeper retries, Store enforces rules

### Critical Bugs to Avoid

#### 1. Workers Directly Mutating State
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

#### 2. Infinite Retries
```go
// ❌ BAD: No retry limit check
if job.Status == domain.StatusFailed {
    job.Status = domain.StatusPending  // Retry forever!
}

// ✅ GOOD: Check retry limit
if job.Status == domain.StatusFailed && job.Attempts < job.MaxRetries {
    job.Status = domain.StatusPending  // Retry only if allowed
}
```

#### 3. Missing Transition Validation
```go
// ❌ BAD: No validation
job.Status = newStatus  // Could be invalid!

// ✅ GOOD: Validate transition
if !canTransition(job.Status, newStatus) {
    return errors.New("invalid transition")
}
job.Status = newStatus
```

#### 4. Not Saving After Updating Fields
```go
// ❌ BAD: LastError not saved
job.Status = status
s.jobs[jobID] = job  // Save here
if lastError != nil {
    job.LastError = lastError  // Update but never save!
}

// ✅ GOOD: Save after all updates
job.Status = status
if lastError != nil {
    job.LastError = lastError
}
s.jobs[jobID] = job  // Save after all updates
```

### Performance Impact

- **Memory Usage**: 80% reduction (IDs vs full objects in channel)
- **Retry Behavior**: Failed jobs retry automatically (up to limit)
- **State Consistency**: All transitions validated, no invalid states

### Detailed Documentation

For comprehensive explanations, see:

- [Task 6 Summary](./task6/summary.md) - Quick reference
- [Task 6 README](./task6/README.md) - Complete overview
- [Task 6 Concepts Documentation](./task6/concepts/README.md) - Detailed concept explanations

---

## Task 7 — Observability: Structured Logging & Metrics

### Quick Setup Commands

- No new dependencies needed (uses standard library: `log/slog`)

### Structured Logging Pattern (Memorize This)

```go
// 1. Create logger in main
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

// 2. Inject logger into components
type JobHandler struct {
    logger *slog.Logger
}

func NewJobHandler(logger *slog.Logger) *JobHandler {
    return &JobHandler{logger: logger}
}

// 3. Use structured logging with event names
func (h *JobHandler) CreateJob(...) {
    h.logger.Info("Job created", "event", "job_created", "job_id", jobID)
}
```

### Metrics Collection Pattern

```go
// 1. Create metric store in main
metricStore := store.NewInMemoryMetricStore()

// 2. Inject into components
type JobHandler struct {
    metricStore store.MetricStore
}

// 3. Update metrics (not directly, through store)
func (h *JobHandler) CreateJob(...) {
    h.metricStore.IncrementJobsCreated(ctx)
}
```

### Metrics Store Pattern

```go
type InMemoryMetricStore struct {
    mu      sync.RWMutex
    metrics *domain.Metric
}

func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Return a copy to prevent external mutation
    m := *s.metrics
    return &m, nil
}

func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.metrics.TotalJobsCreated++
    return nil
}
```

### Event-Based Logging Pattern

```go
// Every log includes event name
logger.Info("Job created", "event", "job_created", "job_id", jobID)
logger.Info("Job started", "event", "job_started", "worker_id", workerID, "job_id", jobID)
logger.Info("Job completed", "event", "job_completed", "worker_id", workerID, "job_id", jobID)
logger.Error("Failed to process", "event", "job_process_error", "error", err)
```

### Metrics Endpoint Pattern

```go
func (h *MetricHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    metrics, err := h.metricStore.GetMetrics(r.Context())
    if err != nil {
        ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
        return
    }
    
    response := MetricResponse{
        TotalJobsCreated: metrics.TotalJobsCreated,
        JobsCompleted:    metrics.JobsCompleted,
        JobsFailed:       metrics.JobsFailed,
        JobsRetried:      metrics.JobsRetried,
        JobsInProgress:   metrics.JobsInProgress,
    }
    
    responseBytes, _ := json.Marshal(response)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(responseBytes)
}
```

### Key Patterns Learned

#### Logger Initialization

```go
// In main.go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

// Inject into all components
jobHandler := internalhttp.NewJobHandler(jobStore, metricStore, logger, jobQueue)
worker := worker.NewWorker(workerID, jobStore, metricStore, logger, jobQueue)
```

#### Structured Logging

```go
// Info level - normal events
logger.Info("Job created", "event", "job_created", "job_id", jobID)

// Error level - errors
logger.Error("Failed to create job", "event", "job_create_error", "error", err)

// Always include event name
// Use consistent field naming (snake_case)
```

#### Metrics Updates

```go
// Counter - only increments
metricStore.IncrementJobsCreated(ctx)

// Gauge - increments and decrements
metricStore.IncrementJobsInProgress(ctx)  // When job starts
metricStore.IncrementJobsCompleted(ctx)   // Decrements in_progress inside
```

#### Returning Copies

```go
// ❌ BAD: Returns pointer to internal state
return s.metrics  // External code can mutate!

// ✅ GOOD: Returns copy
m := *s.metrics  // Copy
return &m, nil   // Return pointer to copy
```

### Important Concepts

- **Structured Logging**: Key-value pairs instead of free-form text
- **Event-Based Logging**: Every log includes event name for searchability
- **Metrics Collection**: Numerical measurements of system behavior
- **Dependency Injection**: Pass logger and metrics, don't use globals
- **Concurrency Safety**: Mutex protection for shared metrics
- **Encapsulation**: Return copies to prevent external mutation
- **Counter vs Gauge**: Counters only increment, gauges increment/decrement
- **RWMutex**: Allows concurrent reads, exclusive writes

### Project Structure

- `internal/domain/metric.go` - Metric domain model
- `internal/store/metric_store.go` - Metrics storage separated
- `internal/http/metric_handler.go` - Metrics endpoint handler
- Logger and metrics injected throughout

### Critical Bugs to Avoid

#### 1. Global Logger
```go
// ❌ BAD: Global logger
var logger = slog.Default()

func handler() {
    logger.Info("message")
}

// ✅ GOOD: Injected logger
type Handler struct {
    logger *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
    return &Handler{logger: logger}
}
```

#### 2. Returning Pointer to Internal State
```go
// ❌ BAD: Returns pointer to internal state
func (s *MetricStore) GetMetrics() *Metric {
    return s.metrics  // External code can mutate!
}

// ✅ GOOD: Returns copy
func (s *MetricStore) GetMetrics(ctx context.Context) (*Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    m := *s.metrics  // Copy
    return &m, nil
}
```

#### 3. Updating Metrics from Handlers
```go
// ❌ BAD: Handler directly updates metrics
func (h *Handler) CreateJob() {
    h.metrics.JobsCreated++  // Direct mutation!
}

// ✅ GOOD: Handler calls metric store
func (h *Handler) CreateJob() {
    h.metricStore.IncrementJobsCreated(ctx)  // Store handles it
}
```

#### 4. Missing Event Names
```go
// ❌ BAD: No event name
logger.Info("Job created", "job_id", jobID)

// ✅ GOOD: With event name
logger.Info("Job created", "event", "job_created", "job_id", jobID)
```

#### 5. Inconsistent Field Naming
```go
// ❌ BAD: Mixed naming
logger.Info("Job created", "jobId", id, "worker_id", wid)

// ✅ GOOD: Consistent snake_case
logger.Info("Job created", "event", "job_created", "job_id", id, "worker_id", wid)
```

### Performance Impact

- **Logging**: Structured logs slightly slower, but worth it for observability
- **Metrics**: Minimal overhead (mutex locks are fast)
- **Memory**: In-memory storage is very efficient

### Detailed Documentation

For comprehensive explanations, see:

- [Task 7 Summary](./task7/summary.md) - Quick reference
- [Task 7 README](./task7/README.md) - Complete overview
- [Task 7 Concepts Documentation](./task7/concepts/README.md) - Detailed concept explanations
