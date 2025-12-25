# Context in HTTP Handlers

## Table of Contents

1. [Do We Need Context in Handlers?](#do-we-need-context-in-handlers)
2. [What is Request Context?](#what-is-request-context)
3. [How to Use Context in Handlers](#how-to-use-context-in-handlers)
4. [Context Cancellation in Handlers](#context-cancellation-in-handlers)
5. [Our Handler: Should We Add Context?](#our-handler-should-we-add-context)
6. [Best Practices](#best-practices)
7. [Common Patterns](#common-patterns)

---

## Do We Need Context in Handlers?

### Short Answer

**For simple handlers (like health check):** Not strictly necessary, but **good practice**.

**For production handlers:** **Yes, you should use it.**

### Why?

1. **Request cancellation** - Client disconnects, request should stop
2. **Timeout handling** - Long-running requests should timeout
3. **Propagation** - Pass context to downstream calls (DB, API calls)
4. **Graceful shutdown** - Server shutdown cancels in-flight requests

### When Context Matters

**Simple handler (our health check):**

- Returns immediately
- No external calls
- No long operations
- **Context less critical** (but still good to have)

**Complex handler:**

- Database queries
- External API calls
- File operations
- Long computations
- **Context essential** for cancellation/timeout

---

## What is Request Context?

### The Request's Built-in Context

Every `http.Request` has a context:

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // Get the request's context
    // Use ctx...
}
```

### What This Context Contains

**1. Request-scoped values**

- Request ID (if set)
- User authentication (if set)
- Tracing information (if set)

**2. Cancellation signal**

- Cancels when client disconnects
- Cancels when server shuts down
- Cancels when timeout expires

**3. Deadline (if set)**

- Request timeout
- Server timeout
- Custom deadline

### How Request Context is Created

**Go's HTTP server automatically:**

1. Creates a context for each request
2. Cancels it when client disconnects
3. Cancels it during server shutdown
4. Sets timeouts (if configured)

**You don't create it - it's already there!**

---

## How to Use Context in Handlers

### Pattern 1: Get and Use Context

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Pass context to functions that need it
    result, err := doSomething(ctx)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Use result...
}
```

### Pattern 2: Check for Cancellation

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Check if context is canceled
    select {
    case <-ctx.Done():
        // Request was canceled (client disconnected, server shutdown, etc.)
        return
    default:
        // Continue processing
    }

    // Do work...
}
```

### Pattern 3: Pass to Downstream Calls

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Pass context to database query
    user, err := db.GetUser(ctx, userID)
    if err != nil {
        if err == context.Canceled {
            // Request was canceled, don't process
            return
        }
        http.Error(w, err.Error(), 500)
        return
    }

    // Use user...
}
```

---

## Context Cancellation in Handlers

### When Context Gets Canceled

**1. Client disconnects**

```
Client sends request
    |
    v
Client closes connection (browser closed, network issue)
    |
    v
Request context is canceled
    |
    v
Handler should stop processing
```

**2. Server shutdown**

```
Server receives shutdown signal
    |
    v
Server calls srv.Shutdown(ctx)
    |
    v
All in-flight request contexts are canceled
    |
    v
Handlers should stop processing
```

**3. Request timeout**

```
Server has ReadTimeout configured
    |
    v
Request takes too long
    |
    v
Request context is canceled
    |
    v
Handler should stop processing
```

### How to Handle Cancellation

**Option 1: Check before long operations**

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Check before expensive operation
    select {
    case <-ctx.Done():
        return  // Client disconnected, stop
    default:
    }

    // Do expensive operation...
}
```

**Option 2: Pass context and let it cancel**

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Pass context - if it cancels, this will stop
    result, err := longOperation(ctx)
    if err != nil {
        if err == context.Canceled {
            return  // Request canceled, don't send response
        }
        http.Error(w, err.Error(), 500)
        return
    }

    // Use result...
}
```

**Option 3: Use context-aware functions**

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Most Go libraries accept context
    user, err := db.QueryContext(ctx, "SELECT ...")
    if err != nil {
        if err == context.Canceled {
            return
        }
        http.Error(w, err.Error(), 500)
        return
    }

    // Use user...
}
```

---

## Our Handler: Should We Add Context?

### Current Handler

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    responseData := HealthCheckResponse{
        Status: "ok",
    }

    buffer := bytes.NewBuffer(nil)
    encoder := json.NewEncoder(buffer)
    encoder.SetIndent("", "  ")
    err := encoder.Encode(responseData)
    if err != nil {
        http.Error(w, "Failed to encode response", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(buffer.Bytes())
}
```

### Analysis

**Current state:**

- ✅ Simple and fast
- ✅ No external calls
- ✅ Returns immediately
- ❌ Doesn't check context
- ❌ Doesn't handle cancellation

**Should we add context handling?**

**Arguments FOR:**

- ✅ Good practice for all handlers
- ✅ Handles client disconnects gracefully
- ✅ Works during server shutdown
- ✅ Consistent pattern for future handlers

**Arguments AGAINST:**

- ❌ Health check is very fast (no need)
- ❌ No external calls to cancel
- ❌ Adds complexity for minimal benefit

### Recommendation

**For this specific handler:** Optional, but **good to add for consistency**.

**For future handlers:** **Definitely add context handling.**

### Updated Handler with Context

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Check if request was canceled (client disconnected, server shutdown)
    select {
    case <-ctx.Done():
        // Request canceled, don't send response
        return
    default:
        // Continue processing
    }

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    responseData := HealthCheckResponse{
        Status: "ok",
    }

    buffer := bytes.NewBuffer(nil)
    encoder := json.NewEncoder(buffer)
    encoder.SetIndent("", "  ")
    err := encoder.Encode(responseData)
    if err != nil {
        http.Error(w, "Failed to encode response", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(buffer.Bytes())
}
```

**Or simpler (if encoding is fast):**

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // For very fast handlers, context check is optional
    // But it's good practice to get it for future use

    ctx := r.Context()  // Get context (even if not used yet)

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // ... rest of handler
    // If you add DB calls later, you'll already have ctx
}
```

---

## Best Practices

### 1. Always Get Context from Request

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // ✅ Always get it
    // Use ctx...
}
```

**Why:**

- It's free (already exists)
- Ready if you need it
- Consistent pattern

### 2. Pass Context to Downstream Calls

```go
// ✅ Good
user, err := db.GetUser(ctx, userID)
apiResult, err := httpClient.Get(ctx, url)

// ❌ Bad
user, err := db.GetUser(userID)  // No context!
```

**Why:**

- Allows cancellation
- Respects timeouts
- Works with graceful shutdown

### 3. Check Cancellation Before Long Operations

```go
select {
case <-ctx.Done():
    return  // Stop if canceled
default:
    // Do long operation
}
```

**Why:**

- Avoids wasted work
- Responds quickly to cancellation
- Better resource usage

### 4. Handle Context Errors

```go
result, err := doSomething(ctx)
if err != nil {
    if err == context.Canceled {
        return  // Request canceled, don't send error response
    }
    if err == context.DeadlineExceeded {
        http.Error(w, "Request timeout", 408)
        return
    }
    http.Error(w, err.Error(), 500)
    return
}
```

**Why:**

- Distinguishes cancellation from real errors
- Appropriate HTTP status codes
- Better client experience

### 5. Don't Create New Background Context

```go
// ❌ Bad
ctx := context.Background()
db.Query(ctx, ...)

// ✅ Good
ctx := r.Context()
db.Query(ctx, ...)
```

**Why:**

- Request context has cancellation/timeout
- Background context never cancels
- Loses request-scoped values

---

## Common Patterns

### Pattern 1: Simple Handler (Our Case)

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Fast operation, context check optional but good practice
    // ... handler logic
}
```

### Pattern 2: Handler with Database

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    user, err := db.GetUser(ctx, userID)
    if err != nil {
        if err == context.Canceled {
            return
        }
        http.Error(w, err.Error(), 500)
        return
    }

    // Use user...
}
```

### Pattern 3: Handler with External API

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := httpClient.Do(req)
    if err != nil {
        if err == context.Canceled {
            return
        }
        http.Error(w, err.Error(), 500)
        return
    }
    defer resp.Body.Close()

    // Process response...
}
```

### Pattern 4: Handler with Timeout

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Add custom timeout for this operation
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    result, err := longOperation(ctx)
    if err != nil {
        if err == context.DeadlineExceeded {
            http.Error(w, "Operation timeout", 408)
            return
        }
        http.Error(w, err.Error(), 500)
        return
    }

    // Use result...
}
```

---

## Key Takeaways

1. **Request context exists automatically** - Get it with `r.Context()`
2. **Use context for cancellation** - Handles client disconnects and server shutdown
3. **Pass context downstream** - Database, API calls, etc.
4. **Check cancellation** - Before long operations
5. **Handle context errors** - Distinguish cancellation from real errors
6. **For simple handlers** - Context is optional but good practice
7. **For complex handlers** - Context is essential

---

## Common Mistakes

❌ **Ignoring context**

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    user, err := db.GetUser(userID)  // No context!
}
```

✅ **Use request context**

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user, err := db.GetUser(ctx, userID)
}
```

❌ **Creating new background context**

```go
ctx := context.Background()  // Loses request cancellation!
```

✅ **Use request context**

```go
ctx := r.Context()  // Has request cancellation
```

❌ **Not handling cancellation**

```go
result, err := doSomething(ctx)
if err != nil {
    http.Error(w, err.Error(), 500)  // Treats cancellation as error
}
```

✅ **Handle cancellation separately**

```go
result, err := doSomething(ctx)
if err != nil {
    if err == context.Canceled {
        return  // Don't send error for cancellation
    }
    http.Error(w, err.Error(), 500)
}
```

---

## Next Steps

- Review [Context](./01-context.md) - Deep dive into context
- Understand [HTTP Server](./03-http-server.md) - How request context is created
- Learn about [Error Handling](./05-error-handling.md) - Context errors
