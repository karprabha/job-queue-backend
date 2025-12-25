# Task 1 — Service Skeleton & Health Endpoint

## Overview

This task establishes the foundation for the job queue backend service by implementing a minimal but production-ready Go HTTP service with a health check endpoint and graceful shutdown handling.

## ✅ Completed Requirements

### Functional Requirements
- ✅ `GET /health` endpoint implemented
- ✅ Returns `200 OK` status
- ✅ JSON response: `{"status": "ok"}`
- ✅ Sets `Content-Type: application/json` header

### Technical Requirements
- ✅ Uses standard library only (`net/http`, `encoding/json`)
- ✅ Configurable port via `PORT` environment variable (default: 8080)
- ✅ Uses `http.Server` struct (not `http.ListenAndServe` directly)
- ✅ Graceful shutdown handling (SIGINT and SIGTERM)
- ✅ Context-based shutdown with 10-second timeout
- ✅ Proper error handling (no panics, no ignored errors)
- ✅ HTTP method validation (only GET allowed)

## 📁 Project Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   └── http/
│       └── handler.go       # HTTP handlers
├── docs/
│   ├── task1/
│   │   ├── README.md        # This file
│   │   └── concepts/        # Detailed concept explanations
│   └── learnings.md         # Quick reference
└── go.mod                   # Go module definition
```

**Structure is idiomatic Go:**
- `cmd/` for main applications
- `internal/` for private application code
- Clear separation of concerns

## 🔑 Key Concepts Learned

### 1. HTTP Server Setup
- **`http.Server` vs `http.ListenAndServe`**: Use `http.Server` struct for production (allows graceful shutdown)
- **Goroutines**: Start server in goroutine so main can handle signals
- **Environment variables**: Read port from `PORT` env var with fallback to 8080

### 2. Graceful Shutdown
- **Signal handling**: Catch SIGINT (Ctrl+C) and SIGTERM (container shutdown)
- **Context with timeout**: Use `context.WithTimeout()` to limit shutdown duration
- **Server shutdown**: `srv.Shutdown(ctx)` stops accepting new connections and waits for in-flight requests

### 3. HTTP Handlers
- **Handler signature**: `func(w http.ResponseWriter, r *http.Request)`
- **Method validation**: Check `r.Method` to enforce HTTP method restrictions
- **Response writing order**: Set headers → Set status → Write body
- **Context handling**: Use `r.Context()` to handle request cancellation

### 4. JSON Encoding
- **Marshal vs Encoder**: 
  - `json.Marshal()` - Memory-based, good for small data
  - `json.NewEncoder()` - Stream-based, good for large data
- **Error handling**: Check encoding errors before writing to response
- **Current approach**: Using `json.Marshal()` for simple health check response

### 5. Error Handling
- **No ignored errors**: Always check `err != nil`
- **Distinguish error types**: `http.ErrServerClosed` is expected during shutdown
- **Proper status codes**: Use appropriate HTTP status codes (200, 405, 500)

## 📝 Implementation Details

### Server Startup Flow

```
1. Read PORT from environment (default: 8080)
2. Register HTTP routes (GET /health)
3. Create http.Server instance
4. Start server in goroutine (non-blocking)
5. Set up signal handling (SIGINT, SIGTERM)
6. Wait for shutdown signal (blocks here)
7. On signal: Graceful shutdown with 10s timeout
8. Exit cleanly
```

### Health Check Handler Flow

```
1. Get request context
2. Check for context cancellation (client disconnect, server shutdown)
3. Validate HTTP method (must be GET)
4. Create response data
5. Marshal to JSON
6. Set Content-Type header
7. Write JSON response
8. Handle errors at each step
```

## 🎓 Learning Resources

Detailed explanations of all concepts are available in the [`concepts/`](./concepts/) directory:

1. **[Context](./concepts/01-context.md)** - Understanding context, timeouts, and cancellation
2. **[Goroutines and Channels](./concepts/02-goroutines-channels.md)** - Concurrency in Go
3. **[HTTP Server](./concepts/03-http-server.md)** - HTTP server implementation details
4. **[Signal Handling](./concepts/04-signal-handling.md)** - OS signals and graceful shutdown
5. **[Error Handling](./concepts/05-error-handling.md)** - Go's error handling philosophy
6. **[JSON Encoding/Decoding](./concepts/06-json-encoding-decoding.md)** - Marshal vs Encoder
7. **[Project Structure](./concepts/07-project-structure.md)** - Go project layout best practices
8. **[Context in Handlers](./concepts/08-context-in-handlers.md)** - Using context in HTTP handlers

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

### Graceful Shutdown
- Press `Ctrl+C` (sends SIGINT)
- Or send SIGTERM: `kill -TERM <pid>`
- Server waits up to 10 seconds for in-flight requests to complete

## 📋 Quick Reference Checklist

### JSON Response (Current Implementation)
- ✅ Set Content-Type: `application/json`
- ✅ Marshal data to JSON bytes
- ✅ Check for encoding errors
- ✅ Write JSON bytes to response
- ✅ Handle write errors

### Server Setup
- ✅ Read port from environment variable
- ✅ Use `http.Server` struct
- ✅ Start server in goroutine
- ✅ Handle SIGINT and SIGTERM
- ✅ Implement graceful shutdown with timeout

### Error Handling
- ✅ Check all errors
- ✅ Distinguish expected vs unexpected errors
- ✅ Use appropriate HTTP status codes
- ✅ No panics in server code

## 🔄 Future Improvements

Potential enhancements for future tasks:
- Add request logging middleware
- Add structured logging
- Add request ID tracking
- Add metrics endpoint
- Add configuration management
- Add database connection handling
- Add request validation helpers

## 📚 Additional Notes

- **Go version**: 1.25+
- **Dependencies**: Standard library only
- **Project structure**: Follows Go best practices
- **Code style**: Idiomatic Go patterns

---

For detailed explanations of any concept, see the [concepts documentation](./concepts/README.md).

