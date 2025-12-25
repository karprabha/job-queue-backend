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

- [Task 1 Summary](../task1/summary.md) - Quick reference
- [Task 1 README](../task1/README.md) - Complete overview
- [Concepts Documentation](../task1/concepts/README.md) - Detailed concept explanations
