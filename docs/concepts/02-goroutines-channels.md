# Understanding Goroutines and Channels in Go

## Table of Contents
1. [Why Goroutines Exist](#why-goroutines-exist)
2. [What is a Goroutine?](#what-is-a-goroutine)
3. [How to Create Goroutines](#how-to-create-goroutines)
4. [What is a Channel?](#what-is-a-channel)
5. [Channel Operations](#channel-operations)
6. [Blocking and Non-Blocking](#blocking-and-non-blocking)
7. [Real Example: Server in Goroutine](#real-example-server-in-goroutine)
8. [Signal Channel Deep Dive](#signal-channel-deep-dive)

---

## Why Goroutines Exist

### The Problem: Concurrency

Imagine your web server needs to:
- Handle multiple HTTP requests simultaneously
- Process database queries
- Send emails
- Log events

**Without concurrency:** You handle one request, finish it, then handle the next. Slow! 😢

**With concurrency:** You handle many requests at the same time. Fast! 🚀

### Traditional Solutions (Other Languages)

**Threads (Java, C++):**
- Heavyweight (each thread uses ~1-2MB memory)
- Expensive to create/destroy
- Limited number (hundreds, maybe thousands)
- Complex synchronization

**Async/Await (JavaScript, Python):**
- Single-threaded event loop
- Can't use multiple CPU cores
- Callback hell or promise chains

### Go's Solution: Goroutines

- **Lightweight:** Each goroutine uses ~2KB memory
- **Cheap:** Can create millions of them
- **Simple:** Just use the `go` keyword
- **Efficient:** Uses multiple CPU cores automatically

---

## What is a Goroutine?

### Simple Definition

A **goroutine** is a lightweight thread managed by Go's runtime.

### Key Characteristics

1. **Independent execution** - Runs concurrently with other code
2. **Lightweight** - Very little memory overhead
3. **Managed by Go runtime** - You don't manage threads yourself
4. **Can run on different CPU cores** - True parallelism

### Mental Model

Think of a goroutine like a **worker**:

```
Main Program (main goroutine)
  ├── Worker 1 (goroutine) - handling request A
  ├── Worker 2 (goroutine) - handling request B
  ├── Worker 3 (goroutine) - handling request C
  └── Worker 4 (goroutine) - doing background task
```

All workers can work **at the same time**.

---

## How to Create Goroutines

### The `go` Keyword

```go
go functionName()
```

That's it! Just put `go` before a function call.

### Example 1: Simple Goroutine

```go
func main() {
    fmt.Println("Start")
    
    go sayHello() // This runs in a new goroutine
    
    fmt.Println("End")
    // Note: main might exit before sayHello finishes!
}

func sayHello() {
    fmt.Println("Hello from goroutine")
}
```

**What happens:**
- `main()` starts
- Prints "Start"
- Starts `sayHello()` in a new goroutine
- Immediately continues to "End"
- Program might exit before "Hello" is printed!

**Problem:** Main goroutine doesn't wait for other goroutines.

### Example 2: Anonymous Function Goroutine

```go
go func() {
    fmt.Println("Running in goroutine")
}()
```

This creates a goroutine from an **anonymous function** (function without a name).

### Example 3: Our Server Code

```go
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Server failed: %v", err)
    }
}()
```

**Why this is necessary:**
- `srv.ListenAndServe()` **blocks forever** (waits for requests)
- If we didn't use `go`, the program would never reach the shutdown code
- By using `go`, the server runs in the background
- Main goroutine can continue to signal handling code

---

## What is a Channel?

### The Problem Goroutines Need to Solve

Goroutines run independently. How do they **communicate** with each other?

**Answer:** Channels!

### Simple Definition

A **channel** is a typed conduit (pipe) that lets goroutines send and receive values to each other.

### Mental Model

Think of a channel like a **mailbox** or **queue**:

```
Goroutine A ──[send]──> [Channel] ──[receive]──> Goroutine B
```

- Goroutine A puts a value into the channel
- Goroutine B takes the value out
- The channel ensures safe communication

### Channel Types

```go
ch := make(chan int)        // Channel that carries integers
ch := make(chan string)     // Channel that carries strings
ch := make(chan os.Signal)  // Channel that carries OS signals
```

### Why Channels Are Safe

Channels are **thread-safe** (goroutine-safe):
- Only one goroutine can send/receive at a time
- Go handles all the locking internally
- No race conditions
- No manual mutexes needed

---

## Channel Operations

### Creating a Channel

```go
ch := make(chan int)
```

**Parameters:**
- `chan` - keyword for channel
- `int` - type of values this channel carries
- No size specified = **unbuffered channel** (size 0)

### Buffered vs Unbuffered

**Unbuffered (size 0):**
```go
ch := make(chan int)
```
- Sender **blocks** until receiver is ready
- Receiver **blocks** until sender is ready
- Synchronous communication

**Buffered (size > 0):**
```go
ch := make(chan int, 10)
```
- Can hold up to 10 values
- Sender only blocks if buffer is full
- Receiver only blocks if buffer is empty
- Asynchronous communication

### Sending to a Channel

```go
ch <- value
```

**What this does:**
- Puts `value` into the channel
- **Blocks** if channel is full (unbuffered or buffered and full)
- Waits until someone receives it

**Example:**
```go
ch <- 42  // Send the number 42 into channel ch
```

### Receiving from a Channel

```go
value := <-ch
```

**What this does:**
- Takes a value out of the channel
- **Blocks** if channel is empty
- Waits until someone sends a value

**Example:**
```go
signal := <-sigChan  // Receive a signal from sigChan
```

### The Arrow Direction

Remember: **Arrow points to where data goes**

```go
ch <- value  // Arrow points INTO channel (send)
value := <-ch  // Arrow points OUT OF channel (receive)
```

---

## Blocking and Non-Blocking

### What Does "Block" Mean?

When a goroutine **blocks**, it:
- Stops executing
- Waits for something to happen
- Doesn't use CPU while waiting
- Resumes when the condition is met

### Blocking Operations

**Sending to unbuffered channel:**
```go
ch := make(chan int)
ch <- 42  // BLOCKS here until someone receives
```

**Receiving from empty channel:**
```go
ch := make(chan int)
value := <-ch  // BLOCKS here until someone sends
```

### Why Blocking Is Useful

**Example: Waiting for shutdown signal**
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

<-sigChan  // BLOCKS here, waiting for signal
// Program pauses at this line
// When signal arrives, execution continues
log.Println("Shutting down...")
```

**What happens:**
1. Program reaches `<-sigChan`
2. **Blocks** (pauses) here
3. Waits for OS to send SIGINT or SIGTERM
4. When signal arrives, channel receives it
5. Execution continues to the next line

**This is exactly what we want!** The program waits patiently until it's told to shut down.

---

## Real Example: Server in Goroutine

### The Code

```go
// 4. Start server in goroutine
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Server failed: %v", err)
    }
}()
```

### Line-by-Line Breakdown

**Line 1: Create anonymous function**
```go
go func() {
```

- `func()` - anonymous function (no name)
- `go` - run this function in a new goroutine
- `{` - start of function body

**Line 2: Start the server**
```go
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
```

**What `ListenAndServe()` does:**
- Starts listening on the address (e.g., `:8080`)
- Accepts incoming HTTP connections
- **Blocks forever** (never returns unless error or shutdown)

**Why it blocks:**
- Server needs to keep running
- It's waiting for requests
- It's doing its job!

**Error handling:**
- `err != nil` - check if there was an error
- `err != http.ErrServerClosed` - but ignore the "server closed" error
- `http.ErrServerClosed` is **expected** when you call `Shutdown()`
- Any other error is a real problem

**Line 3: Handle real errors**
```go
log.Fatalf("Server failed: %v", err)
```

- Only reached if there's a real error (not shutdown)
- Logs the error and exits the program

**Line 4: Close the function**
```go
}()
```

- `}` - end of function body
- `()` - immediately call this function

### What Happens at Runtime

**Timeline:**

```
Time 0ms:  main() starts
Time 1ms:  Creates http.Server
Time 2ms:  Starts goroutine with go func() { ... }()
           ├─> New goroutine begins
           ├─> Calls srv.ListenAndServe()
           └─> Server starts listening on :8080
Time 3ms:  Main goroutine continues (doesn't wait)
           ├─> Sets up signal handling
           └─> Waits for shutdown signal
```

**Two goroutines now running:**
1. **Main goroutine:** Waiting for shutdown signal
2. **Server goroutine:** Handling HTTP requests

**When shutdown happens:**
```
Main goroutine:  Receives signal → calls srv.Shutdown()
Server goroutine: ListenAndServe() returns http.ErrServerClosed
                 └─> Error is ignored (expected)
                 └─> Goroutine exits
Main goroutine:  Continues and exits
```

### Why This Pattern Works

1. **Server runs independently** - Doesn't block main
2. **Main can handle signals** - Can receive shutdown requests
3. **Clean separation** - Server logic separate from signal logic
4. **Graceful shutdown** - Main can tell server to stop

---

## Signal Channel Deep Dive

### The Code

```go
// 5. Set up signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// 6. Wait for shutdown signal
<-sigChan
```

### Line 1: Create Channel

```go
sigChan := make(chan os.Signal, 1)
```

**Breaking this down:**
- `make(chan ...)` - create a channel
- `os.Signal` - type of values this channel carries (OS signals)
- `1` - buffer size of 1

**Why buffer size 1?**
- OS might send signal quickly
- We want to receive it even if we're not immediately reading
- Buffer of 1 prevents signal loss

**What `os.Signal` is:**
- It's an **interface** representing OS signals
- Examples: SIGINT (Ctrl+C), SIGTERM (termination request)

### Line 2: Register for Signals

```go
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
```

**Function signature:**
```go
func Notify(c chan<- os.Signal, sig ...os.Signal)
```

**Parameters:**
- `c chan<- os.Signal` - channel to send signals to (send-only channel)
- `sig ...os.Signal` - which signals to listen for (variadic)

**What this does:**
- Tells Go: "When the OS sends SIGINT or SIGTERM, put it into `sigChan`"
- Sets up the connection between OS signals and your channel

**What signals mean:**
- `os.Interrupt` - Usually SIGINT, sent when you press Ctrl+C
- `syscall.SIGTERM` - Termination request (from Docker, Kubernetes, etc.)

### Line 3: Receive Signal

```go
<-sigChan
```

**What this does:**
- **Blocks** (waits) until a signal arrives
- When signal arrives, receives it from channel
- Discards the value (we don't need it, just knowing it arrived is enough)
- Execution continues to next line

**Why we discard the value:**
- We don't care which specific signal it was
- We just need to know "shutdown requested"
- The value itself isn't used

### Complete Flow

```
1. Create channel: sigChan := make(chan os.Signal, 1)
   └─> Channel exists, empty, ready to receive

2. Register: signal.Notify(sigChan, ...)
   └─> OS now knows: "Send signals to sigChan"

3. Wait: <-sigChan
   └─> Program blocks here, waiting...

4. User presses Ctrl+C (or Docker sends SIGTERM)
   └─> OS sends SIGINT/SIGTERM

5. Go runtime receives signal
   └─> Puts signal into sigChan

6. <-sigChan receives the signal
   └─> Blocking ends, execution continues

7. Next line executes: log.Println("Shutting down...")
```

---

## Key Takeaways

1. **Goroutines are lightweight threads** - Use `go` keyword to create
2. **Channels enable communication** - Safe way for goroutines to talk
3. **Blocking is intentional** - Used to wait for events
4. **Server in goroutine** - Allows main to handle signals
5. **Signal channel pattern** - Standard way to handle OS signals

---

## Common Patterns

### Pattern 1: Background Worker

```go
go func() {
    for {
        doWork()
        time.Sleep(1 * time.Second)
    }
}()
```

### Pattern 2: Wait for Event

```go
eventChan := make(chan Event)
go func() {
    event := waitForEvent()
    eventChan <- event
}()

event := <-eventChan  // Blocks until event arrives
```

### Pattern 3: Timeout with Select

```go
select {
case result := <-resultChan:
    // Got result
case <-time.After(5 * time.Second):
    // Timeout after 5 seconds
}
```

---

## Common Mistakes

❌ **Not waiting for goroutines**
```go
go doWork()
// Program exits immediately, goroutine might not finish
```

✅ **Use channels or sync.WaitGroup to wait**
```go
done := make(chan bool)
go func() {
    doWork()
    done <- true
}()
<-done  // Wait for completion
```

❌ **Deadlock (both sides blocking)**
```go
ch := make(chan int)
ch <- 42      // Blocks waiting for receiver
value := <-ch // Never reached!
```

✅ **Use goroutines or buffered channels**
```go
ch := make(chan int, 1)  // Buffered
ch <- 42  // Doesn't block
value := <-ch
```

---

## Next Steps

- Understand [Context](./01-context.md) - How goroutines know when to stop
- Learn about [Signal Handling](./04-signal-handling.md) - OS signals in detail
- Read [HTTP Server Concepts](./03-http-server.md) - How servers work

