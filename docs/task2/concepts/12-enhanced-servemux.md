# Understanding Enhanced ServeMux (Go 1.22+)

## Table of Contents

1. [What is Enhanced ServeMux?](#what-is-enhanced-servemux)
2. [Before: Manual Method Checking](#before-manual-method-checking)
3. [After: Method-Specific Routing](#after-method-specific-routing)
4. [How It Works](#how-it-works)
5. [Benefits](#benefits)
6. [Our Refactoring](#our-refactoring)
7. [Common Mistakes](#common-mistakes)

---

## What is Enhanced ServeMux?

### Two Concepts

**1. Explicit Mux vs Default Mux**

- **Default mux**: `http.HandleFunc()` uses global `http.DefaultServeMux`
- **Explicit mux**: `mux := http.NewServeMux()` creates a new instance

**2. Method-Specific Routing (Go 1.22+)**

- **Before**: Routes match any HTTP method, must check manually
- **After**: Routes can specify HTTP method in pattern

### The Syntax

**Explicit Mux:**

```go
mux := http.NewServeMux()  // Create mux instance
mux.HandleFunc("/path", handler)
```

**Method-Specific Routing:**

```go
mux.HandleFunc("GET /health", handler)      // Only GET requests
mux.HandleFunc("POST /jobs", handler)       // Only POST requests
mux.HandleFunc("/path", handler)            // Any method (backward compatible)
```

**Combined (Our Approach):**

```go
mux := http.NewServeMux()  // Explicit mux
mux.HandleFunc("GET /health", handler)  // Method-specific routing
```

---

## Before: Default Mux + Manual Method Checking

### The Old Way

**Routing (Default Mux):**

```go
// Uses global http.DefaultServeMux
http.HandleFunc("/health", HealthCheckHandler)

// Server uses default mux
srv := &http.Server{
    Addr: ":" + port,
    // Handler defaults to http.DefaultServeMux
}
```

**Handler:**

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // Must check method manually
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Handler logic...
}
```

### Problems

**1. Global State (Default Mux)**

- `http.HandleFunc()` uses global `http.DefaultServeMux`
- Harder to test
- Less explicit

**2. Boilerplate Code**

- Every handler needs method checking
- Repetitive code
- Easy to forget

**3. Inconsistent**

- Some handlers might forget to check
- Different error messages
- Different status codes

**4. Runtime Errors**

- Method mismatch discovered at runtime
- Not caught at compile time

---

## After: Explicit Mux + Method-Specific Routing

### The New Way

**Routing (Explicit Mux + Method-Specific):**

```go
// 1. Create explicit mux instance
mux := http.NewServeMux()

// 2. Use method-specific routing
mux.HandleFunc("GET /health", HealthCheckHandler)
mux.HandleFunc("POST /jobs", CreateJobHandler)

// 3. Explicitly set mux as handler
srv := &http.Server{
    Addr:    ":" + port,
    Handler: mux,  // Explicit mux instance
}
```

**Handler:**

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // No method checking needed!
    // Mux already validated it's a GET request
    // Handler logic...
}
```

### Benefits

**1. No Global State**

- Explicit mux instance
- Better testability
- More explicit and clear

**2. Less Boilerplate**

- No manual method checking
- Cleaner handlers
- Less code to maintain

**3. Consistent**

- Method validation in one place (routing)
- Standard error handling
- Standard status codes

**4. Compile-Time Safety**

- Method specified at route registration
- Clear intent
- Less room for error

---

## How It Works

### Route Pattern Syntax

**Format:** `"METHOD /path"` or `"/path"`

**Examples:**

```go
mux.HandleFunc("GET /health", handler)        // GET only
mux.HandleFunc("POST /jobs", handler)         // POST only
mux.HandleFunc("PUT /jobs/:id", handler)      // PUT only (with param)
mux.HandleFunc("/path", handler)              // Any method (backward compatible)
```

### Method Matching

**When a request comes in:**

1. Mux checks the HTTP method
2. Matches against route patterns
3. If method matches, route to handler
4. If method doesn't match, return `405 Method Not Allowed`

**Example:**

```go
mux.HandleFunc("GET /health", handler)

// Request: GET /health → Routes to handler ✅
// Request: POST /health → 405 Method Not Allowed ❌
```

### Backward Compatibility

**Old patterns still work:**

```go
mux.HandleFunc("/path", handler)  // Matches any method
```

**Why?**

- Backward compatible with existing code
- Useful for handlers that accept multiple methods
- Can still use manual checking if needed

---

## Benefits

### 1. Cleaner Handlers

**Before:**

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", 405)
        return
    }
    // Handler logic
}
```

**After:**

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    // No method checking needed!
    // Handler logic
}
```

### 2. Centralized Validation

**All method validation in one place:**

```go
mux.HandleFunc("GET /health", HealthCheckHandler)
mux.HandleFunc("POST /jobs", CreateJobHandler)
mux.HandleFunc("GET /jobs/:id", GetJobHandler)
```

**Easy to see all routes and methods at a glance.**

### 3. Standard Error Handling

**Mux automatically returns:**

- `405 Method Not Allowed` for wrong method
- `404 Not Found` for unknown routes
- Consistent error format

### 4. Less Error-Prone

**Can't forget method checking:**

- Method specified at route registration
- Compile-time specification
- Less runtime errors

---

## Our Refactoring

### Before (Task 1)

**main.go:**

```go
// Using default mux (package-level)
http.HandleFunc("/health", internalhttp.HealthCheckHandler)

// Server uses default mux (no Handler field specified)
srv := &http.Server{
    Addr: ":" + port,
    // Handler defaults to http.DefaultServeMux
}
```

**handler.go:**

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    select {
    case <-ctx.Done():
        http.Error(w, "Context cancelled", http.StatusInternalServerError)
        return
    default:
    }

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Handler logic...
}
```

**Issues:**

- Using default mux (global state)
- Manual method checking in handler
- Context checking (unnecessary upfront)
- Boilerplate code

### After (Task 2)

**main.go:**

```go
// 1. Explicitly create mux instance
mux := http.NewServeMux()

// 2. Use method-specific routing (Go 1.22+)
mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)
mux.HandleFunc("POST /jobs", internalhttp.CreateJobHandler)

// 3. Explicitly set mux as server handler
srv := &http.Server{
    Addr:    ":" + port,
    Handler: mux,  // Explicit mux instance
}
```

**handler.go:**

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // No method checking needed!
    // No upfront context checking needed!
    // Just handler logic
    responseData := HealthCheckResponse{
        Status: "ok",
    }
    // ... rest of handler
}
```

**Two Refactorings:**

1. **Explicit Mux Creation**

   - Changed from default mux (`http.HandleFunc`) to explicit mux (`mux := http.NewServeMux()`)
   - Avoids global state
   - More explicit and testable

2. **Method-Specific Routing**
   - Changed from `"/health"` to `"GET /health"`
   - Method validation in routing, not handler
   - Cleaner handlers

**Benefits:**

- No global state (explicit mux)
- Cleaner handlers (no method checking)
- Method validation in routing
- Less boilerplate
- More declarative
- Better testability

---

## Common Mistakes

### Mistake 1: Forgetting to Use Mux

```go
// ❌ BAD: Still using old pattern
http.HandleFunc("/health", handler)  // No method specification
```

**Fix:**

```go
// ✅ GOOD: Use mux with method
mux := http.NewServeMux()
mux.HandleFunc("GET /health", handler)
```

### Mistake 2: Still Checking Method in Handler

```go
// ❌ BAD: Redundant checking
mux.HandleFunc("GET /health", handler)

func handler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {  // Unnecessary!
        return
    }
}
```

**Fix:**

```go
// ✅ GOOD: Trust the mux
mux.HandleFunc("GET /health", handler)

func handler(w http.ResponseWriter, r *http.Request) {
    // No method checking needed
}
```

### Mistake 3: Wrong Method in Pattern

```go
// ❌ BAD: Typo in method
mux.HandleFunc("GTE /health", handler)  // Typo: GTE instead of GET
```

**Fix:**

```go
// ✅ GOOD: Correct method
mux.HandleFunc("GET /health", handler)
```

### Mistake 4: Not Using Mux Instance

```go
// ❌ BAD: Using package-level functions
http.HandleFunc("GET /health", handler)  // Won't work with method pattern
```

**Fix:**

```go
// ✅ GOOD: Create mux instance
mux := http.NewServeMux()
mux.HandleFunc("GET /health", handler)
```

---

## Key Takeaways

1. **Explicit Mux** = Create mux instance instead of using default mux
2. **Method-Specific Routing** = `"METHOD /path"` syntax (Go 1.22+)
3. **Benefits** = No global state, less boilerplate, centralized validation, cleaner handlers
4. **Backward compatible** = Old patterns still work
5. **Automatic** = Mux handles method validation and 405 errors
6. **Declarative** = Method specified at route registration
7. **Better testability** = Explicit mux is easier to test than global state

---

## Next Steps

- Read [HTTP Handler Patterns](./11-http-handler-patterns.md) to see handlers without method checking
- Read [HTTP Request Parsing](./03-http-request-parsing.md) to see request handling patterns
