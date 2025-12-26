# Task 2 — Job Creation Endpoint

## Objective

Introduce the first domain concept — a **Job** — and implement an HTTP endpoint to create jobs synchronously.

This task focuses on:

- Request parsing & validation
- Domain modeling
- Handler-level error handling
- Clean separation of concerns

**Note:** No background processing yet. This is intentionally synchronous and naive.

---

## Scope

- Define a `Job` domain model
- Implement `POST /jobs` endpoint
- Validate incoming request
- Return a created job response

---

## Functional Requirements

### Create Job Endpoint

- **Endpoint:** `POST /jobs`
- **Content-Type:** `application/json`
- **Request body:**

  ```json
  {
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Hello"
    }
  }
  ```

- **Response:**
  - **Status:** `201 Created`
  - **Body:**
    ```json
    {
      "id": "<generated-id>",
      "type": "email",
      "status": "pending",
      "created_at": "<timestamp>"
    }
    ```

---

## Domain Requirements

### Job Model

At minimum, a Job must have:

- **ID** (string or UUID)
- **Type** (string)
- **Status** (string)
- **Payload** (opaque JSON)
- **CreatedAt** (timestamp)

**Status values (for now):**

- `pending`

---

## Technical Constraints

### Project Structure

You must introduce **domain separation**.

Expected additions:

```
internal/
├── domain/
│   └── job.go
├── http/
│   ├── handler.go
│   └── job_handler.go
```

### Validation Rules

- `type` must be non-empty
- `payload` must be valid JSON
- Invalid requests must return:
  - `400 Bad Request`
  - JSON error response

### Error Handling

- No panics
- No stringly-typed error responses
- Centralize error response logic if possible
- Use appropriate HTTP status codes

### ID & Time

- ID generation must be deterministic or random (your choice)
- Time must use `time.Time` in UTC

---

## Explicit Non-Goals

- Persistence (no DB, no files)
- Background workers
- Authentication
- Job processing logic
- Retry logic
- Tests

---

## Review Criteria

**PR will be blocked if:**

- Business logic lives in `main.go`
- Validation is scattered and duplicated
- Handler becomes excessively large
- Errors are returned as plain strings
- Response types are inconsistent
- Global mutable state is introduced

**Will be commented on:**

- Domain model design
- Validation approach
- Error handling patterns
- Code organization

---

## Definition of Done

- `go build ./...` succeeds
- `POST /jobs` returns `201` on valid input
- Invalid input returns structured JSON errors
- `GET /health` continues to work

---

## Deliverables

1. Feature branch: `feature/create-job-endpoint`
2. Pull request into `main`
3. PR description must include:
   - What validation logic felt awkward
   - What error handling decisions you made
   - One thing you would refactor later
