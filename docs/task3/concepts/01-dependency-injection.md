# Understanding Dependency Injection in Go

## Table of Contents

1. [What is Dependency Injection?](#what-is-dependency-injection)
2. [The Problem Without Dependency Injection](#the-problem-without-dependency-injection)
3. [Dependency Injection in Go](#dependency-injection-in-go)
4. [Constructor Functions for DI](#constructor-functions-for-di)
5. [Why Dependency Injection Matters](#why-dependency-injection-matters)
6. [Real Example: Our Job Handler](#real-example-our-job-handler)
7. [Dependency Injection Patterns](#dependency-injection-patterns)
8. [Common Mistakes](#common-mistakes)

---

## What is Dependency Injection?

### The Core Idea

**Dependency Injection (DI)** is a design pattern where an object receives its dependencies from the outside, rather than creating them internally.

**Simple analogy:**

- **Without DI:** A chef goes to the pantry and gets ingredients themselves
- **With DI:** Someone hands the chef the ingredients they need

### The Three Parts

1. **Dependency** - Something your code needs (like a database, store, logger)
2. **Injection** - Passing the dependency into your code
3. **Inversion** - Your code doesn't control what it gets, it receives it

### Why It's Called "Injection"

The dependency is "injected" into your code from the outside, rather than being created inside.

---

## The Problem Without Dependency Injection

### Example: Handler Creating Its Own Store

```go
// ❌ BAD: Handler creates its own store
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // Handler creates store internally
    store := store.NewInMemoryJobStore()  // Problem!

    job := domain.NewJob(...)
    store.CreateJob(job)
    // ...
}
```

### Problems with This Approach

**1. Hard to Test**

```go
// How do you test with a mock store?
// You can't - the handler always creates a real store!
func TestCreateJob(t *testing.T) {
    // Can't inject a test store - handler creates its own
}
```

**2. Tight Coupling**

- Handler is tightly coupled to `InMemoryJobStore`
- Can't swap implementations (e.g., database store)
- Changes to store creation affect handler

**3. No Control Over Store**

- Can't configure the store before passing it
- Can't share a store across multiple handlers
- Can't use a store that was created elsewhere

**4. Global State Risk**

```go
// ❌ BAD: Global variable
var globalStore = store.NewInMemoryJobStore()

func CreateJobHandler(...) {
    globalStore.CreateJob(...)  // Global mutable state!
}
```

**Problems:**

- Hard to test (can't replace with mock)
- Race conditions possible
- Hidden dependencies

---

## Dependency Injection in Go

### The Go Way

Go doesn't have fancy DI frameworks. Instead, we use **simple constructor functions** that accept dependencies as parameters.

### Basic Pattern

```go
// Step 1: Define a struct that holds dependencies
type Handler struct {
    store Store  // Dependency stored as field
}

// Step 2: Create constructor that accepts dependencies
func NewHandler(store Store) *Handler {
    return &Handler{
        store: store,  // Dependency injected here
    }
}

// Step 3: Use dependency in methods
func (h *Handler) DoSomething() {
    h.store.Create(...)  // Use injected dependency
}
```

### Breaking It Down

**Step 1: Struct with Dependency Field**

```go
type Handler struct {
    store Store  // Dependency is a field
}
```

- The struct **holds** the dependency
- Field is typically unexported (lowercase) if it shouldn't be accessed directly
- Type is usually an interface (for flexibility)

**Step 2: Constructor Function**

```go
func NewHandler(store Store) *Handler {
    return &Handler{
        store: store,
    }
}
```

- Function name starts with `New` (Go convention)
- Takes dependency as parameter
- Creates struct and sets dependency
- Returns pointer to struct

**Step 3: Using the Dependency**

```go
func (h *Handler) DoSomething() {
    h.store.Create(...)  // Access via struct field
}
```

- Access dependency through struct field
- No need to create it - it's already there

---

## Constructor Functions for DI

### Why "Constructor"?

In Go, there's no special constructor syntax. We use regular functions, typically named `New...`.

### Constructor Pattern

```go
func NewHandler(dependency Dependency) *Handler {
    return &Handler{
        dependency: dependency,
    }
}
```

### What Makes It a Constructor?

1. **Name:** Starts with `New` (Go convention)
2. **Purpose:** Creates and initializes a new instance
3. **Returns:** Pointer to the type (`*Handler`)
4. **Dependencies:** Accepts dependencies as parameters

### Multiple Dependencies

```go
type Handler struct {
    store    Store
    logger   Logger
    metrics  Metrics
}

func NewHandler(store Store, logger Logger, metrics Metrics) *Handler {
    return &Handler{
        store:    store,
        logger:   logger,
        metrics:  metrics,
    }
}
```

**Key Point:** All dependencies are passed in, none are created inside.

---

## Why Dependency Injection Matters

### 1. Testability

**Without DI:**

```go
// ❌ Can't test with mock store
func CreateJobHandler(...) {
    store := store.NewInMemoryJobStore()  // Always real store
    // ...
}
```

**With DI:**

```go
// ✅ Can inject mock store for testing
func TestCreateJob(t *testing.T) {
    mockStore := &MockStore{}  // Test implementation
    handler := NewJobHandler(mockStore)  // Inject mock

    // Test handler with mock
    handler.CreateJob(...)

    // Verify mock was called correctly
    assert.True(t, mockStore.CreateCalled)
}
```

### 2. Flexibility

**Swap Implementations:**

```go
// In-memory store for development
store := store.NewInMemoryJobStore()
handler := NewJobHandler(store)

// Database store for production
dbStore := store.NewDatabaseStore(db)
handler := NewJobHandler(dbStore)

// Same handler code, different store!
```

### 3. Control Over Lifecycle

**Create Once, Share Many:**

```go
// Create store once
jobStore := store.NewInMemoryJobStore()

// Share across multiple handlers
jobHandler := NewJobHandler(jobStore)
adminHandler := NewAdminHandler(jobStore)  // Same store!

// Both handlers use the same store instance
```

### 4. Explicit Dependencies

**Clear What's Needed:**

```go
// ✅ Clear: Handler needs a store
handler := NewJobHandler(store)

// ❌ Unclear: What does handler need?
handler := NewJobHandler()  // Where does store come from?
```

---

## Real Example: Our Job Handler

### Before: Function Handler (Task 2)

```go
// Task 2: Function handler, no dependencies
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // No store - jobs weren't persisted
    job := domain.NewJob(...)
    // Just return response
}
```

### After: Struct Handler with DI (Task 3)

```go
// Step 1: Define handler struct
type JobHandler struct {
    store store.JobStore  // Dependency field
}

// Step 2: Constructor with dependency injection
func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{
        store: store,  // Dependency injected here
    }
}

// Step 3: Methods use injected dependency
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
    job := domain.NewJob(...)
    h.store.CreateJob(r.Context(), job)  // Use injected store
}
```

### How It's Wired in main.go

```go
func main() {
    // 1. Create dependencies first
    jobStore := store.NewInMemoryJobStore()

    // 2. Inject dependencies into handlers
    jobHandler := internalhttp.NewJobHandler(jobStore)

    // 3. Register handlers
    mux.HandleFunc("POST /jobs", jobHandler.CreateJob)
    mux.HandleFunc("GET /jobs", jobHandler.GetJobs)
}
```

### The Flow

```
main() creates store
    ↓
main() creates handler (injects store)
    ↓
main() registers handler methods
    ↓
HTTP request arrives
    ↓
Handler method uses injected store
```

**Key Point:** Handler never creates store - it receives it.

---

## Dependency Injection Patterns

### Pattern 1: Constructor Injection (Our Pattern)

```go
type Handler struct {
    store Store
}

func NewHandler(store Store) *Handler {
    return &Handler{store: store}
}
```

**When to use:**

- Most common pattern
- Dependencies are required
- Clear and explicit

### Pattern 2: Optional Dependencies

```go
type Handler struct {
    store  Store
    logger Logger  // Optional
}

func NewHandler(store Store, logger Logger) *Handler {
    h := &Handler{store: store}
    if logger != nil {
        h.logger = logger
    } else {
        h.logger = defaultLogger  // Default if not provided
    }
    return h
}
```

**When to use:**

- Some dependencies are optional
- Want to provide defaults

### Pattern 3: Interface Dependencies

```go
type Handler struct {
    store Store  // Interface, not concrete type
}

func NewHandler(store Store) *Handler {
    return &Handler{store: store}
}
```

**Benefits:**

- Can inject any implementation
- Easy to mock for testing
- Flexible and testable

**Our example:**

```go
type JobHandler struct {
    store store.JobStore  // Interface!
}

// Can inject InMemoryJobStore, DatabaseStore, MockStore, etc.
```

---

## Common Mistakes

### Mistake 1: Creating Dependencies Inside

```go
// ❌ BAD: Creates dependency inside
func (h *JobHandler) CreateJob(...) {
    store := store.NewInMemoryJobStore()  // Don't do this!
    store.CreateJob(...)
}
```

**Fix:** Use injected dependency

```go
// ✅ GOOD: Uses injected dependency
func (h *JobHandler) CreateJob(...) {
    h.store.CreateJob(...)  // Use injected store
}
```

### Mistake 2: Global Variables

```go
// ❌ BAD: Global mutable state
var globalStore = store.NewInMemoryJobStore()

func CreateJobHandler(...) {
    globalStore.CreateJob(...)
}
```

**Fix:** Inject via constructor

```go
// ✅ GOOD: Dependency injection
type JobHandler struct {
    store store.JobStore
}

func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{store: store}
}
```

### Mistake 3: Forgetting to Pass Dependencies

```go
// ❌ BAD: Missing dependency
func main() {
    handler := NewJobHandler()  // Missing store parameter!
}
```

**Fix:** Always pass dependencies

```go
// ✅ GOOD: Pass all dependencies
func main() {
    store := store.NewInMemoryJobStore()
    handler := NewJobHandler(store)  // Pass store
}
```

### Mistake 4: Concrete Types Instead of Interfaces

```go
// ❌ BAD: Concrete type (hard to test)
type JobHandler struct {
    store *store.InMemoryJobStore  // Can't swap implementations
}
```

**Fix:** Use interface

```go
// ✅ GOOD: Interface (flexible)
type JobHandler struct {
    store store.JobStore  // Can inject any implementation
}
```

### Mistake 5: Creating Dependencies in main() Wrong Order

```go
// ❌ BAD: Handler created before store
func main() {
    handler := NewJobHandler(nil)  // Store doesn't exist yet!
    store := store.NewInMemoryJobStore()
}
```

**Fix:** Create dependencies first

```go
// ✅ GOOD: Create dependencies, then inject
func main() {
    store := store.NewInMemoryJobStore()  // Create first
    handler := NewJobHandler(store)       // Then inject
}
```

---

## Key Takeaways

1. **Dependency Injection** = Dependencies come from outside, not created inside
2. **Constructor functions** = Go's way of doing DI (`NewHandler(dep)`)
3. **Struct fields** = Store dependencies in struct
4. **Interfaces** = Use interfaces for dependencies (flexibility)
5. **Testability** = DI makes code testable (can inject mocks)
6. **Explicit** = Dependencies are clear in constructor signature
7. **No globals** = DI avoids global mutable state

---

## The Go Philosophy

Go favors **simplicity** and **explicitness**:

- ✅ Simple constructor functions
- ✅ Explicit dependencies in function signatures
- ✅ No magic frameworks
- ✅ Clear and readable code

**Go's approach:**

- Simple is better than complex
- Explicit is better than implicit
- Readability counts

---

## Next Steps

- Read [Handler Struct Pattern](./02-handler-struct-pattern.md) to see how DI enables struct handlers
- Read [Interface Design for Storage](./07-interface-design.md) to understand why interfaces matter
- Read [In-Memory Storage](./03-in-memory-storage.md) to see the store implementation
