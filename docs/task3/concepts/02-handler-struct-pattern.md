# Handler Struct Pattern in Go

## Table of Contents

1. [Function Handlers vs Struct Handlers](#function-handlers-vs-struct-handlers)
2. [Why Use Struct Handlers?](#why-use-struct-handlers)
3. [The Struct Handler Pattern](#the-struct-handler-pattern)
4. [Method Receivers Explained](#method-receivers-explained)
5. [Real Example: Our Refactoring](#real-example-our-refactoring)
6. [When to Use Struct Handlers](#when-to-use-struct-handlers)
7. [Common Patterns](#common-patterns)
8. [Common Mistakes](#common-mistakes)

---

## Function Handlers vs Struct Handlers

### Function Handlers (Task 2)

```go
// Function handler - no state, no dependencies
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // All logic here
    job := domain.NewJob(...)
    // ...
}
```

**Characteristics:**

- Standalone function
- No state between calls
- No dependencies
- Simple but limited

### Struct Handlers (Task 3)

```go
// Struct handler - has state, has dependencies
type JobHandler struct {
    store store.JobStore
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // Can use h.store
    job := domain.NewJob(...)
    h.store.CreateJob(r.Context(), job)
}
```

**Characteristics:**

- Method on a struct
- Can hold state
- Can have dependencies
- More flexible

---

## Why Use Struct Handlers?

### The Problem with Function Handlers

**Scenario:** Handler needs a store

**Option 1: Global Variable (Bad)**

```go
// ❌ BAD: Global mutable state
var globalStore = store.NewInMemoryJobStore()

func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    globalStore.CreateJob(...)  // Uses global
}
```

**Problems:**

- Hard to test (can't replace with mock)
- Race conditions possible
- Hidden dependencies

**Option 2: Create Inside Handler (Bad)**

```go
// ❌ BAD: Creates dependency inside
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    store := store.NewInMemoryJobStore()  // New store every request!
    store.CreateJob(...)
}
```

**Problems:**

- New store for every request (loses data!)
- Can't share store across handlers
- Can't test with mock

**Option 3: Struct Handler (Good)**

```go
// ✅ GOOD: Dependency injected
type JobHandler struct {
    store store.JobStore
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    h.store.CreateJob(...)  // Uses injected store
}
```

**Benefits:**

- Store injected from outside
- Can share store across handlers
- Easy to test (can inject mock)
- No global state

---

## The Struct Handler Pattern

### The Pattern

```go
// Step 1: Define struct
type Handler struct {
    // Dependencies as fields
    dependency Dependency
}

// Step 2: Constructor
func NewHandler(dependency Dependency) *Handler {
    return &Handler{
        dependency: dependency,
    }
}

// Step 3: Handler methods
func (h *Handler) HandleMethod(w http.ResponseWriter, r *http.Request) {
    // Use h.dependency
}
```

### Breaking It Down

**Step 1: Define the Struct**

```go
type JobHandler struct {
    store store.JobStore
}
```

- Struct holds dependencies
- Fields are typically unexported (lowercase)
- Type is usually an interface

**Step 2: Constructor Function**

```go
func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{
        store: store,
    }
}
```

- Creates handler with dependencies
- Returns pointer to handler
- Dependencies injected here

**Step 3: Handler Methods**

```go
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // h is the receiver
    // Can access h.store
}
```

- Methods on the struct
- Use `h` to access struct fields
- Standard HTTP handler signature

---

## Method Receivers Explained

### What is a Receiver?

A **receiver** is a special parameter that makes a function a method.

```go
func (h *JobHandler) CreateJob(...) {
    // h is the receiver
    // This function is a method on JobHandler
}
```

### Receiver Syntax

```go
func (receiver Type) MethodName(params) returns {
    // Method body
}
```

**Parts:**

- `(receiver Type)` - The receiver declaration
- `receiver` - Name (convention: short, like `h` for handler)
- `Type` - The type this method belongs to
- `MethodName` - The method name

### Pointer Receivers

```go
func (h *JobHandler) CreateJob(...) {
    // h is a pointer to JobHandler
    // Can modify h.store, etc.
}
```

**Why pointer (`*`)?**

- Allows modifying struct fields
- More efficient (doesn't copy struct)
- Standard for handlers

### Value Receivers (Rare for Handlers)

```go
func (h JobHandler) CreateJob(...) {
    // h is a copy of JobHandler
    // Changes don't affect original
}
```

**When to use:**

- Rarely for handlers
- Usually use pointer receivers

---

## Real Example: Our Refactoring

### Before: Function Handler (Task 2)

```go
// Function handler - no dependencies
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // Parse request
    bodyBytes, _ := io.ReadAll(r.Body)
    var request CreateJobRequest
    json.Unmarshal(bodyBytes, &request)

    // Create job
    job := domain.NewJob(request.Type, request.Payload)

    // Return response (no storage)
    response := CreateJobResponse{
        ID:        job.ID,
        Type:      job.Type,
        Status:    string(job.Status),
        CreatedAt: job.CreatedAt.Format(time.RFC3339),
    }

    json.NewEncoder(w).Encode(response)
}
```

**Limitations:**

- No way to store jobs
- No dependencies
- Can't share state

### After: Struct Handler (Task 3)

```go
// Step 1: Define struct
type JobHandler struct {
    store store.JobStore  // Dependency
}

// Step 2: Constructor
func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{
        store: store,
    }
}

// Step 3: Handler method
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    // Parse request (same as before)
    bodyBytes, _ := io.ReadAll(r.Body)
    var request CreateJobRequest
    json.Unmarshal(bodyBytes, &request)

    // Create job (same as before)
    job := domain.NewJob(request.Type, request.Payload)

    // NEW: Store job using injected store
    err := h.store.CreateJob(r.Context(), job)
    if err != nil {
        ErrorResponse(w, "Failed to create job", http.StatusInternalServerError)
        return
    }

    // Return response (same as before)
    response := jobToResponse(job)
    json.NewEncoder(w).Encode(response)
}

// NEW: Additional handler method
func (h *JobHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
    jobs, err := h.store.GetJobs(r.Context())
    if err != nil {
        ErrorResponse(w, "Failed to get jobs", http.StatusInternalServerError)
        return
    }

    response := make([]JobResponse, len(jobs))
    for i, job := range jobs {
        response[i] = jobToResponse(&job)
    }

    json.NewEncoder(w).Encode(response)
}
```

### How It's Used in main.go

```go
func main() {
    // Create store
    jobStore := store.NewInMemoryJobStore()

    // Create handler (inject store)
    jobHandler := internalhttp.NewJobHandler(jobStore)

    // Register handler methods
    mux.HandleFunc("POST /jobs", jobHandler.CreateJob)
    mux.HandleFunc("GET /jobs", jobHandler.GetJobs)
}
```

**Key Points:**

- Handler created once with store
- Multiple methods can use same store
- Store shared across methods

---

## When to Use Struct Handlers

### Use Struct Handlers When:

**1. You Need Dependencies**

```go
// Handler needs store, logger, etc.
type Handler struct {
    store  Store
    logger Logger
}
```

**2. You Need State**

```go
// Handler needs to remember something
type Handler struct {
    config Config
    cache  Cache
}
```

**3. You Have Multiple Related Handlers**

```go
// Multiple methods share dependencies
type JobHandler struct {
    store store.JobStore
}

func (h *JobHandler) CreateJob(...) { }
func (h *JobHandler) GetJobs(...) { }
func (h *JobHandler) GetJob(...) { }
```

**4. You Need Testability**

```go
// Can inject mocks for testing
mockStore := &MockStore{}
handler := NewJobHandler(mockStore)
```

### Use Function Handlers When:

**1. No Dependencies**

```go
// Simple handler, no dependencies
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // No store, no logger, just return status
}
```

**2. Stateless**

```go
// Handler doesn't need to remember anything
func PingHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("pong"))
}
```

**3. Single Purpose**

```go
// One simple handler, no related handlers
func VersionHandler(w http.ResponseWriter, r *http.Request) {
    // Return version info
}
```

---

## Common Patterns

### Pattern 1: Single Handler Struct

```go
type JobHandler struct {
    store store.JobStore
}

func (h *JobHandler) CreateJob(...) { }
func (h *JobHandler) GetJobs(...) { }
func (h *JobHandler) GetJob(...) { }
```

**When:** Related handlers share dependencies

### Pattern 2: Multiple Handler Structs

```go
type JobHandler struct {
    store store.JobStore
}

type AdminHandler struct {
    store store.JobStore
    auth  AuthService
}
```

**When:** Different handlers have different dependencies

### Pattern 3: Shared Dependencies

```go
// Create once, share many
store := store.NewInMemoryJobStore()

jobHandler := NewJobHandler(store)
adminHandler := NewAdminHandler(store)  // Same store!
```

**When:** Multiple handlers need same dependency

### Pattern 4: Helper Methods

```go
type JobHandler struct {
    store store.JobStore
}

// Public handler method
func (h *JobHandler) CreateJob(...) {
    // ...
    h.store.CreateJob(...)
}

// Private helper method
func (h *JobHandler) jobToResponse(job *domain.Job) JobResponse {
    // Helper used by multiple handler methods
}
```

**When:** Need to share logic between handler methods

---

## Common Mistakes

### Mistake 1: Forgetting Receiver

```go
// ❌ BAD: No receiver, can't access struct fields
func CreateJob(w http.ResponseWriter, r *http.Request) {
    // Can't use h.store - no receiver!
}
```

**Fix:** Add receiver

```go
// ✅ GOOD: Has receiver
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    h.store.CreateJob(...)  // Can access h.store
}
```

### Mistake 2: Wrong Receiver Type

```go
// ❌ BAD: Receiver type doesn't match struct
type JobHandler struct {
    store store.JobStore
}

func (h *WrongHandler) CreateJob(...) {  // Wrong type!
}
```

**Fix:** Match receiver type

```go
// ✅ GOOD: Receiver type matches struct
func (h *JobHandler) CreateJob(...) {  // Correct type
}
```

### Mistake 3: Creating Handler Per Request

```go
// ❌ BAD: New handler every request
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    store := store.NewInMemoryJobStore()
    handler := NewJobHandler(store)  // New handler!
    handler.CreateJob(w, r)
}
```

**Fix:** Create handler once, reuse

```go
// ✅ GOOD: Handler created once
func main() {
    store := store.NewInMemoryJobStore()
    handler := NewJobHandler(store)  // Create once

    mux.HandleFunc("POST /jobs", handler.CreateJob)  // Reuse
}
```

### Mistake 4: Not Using Pointer Receiver

```go
// ❌ BAD: Value receiver (copies struct)
func (h JobHandler) CreateJob(...) {
    // Less efficient, can't modify struct
}
```

**Fix:** Use pointer receiver

```go
// ✅ GOOD: Pointer receiver
func (h *JobHandler) CreateJob(...) {
    // Efficient, can modify struct
}
```

### Mistake 5: Accessing Fields Without Receiver

```go
// ❌ BAD: Trying to access store without receiver
func CreateJob(w http.ResponseWriter, r *http.Request) {
    store.CreateJob(...)  // store doesn't exist here!
}
```

**Fix:** Use receiver

```go
// ✅ GOOD: Access via receiver
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    h.store.CreateJob(...)  // Access via h
}
```

---

## Key Takeaways

1. **Struct handlers** = Methods on structs that hold dependencies
2. **Method receivers** = `(h *Handler)` makes function a method
3. **Pointer receivers** = Standard for handlers (`*Handler`)
4. **Dependencies** = Stored as struct fields
5. **Constructor** = `NewHandler(dep)` creates handler with dependencies
6. **Multiple methods** = Can share dependencies across methods
7. **Testability** = Can inject mocks via constructor

---

## The Go Philosophy

Go favors **simplicity** and **explicitness**:

- ✅ Simple structs with methods
- ✅ Explicit dependencies in constructors
- ✅ No magic, just clear code
- ✅ Easy to understand and test

**Go's approach:**

- Methods on types (not classes)
- Explicit is better than implicit
- Simple patterns over complex frameworks

---

## Next Steps

- Read [Dependency Injection](./01-dependency-injection.md) to understand how dependencies are injected
- Read [In-Memory Storage](./03-in-memory-storage.md) to see what the handler depends on
- Read [Interface Design](./07-interface-design.md) to understand why interfaces matter
