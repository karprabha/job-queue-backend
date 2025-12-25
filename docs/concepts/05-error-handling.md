# Understanding Error Handling in Go

## Table of Contents
1. [Go's Error Philosophy](#gos-error-philosophy)
2. [What Are Errors in Go?](#what-are-errors-in-go)
3. [Error Handling Patterns](#error-handling-patterns)
4. [Error Handling in Our Code](#error-handling-in-our-code)
5. [Common Error Handling Mistakes](#common-error-handling-mistakes)
6. [Best Practices](#best-practices)

---

## Go's Error Philosophy

### No Exceptions!

**Unlike other languages:**
- Java, Python, JavaScript use **exceptions** (try/catch)
- Exceptions can be thrown anywhere
- Exceptions can be ignored (caught and swallowed)

**Go's approach:**
- **No exceptions** (except for panics, which are for unrecoverable errors)
- Errors are **explicit return values**
- You **must** handle errors (or explicitly ignore them)
- Errors are **just values** - not special language constructs

### Why This Design?

**Benefits:**
1. **Explicit** - You see errors in function signatures
2. **Predictable** - Errors don't hide in try/catch blocks
3. **Simple** - No complex exception hierarchies
4. **Performance** - No stack unwinding overhead

**Trade-off:**
- More verbose (must check errors everywhere)
- But: More explicit and clear

---

## What Are Errors in Go?

### The error Interface

```go
type error interface {
    Error() string
}
```

**That's it!** An error is anything that has an `Error() string` method.

### Creating Errors

**Method 1: errors.New()**
```go
err := errors.New("something went wrong")
```

**Method 2: fmt.Errorf()**
```go
err := fmt.Errorf("failed to connect: %v", connectionErr)
```

**Method 3: Returning nil (no error)**
```go
func doSomething() error {
    // Everything worked
    return nil  // No error
}
```

### Error Values

**Important concept:** Errors are **values**, not exceptions.

```go
if err != nil {
    // Handle error
}
```

**What `err != nil` means:**
- `nil` = no error (success)
- Non-`nil` = error occurred
- This is just a value comparison, not special syntax

---

## Error Handling Patterns

### Pattern 1: Check and Return

```go
func doSomething() error {
    result, err := someFunction()
    if err != nil {
        return err  // Pass error up to caller
    }
    // Continue with result
    return nil
}
```

**When to use:**
- Function can't handle the error itself
- Let caller decide what to do

### Pattern 2: Check and Handle

```go
result, err := someFunction()
if err != nil {
    log.Printf("Error: %v", err)
    // Handle error (maybe use default value, retry, etc.)
    return defaultValue
}
```

**When to use:**
- Function can handle the error
- Can recover or use fallback

### Pattern 3: Check and Wrap

```go
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

**What `%w` does:**
- Wraps the original error
- Preserves the error chain
- Caller can use `errors.Unwrap()` or `errors.Is()`

**When to use:**
- Adding context to error
- Preserving original error for debugging

### Pattern 4: Ignore (Rare!)

```go
result, _ := someFunction()  // _ discards the error
```

**⚠️ Warning:** Only do this if you're **absolutely sure** the error doesn't matter.

**Better:**
```go
result, err := someFunction()
if err != nil {
    // Log it at least
    log.Printf("Warning: %v", err)
}
```

---

## Error Handling in Our Code

### Example 1: Server Startup

```go
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
    log.Fatalf("Server failed: %v", err)
}
```

**Breaking this down:**

**`srv.ListenAndServe()`**
- Returns an `error`
- We **must** check it

**`err != nil`**
- Check if there was an error

**`err != http.ErrServerClosed`**
- `http.ErrServerClosed` is a **special error**
- It's returned when you call `Shutdown()`
- This is **expected**, not a real error
- We want to ignore this specific error

**`&&` (logical AND)**
- Both conditions must be true:
  - There IS an error (`err != nil`)
  - AND it's NOT the "server closed" error
- Only then do we treat it as a real problem

**`log.Fatalf()`**
- Logs error and **exits program**
- Used for startup failures (can't recover)

**Why this pattern:**
- Distinguishes expected errors from real failures
- Only fatal errors cause program exit
- Expected errors are silently ignored

### Example 2: JSON Encoding

```go
if err := json.NewEncoder(w).Encode(responseData); err != nil {
    http.Error(w, "Failed to encode response", http.StatusInternalServerError)
    return
}
```

**Breaking this down:**

**`json.NewEncoder(w).Encode(responseData)`**
- Returns an `error`
- Encoding can fail (rare, but possible)

**`if err != nil`**
- Check if encoding failed

**`http.Error()`**
- Sets HTTP status to 500
- Writes error message to response
- Tells client something went wrong

**`return`**
- Stop handler execution
- Don't try to write more

**Why this matters:**
- Encoding failures are real errors
- Client needs to know request failed
- We can't continue with corrupted response

### Example 3: Server Shutdown

```go
if err := srv.Shutdown(ctx); err != nil {
    log.Fatalf("Shutdown error: %v", err)
}
```

**Breaking this down:**

**`srv.Shutdown(ctx)`**
- Returns an `error`
- Can fail if:
  - Context timeout expires
  - Server is already closed
  - Other shutdown issues

**`if err != nil`**
- Check if shutdown failed

**`log.Fatalf()`**
- Logs error and exits
- Shutdown failure is critical
- Program can't continue safely

**Why fatal:**
- If shutdown fails, server might be in bad state
- Better to exit than continue with broken shutdown
- Operator needs to investigate

---

## Common Error Handling Mistakes

### Mistake 1: Ignoring Errors

❌ **Bad:**
```go
srv.ListenAndServe()  // Error ignored!
```

✅ **Good:**
```go
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
    log.Fatalf("Server failed: %v", err)
}
```

**Why:** Errors tell you what went wrong. Ignoring them hides problems.

### Mistake 2: Using _ to Discard

❌ **Bad:**
```go
_, _ = doSomething()  // Discarding both return values
```

✅ **Good:**
```go
result, err := doSomething()
if err != nil {
    // Handle error
}
```

**Why:** Even if you don't need the value, you might need to know about errors.

### Mistake 3: Not Distinguishing Error Types

❌ **Bad:**
```go
if err := srv.ListenAndServe(); err != nil {
    log.Fatalf("Error: %v", err)  // Treats all errors the same
}
```

✅ **Good:**
```go
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
    log.Fatalf("Server failed: %v", err)  // Ignores expected error
}
```

**Why:** Some errors are expected (like shutdown). Don't treat them as failures.

### Mistake 4: Not Adding Context

❌ **Bad:**
```go
if err := doSomething(); err != nil {
    return err  // No context about where error occurred
}
```

✅ **Good:**
```go
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to process request: %w", err)  // Adds context
}
```

**Why:** Context helps debug. "Failed to process request" tells you WHERE the error happened.

### Mistake 5: Using panic() for Errors

❌ **Bad:**
```go
if err != nil {
    panic(err)  // Don't do this for normal errors!
}
```

✅ **Good:**
```go
if err != nil {
    return err  // Return error, let caller handle
}
```

**Why:** `panic()` is for **unrecoverable** errors. Normal errors should be returned.

---

## Best Practices

### 1. Always Check Errors

```go
// ✅ Good
result, err := doSomething()
if err != nil {
    return err
}

// ❌ Bad
result, _ := doSomething()  // Error ignored
```

### 2. Handle Errors at the Right Level

**Low-level functions:** Return errors
```go
func readFile(filename string) ([]byte, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err  // Just return, don't handle here
    }
    return data, nil
}
```

**High-level functions:** Handle or wrap errors
```go
func processFile(filename string) error {
    data, err := readFile(filename)
    if err != nil {
        return fmt.Errorf("failed to process file %s: %w", filename, err)
    }
    // Process data...
    return nil
}
```

### 3. Add Context to Errors

```go
// ❌ Bad
if err != nil {
    return err
}

// ✅ Good
if err != nil {
    return fmt.Errorf("failed to connect to database: %w", err)
}
```

### 4. Use errors.Is() for Specific Errors

```go
if errors.Is(err, os.ErrNotExist) {
    // File doesn't exist - handle specially
}
```

### 5. Use errors.As() for Error Types

```go
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    // err is a PathError, can access pathErr.Path
}
```

### 6. Distinguish Expected vs Unexpected Errors

```go
// Expected error (shutdown) - ignore
if err != nil && err != http.ErrServerClosed {
    // Unexpected error - handle it
    log.Fatalf("Server failed: %v", err)
}
```

### 7. Don't Use log.Fatal() Except in main()

```go
// ✅ Good (in main)
if err != nil {
    log.Fatalf("Failed to start: %v", err)
}

// ❌ Bad (in other functions)
func helper() {
    if err != nil {
        log.Fatalf("Error: %v", err)  // Kills entire program!
    }
}

// ✅ Good (in other functions)
func helper() error {
    if err != nil {
        return err  // Let caller decide
    }
    return nil
}
```

---

## Error Handling Checklist

When you see an error-returning function:

- [ ] Did I check the error?
- [ ] Did I handle it appropriately?
- [ ] Did I add context if returning it?
- [ ] Did I distinguish expected vs unexpected errors?
- [ ] Did I use the right handling level (return vs handle)?

---

## Key Takeaways

1. **Errors are values** - Not exceptions, just return values
2. **Always check errors** - Don't ignore them
3. **Handle at right level** - Low-level returns, high-level handles
4. **Add context** - Wrap errors with `fmt.Errorf` and `%w`
5. **Distinguish error types** - Some errors are expected
6. **Don't panic for errors** - Return them instead

---

## Next Steps

- Review [Context](./01-context.md) - Context cancellation returns errors
- Understand [HTTP Server](./03-http-server.md) - HTTP error handling
- Learn about [Goroutines](./02-goroutines-channels.md) - Error handling in concurrent code

