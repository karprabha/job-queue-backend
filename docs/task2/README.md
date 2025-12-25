# Task 2 — Job Creation Endpoint

## Overview

This task introduces the first domain concept — a **Job** — and implements an HTTP endpoint to create jobs synchronously. The focus is on request parsing, validation, domain modeling, handler-level error handling, and clean separation of concerns.

## ✅ Completed Requirements

### Functional Requirements

- ✅ `POST /jobs` endpoint implemented
- ✅ Accepts JSON request body with `type` and `payload` fields
- ✅ Returns `201 Created` status on success
- ✅ Returns structured JSON response with job details
- ✅ Validates request (type required, payload must be valid JSON)
- ✅ Returns `400 Bad Request` for invalid requests
- ✅ `GET /health` endpoint continues to work

### Technical Requirements

- ✅ Domain model (`Job`) defined in `internal/domain/job.go`
- ✅ Typed constants for job status (`JobStatus`)
- ✅ Opaque JSON payloads using `json.RawMessage`
- ✅ UUID generation for job IDs
- ✅ UTC timestamps for `created_at`
- ✅ Request body size limiting (1MB max)
- ✅ Centralized error response function
- ✅ Appropriate HTTP status codes
- ✅ Clean separation: domain vs HTTP layer
- ✅ Enhanced ServeMux with method-specific routing (Go 1.22+)

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go              # Server setup with mux
├── internal/
│   ├── domain/
│   │   └── job.go               # Domain model (Job, NewJob)
│   └── http/
│       ├── handler.go           # Health check handler
│       ├── job_handler.go       # Job creation handler
│       └── response.go          # Error response helper
├── docs/
│   ├── task2/
│   │   ├── README.md            # This file
│   │   ├── summary.md           # Quick reference
│   │   └── concepts/            # Detailed concept explanations
│   └── learnings.md             # Overall learnings
└── go.mod                       # Go module (includes google/uuid)
```

**Structure improvements:**
- `internal/domain/` - Business logic separated from infrastructure
- `internal/http/` - HTTP layer handles translation
- Clear dependency direction: HTTP → Domain

## 🔑 Key Concepts Learned

### 1. Domain Modeling

- **Domain separation**: Business logic separate from HTTP layer
- **Typed constants**: `type JobStatus string` prevents typos
- **Constructor functions**: `NewJob()` encapsulates initialization
- **Opaque types**: `json.RawMessage` for flexible payloads

### 2. JSON RawMessage

- **What**: Stores raw JSON bytes without parsing
- **Why**: Domain doesn't need to know payload structure
- **Benefits**: Flexible, preserves structure, loose coupling
- **Use case**: Varying payload structures by job type

### 3. HTTP Request Parsing

- **Size limiting**: `http.MaxBytesReader()` prevents DoS
- **Body reading**: `io.ReadAll()` reads entire body
- **JSON parsing**: `json.Unmarshal()` converts to struct
- **Validation**: Check after parsing (empty strings, etc.)

### 4. Request Validation

- **Validate early**: Check at HTTP boundary
- **Fail fast**: Return errors immediately
- **Clear messages**: User-friendly error messages
- **Right status codes**: 4xx for client errors

### 5. Error Response Centralization

- **Consistent format**: `{"error": "message"}` JSON
- **Centralized function**: `ErrorResponse()` for all errors
- **Appropriate codes**: 4xx vs 5xx distinction
- **Fallback handling**: `http.Error()` if can't marshal

### 6. HTTP Status Codes

- **201 Created**: Resource created successfully
- **400 Bad Request**: Invalid request format
- **413 Request Entity Too Large**: Body too large
- **500 Internal Server Error**: Server errors

### 7. UUID Generation

- **Package**: `github.com/google/uuid`
- **Usage**: `uuid.New().String()`
- **Why**: No database needed, distributed-friendly

### 8. Time Handling

- **Always UTC**: `time.Now().UTC()` for consistency
- **RFC3339 format**: Standard for JSON APIs
- **Store as time.Time**: Rich API, timezone-aware

### 9. Enhanced ServeMux (Go 1.22+)

- **Method-specific routing**: `mux.HandleFunc("GET /health", ...)`
- **No manual checking**: Mux handles method validation
- **Cleaner handlers**: Less boilerplate code

## 📝 Implementation Details

### Server Setup with Enhanced Mux

```go
mux := http.NewServeMux()

// Method-specific routing (Go 1.22+)
mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)
mux.HandleFunc("POST /jobs", internalhttp.CreateJobHandler)

srv := &http.Server{
    Addr:    ":" + port,
    Handler: mux,
}
```

**Benefits:**
- Method validation handled by mux
- No manual `r.Method` checking needed
- Cleaner, more declarative routing

### Create Job Handler Flow

```
1. Limit body size (MaxBytesReader - 1MB)
   ↓
2. Read request body (io.ReadAll)
   ↓
3. Parse JSON (json.Unmarshal to CreateJobRequest)
   ↓
4. Validate (type required, payload valid JSON)
   ↓
5. Create domain object (domain.NewJob)
   ↓
6. Format response (CreateJobResponse)
   ↓
7. Marshal to JSON
   ↓
8. Set headers and status (201 Created)
   ↓
9. Write response
```

### Domain Model

```go
type JobStatus string

const (
    StatusPending JobStatus = "pending"
)

