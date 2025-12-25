# Understanding Request Body Size Limiting

## Table of Contents

1. [Why Limit Request Body Size?](#why-limit-request-body-size)
2. [http.MaxBytesReader Explained](#httpmaxbytesreader-explained)
3. [Security Implications](#security-implications)
4. [Error Detection](#error-detection)
5. [Choosing Appropriate Limits](#choosing-appropriate-limits)
6. [Common Mistakes](#common-mistakes)

---

## Why Limit Request Body Size?

### The Problem: DoS Attacks

**Without limits:**
- Client can send huge request bodies
- Server reads entire body into memory
- Memory exhaustion = server crash
- Easy DoS (Denial of Service) attack

**Example attack:**
```bash
# Attacker sends 10GB request
curl -X POST /jobs -d "$(dd if=/dev/zero bs=1G count=10)"
# Server tries to read 10GB into memory = crash!
```

### The Solution: Size Limits

**With limits:**
- Reject large requests early
- Protect server resources
- Prevent DoS attacks
- Better error messages

---

## http.MaxBytesReader Explained

### What It Does

`http.MaxBytesReader` wraps an `io.Reader` and limits how many bytes can be read.

```go
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB limit
```

### How It Works

**Before:**
```go
r.Body  // Can read unlimited bytes
```

**After:**
```go
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
// Now r.Body can only read 1MB
// If more is read, returns error
```

### Parameters

```go
func MaxBytesReader(w ResponseWriter, r io.ReadCloser, n int64) io.ReadCloser
```

- `w ResponseWriter` - Used to set error status if limit exceeded
- `r io.ReadCloser` - Original reader to wrap
- `n int64` - Maximum bytes (1MB = 1024*1024)

**Returns:** New reader that enforces the limit

---

## Security Implications

### DoS Protection

**Without limit:**
```go
// ❌ BAD: No protection
bodyBytes, err := io.ReadAll(r.Body)  // Can read unlimited!
```

**With limit:**
```go
// ✅ GOOD: Protected
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
bodyBytes, err := io.ReadAll(r.Body)  // Max 1MB
```

### Memory Protection

**Memory usage:**
- Without limit: Unbounded (dangerous!)
- With limit: Bounded (safe)

**Example:**
- 1MB limit = Max 1MB memory per request
- 100 concurrent requests = Max 100MB
- Without limit = Could be GBs!

---

## Error Detection

### The Error

When limit is exceeded, `io.ReadAll` returns an error with message containing "request body too large".

### Our Detection

```go
bodyBytes, err := io.ReadAll(r.Body)
if err != nil {
    if strings.Contains(err.Error(), "request body too large") {
        ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
        return
    }
    ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
    return
}
```

### Why String Matching?

**Limitation:** Go's standard library doesn't export a specific error type for this.

**Our approach:** String matching (pragmatic)

**Alternative (future):** Wrap `MaxBytesReader` to return typed error

### Status Code

**413 Request Entity Too Large:**
- Specific status for size errors
- Client knows the problem
- Better than generic 400

---

## Choosing Appropriate Limits

### Our Choice: 1MB

```go
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB
```

### Why 1MB?

**Considerations:**
- Job payloads are typically small (few KB)
- 1MB allows for reasonable payloads
- Protects against abuse
- Balance between usability and security

### Other Common Limits

**Small APIs:** 64KB - 256KB
- Simple requests
- Tight security

**Medium APIs:** 1MB - 5MB
- Our case
- JSON payloads

**Large APIs:** 10MB - 100MB
- File uploads
- Complex data

**Our choice:** 1MB = Good balance

---

## Common Mistakes

### Mistake 1: No Size Limit

```go
// ❌ BAD: No protection
bodyBytes, err := io.ReadAll(r.Body)
```

**Fix:**
```go
// ✅ GOOD: Add limit
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
bodyBytes, err := io.ReadAll(r.Body)
```

### Mistake 2: Limit Too High

```go
// ❌ BAD: Too permissive
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024) // 1GB!
```

**Fix:**
```go
// ✅ GOOD: Reasonable limit
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB
```

### Mistake 3: Not Handling Size Error

```go
// ❌ BAD: Generic error
bodyBytes, err := io.ReadAll(r.Body)
if err != nil {
    ErrorResponse(w, "Error", 500)  // Doesn't tell user it's a size issue
}
```

**Fix:**
```go
// ✅ GOOD: Specific error
if strings.Contains(err.Error(), "request body too large") {
    ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
    return
}
```

---

## Key Takeaways

1. **Always limit** request body size
2. **MaxBytesReader** = Wraps reader with limit
3. **Security** = Prevents DoS attacks
4. **1MB** = Good default for JSON APIs
5. **413 status** = Use for size errors
6. **Detect errors** = Check for "request body too large"

---

## Next Steps

- Read [HTTP Request Parsing](./03-http-request-parsing.md) to see size limiting in context
- Read [HTTP Status Codes](./06-http-status-codes.md) to understand 413 status code

