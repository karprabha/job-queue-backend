# Task 1 — Service Skeleton & Health Endpoint

## Objective

Set up the initial Go service structure and implement a basic health check endpoint to validate that the server starts, responds to HTTP requests, and shuts down gracefully.

This task establishes the foundation for all future work.

---

## Scope

- Create a minimal but production-minded Go HTTP service
- Implement a `GET /health` endpoint
- Introduce a standard project layout
- Add graceful shutdown handling

---

## Functional Requirements

### Health Endpoint

- **Endpoint:** `GET /health`
- **Response status:** `200 OK`
- **Response body (JSON):**
  ```json
  {
    "status": "ok"
  }
  ```
- **Content-Type:** `application/json`

---

## Technical Constraints

### Language & Libraries

- **Language:** Go
- **HTTP server:** `net/http` (standard library only)
- **JSON handling:** `encoding/json`
- **No third-party routing frameworks**

### Project Structure

At minimum, the project must follow this structure:

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   └── http/
│       └── handler.go
├── go.mod
```

**Requirements:**

- Do not put everything in `main.go`
- Separate handlers from main application
- Use `internal/` for private application code

### Server Configuration

- HTTP port must be configurable via an environment variable
- Default port: `8080`
- Use `http.Server` struct
- Do not call `http.ListenAndServe` directly

### Graceful Shutdown

- Handle `SIGINT` and `SIGTERM`
- Shutdown the HTTP server using context with timeout
- In-flight requests must be allowed to complete
- Do not ignore context cancellation

### Error Handling

- No panics in server code
- No ignored errors
- Errors must be handled or returned explicitly
- Avoid `log.Fatal` except in `main` during startup failures

---

## Explicit Non-Goals

- Logging framework
- Middleware
- Routing framework
- Docker
- Tests (for now)

---

## Review Criteria

**PR will be blocked if:**

- God `main.go` (all logic in main)
- Global variables
- `log.Fatal` everywhere
- No context usage
- Hardcoded port
- Non-idiomatic naming

**Will be commented on:**

- Project layout
- Handler design
- Shutdown logic
- Error handling style
- Go idioms

---

## Definition of Done

- `curl localhost:8080/health` returns `200 OK` with JSON response
- Server shuts down cleanly with `Ctrl+C`
- Code compiles with `go build ./...`
- No linter errors

---

## Deliverables

1. Feature branch: `feature/health-endpoint`
2. Pull request into `main`
3. PR description must include:
   - What felt confusing?
   - What felt ugly?
   - What are you unsure about?
