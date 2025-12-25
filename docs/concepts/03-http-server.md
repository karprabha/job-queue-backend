# Understanding HTTP Server in Go

## Table of Contents
1. [HTTP Server Basics](#http-server-basics)
2. [http.Server vs http.ListenAndServe](#httpserver-vs-httplistenandserve)
3. [HTTP Handlers Explained](#http-handlers-explained)
4. [Request and Response](#request-and-response)
5. [Our Health Check Handler](#our-health-check-handler)
6. [Error Handling in HTTP](#error-handling-in-http)
7. [Server Shutdown Process](#server-shutdown-process)

---

## HTTP Server Basics

### What is an HTTP Server?

An HTTP server is a program that:
1. **Listens** on a network port (e.g., 8080)
2. **Receives** HTTP requests from clients
3. **Processes** the requests
4. **Sends** HTTP responses back

### The Request-Response Cycle

```
Client (Browser)          Server
    |                        |
    |-- GET /health -------->|
    |                        | Process request
    |                        | Create response
    |<-- 200 OK {status:ok}--|
    |                        |
```

### Why We Need a Server

- Web applications need to serve content
- APIs need to handle requests
- Services need to communicate over HTTP
- Health checks need endpoints

---

## http.Server vs http.ListenAndServe

### The Simple Way (What We Avoided)

```go
http.ListenAndServe(":8080", nil)
```

**What this does:**
- Creates a server internally
- Starts listening immediately
- **Blocks forever** (never returns)
- **No way to shut down gracefully**

**Problems:**
- Can't control shutdown
- Can't set timeouts
- Can't customize behavior
- Not suitable for production

### The Right Way (What We Use)

```go
srv := &http.Server{
    Addr: ":8080",
}

go func() {
    srv.ListenAndServe()
}()

// Later...
srv.Shutdown(ctx)
```

**What this does:**
- Creates a **server instance** we control
- Can call methods on it (like `Shutdown()`)
- Can set timeouts and other options
- Allows graceful shutdown

### Why http.Server is Better

**http.Server struct fields:**
```go
type Server struct {
    Addr         string        // Address to listen on
    Handler      Handler       // Handler to use (nil = DefaultServeMux)
    ReadTimeout  time.Duration // Max time to read request
    WriteTimeout time.Duration // Max time to write response
    IdleTimeout  time.Duration // Max time for idle connections
    // ... more fields
}
```

**Benefits:**
- **Configurable** - Set timeouts, addresses, etc.
- **Controllable** - Can call `Shutdown()` method
- **Production-ready** - Handles edge cases
- **Testable** - Can create server instances in tests

### The Key Difference

| Feature | http.ListenAndServe | http.Server |
|---------|---------------------|-------------|
| Control | None | Full control |
| Shutdown | Impossible | `Shutdown()` method |
| Configuration | Limited | Extensive |
| Use case | Quick prototypes | Production code |

---

## HTTP Handlers Explained

### What is a Handler?

A **handler** is a function that processes an HTTP request and writes a response.

### Handler Signature

```go
func Handler(w http.ResponseWriter, r *http.Request)
```

**Parameters:**
- `w http.ResponseWriter` - Used to write the response
- `r *http.Request` - Contains the incoming request data

**Return:** Nothing (void function)

### Registering Handlers

**Method 1: Using HandleFunc**
```go
http.HandleFunc("/health", HealthCheckHandler)
```

**What this does:**
- Registers `HealthCheckHandler` to handle requests to `/health`
- Uses Go's default multiplexer (`DefaultServeMux`)

**Method 2: Using Handle (with custom types)**
```go
type MyHandler struct{}

func (h MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Handle request
}

http.Handle("/health", MyHandler{})
```

### How Routing Works

When a request comes in:

1. Server receives: `GET /health`
2. Looks up handler for `/health` path
3. Calls the registered handler function
4. Handler processes and writes response

**Important:** Go matches paths **exactly** (not like regex). `/health` matches `/health`, not `/health/check`.

---

## Request and Response

### http.Request (The Incoming Request)

**What it contains:**
- HTTP method (GET, POST, etc.)
- URL path (`/health`)
- Headers
- Query parameters
- Body (for POST/PUT)
- Context (for cancellation)

**Common fields we use:**
```go
r.Method        // "GET", "POST", etc.
r.URL.Path      // "/health"
r.Header        // Request headers
r.Context()     // Request context (for cancellation)
```

### http.ResponseWriter (The Outgoing Response)

**What it does:**
- Writes HTTP status code
- Sets response headers
- Writes response body

**Important methods:**
```go
w.Header()                    // Get header map
w.Header().Set("Key", "Val") // Set a header
w.WriteHeader(200)           // Set status code
w.Write([]byte("data"))       // Write body
```

**Critical rule:** Once you call `WriteHeader()` or `Write()`, you **cannot** change the status code or headers!

### Response Writing Order

**Correct order:**
```go
// 1. Set headers first
w.Header().Set("Content-Type", "application/json")

// 2. Write status (optional, defaults to 200)
w.WriteHeader(http.StatusOK)

// 3. Write body last
w.Write(data)
```

**Wrong order:**
```go
w.Write(data)  // This writes headers automatically!
w.Header().Set("Content-Type", "application/json")  // Too late! ❌
```

---

## Our Health Check Handler

### The Complete Code

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    responseData := HealthCheckResponse{
        Status: "ok",
    }

    w.Header().Set("Content-Type", "application/json")

    if err := json.NewEncoder(w).Encode(responseData); err != nil {
        http.Error(w, "Failed to encode response", http.StatusInternalServerError)
        return
    }
}
```

### Line-by-Line Breakdown

**Line 1: Function signature**
```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request)
```

- Standard Go HTTP handler signature
- `w` - write response here
- `r` - read request from here

**Lines 2-6: Method validation**
```go
if r.Method != http.MethodGet {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
}
```

**What `r.Method` is:**
- String containing HTTP method: `"GET"`, `"POST"`, `"PUT"`, etc.

**What `http.MethodGet` is:**
- Constant defined in Go: `"GET"`

**Why we check:**
- `/health` should only accept GET requests
- POST, PUT, DELETE should be rejected
- Returns `405 Method Not Allowed` for wrong methods

**What `http.Error()` does:**
- Sets status code
- Sets `Content-Type: text/plain`
- Writes error message to body
- Convenient helper function

**Lines 8-10: Create response data**
```go
responseData := HealthCheckResponse{
    Status: "ok",
}
```

**What this is:**
- Creating a struct instance
- `HealthCheckResponse` is defined elsewhere:
  ```go
  type HealthCheckResponse struct {
      Status string `json:"status"`
  }
  ```

**JSON tags explained:**
- `` `json:"status"` `` - Tells Go: "When encoding to JSON, use key 'status'"
- Without tag, JSON key would be `"Status"` (capitalized)

**Line 12: Set content type**
```go
w.Header().Set("Content-Type", "application/json")
```

**Why this matters:**
- Tells client: "This response is JSON"
- Browsers/APIs know how to parse it
- **Must be set before writing body**

**What `w.Header()` returns:**
- A `Header` type (which is `map[string][]string`)
- `Set()` method sets a header value
- Headers are case-insensitive, but Go normalizes them

**Lines 14-18: Encode and write JSON**
```go
if err := json.NewEncoder(w).Encode(responseData); err != nil {
    http.Error(w, "Failed to encode response", http.StatusInternalServerError)
    return
}
```

**What `json.NewEncoder(w)` does:**
- Creates a JSON encoder
- Encoder writes directly to `w` (the response writer)
- More efficient than encoding to memory first

**What `Encode(responseData)` does:**
- Converts Go struct to JSON
- Writes JSON directly to response
- Returns error if encoding fails

**Why error handling matters:**
- JSON encoding can fail (rare, but possible)
- If it fails, we can still set error status
- This works because `Encode()` writes headers automatically if not set

**Implicit status code:**
- If no error, Go uses `200 OK` automatically
- We don't need `w.WriteHeader(200)` explicitly

---

## Error Handling in HTTP

### The Problem We Solved

**Original problematic code:**
```go
w.WriteHeader(http.StatusOK)  // Set status to 200
encoder := json.NewEncoder(w)
if err := encoder.Encode(&responseData); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)  // ❌ Too late!
}
```

**Why this is wrong:**
- Status 200 already written
- Can't change to 500 if encoding fails
- Client sees 200 but might get partial/corrupted response

**Our fixed code:**
```go
w.Header().Set("Content-Type", "application/json")  // Set header only
if err := json.NewEncoder(w).Encode(responseData); err != nil {
    http.Error(w, "Failed to encode response", http.StatusInternalServerError)  // ✅ Can still set 500
    return
}
// Status 200 is implicit if no error
```

**Why this works:**
- Headers set, but status not written yet
- If encoding fails, `http.Error()` can set 500 status
- If encoding succeeds, 200 is used automatically

### http.Error() Explained

```go
func Error(w ResponseWriter, error string, code int)
```

**What it does:**
1. Sets status code to `code`
2. Sets `Content-Type: text/plain`
3. Writes `error` string as body
4. **Important:** This writes headers, so call it before other writes

### Status Codes We Use

- `200 OK` - Success (implicit, default)
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error (encoding failure)

---

## Server Shutdown Process

### What Happens During Shutdown

When you call `srv.Shutdown(ctx)`:

**Step 1: Stop accepting new connections**
```
New request arrives → Server rejects it
```

**Step 2: Wait for existing requests**
```
Request 1: Still processing... (wait)
Request 2: Still processing... (wait)
Request 3: Finished ✅ (can close)
```

**Step 3: Respect context timeout**
```
If all requests finish in 2 seconds → Shutdown completes ✅
If requests take 15 seconds → Context cancels → Shutdown stops ⏰
```

**Step 4: Close connections**
```
All connections closed
Server stops
```

### Why This Is "Graceful"

**Ungraceful shutdown:**
```
Server stops immediately
├─> Active requests: KILLED ❌
├─> Database transactions: ABORTED ❌
└─> Responses: LOST ❌
```

**Graceful shutdown:**
```
Server stops accepting new requests
├─> Active requests: ALLOWED TO FINISH ✅
├─> Database transactions: COMPLETED ✅
└─> Responses: SENT ✅
```

### The Timeout Purpose

**Without timeout:**
```go
srv.Shutdown(context.Background())  // Waits forever
// If a request hangs, server never shuts down!
```

**With timeout:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
srv.Shutdown(ctx)  // Waits max 10 seconds
// If requests finish in 2s → done ✅
// If requests take 15s → timeout, force stop ⏰
```

**Why 10 seconds?**
- Enough time for most requests to finish
- Not so long that shutdown feels stuck
- Balances user experience with server availability

---

## Key Takeaways

1. **Use `http.Server` struct** - Not `http.ListenAndServe` directly
2. **Handlers process requests** - Standard signature: `func(w, r)`
3. **Set headers before body** - Order matters!
4. **Validate HTTP methods** - Reject invalid methods
5. **Handle encoding errors** - Before writing status code
6. **Graceful shutdown** - Let requests finish, but with timeout

---

## Common Patterns

### Pattern 1: Method Validation

```go
if r.Method != http.MethodGet {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
}
```

### Pattern 2: JSON Response

```go
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(data)
```

### Pattern 3: Error Handling

```go
if err := doSomething(); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}
```

---

## Common Mistakes

❌ **Writing body before headers**
```go
w.Write(data)
w.Header().Set("Content-Type", "application/json")  // Too late!
```

✅ **Set headers first**
```go
w.Header().Set("Content-Type", "application/json")
w.Write(data)
```

❌ **Ignoring encoding errors**
```go
json.NewEncoder(w).Encode(data)  // What if this fails?
```

✅ **Handle encoding errors**
```go
if err := json.NewEncoder(w).Encode(data); err != nil {
    http.Error(w, "Encoding failed", http.StatusInternalServerError)
    return
}
```

❌ **Using http.ListenAndServe directly**
```go
http.ListenAndServe(":8080", nil)  // Can't shut down!
```

✅ **Use http.Server**
```go
srv := &http.Server{Addr: ":8080"}
go srv.ListenAndServe()
// Later: srv.Shutdown(ctx)
```

---

## Next Steps

- Understand [Context](./01-context.md) - How shutdown timeout works
- Learn about [Goroutines](./02-goroutines-channels.md) - How server runs concurrently
- Read [Signal Handling](./04-signal-handling.md) - How OS signals trigger shutdown

