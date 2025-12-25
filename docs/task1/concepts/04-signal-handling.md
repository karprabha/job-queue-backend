# Understanding Signal Handling in Go

## Table of Contents
1. [What Are OS Signals?](#what-are-os-signals)
2. [Why We Need Signal Handling](#why-we-need-signal-handling)
3. [Common Signals Explained](#common-signals-explained)
4. [Go's signal Package](#gos-signal-package)
5. [Our Signal Handling Code](#our-signal-handling-code)
6. [How signal.Notify Works](#how-signalnotify-works)
7. [The Complete Shutdown Flow](#the-complete-shutdown-flow)

---

## What Are OS Signals?

### Simple Definition

An **OS signal** is a notification sent by the operating system (or another process) to your program to tell it something happened.

### Real-World Analogy

Think of signals like **interrupts** in real life:

- **Phone call** - Interrupts what you're doing
- **Doorbell** - Someone wants attention
- **Fire alarm** - Emergency, stop everything

OS signals work the same way - they interrupt your program to tell it something important happened.

### How Signals Work

```
Operating System
    |
    | (sends signal)
    v
Your Program
    |
    | (receives signal)
    v
Your Code (handles it)
```

**Key point:** Signals are **asynchronous** - they can arrive at any time, interrupting your program's normal flow.

---

## Why We Need Signal Handling

### The Problem Without Signal Handling

**Scenario:** Your server is running, and you want to stop it.

**Without signal handling:**
- Press Ctrl+C → Program **crashes** immediately
- Active requests are **killed** mid-processing
- Database connections are **aborted**
- Data might be **corrupted**
- No cleanup happens

**This is bad!** 😢

### The Solution With Signal Handling

**With signal handling:**
- Press Ctrl+C → Program **receives signal**
- Program **stops accepting new requests**
- Program **waits for active requests to finish**
- Program **cleans up resources**
- Program **exits gracefully**

**This is good!** ✅

### Why This Matters in Production

**Containers (Docker, Kubernetes):**
- When you stop a container, it sends SIGTERM
- Your program has a few seconds to clean up
- If you don't handle it, container is **force-killed**

**Process managers:**
- Systemd, supervisord send signals
- They expect graceful shutdown
- Force-killing looks like a crash

**User experience:**
- Users don't lose their requests
- Transactions complete
- Data stays consistent

---

## Common Signals Explained

### SIGINT (Interrupt)

**What it is:**
- Sent when user presses **Ctrl+C** in terminal
- Standard way to stop a program interactively

**When it's sent:**
- User presses Ctrl+C
- Terminal sends SIGINT to foreground process

**What programs usually do:**
- Stop what they're doing
- Clean up
- Exit

**In Go:**
```go
os.Interrupt  // This is SIGINT
```

### SIGTERM (Terminate)

**What it is:**
- **Polite** request to terminate
- Sent by system/containers to ask program to stop

**When it's sent:**
- Docker: `docker stop` sends SIGTERM
- Kubernetes: Pod termination sends SIGTERM
- Systemd: Service stop sends SIGTERM
- Process managers: Shutdown sends SIGTERM

**What programs should do:**
- Stop accepting new work
- Finish current work
- Clean up resources
- Exit gracefully

**In Go:**
```go
syscall.SIGTERM  // This is SIGTERM
```

### SIGKILL (Kill - Can't Be Caught!)

**What it is:**
- **Force kill** - cannot be caught or ignored
- Last resort when program won't stop

**Important:** Your program **cannot** handle SIGKILL. It's immediately terminated.

**When it's sent:**
- `kill -9 <pid>` command
- System out of memory (OOM killer)
- After graceful shutdown timeout expires

**Why we handle SIGTERM:**
- To avoid getting SIGKILL
- If we handle SIGTERM gracefully, SIGKILL won't be needed

### Signal Comparison

| Signal | Can Catch? | Politeness | Common Source |
|--------|------------|------------|---------------|
| SIGINT | ✅ Yes | Polite | User (Ctrl+C) |
| SIGTERM | ✅ Yes | Polite | System/Containers |
| SIGKILL | ❌ No | Force | System (last resort) |

---

## Go's signal Package

### The signal Package

Go provides the `os/signal` package for handling OS signals.

**Main function we use:**
```go
signal.Notify(c chan<- os.Signal, sig ...os.Signal)
```

### Import Statement

```go
import (
    "os/signal"
    "syscall"
)
```

**Why both?**
- `os/signal` - Go's signal handling package
- `syscall` - Contains signal constants (like `SIGTERM`)

---

## Our Signal Handling Code

### The Complete Code

```go
// 5. Set up signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// 6. Wait for shutdown signal
<-sigChan
log.Println("Shutting down...")
```

### Line 1: Create Signal Channel

```go
sigChan := make(chan os.Signal, 1)
```

**Breaking this down:**

**`make(chan os.Signal, 1)`**
- `make` - Allocates memory for a channel
- `chan` - Keyword for channel type
- `os.Signal` - Type of values this channel carries
- `1` - Buffer size

**What `os.Signal` is:**
- It's an **interface** in Go
- Represents any OS signal
- Both `SIGINT` and `SIGTERM` implement this interface

**Why buffer size 1?**
- Signals can arrive quickly
- If channel is full and signal arrives, it might be lost
- Buffer of 1 ensures we can receive at least one signal even if we're not immediately reading
- Prevents signal loss

**What the channel does:**
- Acts as a **mailbox** for signals
- OS will put signals into this channel
- Our code reads from this channel

### Line 2: Register for Signals

```go
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
```

**Function signature:**
```go
func Notify(c chan<- os.Signal, sig ...os.Signal)
```

**Parameters explained:**

**`c chan<- os.Signal`**
- `chan<-` means "send-only channel"
- We're giving Go permission to **send** signals to our channel
- Go cannot read from this channel (we do that)
- This is a type safety feature

**`sig ...os.Signal`**
- `...` means "variadic" - can accept multiple signals
- We're telling Go: "Notify me about these specific signals"
- `os.Interrupt` - SIGINT (Ctrl+C)
- `syscall.SIGTERM` - SIGTERM (termination request)

**What this function does:**

1. **Registers interest** - Tells Go runtime: "I want to know about these signals"
2. **Sets up handler** - Go installs a signal handler in the OS
3. **Routes signals** - When OS sends signal, Go puts it in `sigChan`

**Important:** Before calling `signal.Notify`, signals would **kill your program**. After calling it, signals go to your channel instead.

### Line 3: Wait for Signal

```go
<-sigChan
```

**What this syntax means:**

**`<-sigChan`**
- `<-` is the "receive" operator
- Reads a value from the channel
- **Blocks** (waits) until a value is available

**What happens:**
1. Program reaches this line
2. **Blocks** (pauses execution)
3. Waits for a signal to arrive
4. When signal arrives, Go puts it in channel
5. This line receives it (we discard the value)
6. Execution continues to next line

**Why we discard the value:**
- We don't care which specific signal it was
- We just need to know "a shutdown signal arrived"
- The signal value itself isn't used

**Alternative (if you wanted to check which signal):**
```go
sig := <-sigChan
if sig == os.Interrupt {
    log.Println("Received SIGINT (Ctrl+C)")
} else if sig == syscall.SIGTERM {
    log.Println("Received SIGTERM")
}
```

But for our use case, we don't need this - both signals mean "shutdown".

---

## How signal.Notify Works

### The Internal Mechanism

**Step 1: Before signal.Notify**
```
OS sends SIGTERM → Program is killed immediately ❌
```

**Step 2: After signal.Notify**
```
OS sends SIGTERM → Go runtime catches it → Puts in sigChan → Your code receives it ✅
```

### What Go Does Internally

1. **Installs signal handler** - Registers with OS to catch signals
2. **Creates goroutine** - Background goroutine listens for signals
3. **Routes to channel** - When signal arrives, sends to your channel
4. **Preserves behavior** - Your program doesn't die, signal goes to channel

### Visual Flow

```
User presses Ctrl+C
    |
    v
OS sends SIGINT
    |
    v
Go runtime signal handler catches it
    |
    v
Go puts signal into sigChan
    |
    v
Your code: <-sigChan receives it
    |
    v
Execution continues (shutdown logic)
```

### Why This Is Non-Blocking (Before Receive)

**Before the receive:**
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
// Program continues normally here
// Can do other work
// Signals are being caught in background
```

**At the receive:**
```go
<-sigChan  // NOW it blocks, waiting for signal
```

**Key point:** `signal.Notify` itself doesn't block. Only the channel receive (`<-sigChan`) blocks.

---

## The Complete Shutdown Flow

### End-to-End Timeline

**Time 0s: Program starts**
```
1. Create server
2. Start server in goroutine
3. Create signal channel
4. Register for signals (signal.Notify)
5. Wait for signal (<-sigChan) ← BLOCKS HERE
```

**Time 30s: User presses Ctrl+C**
```
1. OS sends SIGINT
2. Go runtime catches it
3. Puts SIGINT into sigChan
4. <-sigChan receives it (unblocks)
5. Log: "Shutting down..."
6. Create shutdown context (10 second timeout)
7. Call srv.Shutdown(ctx)
   ├─> Stop accepting new connections
   ├─> Wait for active requests (max 10 seconds)
   └─> Close connections
8. Log: "Server stopped"
9. Program exits
```

### What Happens to Active Requests

**Scenario: 3 active requests when shutdown starts**

```
Request 1: Processing... (takes 2 seconds) → Finishes ✅
Request 2: Processing... (takes 5 seconds) → Finishes ✅
Request 3: Processing... (takes 15 seconds) → Timeout, interrupted ⏰
```

**Why Request 3 is interrupted:**
- Shutdown timeout is 10 seconds
- Request 3 takes 15 seconds
- Context cancels after 10 seconds
- Server stops waiting, closes connection

**This is the trade-off:**
- We give requests time to finish (graceful)
- But we don't wait forever (timeout prevents hanging)

### Alternative: Docker/Kubernetes Scenario

**Instead of Ctrl+C, container orchestrator stops the pod:**

```
1. Kubernetes sends SIGTERM to container
2. Container receives SIGTERM
3. Go runtime catches it (via signal.Notify)
4. Puts SIGTERM into sigChan
5. <-sigChan receives it
6. Graceful shutdown begins
7. Kubernetes waits (grace period, e.g., 30 seconds)
8. If shutdown completes → Container exits cleanly ✅
9. If shutdown takes too long → Kubernetes sends SIGKILL ❌
```

**Why this matters:**
- If we handle SIGTERM properly, we finish within grace period
- Container exits cleanly
- No force-kill needed
- Better for production systems

---

## Key Takeaways

1. **Signals are OS notifications** - Interrupt your program
2. **SIGINT = Ctrl+C** - User-initiated stop
3. **SIGTERM = Termination request** - System/container-initiated stop
4. **signal.Notify registers interest** - Tells Go to catch signals
5. **Channel receives signals** - Non-blocking setup, blocking receive
6. **Graceful shutdown** - Handle signals to clean up properly

---

## Common Patterns

### Pattern 1: Basic Signal Handling

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
// Shutdown logic here
```

### Pattern 2: Handle Multiple Signals Differently

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

sig := <-sigChan
switch sig {
case os.Interrupt:
    log.Println("Received SIGINT")
case syscall.SIGTERM:
    log.Println("Received SIGTERM")
case syscall.SIGHUP:
    log.Println("Received SIGHUP (reload config)")
}
```

### Pattern 3: Signal with Context

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

// Use ctx in your code
<-ctx.Done()  // Blocks until signal received
```

---

## Common Mistakes

❌ **Not handling signals**
```go
// Program just runs, Ctrl+C kills it immediately
srv.ListenAndServe()
```

✅ **Handle signals properly**
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
srv.Shutdown(ctx)
```

❌ **Unbuffered channel (might lose signals)**
```go
sigChan := make(chan os.Signal)  // No buffer
signal.Notify(sigChan, ...)
// If signal arrives when not reading, might be lost
```

✅ **Buffered channel**
```go
sigChan := make(chan os.Signal, 1)  // Buffer of 1
signal.Notify(sigChan, ...)
// Can receive signal even if not immediately reading
```

❌ **Only handling SIGINT (not SIGTERM)**
```go
signal.Notify(sigChan, os.Interrupt)  // Missing SIGTERM!
// Docker/Kubernetes send SIGTERM, won't be caught
```

✅ **Handle both**
```go
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
// Handles both user (Ctrl+C) and system (containers) signals
```

---

## Advanced: signal.NotifyContext

Go 1.16+ provides a convenience function:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

// Use ctx like any other context
<-ctx.Done()  // Blocks until signal received
```

**Benefits:**
- Combines signal handling with context
- Can be passed to other functions
- Works with context-based APIs
- Cleaner code

**Our code could be:**
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

// Start server...
go srv.ListenAndServe()

// Wait for signal
<-ctx.Done()
log.Println("Shutting down...")

// Shutdown with new timeout context
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(shutdownCtx)
```

---

## Next Steps

- Understand [Context](./01-context.md) - How shutdown timeout works
- Learn about [Goroutines and Channels](./02-goroutines-channels.md) - How signals are communicated
- Read [HTTP Server Concepts](./03-http-server.md) - How server shutdown works

