# Understanding Error Response Centralization

## Table of Contents

1. [Why Centralize Error Responses?](#why-centralize-error-responses)
2. [The ErrorResponse Function](#the-errorresponse-function)
3. [Error Response Format](#error-response-format)
4. [HTTP Status Codes in Errors](#http-status-codes-in-errors)
5. [Fallback Error Handling](#fallback-error-handling)
6. [When Headers Are Already Written](#when-headers-are-already-written)
7. [Common Mistakes](#common-mistakes)

---

## Why Centralize Error Responses?

### The Problem: Inconsistent Errors

**Without centralization:**
```go
// Handler 1
http.Error(w, "Error message", http.StatusBadRequest)

// Handler 2
w.WriteHeader(http.StatusBadRequest)
w.Write([]byte(`{"error":"Error message"}`))

// Handler 3
json.NewEncoder(w).Encode(map[string]string{"error": "Error message"})
```

**Problems:**
1. **Inconsistent format** - Different handlers return different formats
2. **Code duplication** - Same error handling code repeated
3. **Hard to change** - Need to update every handler
4. **Easy to forget** - Forgot to set Content-Type? Forgot status code?

### The Solution: Centralized Function

```go
// ✅ GOOD: One function for all errors
ErrorResponse(w, "Error message", http.StatusBadRequest)
```

**Benefits:**
1. **Consistent format** - All errors look the same
2. **DRY** - Don't Repeat Yourself
3. **Easy to change** - Update one function
4. **Less error-prone** - Can't forget headers or format

---

## The ErrorResponse Function

### Our Implementation

```go
func ErrorResponse(w http.ResponseWriter, message string, statusCode int) {
    jsonBytes, err := json.Marshal(map[string]string{"error": message})
    if err != nil {
        // If we can't marshal, fall back to plain text error
        // Headers haven't been written yet, so http.Error is safe
        http.Error(w, "Failed to marshal error response", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    if _, err := w.Write(jsonBytes); err != nil {
        // Headers already written, can't send another response
        // Client may have disconnected - just return
        return
    }
}
```

### Breaking It Down

**Function Signature**
```go
func ErrorResponse(w http.ResponseWriter, message string, statusCode int)
```

**Parameters:**
- `w http.ResponseWriter` - Where to write the response
- `message string` - Error message for the client
- `statusCode int` - HTTP status code (400, 500, etc.)

**Why these parameters?**
- `w` - Need to write response
- `message` - User-friendly error message
- `statusCode` - HTTP semantics (400 = client error, 500 = server error)

### Step-by-Step Execution

**Step 1: Marshal Error to JSON**
```go
jsonBytes, err := json.Marshal(map[string]string{"error": message})
```

**What this does:**
- Creates a map: `{"error": "message"}`
- Marshals to JSON bytes
- Returns bytes and error

**Why a map?**
- Simple structure
- Easy to extend later (can add more fields)
- Standard JSON error format

**Step 2: Handle Marshal Error**
```go
if err != nil {
    http.Error(w, "Failed to marshal error response", http.StatusInternalServerError)
    return
}
```

**Why this fallback?**
- If we can't create JSON, we're in a bad state
- Fall back to plain text error
- Headers haven't been written yet, so `http.Error` is safe

**Step 3: Set Headers**
```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(statusCode)
```

**Why in this order?**
- Set headers BEFORE writing body
- `WriteHeader` must be called before `Write`
- Once `WriteHeader` is called, headers are sent

**Step 4: Write Response**
```go
if _, err := w.Write(jsonBytes); err != nil {
    return
}
```

**Why check error?**
- Write might fail (client disconnected, network issue)
- But headers are already written, can't send another response
- Just return (nothing more we can do)

---

## Error Response Format

### Our Format

```json
{
  "error": "Error message here"
}
```

### Why This Format?

**Simple:**
- One field, easy to parse
- Clear and readable

**Extensible:**
- Can add more fields later:
  ```json
  {
    "error": "Validation failed",
    "field": "type",
    "code": "REQUIRED"
  }
  ```

**Standard:**
- Many APIs use this format
- Clients expect it

### Alternative Formats

**Multiple errors:**
```json
{
  "errors": [
    {"field": "type", "message": "required"},
    {"field": "payload", "message": "invalid"}
  ]
}
```

**With error code:**
```json
{
  "error": {
    "message": "Validation failed",
    "code": "VALIDATION_ERROR"
  }
}
```

**Our choice:** Start simple, extend when needed.

---

## HTTP Status Codes in Errors

### Status Code Categories

**2xx - Success** (not used in errors)

**4xx - Client Errors**
- `400 Bad Request` - Invalid request format
- `401 Unauthorized` - Not authenticated
- `403 Forbidden` - Not authorized
- `404 Not Found` - Resource doesn't exist
- `413 Request Entity Too Large` - Body too big
- `422 Unprocessable Entity` - Valid format, invalid data

**5xx - Server Errors**
- `500 Internal Server Error` - Server problem

### Our Usage

```go
// Client errors (4xx)
ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
ErrorResponse(w, "Job type is required", http.StatusBadRequest)
ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)

// Server errors (5xx)
ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
```

### Choosing the Right Status Code

**400 Bad Request:**
- Invalid JSON
- Missing required fields
- Wrong data types

**413 Request Entity Too Large:**
- Body exceeds size limit

**500 Internal Server Error:**
- Server can't read body
- Server can't marshal response
- Unexpected server errors

---

## Fallback Error Handling

### The Problem

What if `ErrorResponse` itself fails?

**Scenario 1: Can't Marshal JSON**
```go
jsonBytes, err := json.Marshal(map[string]string{"error": message})
if err != nil {
    // What do we do?
}
```

**Our solution:**
```go
if err != nil {
    http.Error(w, "Failed to marshal error response", http.StatusInternalServerError)
    return
}
```

**Why `http.Error`?**
- Headers haven't been written yet
- `http.Error` sets status code and writes plain text
- Safe to use here

**Scenario 2: Can't Write Response**
```go
if _, err := w.Write(jsonBytes); err != nil {
    // Headers already written!
    // Can't send another response
}
```

**Our solution:**
```go
if _, err := w.Write(jsonBytes); err != nil {
    return  // Just return, nothing more we can do
}
```

**Why just return?**
- Headers are already written
- Can't send another response
- Client may have disconnected
- Logging would be good here (future improvement)

---

## When Headers Are Already Written

### The HTTP Response Lifecycle

**Step 1: Set Headers (Optional)**
```go
w.Header().Set("Content-Type", "application/json")
```

**Step 2: Write Status Code**
```go
w.WriteHeader(http.StatusCreated)
```

**Step 3: Write Body**
```go
w.Write(responseBytes)
```

**Important:** Once `WriteHeader` is called, headers are sent to the client. You can't change them after that.

### In ErrorResponse

**Before WriteHeader:**
```go
// Can still change headers
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(statusCode)  // Headers sent here
```

**After WriteHeader:**
```go
w.Write(jsonBytes)
if err != nil {
    // Headers already sent!
    // Can't send another response
    return
}
```

### Why This Matters

**If we tried to send another response:**
```go
// ❌ BAD: Can't do this after headers written
if err != nil {
    ErrorResponse(w, "Write failed", http.StatusInternalServerError)  // Error!
    // This would try to write headers again - won't work
}
```

**Our approach:**
```go
// ✅ GOOD: Just return
if err != nil {
    return  // Accept that we can't send error response
}
```

---

## Common Mistakes

### Mistake 1: Inconsistent Error Format

```go
// ❌ BAD: Different formats in different handlers
http.Error(w, "Error", 400)  // Handler 1
w.Write([]byte("Error"))     // Handler 2
```

**Fix:** Use `ErrorResponse` everywhere:
```go
// ✅ GOOD: Consistent format
ErrorResponse(w, "Error", http.StatusBadRequest)
```

### Mistake 2: Forgetting Content-Type

```go
// ❌ BAD: No Content-Type header
w.WriteHeader(400)
w.Write(jsonBytes)  // Client doesn't know it's JSON!
```

**Fix:** Always set Content-Type:
```go
// ✅ GOOD: Set Content-Type
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(400)
w.Write(jsonBytes)
```

### Mistake 3: Wrong Order of Operations

```go
// ❌ BAD: WriteHeader before setting headers
w.WriteHeader(400)
w.Header().Set("Content-Type", "application/json")  // Too late!
```

**Fix:** Set headers before WriteHeader:
```go
// ✅ GOOD: Headers before WriteHeader
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(400)
```

### Mistake 4: Trying to Send Error After Headers Written

```go
// ❌ BAD: Can't send error after headers written
w.WriteHeader(200)
w.Write(responseBytes)
if err != nil {
    ErrorResponse(w, "Error", 500)  // Won't work!
}
```

**Fix:** Check errors before writing:
```go
// ✅ GOOD: Check before writing
responseBytes, err := json.Marshal(response)
if err != nil {
    ErrorResponse(w, "Error", 500)  // Headers not written yet
    return
}
w.WriteHeader(200)
w.Write(responseBytes)
```

### Mistake 5: Not Handling Marshal Errors

```go
// ❌ BAD: Ignoring marshal error
jsonBytes, _ := json.Marshal(map[string]string{"error": message})
w.Write(jsonBytes)  // What if marshal failed? jsonBytes might be nil!
```

**Fix:** Always check errors:
```go
// ✅ GOOD: Check marshal error
jsonBytes, err := json.Marshal(map[string]string{"error": message})
if err != nil {
    http.Error(w, "Failed to marshal error", 500)
    return
}
```

---

## Key Takeaways

1. **Centralize error responses** = Consistent format, DRY, maintainable
2. **ErrorResponse function** = One place for all error handling
3. **Consistent format** = `{"error": "message"}` JSON
4. **Appropriate status codes** = 4xx for client errors, 5xx for server errors
5. **Fallback handling** = `http.Error` if can't marshal JSON
6. **Headers order matters** = Set headers before WriteHeader
7. **Can't send error after headers written** = Check errors early

---

## Next Steps

- Read [HTTP Status Codes](./06-http-status-codes.md) to understand status code semantics
- Read [HTTP Handler Patterns](./11-http-handler-patterns.md) to see error handling in context
- Read [Request Validation](./04-request-validation.md) to see validation error handling

