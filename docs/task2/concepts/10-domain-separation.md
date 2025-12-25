# Understanding Domain Separation

## Table of Contents

1. [What is Domain Separation?](#what-is-domain-separation)
2. [Why Separate Domain from HTTP?](#why-separate-domain-from-http)
3. [The internal/domain Pattern](#the-internaldomain-pattern)
4. [Domain vs Infrastructure](#domain-vs-infrastructure)
5. [Clean Architecture Principles](#clean-architecture-principles)
6. [Our Implementation](#our-implementation)
7. [Common Mistakes](#common-mistakes)

---

## What is Domain Separation?

### The Core Idea

**Domain separation** = Keeping business logic separate from technical concerns.

**Layers:**

- **Domain** = Business concepts (Job, Status, etc.)
- **Infrastructure** = Technical details (HTTP, Database, etc.)

### The Goal

**Domain should:**

- Not know about HTTP
- Not know about database
- Not know about external systems
- Only know about business concepts

**Infrastructure should:**

- Know about domain
- Translate between domain and external world
- Handle technical concerns

---

## Why Separate Domain from HTTP?

### The Problem: Mixed Concerns

**Without separation:**

```go
// ❌ BAD: Everything mixed
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP code
    bodyBytes, _ := io.ReadAll(r.Body)

    // Business logic mixed in
    id := uuid.New().String()
    status := "pending"

    // HTTP response
    json.NewEncoder(w).Encode(job)
}
```

**Problems:**

- Can't reuse business logic
- Hard to test
- Changes to HTTP affect business logic
- Tight coupling

### The Solution: Separation

**With separation:**

```go
// Domain (business logic)
func NewJob(jobType string, payload json.RawMessage) *Job {
    // Pure business logic
}

// HTTP (infrastructure)
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP concerns
    job := domain.NewJob(request.Type, request.Payload)
    // HTTP response
}
```

**Benefits:**

- Reusable business logic
- Testable domain
- Independent layers
- Loose coupling

---

## The internal/domain Pattern

### Package Structure

```
internal/
└── domain/
    └── job.go
```

### Why `internal/`?

**Go's special package:**

- Cannot be imported by other modules
- Protects internal code
- Standard Go pattern

### Why `domain/`?

**Standard name:**

- Clear purpose: business logic
- Separated from infrastructure
- Easy to find

### Our Structure

```
internal/
├── domain/          # Business logic
│   └── job.go
└── http/            # Infrastructure
    ├── handler.go
    └── job_handler.go
```

**Separation:**

- `domain/` = Pure business logic
- `http/` = HTTP translation layer

---

## Domain vs Infrastructure

### Domain Layer

**What it contains:**

- Business concepts (Job, Status)
- Business rules
- Domain logic

**What it doesn't contain:**

- HTTP code
- Database code
- External API calls
- Technical details

**Our domain:**

```go
// internal/domain/job.go
type Job struct {
    ID        string
    Type      string
    Status    JobStatus
    Payload   json.RawMessage
    CreatedAt time.Time
}

func NewJob(jobType string, payload json.RawMessage) *Job {
    // Pure business logic
}
```

### Infrastructure Layer

**What it contains:**

- HTTP handlers
- Request/response translation
- External system integration
- Technical concerns

**Our infrastructure:**

```go
// internal/http/job_handler.go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP: Read request
    // Domain: Create job
    job := domain.NewJob(request.Type, request.Payload)
    // HTTP: Write response
}
```

---

## Clean Architecture Principles

### Dependency Rule

**Dependencies point inward:**

```
HTTP → Domain
```

**Not:**

```
Domain → HTTP  ❌
```

**Why?**

- Domain is core
- Infrastructure depends on domain
- Domain doesn't depend on infrastructure

### Our Dependencies

**HTTP depends on Domain:**

```go
import "github.com/karprabha/job-queue-backend/internal/domain"

job := domain.NewJob(...)  // HTTP uses domain
```

**Domain doesn't depend on HTTP:**

```go
// domain/job.go has no HTTP imports!
```

---

## Our Implementation

### Domain Layer

```go
// internal/domain/job.go
package domain

type Job struct {
    ID        string
    Type      string
    Status    JobStatus
    Payload   json.RawMessage
    CreatedAt time.Time
}

func NewJob(jobType string, jobPayload json.RawMessage) *Job {
    // Pure business logic
    // No HTTP, no database, no external dependencies
}
```

**Characteristics:**

- Pure Go types
- No HTTP imports
- No external dependencies (except standard library)
- Testable independently

### HTTP Layer

```go
// internal/http/job_handler.go
package http

import "github.com/karprabha/job-queue-backend/internal/domain"

func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // 1. HTTP: Read and parse request
    // 2. Domain: Create job
    job := domain.NewJob(request.Type, request.Payload)
    // 3. HTTP: Format and write response
}
```

**Characteristics:**

- Depends on domain
- Handles HTTP concerns
- Translates between HTTP and domain

---

## Common Mistakes

### Mistake 1: Domain Knows About HTTP

```go
// ❌ BAD: Domain imports HTTP
import "net/http"

func NewJob(w http.ResponseWriter, ...) *Job {
    // Domain shouldn't know about HTTP!
}
```

**Fix:**

```go
// ✅ GOOD: Domain is pure
func NewJob(jobType string, payload json.RawMessage) *Job {
    // No HTTP!
}
```

### Mistake 2: Business Logic in HTTP

```go
// ❌ BAD: Business logic in handler
func CreateJobHandler(...) {
    id := uuid.New().String()  // Should be in domain!
    status := "pending"         // Should be in domain!
}
```

**Fix:**

```go
// ✅ GOOD: Business logic in domain
job := domain.NewJob(request.Type, request.Payload)
// ID and status set in NewJob
```

### Mistake 3: No Separation

```go
// ❌ BAD: Everything in one file
func main() {
    // HTTP setup
    // Business logic
    // Everything mixed
}
```

**Fix:**

```go
// ✅ GOOD: Separate packages
// domain/job.go - Business logic
// http/handler.go - HTTP layer
```

---

## Key Takeaways

1. **Domain separation** = Business logic separate from infrastructure
2. **Domain** = Business concepts, no HTTP/database
3. **Infrastructure** = HTTP layer, depends on domain
4. **internal/domain** = Standard Go pattern
5. **Dependencies point inward** = Domain is core
6. **Testable** = Domain can be tested independently

---

## Next Steps

- Read [Domain Modeling](./01-domain-modeling.md) to see domain design
- Read [HTTP Handler Patterns](./11-http-handler-patterns.md) to see HTTP layer
