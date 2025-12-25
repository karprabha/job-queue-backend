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
