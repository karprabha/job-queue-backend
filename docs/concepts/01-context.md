# Understanding Context in Go

## Table of Contents

1. [Why Context Exists](#why-context-exists)
2. [What is a Context?](#what-is-a-context)
3. [The Three Capabilities of Context](#the-three-capabilities-of-context)
4. [context.Background() Explained](#contextbackground-explained)
5. [context.WithTimeout() Deep Dive](#contextwithtimeout-deep-dive)
6. [The Cancel Function](#the-cancel-function)
7. [Why defer cancel() Matters](#why-defer-cancel-matters)
8. [How Context Works Internally](#how-context-works-internally)
9. [Real-World Example: Server Shutdown](#real-world-example-server-shutdown)

---

## Why Context Exists

### The Problem Context Solves

Imagine you're running a web server and you need to shut it down. Here's what might be happening:

- Some HTTP requests are still being processed
- Database queries are running
- File operations are in progress
- Multiple goroutines are doing work

**Question:** How do you tell ALL of these operations to stop?

**Before Context:** There was no standard way. Each library had its own cancellation mechanism.

**With Context:** Go provides a single, standard way to signal cancellation across your entire program.

### The Core Idea

Context is like a **shared stop signal** that can be passed through function calls. It's a way for code to cooperatively agree on when to stop working.

**Key Point:** Context doesn't force-stop code. Code must check the context and stop itself. It's a polite request, not a command.

---

## What is a Context?

A `context.Context` is an **interface** (a contract) in Go that represents:

1. **A cancellation signal** - "Stop what you're doing"
2. **A deadline** - "You have until this time"
3. **Request-scoped values** - "Here's some data for this request"

### The Context Interface

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{}
    Err() error
    Value(key interface{}) interface{}
}
```

Don't worry about understanding this interface yet. Just know that any type implementing these methods can be used as a context.

---

## The Three Capabilities of Context

### 1. Cancellation Signal

A context can be **canceled**, meaning it's been told to stop.

**How code checks for cancellation:**

```go
select {
case <-ctx.Done():
    // Context was canceled, stop working
    return ctx.Err()
default:
    // Context is still valid, continue working
}
```

**What `ctx.Done()` is:**

- It's a **channel** (we'll explain channels later)
- When the context is canceled, this channel is **closed**
- Code can listen to this channel to know when to stop

**What `ctx.Err()` returns:**

- `nil` if context is still active
- `context.Canceled` if manually canceled
- `context.DeadlineExceeded` if timeout expired

### 2. Deadline / Timeout

A context can have a **deadline** - a specific time when it automatically cancels.

**Example:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// This context will automatically cancel after 10 seconds
```

**Why this is useful:**

- Prevents operations from running forever
- Ensures shutdown completes within a time limit
- Protects against hanging operations

### 3. Request-Scoped Values (Advanced)

Context can carry key-value pairs that are specific to a request.

**Example:**

```go
ctx := context.WithValue(parentCtx, "userID", 12345)
userID := ctx.Value("userID") // Gets 12345
```

**When to use:**

- Passing request IDs for logging
- Authentication tokens
- Tracing information

**⚠️ Warning:** Don't use this for passing function parameters. That's an anti-pattern.

---

## context.Background() Explained

### What It Is

```go
context.Background()
```

This is the **root context** - the starting point for all contexts.

### Properties

- **Never canceled** - It has no cancellation signal
- **No deadline** - It never expires
- **No values** - It carries no data

### Why It Exists

Contexts form a **tree structure**:

```
Background (root)
  ├── WithTimeout (child)
  │     ├── WithValue (grandchild)
  │     └── WithCancel (grandchild)
  └── WithCancel (child)
```

Every context must have a parent. `Background()` is the ultimate parent.

### When to Use It

- At the **top level** of your program (like in `main()`)
- When you need a context but don't have one yet
- As the parent for creating other contexts

**Example:**

```go
// In main() - you don't have a context yet, so start with Background
ctx := context.Background()

// Now create a timeout context from it
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
```

---

## context.WithTimeout() Deep Dive

### The Function Signature

```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
```

Let's break this down:

**Parameters:**

- `parent Context` - The parent context to build from (usually `context.Background()`)
- `timeout time.Duration` - How long until automatic cancellation (e.g., `10*time.Second`)

**Returns:**

- `Context` - A new context with a deadline
- `CancelFunc` - A function to cancel it early

### What Happens Step-by-Step

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
```

**Step 1:** Go creates a new context struct internally

- This new context is a **child** of `Background()`
- It stores the deadline: `time.Now().Add(10 * time.Second)`

**Step 2:** Go starts an internal timer

- When 10 seconds pass, the timer fires
- The context is automatically marked as canceled
- `ctx.Done()` channel is closed

**Step 3:** Go returns two things

- `ctx` - The new context you can use
- `cancel` - A function to cancel it manually (before timeout)

### Visual Timeline

```
Time 0s:  ctx created, timer starts
          └─> ctx is active ✅

Time 5s:  ctx is still active ✅
          └─> (You could call cancel() here to stop early)

Time 10s: Timer fires ⏰
          └─> ctx is automatically canceled ❌
          └─> ctx.Done() channel closes
          └─> ctx.Err() returns context.DeadlineExceeded
```

### Why Return Both ctx AND cancel?

You get two ways to stop:

1. **Automatic:** Wait for timeout (10 seconds)
2. **Manual:** Call `cancel()` immediately when done

**Example scenario:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

// Start a long operation
go doSomething(ctx)

// If it finishes in 2 seconds, cancel early (don't wait 10 seconds)
if quickFinish {
    cancel() // Manual cancellation
}

// Otherwise, wait for 10-second timeout (automatic cancellation)
```

---

## The Cancel Function

### What Is It?

```go
cancel()
```

This is a **function** (specifically a `CancelFunc`) that, when called, immediately cancels the context.

### What Happens When You Call cancel()

1. The context is marked as **canceled**
2. The `ctx.Done()` channel is **closed**
3. `ctx.Err()` returns `context.Canceled`
4. All code listening to this context is notified

### Manual vs Automatic Cancellation

**Manual (you call cancel()):**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// ... do work ...
cancel() // You decide: "We're done, stop now"
```

**Automatic (timeout expires):**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// ... do work ...
// 10 seconds pass automatically
// Context cancels itself
```

### Why You Might Cancel Early

**Scenario:** Server shutdown finishes in 2 seconds, but timeout is 10 seconds

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel() // Always clean up

if err := srv.Shutdown(ctx); err != nil {
    // Handle error
}

// If shutdown finished in 2 seconds, cancel() runs here
// This cleans up the internal timer (doesn't wait 8 more seconds)
```

**Key Point:** Calling `cancel()` after the context is already canceled (by timeout) is safe - it does nothing.

---

## Why defer cancel() Matters

### What defer Does

```go
defer cancel()
```

`defer` schedules a function to run **when the surrounding function exits**, no matter how it exits (normal return, error, panic).

### Why This Is Critical

**Without defer:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

if err := srv.Shutdown(ctx); err != nil {
    log.Fatalf("Error: %v", err) // Program exits here!
    // cancel() never called - RESOURCE LEAK! 💥
}

cancel() // Only reached if no error
```

**Problems:**

- If an error occurs, `cancel()` is never called
- The internal timer keeps running
- Resources aren't cleaned up
- This is a **memory leak**

**With defer:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel() // ✅ Always called, no matter what

if err := srv.Shutdown(ctx); err != nil {
    log.Fatalf("Error: %v", err) // Program exits
    // cancel() still runs! ✅
}
```

**Benefits:**

- `cancel()` is **guaranteed** to run
- Resources are always cleaned up
- No memory leaks
- Code is safer

### The Rule

**Always use `defer cancel()` immediately after creating a context with `WithTimeout` or `WithCancel`.**

This is a Go best practice and prevents resource leaks.

---

## How Context Works Internally

### The Internal Mechanism (Simplified)

When you create a context with timeout:

1. **Go creates a timer goroutine** that waits for the duration
2. **Go creates a channel** (`done`) that will be closed when canceled
3. **When timeout expires:**
   - Timer goroutine closes the `done` channel
   - All code listening to `ctx.Done()` receives the signal
4. **When you call cancel():**
   - The `done` channel is closed immediately
   - Timer goroutine is stopped

### How Code Listens to Context

```go
func doWork(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            // Context was canceled, stop working
            return ctx.Err()
        default:
            // Do actual work here
            processItem()
        }
    }
}
```

**What's happening:**

- `select` waits for either:
  - `ctx.Done()` to be closed (context canceled) → stop
  - `default` case → continue working
- This is **non-blocking** - it checks and continues immediately

### Why This Is Cooperative

Context doesn't magically stop your code. Your code must:

1. **Accept a context parameter**
2. **Check `ctx.Done()` periodically**
3. **Stop working when canceled**

This is why it's called "cooperative cancellation" - both sides must cooperate.

---

## Real-World Example: Server Shutdown

### The Code

```go
// 7. Graceful shutdown with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
    log.Fatalf("Shutdown error: %v", err)
}
```

### Line-by-Line Breakdown

**Line 1: Create timeout context**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
```

- Start with `Background()` (root context)
- Create a child context with 10-second timeout
- Get back: `ctx` (the context) and `cancel` (function to cancel early)

**Line 2: Ensure cleanup**

```go
defer cancel()
```

- Schedule `cancel()` to run when `main()` exits
- Guarantees cleanup no matter what happens

**Line 3-5: Shutdown with context**

```go
if err := srv.Shutdown(ctx); err != nil {
    log.Fatalf("Shutdown error: %v", err)
}
```

- Pass context to `Shutdown()`
- `Shutdown()` will:
  - Stop accepting new connections
  - Wait for existing requests to finish
  - **BUT** only until context is canceled (10 seconds max)

### What Happens During Shutdown

**Scenario 1: Shutdown completes quickly (2 seconds)**

```
Time 0s:  Shutdown starts
Time 2s:  All requests finished ✅
          └─> Shutdown returns success
          └─> defer cancel() runs (cleans up timer)
```

**Scenario 2: Shutdown takes too long (15 seconds)**

```
Time 0s:  Shutdown starts
Time 10s: Timeout expires ⏰
          └─> Context is canceled
          └─> Shutdown() stops waiting
          └─> Returns error (timeout exceeded)
          └─> Some requests might be interrupted
```

### Why This Pattern Works

1. **Prevents hanging:** Server won't wait forever
2. **Allows completion:** Gives time for requests to finish
3. **Clean resource management:** `defer cancel()` ensures cleanup
4. **Predictable behavior:** Always completes within 10 seconds

---

## Key Takeaways

1. **Context is a cancellation signal** - not a force-stop mechanism
2. **Always start with `context.Background()`** - it's the root
3. **Use `WithTimeout()` for deadlines** - prevents infinite waits
4. **Always `defer cancel()`** - prevents resource leaks
5. **Code must cooperate** - check `ctx.Done()` to respect cancellation
6. **Contexts form a tree** - every context has a parent

---

## Common Mistakes to Avoid

❌ **Forgetting defer cancel()**

```go
ctx, cancel := context.WithTimeout(...)
// Missing: defer cancel()
```

✅ **Always defer cancel()**

```go
ctx, cancel := context.WithTimeout(...)
defer cancel()
```

❌ **Using context for function parameters**

```go
ctx := context.WithValue(ctx, "userID", id) // Don't do this
```

✅ **Pass values as function parameters**

```go
func processUser(userID int) { ... } // Do this instead
```

❌ **Ignoring context cancellation**

```go
func doWork(ctx context.Context) {
    for {
        work() // Never checks ctx.Done() - bad!
    }
}
```

✅ **Check context periodically**

```go
func doWork(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            work()
        }
    }
}
```

---

## Next Steps

- Read about [Goroutines and Channels](./02-goroutines-channels.md)
- Understand [Signal Handling](./04-signal-handling.md)
- Learn about [HTTP Server Concepts](./03-http-server.md)
