# Interface Design for Storage

## Table of Contents

1. [What is an Interface?](#what-is-an-interface)
2. [Why Use Interfaces for Storage?](#why-use-interfaces-for-storage)
3. [Our JobStore Interface](#our-jobstore-interface)
4. [Interface vs Concrete Type](#interface-vs-concrete-type)
5. [Dependency Injection with Interfaces](#dependency-injection-with-interfaces)
6. [Testing with Interfaces](#testing-with-interfaces)
7. [Future Implementations](#future-implementations)
8. [Interface Design Principles](#interface-design-principles)
9. [Common Mistakes](#common-mistakes)

---

## What is an Interface?

### The Concept

An **interface** defines a contract - it specifies what methods a type must have, but not how they're implemented.

**Simple analogy:**
- Interface = A job description (what you need to do)
- Implementation = How you actually do it

### Interface Syntax

```go
type InterfaceName interface {
    Method1(param1 Type1) ReturnType
    Method2(param2 Type2) (ReturnType1, ReturnType2)
}
```

**Key points:**
- Defines method signatures (name, parameters, return types)
- Doesn't define implementation
- Any type with these methods satisfies the interface

### Example

```go
// Interface definition
type Writer interface {
    Write(data []byte) (int, error)
}

// Any type with Write() method satisfies Writer
type File struct { }
func (f *File) Write(data []byte) (int, error) { ... }

type Buffer struct { }
func (b *Buffer) Write(data []byte) (int, error) { ... }

// Both File and Buffer satisfy Writer interface!
```

---

## Why Use Interfaces for Storage?

### The Problem: Tight Coupling

**Without interface:**
```go
// Handler depends on concrete type
type JobHandler struct {
    store *store.InMemoryJobStore  // Concrete type
}

func NewJobHandler(store *store.InMemoryJobStore) *JobHandler {
    return &JobHandler{store: store}
}
```

**Problems:**
- Can't swap implementations
- Hard to test (can't use mock)
- Tightly coupled to InMemoryJobStore

### The Solution: Interface

**With interface:**
```go
// Handler depends on interface
type JobHandler struct {
    store store.JobStore  // Interface!
}

func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{store: store}
}
```

**Benefits:**
- Can swap implementations
- Easy to test (can use mock)
- Loosely coupled

---

## Our JobStore Interface

### The Interface Definition

```go
type JobStore interface {
    CreateJob(ctx context.Context, job *domain.Job) error
    GetJobs(ctx context.Context) ([]domain.Job, error)
}
```

### Breaking It Down

**Method 1: CreateJob**
```go
CreateJob(ctx context.Context, job *domain.Job) error
```

- `ctx context.Context` - Context for cancellation/timeout
- `job *domain.Job` - Job to create (pointer)
- `error` - Returns error if creation fails

**Method 2: GetJobs**
```go
GetJobs(ctx context.Context) ([]domain.Job, error)
```

- `ctx context.Context` - Context for cancellation/timeout
- `[]domain.Job` - Returns slice of all jobs
- `error` - Returns error if retrieval fails

### What This Interface Says

**Contract:**
- "Any storage implementation must be able to create jobs"
- "Any storage implementation must be able to get all jobs"
- "All methods must accept context"
- "All methods must return errors"

**What it doesn't say:**
- How jobs are stored (memory, database, file, etc.)
- Whether storage is persistent
- Performance characteristics
- Internal implementation details

---

## Interface vs Concrete Type

### Using Interface (Our Approach)

```go
// Handler uses interface
type JobHandler struct {
    store store.JobStore  // Interface
}

func NewJobHandler(store store.JobStore) *JobHandler {
    return &JobHandler{store: store}
}
```

**Benefits:**
- ✅ Flexible (can inject any implementation)
- ✅ Testable (can inject mock)
- ✅ Loosely coupled

### Using Concrete Type

```go
// Handler uses concrete type
type JobHandler struct {
    store *store.InMemoryJobStore  // Concrete type
}

func NewJobHandler(store *store.InMemoryJobStore) *JobHandler {
    return &JobHandler{store: store}
}
```

**Problems:**
- ❌ Inflexible (only InMemoryJobStore)
- ❌ Hard to test (can't inject mock)
- ❌ Tightly coupled

### The Implementation

```go
// InMemoryJobStore implements JobStore interface
type InMemoryJobStore struct {
    jobs map[string]domain.Job
    mu   sync.RWMutex
}

// These methods satisfy the JobStore interface
func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // Implementation
}

func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    // Implementation
}
```

**Key point:** `InMemoryJobStore` automatically satisfies `JobStore` interface (no explicit declaration needed in Go!)

---

## Dependency Injection with Interfaces

### How It Works

**Step 1: Create implementation**
```go
// In main.go
jobStore := store.NewInMemoryJobStore()  // Concrete type
```

**Step 2: Inject as interface**
```go
// In main.go
jobHandler := internalhttp.NewJobHandler(jobStore)  // Interface type
//            ↑
//      jobStore (InMemoryJobStore) is automatically
//      treated as JobStore interface
```

**Step 3: Use interface**
```go
// In handler
func (h *JobHandler) CreateJob(...) {
    h.store.CreateJob(...)  // Calls interface method
    //     ↑
    //  Can be any implementation
}
```

### The Magic: Implicit Implementation

**Go's interfaces are implicit:**
- No need to declare "InMemoryJobStore implements JobStore"
- If type has the methods → it satisfies the interface
- Automatic and flexible

**Example:**
```go
// InMemoryJobStore has CreateJob() and GetJobs()
// → Automatically satisfies JobStore interface
// → Can be used wherever JobStore is expected
```

---

## Testing with Interfaces

### The Problem Without Interfaces

**Hard to test:**
```go
// Handler uses concrete type
type JobHandler struct {
    store *store.InMemoryJobStore  // Can't replace!
}

func TestCreateJob(t *testing.T) {
    // Must use real InMemoryJobStore
    store := store.NewInMemoryJobStore()
    handler := NewJobHandler(store)
    
    // Can't verify if CreateJob was called
    // Can't control store behavior
}
```

### The Solution: Mock with Interface

**Create mock:**
```go
// Mock implementation for testing
type MockJobStore struct {
    CreateJobFunc func(ctx context.Context, job *domain.Job) error
    GetJobsFunc   func(ctx context.Context) ([]domain.Job, error)
}

func (m *MockJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    if m.CreateJobFunc != nil {
        return m.CreateJobFunc(ctx, job)
    }
    return nil
}

func (m *MockJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    if m.GetJobsFunc != nil {
        return m.GetJobsFunc(ctx)
    }
    return []domain.Job{}, nil
}
```

**Use in tests:**
```go
func TestCreateJob(t *testing.T) {
    // Create mock
    mockStore := &MockJobStore{}
    called := false
    
    mockStore.CreateJobFunc = func(ctx context.Context, job *domain.Job) error {
        called = true
        assert.Equal(t, "email", job.Type)
        return nil
    }
    
    // Inject mock
    handler := NewJobHandler(mockStore)
    
    // Test handler
    handler.CreateJob(...)
    
    // Verify mock was called
    assert.True(t, called)
}
```

**Benefits:**
- ✅ Can verify method calls
- ✅ Can control behavior
- ✅ Fast (no real storage)
- ✅ Isolated (no side effects)

---

## Future Implementations

### Current: In-Memory Store

```go
type InMemoryJobStore struct {
    jobs map[string]domain.Job
    mu   sync.RWMutex
}
```

**Characteristics:**
- Fast (in memory)
- Temporary (lost on restart)
- Simple

### Future: Database Store

```go
type DatabaseJobStore struct {
    db *sql.DB
}

func (s *DatabaseJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // Insert into database
    _, err := s.db.ExecContext(ctx, "INSERT INTO jobs ...", ...)
    return err
}

func (s *DatabaseJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    // Query database
    rows, err := s.db.QueryContext(ctx, "SELECT * FROM jobs")
    // ...
}
```

**Same interface, different implementation!**

### Future: File Store

```go
type FileJobStore struct {
    filePath string
}

func (s *FileJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
    // Write to file
    // ...
}

func (s *FileJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
    // Read from file
    // ...
}
```

**Same interface, different implementation!**

### Switching Implementations

**No code changes needed in handler:**
```go
// Development: In-memory
store := store.NewInMemoryJobStore()
handler := NewJobHandler(store)

// Production: Database
dbStore := store.NewDatabaseStore(db)
handler := NewJobHandler(dbStore)  // Same handler code!
```

---

## Interface Design Principles

### Principle 1: Keep Interfaces Small

**❌ BAD: Large interface**
```go
type JobStore interface {
    CreateJob(...)
    GetJobs(...)
    GetJob(...)
    UpdateJob(...)
    DeleteJob(...)
    Count(...)
    Exists(...)
    // Too many methods!
}
```

**✅ GOOD: Small, focused interface**
```go
type JobStore interface {
    CreateJob(...)
    GetJobs(...)
}
```

**Why:** Smaller interfaces are easier to implement and test

### Principle 2: Accept Interfaces, Return Structs

**✅ GOOD: Accept interface**
```go
func NewJobHandler(store store.JobStore) *JobHandler
//                  ↑
//            Accept interface
```

**✅ GOOD: Return concrete type**
```go
func NewJobHandler(...) *JobStore
//                    ↑
//              Return concrete type
```

**Why:** Maximum flexibility for callers, clear return type

### Principle 3: Interface in Consumer, Implementation in Provider

**Consumer (handler):**
```go
// internal/http/job_handler.go
type JobHandler struct {
    store store.JobStore  // Interface (consumer)
}
```

**Provider (store):**
```go
// internal/store/job_store.go
type JobStore interface { ... }  // Interface definition

type InMemoryJobStore struct { ... }  // Implementation
```

**Why:** Consumer doesn't depend on implementation

### Principle 4: Don't Over-Interface

**❌ BAD: Interface for everything**
```go
type Stringer interface {
    String() string
}
// Go already has this! Don't redefine.
```

**✅ GOOD: Interface when it adds value**
```go
type JobStore interface {
    CreateJob(...)
    GetJobs(...)
}
// Needed for flexibility and testing
```

**Why:** Interfaces add complexity - only use when needed

---

## Common Mistakes

### Mistake 1: Interface Too Large

```go
// ❌ BAD: Too many methods
type JobStore interface {
    CreateJob(...)
    GetJobs(...)
    GetJob(...)
    UpdateJob(...)
    DeleteJob(...)
    Count(...)
    Exists(...)
    Search(...)
    Filter(...)
    // ...
}
```

**Fix:** Split into smaller interfaces
```go
// ✅ GOOD: Small, focused interfaces
type JobReader interface {
    GetJobs(...)
    GetJob(...)
}

type JobWriter interface {
    CreateJob(...)
    UpdateJob(...)
    DeleteJob(...)
}

type JobStore interface {
    JobReader
    JobWriter
}
```

### Mistake 2: Interface in Wrong Package

```go
// ❌ BAD: Interface in handler package
// internal/http/job_store.go
type JobStore interface { ... }
```

**Fix:** Interface in store package
```go
// ✅ GOOD: Interface in store package
// internal/store/job_store.go
type JobStore interface { ... }
```

### Mistake 3: Returning Interface

```go
// ❌ BAD: Returning interface
func NewInMemoryJobStore() store.JobStore {
    return &InMemoryJobStore{...}
}
```

**Fix:** Return concrete type
```go
// ✅ GOOD: Return concrete type
func NewInMemoryJobStore() *InMemoryJobStore {
    return &InMemoryJobStore{...}
}
```

### Mistake 4: Empty Interface

```go
// ❌ BAD: interface{} (too generic)
func Process(store interface{}) {
    // Can't call methods without type assertion
}
```

**Fix:** Use specific interface
```go
// ✅ GOOD: Specific interface
func Process(store store.JobStore) {
    store.CreateJob(...)  // Can call methods
}
```

---

## Key Takeaways

1. **Interface** = Contract defining required methods
2. **Implicit implementation** = Type satisfies interface if it has the methods
3. **Accept interfaces** = Maximum flexibility
4. **Return structs** = Clear return types
5. **Small interfaces** = Easier to implement and test
6. **Interface in consumer** = Handler depends on interface
7. **Implementation in provider** = Store provides implementation

---

## The Go Philosophy

Go's interfaces are **implicit and flexible**:

- ✅ Implicit satisfaction (no explicit declaration)
- ✅ Small interfaces preferred
- ✅ Interface segregation (many small interfaces)
- ✅ "Accept interfaces, return structs"

**Go's approach:**
- Simplicity over complexity
- Flexibility over rigidity
- Composition over inheritance
- "The bigger the interface, the weaker the abstraction"

---

## Next Steps

- Read [Dependency Injection](./01-dependency-injection.md) to see how interfaces enable DI
- Read [Handler Struct Pattern](./02-handler-struct-pattern.md) to see interfaces in action
- Read [In-Memory Storage](./03-in-memory-storage.md) to see the implementation

