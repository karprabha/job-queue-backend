# Task 2 — Summary of Learnings

## Quick Reference

### Domain Model

```go
type Job struct {
    ID        string
    Type      string
    Status    JobStatus
    Payload   json.RawMessage
    CreatedAt time.Time
}

func NewJob(jobType string, jobPayload json.RawMessage) *Job
```

### HTTP Request Parsing

```go
// 1. Limit body size
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB

// 2. Read body
bodyBytes, err := io.ReadAll(r.Body)

// 3. Parse JSON
var request CreateJobRequest
json.Unmarshal(bodyBytes, &request)

// 4. Validate
if request.Type == "" {
    ErrorResponse(w, "Job type is required", http.StatusBadRequest)
    return
}
```

### HTTP Response Pattern

```go
// 1. Create response struct
response := CreateJobResponse{
    ID:        job.ID,
    Type:      job.Type,
    Status:    string(job.Status),
    CreatedAt: job.CreatedAt.Format(time.RFC3339),
}

// 2. Marshal to JSON
responseBytes, err := json.Marshal(response)

// 3. Set headers
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)

// 4. Write response
w.Write(responseBytes)
```

### Server Setup with Explicit Mux

```go
// 1. Create explicit mux instance (not using default mux)
mux := http.NewServeMux()

// 2. Use method-specific routing (Go 1.22+)
mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)
mux.HandleFunc("POST /jobs", internalhttp.CreateJobHandler)

// 3. Explicitly set mux as handler
srv := &http.Server{
    Addr:    ":" + port,
    Handler: mux,  // Explicit mux instance
}
```

**Two improvements:**
- **Explicit mux**: Avoids global state (`http.DefaultServeMux`)
- **Method-specific routing**: Method validation in routing, not handler

### Error Response Pattern

```go
ErrorResponse(w, "Error message", http.StatusBadRequest)
```

## Key Concepts

### Domain Modeling

- **Separation of concerns**: Domain logic separate from HTTP layer
- **Typed constants**: `type JobStatus string` for type safety
- **Constructor functions**: `NewJob()` for encapsulated initialization
- **Opaque payloads**: `json.RawMessage` for flexible JSON storage

### JSON RawMessage

- **What**: `type RawMessage []byte` - stores raw JSON bytes
- **Why**: Domain doesn't need to know payload structure
- **When**: Use for varying structures, opaque data
- **Benefits**: Flexible, preserves JSON structure, loose coupling

### HTTP Request Handling

- **Size limiting**: `http.MaxBytesReader()` prevents DoS attacks
- **Body reading**: `io.ReadAll()` reads entire body into memory
- **JSON parsing**: `json.Unmarshal()` converts bytes to struct
- **Validation**: Check after parsing (empty strings, required fields)

### Error Handling

- **Centralized**: `ErrorResponse()` function for consistent format
- **Status codes**: 4xx for client errors, 5xx for server errors
- **Format**: `{"error": "message"}` JSON structure
- **Fallback**: `http.Error()` if can't marshal JSON

### HTTP Status Codes

- **201 Created**: Resource created successfully
- **400 Bad Request**: Invalid request format
- **413 Request Entity Too Large**: Body exceeds size limit
- **500 Internal Server Error**: Unexpected server error

### UUID Generation

- **Package**: `github.com/google/uuid`
- **Usage**: `uuid.New().String()`
- **Why**: No database needed, distributed-friendly, not guessable

### Time Handling

- **Always UTC**: `time.Now().UTC()` for consistency
- **Format**: `time.RFC3339` for JSON responses
- **Store as time.Time**: Rich API, timezone-aware
- **Format for JSON**: Convert to string in responses

### Server Refactoring: Explicit Mux + Enhanced ServeMux

**Two refactorings:**

1. **Explicit Mux Creation**
   - Changed from default mux (`http.HandleFunc`) to explicit mux (`mux := http.NewServeMux()`)
   - Avoids global state (`http.DefaultServeMux`)
   - More explicit and testable

2. **Method-Specific Routing (Go 1.22+)**
   - Changed from `"/health"` to `"GET /health"`
   - Method validation in routing, not handler
   - No manual method checking needed
   - Cleaner handlers

## Project Structure

```
internal/
├── domain/
│   └── job.go              # Domain model (Job, NewJob)
├── http/
│   ├── handler.go          # Health check handler
│   ├── job_handler.go      # Job creation handler
│   └── response.go         # Error response helper
```

## Common Patterns

### Request Parsing Flow

```go
1. Limit body size (security)
2. Read body bytes
3. Unmarshal JSON to struct
4. Validate required fields
5. Create domain object
6. Format response
7. Write response
```

### Error Handling Pattern

```go
if err != nil {
    ErrorResponse(w, "Clear error message", appropriateStatusCode)
    return  // Early return
}
```

### Domain Separation

```go
// HTTP layer: Read, parse, validate
var request CreateJobRequest
json.Unmarshal(bodyBytes, &request)

// Domain layer: Business logic
job := domain.NewJob(request.Type, request.Payload)

// HTTP layer: Format response
response := CreateJobResponse{...}
```

## Checklist: Create Job Endpoint

- [ ] Limit request body size (MaxBytesReader)
- [ ] Read request body (io.ReadAll)
- [ ] Parse JSON (json.Unmarshal)
- [ ] Validate required fields (type, payload)
- [ ] Create domain object (domain.NewJob)
- [ ] Format response (CreateJobResponse)
- [ ] Marshal response to JSON
- [ ] Set Content-Type header
- [ ] Set status code (201 Created)
- [ ] Write response
- [ ] Handle all errors appropriately

## Checklist: Error Handling

- [ ] Use ErrorResponse() for all errors
- [ ] Appropriate status codes (4xx vs 5xx)
- [ ] Clear, user-friendly error messages
- [ ] Consistent error format ({"error": "message"})
- [ ] Handle marshal errors in ErrorResponse
- [ ] Handle write errors (headers already written)

## Checklist: Domain Model

- [ ] Separate domain from HTTP layer
- [ ] Use typed constants for status values
- [ ] Constructor function for initialization
- [ ] Opaque payloads (json.RawMessage)
- [ ] UTC time for timestamps
- [ ] UUID for IDs

## Important Notes

1. **Always limit body size** - Prevents DoS attacks
2. **Always use UTC** - Consistent across servers
3. **Always validate** - Never trust client input
4. **Separate concerns** - Domain vs HTTP layer
5. **Centralize errors** - Consistent error format
6. **Use appropriate status codes** - 201 for creation, 4xx for client errors
7. **Enhanced ServeMux** - Method-specific routing (Go 1.22+)

## Next Steps

- Review detailed concepts in [`concepts/`](./concepts/) directory
- Understand domain separation and opaque types
- Practice request validation patterns
- Learn about middleware and advanced routing

