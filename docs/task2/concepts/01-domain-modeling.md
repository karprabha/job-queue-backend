# Understanding Domain Modeling in Go

## Table of Contents

1. [What is Domain Modeling?](#what-is-domain-modeling)
2. [Why Separate Domain from HTTP?](#why-separate-domain-from-http)
3. [Our Domain Model: The Job](#our-domain-model-the-job)
4. [Typed Constants: JobStatus](#typed-constants-jobstatus)
5. [Constructor Functions: NewJob](#constructor-functions-newjob)
6. [The `internal/domain` Package](#the-internaldomain-package)
7. [Design Decisions and Trade-offs](#design-decisions-and-trade-offs)
8. [Common Mistakes](#common-mistakes)

---

## What is Domain Modeling?

### The Core Idea

**Domain modeling** is the process of creating a representation of the real-world concepts your application deals with.

Think of it like this:
- **Domain** = The business concepts (Job, User, Order, etc.)
- **Model** = How we represent those concepts in code

### Example: A Job Queue System

In our job queue system, the **domain** includes:
- **Job** - A task to be processed
- **Job Status** - The current state of a job
- **Job Type** - What kind of job it is

These are **business concepts**, not technical implementation details.

### Why Model the Domain?

**Benefits:**
1. **Clarity** - Code reflects business concepts
2. **Maintainability** - Changes to business logic are localized
3. **Testability** - Domain logic can be tested independently
4. **Reusability** - Domain models can be used across different interfaces (HTTP, CLI, gRPC)

---

## Why Separate Domain from HTTP?

### The Problem

Imagine if your domain logic was mixed with HTTP code:

```go
// ❌ BAD: Domain mixed with HTTP
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP-specific code
    bodyBytes, _ := io.ReadAll(r.Body)
    
    // Domain logic mixed in
    jobID := uuid.New().String()
    jobStatus := "pending"
    createdAt := time.Now().UTC()
    
    // HTTP response code
    json.NewEncoder(w).Encode(job)
}
```

**Problems:**
- Can't reuse domain logic in CLI or gRPC
- Hard to test domain logic independently
- HTTP concerns leak into business logic
- Changes to HTTP affect domain logic

### The Solution: Separation of Concerns

```go
// ✅ GOOD: Domain separated
// internal/domain/job.go
func NewJob(jobType string, payload json.RawMessage) *Job {
    // Pure domain logic - no HTTP!
}

// internal/http/job_handler.go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP-specific: read request
    // Call domain function
    job := domain.NewJob(request.Type, request.Payload)
    // HTTP-specific: write response
}
```

**Benefits:**
- Domain logic is reusable
- Domain logic is testable
- HTTP layer is thin (just translation)
- Changes to HTTP don't affect domain

---

## Our Domain Model: The Job

### The Job Struct

```go
type Job struct {
    ID        string
    Type      string
    Status    JobStatus
    Payload   json.RawMessage
    CreatedAt time.Time
}
```

### Breaking It Down

**`ID string`**
- Unique identifier for the job
- Generated using UUID
- Why string? UUIDs are strings, and we might want to use other ID formats later

**`Type string`**
- What kind of job this is (e.g., "email", "sms", "notification")
- Why string? Flexible - can add new types without code changes

**`Status JobStatus`**
- Current state of the job
- Why `JobStatus` type? Type safety! (We'll explain this next)

**`Payload json.RawMessage`**
- The actual data for the job
- Why `json.RawMessage`? Opaque JSON - domain doesn't need to know the structure
- This is a key design decision we'll explore in detail

**`CreatedAt time.Time`**
- When the job was created
- Why `time.Time`? Standard Go time type, supports all time operations

### Why These Fields?

**Question:** Why not include `UpdatedAt`, `CompletedAt`, `Error`?

**Answer:** **YAGNI** (You Aren't Gonna Need It)
- Task 2 only requires creating jobs
- We'll add fields as we need them
- Starting simple prevents over-engineering

---

## Typed Constants: JobStatus

### The Problem with Strings

```go
// ❌ BAD: Stringly-typed
type Job struct {
    Status string  // Can be anything: "pending", "PENDING", "pendin", "invalid"
}
```

**Problems:**
- Typos: `job.Status = "pendin"` (missing 'g')
- Case sensitivity: `"PENDING"` vs `"pending"`
- No compile-time checking
- Hard to refactor

### The Solution: Typed Constants

```go
// ✅ GOOD: Type-safe
type JobStatus string

const (
    StatusPending JobStatus = "pending"
)

type Job struct {
    Status JobStatus  // Can only be JobStatus values
}
```

### How It Works

**Step 1: Define the Type**
```go
type JobStatus string
```
- Creates a new type based on `string`
- `JobStatus` is NOT the same as `string` (type safety!)

**Step 2: Define Constants**
```go
const (
    StatusPending JobStatus = "pending"
)
```
- `StatusPending` is of type `JobStatus`
- Value is `"pending"` (the string)

**Step 3: Use in Struct**
```go
type Job struct {
    Status JobStatus
}
```

### Benefits

**1. Type Safety**
```go
// ❌ This won't compile
job.Status = "invalid"  // Error: cannot use "invalid" as JobStatus

// ✅ This works
job.Status = StatusPending  // OK!
```

**2. Autocomplete**
- IDE can suggest valid status values
- Less typing, fewer mistakes

**3. Refactoring**
- Change `"pending"` to `"queued"`? Change it in one place
- Compiler catches all usages

**4. Documentation**
- Constants document valid values
- Self-documenting code

### Converting to String

When you need a string (e.g., for JSON):

```go
// In handler
Status: string(job.Status),  // Convert JobStatus to string
```

**Why convert?**
- JSON encoding needs strings
- `JobStatus` is a type, but JSON sees it as a string
- Explicit conversion makes intent clear

---

## Constructor Functions: NewJob

### What is a Constructor?

A **constructor** is a function that creates and initializes a new instance of a type.

In Go, there's no special constructor syntax. We use regular functions, typically named `New...`.

### Our Constructor

```go
func NewJob(jobType string, jobPayload json.RawMessage) *Job {
    job := &Job{
        ID:        uuid.New().String(),
        Type:      jobType,
        Status:    StatusPending,
        Payload:   jobPayload,
        CreatedAt: time.Now().UTC(),
    }
    
    return job
}
```

### Breaking It Down

**Function Signature**
```go
func NewJob(jobType string, jobPayload json.RawMessage) *Job
```
- Takes required parameters
- Returns a pointer to `Job` (`*Job`)
- Why pointer? Allows `nil` checks, more flexible

**Creating the Job**
```go
job := &Job{
    ID:        uuid.New().String(),
    Type:      jobType,
    Status:    StatusPending,
    Payload:   jobPayload,
    CreatedAt: time.Now().UTC(),
}
```

**Line by line:**
- `ID: uuid.New().String()` - Generate UUID, convert to string
- `Type: jobType` - Use provided type
- `Status: StatusPending` - Always start as "pending"
- `Payload: jobPayload` - Store the opaque payload
- `CreatedAt: time.Now().UTC()` - Current time in UTC

**Return**
```go
return job
```
- Return the initialized job

### Why Use a Constructor?

**1. Encapsulation**
- All initialization logic in one place
- Can't forget to set required fields

**2. Validation (Future)**
```go
func NewJob(jobType string, jobPayload json.RawMessage) (*Job, error) {
    if jobType == "" {
        return nil, errors.New("job type cannot be empty")
    }
    // ... create job
}
```

**3. Defaults**
- Always sets `Status` to `StatusPending`
- Always uses UTC for time
- Consistent initialization

**4. Flexibility**
- Can change initialization logic without changing callers
- Can add validation later

### Why No Error Return?

**Current implementation:**
```go
func NewJob(...) *Job  // No error return
```

**Why?**
- Currently, `NewJob` can't fail
- All parameters are valid (validated in handler)
- UUID generation can't fail
- Time creation can't fail

**Future:**
If we add validation, we'd change to:
```go
func NewJob(...) (*Job, error)  // Can return error
```

This is a **design decision** - start simple, add complexity when needed.

---

## The `internal/domain` Package

### Package Structure

```
internal/
└── domain/
    └── job.go
```

### Why `internal/domain`?

**`internal/`**
- Go's special package name
- Cannot be imported by code outside this module
- Protects internal code from external dependencies

**`domain/`**
- Standard name for domain/business logic
- Clear purpose: business concepts
- Separated from infrastructure (HTTP, database, etc.)

### Package Design Principles

**1. Single Responsibility**
- `domain` package = business concepts only
- No HTTP, no database, no external dependencies (except standard library)

**2. Dependency Direction**
```
HTTP Layer → Domain Layer
```
- HTTP depends on domain (good!)
- Domain doesn't depend on HTTP (good!)
- Domain is the core, HTTP is the interface

**3. Pure Functions**
- Domain functions should be pure (no side effects)
- Same input = same output
- Easy to test

---

## Design Decisions and Trade-offs

### Decision 1: JobStatus as Typed String

**Choice:** `type JobStatus string`

**Alternatives:**
- `string` - Less type-safe
- `int` enum - Less readable
- `iota` constants - More complex

**Trade-off:**
- ✅ Type-safe
- ✅ Readable
- ✅ Easy to extend
- ❌ Need conversion to string for JSON

**Verdict:** Good choice for this use case.

### Decision 2: Payload as json.RawMessage

**Choice:** `Payload json.RawMessage`

**Alternatives:**
- `map[string]interface{}` - Loses type information
- Concrete struct - Too specific, breaks abstraction
- `[]byte` - Too low-level

**Trade-off:**
- ✅ Opaque (domain doesn't care about structure)
- ✅ Flexible (can store any JSON)
- ✅ Preserves JSON structure
- ❌ Can't validate structure in domain

**Verdict:** Perfect for opaque payloads.

### Decision 3: ID as String

**Choice:** `ID string`

**Alternatives:**
- `int` - Auto-increment (needs database)
- `uuid.UUID` - More type-safe but less flexible

**Trade-off:**
- ✅ Flexible (UUIDs, ULIDs, etc.)
- ✅ No database needed
- ✅ Works with JSON
- ❌ Less type-safe than `uuid.UUID`

**Verdict:** Good for now, might change later.

### Decision 4: CreatedAt as time.Time

**Choice:** `CreatedAt time.Time`

**Alternatives:**
- `int64` (Unix timestamp) - Less readable
- `string` - Loses time operations

**Trade-off:**
- ✅ Standard Go type
- ✅ Rich API (formatting, comparison, etc.)
- ✅ Timezone-aware
- ❌ Need to format for JSON

**Verdict:** Best choice.

---

## Common Mistakes

### Mistake 1: Mixing Domain and HTTP

```go
// ❌ BAD
func NewJob(w http.ResponseWriter, jobType string) *Job {
    // Domain function shouldn't know about HTTP!
}
```

**Fix:** Keep domain pure, handle HTTP in handlers.

### Mistake 2: Stringly-Typed Status

```go
// ❌ BAD
type Job struct {
    Status string  // Can be anything!
}
```

**Fix:** Use typed constants.

### Mistake 3: Forgetting UTC

```go
// ❌ BAD
CreatedAt: time.Now()  // Local time - inconsistent!
```

**Fix:** Always use `time.Now().UTC()`.

### Mistake 4: Exposing Internal Fields

```go
// ❌ BAD: Public fields that shouldn't be changed
type Job struct {
    ID string  // Should be read-only after creation
}
```

**Fix:** Use getters or make fields unexported if needed.

### Mistake 5: Over-Engineering

```go
// ❌ BAD: Too complex for current needs
type Job struct {
    ID          string
    Type        string
    Status      JobStatus
    Payload     json.RawMessage
    CreatedAt   time.Time
    UpdatedAt   time.Time      // Not needed yet!
    CompletedAt *time.Time     // Not needed yet!
    Error       *string         // Not needed yet!
    RetryCount  int             // Not needed yet!
}
```

**Fix:** Start simple, add fields as needed (YAGNI principle).

---

## Key Takeaways

1. **Domain modeling** = Representing business concepts in code
2. **Separate domain from infrastructure** = Reusable, testable code
3. **Typed constants** = Type safety and compile-time checking
4. **Constructor functions** = Encapsulated initialization
5. **`internal/domain`** = Standard pattern for business logic
6. **Start simple** = Add complexity when needed (YAGNI)

---

## Next Steps

- Read [Domain Separation](./10-domain-separation.md) for more on architecture
- Read [JSON RawMessage](./02-json-rawmessage.md) to understand opaque payloads
- Read [Time Handling](./09-time-handling.md) for UTC and formatting

