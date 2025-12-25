# Task 1 — Summary of Learnings

## Quick Reference

### Go Module Setup

```bash
go mod init github.com/karprabha/job-queue-backend
```

### Server Implementation

- Use `http.Server` struct (not `http.ListenAndServe` directly)
- Start server in goroutine for non-blocking execution
- Read port from `PORT` environment variable (default: 8080)

### Graceful Shutdown

- Handle SIGINT (Ctrl+C) and SIGTERM (container shutdown)
- Use `context.WithTimeout()` for shutdown deadline
- Call `srv.Shutdown(ctx)` to stop accepting new connections

### HTTP Handler Pattern

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // Get request context

    // Check for cancellation
    select {
    case <-ctx.Done():
        return  // Request canceled
    default:
    }

    // Validate method
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", 405)
        return
    }

    // Process request...
}
```

### JSON Response Pattern

```go
// 1. Create response data
data := ResponseStruct{...}

// 2. Marshal to JSON
jsonBytes, err := json.Marshal(data)
if err != nil {
    http.Error(w, "Encoding failed", 500)
    return
}

// 3. Set headers
w.Header().Set("Content-Type", "application/json")

// 4. Write response
w.Write(jsonBytes)
```

## Key Concepts

### Context

- **Purpose**: Cancellation, deadlines, request-scoped values
- **Request context**: `r.Context()` - automatically created, cancels on client disconnect/server shutdown
- **Timeout context**: `context.WithTimeout(parent, duration)` - auto-cancels after duration
- **Always defer cancel()**: Prevents resource leaks

### Goroutines

- **Purpose**: Lightweight concurrency
- **Syntax**: `go function()` - runs function in new goroutine
- **Use case**: Start server in background so main can handle signals

### Channels

- **Purpose**: Communication between goroutines
- **Signal channel**: `make(chan os.Signal, 1)` - receives OS signals
- **Blocking receive**: `<-sigChan` - waits until signal arrives

### Error Handling

- **Philosophy**: Errors are values, not exceptions
- **Pattern**: Always check `err != nil`
- **Distinguish**: Expected errors (like `http.ErrServerClosed`) vs unexpected errors

### JSON Encoding

- **Marshal**: `json.Marshal(data)` → `[]byte` - Memory-based, good for small data
- **Encoder**: `json.NewEncoder(writer).Encode(data)` - Stream-based, good for large data
- **Current choice**: Marshal for simple health check response

## Project Structure

```
cmd/          # Main applications
internal/     # Private application code (cannot be imported externally)
docs/         # Documentation
go.mod        # Module definition
```

## Common Patterns

### Server Setup

```go
srv := &http.Server{Addr: ":" + port}
go srv.ListenAndServe()
```

### Signal Handling

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan  // Blocks until signal
```

### Graceful Shutdown

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

### Error Checking

```go
if err != nil && err != http.ErrServerClosed {
    log.Fatalf("Error: %v", err)
}
```

## Checklist: JSON Response

- [ ] Set Content-Type: `application/json`
- [ ] Marshal data to JSON
- [ ] Check encoding errors
- [ ] Write JSON bytes
- [ ] Handle write errors

## Checklist: Server Setup

- [ ] Read port from environment
- [ ] Use `http.Server` struct
- [ ] Start in goroutine
- [ ] Handle signals (SIGINT, SIGTERM)
- [ ] Graceful shutdown with timeout

## Checklist: Error Handling

- [ ] Check all errors
- [ ] Distinguish expected vs unexpected
- [ ] Use appropriate HTTP status codes
- [ ] No panics in server code

## Important Notes

1. **Always use `http.Server`** - Not `http.ListenAndServe` directly
2. **Always defer cancel()** - When creating context with timeout
3. **Always check errors** - Don't ignore them
4. **Set headers before body** - Order matters in HTTP
5. **Use request context** - For cancellation and timeouts

## Next Steps

- Review detailed concepts in [`concepts/`](./concepts/) directory
- Understand each pattern before moving to next task
- Practice implementing similar handlers
- Learn about middleware and request validation
