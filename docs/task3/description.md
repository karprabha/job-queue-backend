# Task 3 — In-Memory Job Store & List Jobs API

## Objective

Introduce **state** into the service by storing jobs in memory and exposing a read API.

This task focuses on:

- in-memory persistence
- concurrency safety
- separating storage from HTTP
- preparing the codebase for future async workers

---

## Scope

- Introduce an in-memory job store
- Store newly created jobs from `POST /jobs`
- Implement a `GET /jobs` endpoint to list all jobs

---

## Functional Requirements

### Store Jobs

- Every job created via `POST /jobs` must be stored in memory
- Jobs must remain available for the lifetime of the process

### List Jobs Endpoint

- **Endpoint:** `GET /jobs`
- **Response status:** `200 OK`
- **Response body (JSON):**
  ```json
  [
    {
      "id": "<id>",
      "type": "<type>",
      "status": "<status>",
      "created_at": "<timestamp>"
    }
  ]
  ```
- **Order:** insertion order is acceptable
- **Empty list:** must return `[]`, not `null`

---

## Technical Constraints

### Storage Design

- Storage must live **outside** the HTTP layer
- Introduce a dedicated package for storage (example):

```
internal/
├── store/
│   └── job_store.go
```

- Store interface is optional, concrete type is acceptable

### Concurrency & Safety

- Storage must be safe for concurrent access
- Use `sync.Mutex` or `sync.RWMutex`
- No race conditions under concurrent requests

### Dependency Wiring

- HTTP handlers must NOT create the store internally
- Store must be injected into handlers during server setup
- No global variables allowed

### Error Handling

- No panics
- No ignored errors
- Handler must return valid JSON on all paths
- Internal errors must map to `500 Internal Server Error`

---

## Explicit Non-Goals

- Persistence to disk or database
- Background workers
- Job status updates
- Pagination
- Filtering or sorting
- Tests
- Performance optimization beyond correctness

---

## Review Criteria

**PR will be blocked if:**

- Global mutable state is introduced
- Store logic lives inside HTTP handlers
- Concurrency primitives are misused
- Handlers become tightly coupled to storage implementation
- `GET /jobs` duplicates logic from `POST /jobs`
- Response formats are inconsistent

**Will be commented on:**

- Store design and structure
- Concurrency safety implementation
- Dependency injection approach
- Error handling patterns

---

## Definition of Done

- `go build ./...` succeeds
- `POST /jobs` stores jobs in memory
- `GET /jobs` returns all stored jobs
- Concurrent requests do not cause data races
- Existing endpoints (`/health`, `POST /jobs`) continue to work

---

## Deliverables

1. Feature branch: `feature/in-memory-job-store`
2. Pull request into `main`
3. PR description must include:
   - Why you chose `Mutex` vs `RWMutex`
   - How the store is injected into handlers
   - One concern you have about this approach