type Job struct {
    ID        string
    Type      string
    Status    JobStatus
    Payload   json.RawMessage
    CreatedAt time.Time
}

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

**Design decisions:**
- `JobStatus` type for type safety
- `json.RawMessage` for opaque payloads
- `NewJob()` constructor for initialization
- UTC time for consistency

### Error Response Helper

```go
func ErrorResponse(w http.ResponseWriter, message string, statusCode int) {
    jsonBytes, err := json.Marshal(map[string]string{"error": message})
    if err != nil {
        http.Error(w, "Failed to marshal error response", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    w.Write(jsonBytes)
}
```

**Features:**
- Consistent JSON error format
- Fallback to `http.Error()` if marshal fails
- Handles header writing correctly

## 🎓 Learning Resources

Detailed explanations of all concepts are available in the [`concepts/`](./concepts/) directory:

1. **[Domain Modeling](./concepts/01-domain-modeling.md)** - Struct design, typed constants, constructors
2. **[JSON RawMessage](./concepts/02-json-rawmessage.md)** - Opaque JSON payloads
3. **[HTTP Request Parsing](./concepts/03-http-request-parsing.md)** - Reading and parsing requests
4. **[Request Validation](./concepts/04-request-validation.md)** - Validation patterns
5. **[Error Response Centralization](./concepts/05-error-response-centralization.md)** - Consistent error handling
6. **[HTTP Status Codes](./concepts/06-http-status-codes.md)** - When to use which codes
7. **[Request Body Size Limiting](./concepts/07-request-body-size-limiting.md)** - Security and DoS protection
8. **[UUID Generation](./concepts/08-uuid-generation.md)** - Generating unique IDs
9. **[Time Handling](./concepts/09-time-handling.md)** - UTC, RFC3339 formatting
10. **[Domain Separation](./concepts/10-domain-separation.md)** - Clean architecture
11. **[HTTP Handler Patterns](./concepts/11-http-handler-patterns.md)** - Complete handler patterns

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

### Test Health Endpoint

```bash
curl http://localhost:8080/health
# Expected: {"status":"ok"}
```

### Test Create Job Endpoint

```bash
# Valid request
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Hello"
    }
  }'

# Expected: 201 Created
# Response: {"id":"...","type":"email","status":"pending","created_at":"2024-01-01T12:00:00Z"}

# Invalid request (missing type)
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"payload": {}}'

# Expected: 400 Bad Request
# Response: {"error":"Job type is required and must be non-empty"}
```

## 📋 Quick Reference Checklist

### Create Job Handler

- ✅ Limit request body size (1MB)
- ✅ Read request body
- ✅ Parse JSON to struct
- ✅ Validate required fields
- ✅ Create domain object
- ✅ Format response
- ✅ Set Content-Type header
- ✅ Set 201 Created status
- ✅ Write JSON response
- ✅ Handle all errors

### Domain Model

- ✅ Separate domain package
- ✅ Typed constants for status
- ✅ Constructor function
- ✅ Opaque payloads (json.RawMessage)
- ✅ UTC timestamps
- ✅ UUID generation

### Error Handling

- ✅ Centralized ErrorResponse function
- ✅ Consistent JSON format
- ✅ Appropriate status codes
- ✅ Clear error messages
- ✅ Fallback handling

## 🔄 Refactoring: Explicit Mux + Enhanced ServeMux

### Before (Task 1)

```go
// Using default mux (global state)
http.HandleFunc("/health", internalhttp.HealthCheckHandler)

// Server uses default mux
srv := &http.Server{
    Addr: ":" + port,
    // Handler defaults to http.DefaultServeMux
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // ... handler logic
}
```

**Issues:**
- Using default mux (global state)
- Manual method checking in handler
- Boilerplate code
- Easy to forget method validation

### After (Task 2)

```go
// 1. Explicitly create mux instance
mux := http.NewServeMux()

// 2. Use method-specific routing (Go 1.22+)
mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)
mux.HandleFunc("POST /jobs", internalhttp.CreateJobHandler)

// 3. Explicitly set mux as handler
srv := &http.Server{
    Addr:    ":" + port,
    Handler: mux,  // Explicit mux instance
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // No method checking needed - mux handles it!
    // ... handler logic
}
```

**Two Refactorings:**

1. **Explicit Mux Creation**
   - Changed from default mux (`http.HandleFunc`) to explicit mux (`mux := http.NewServeMux()`)
   - Avoids global state
   - More explicit and testable

2. **Method-Specific Routing (Go 1.22+)**
   - Changed from `"/health"` to `"GET /health"`
   - Method validation in routing, not handler
   - Cleaner handlers

**Benefits:**
- No global state (explicit mux)
- Method validation in routing (Go 1.22+)
- Cleaner handlers
- Less boilerplate
- Compile-time method specification
- Better testability

## 🔄 Future Improvements

Potential enhancements for future tasks:
- Add request logging middleware
- Add structured logging
- Add request ID tracking
- Add payload validation based on job type
- Add job retrieval endpoint (GET /jobs/:id)
- Add job listing endpoint (GET /jobs)
- Add database persistence
- Add background job processing

## 📚 Additional Notes

- **Go version**: 1.25+ (enhanced ServeMux requires Go 1.22+)
- **Dependencies**: `github.com/google/uuid`
- **Project structure**: Follows Go best practices with domain separation
- **Code style**: Idiomatic Go patterns

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

