# Understanding HTTP Request Parsing in Go

## Table of Contents

1. [The HTTP Request Lifecycle](#the-http-request-lifecycle)
2. [Reading the Request Body](#reading-the-request-body)
3. [Unmarshaling JSON](#unmarshaling-json)
4. [Request Struct Design](#request-struct-design)
5. [Error Handling During Parsing](#error-handling-during-parsing)
6. [Our Implementation](#our-implementation)
7. [Common Mistakes](#common-mistakes)

---

## The HTTP Request Lifecycle

### How HTTP Requests Work

When a client sends a POST request:

```
Client → HTTP Request → Server Handler → Response → Client
```

**Inside the handler:**
1. **Read** the request body (bytes)
2. **Parse** the bytes (JSON, form data, etc.)
3. **Validate** the parsed data
4. **Process** the request
5. **Write** the response

### The Request Body

The request body is an `io.ReadCloser`:

```go
type Request struct {
    Body io.ReadCloser  // Can be read once!
}
```

**Key characteristics:**
- **Stream** - Data flows from client to server
- **One-time read** - Can only read once
- **Closable** - Must be closed when done

---

## Reading the Request Body

### Using io.ReadAll

```go
bodyBytes, err := io.ReadAll(r.Body)
```

### What io.ReadAll Does

**Step by step:**
1. Creates a buffer
2. Reads from `r.Body` until EOF
3. Returns all bytes read
4. Returns error if read fails

**Why use `io.ReadAll`?**
- Simple - one function call
- Reads entire body into memory
- Good for small to medium bodies
- Returns `[]byte` ready for parsing

### Our Code

```go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB limit
    
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        // Handle error
    }
    // Now bodyBytes contains the raw request body
}
```

### Why Read All at Once?

**Alternative:** Stream parsing (read and parse incrementally)

**Our choice:** Read all, then parse

**Why?**
- Simpler code
- Request bodies are typically small
- JSON needs full document anyway
- Easier error handling

**Trade-off:**
- Uses more memory (entire body in memory)
- But: Acceptable for our use case (1MB limit)

---

## Unmarshaling JSON

### What is Unmarshaling?

**Marshaling** = Go struct → JSON bytes
**Unmarshaling** = JSON bytes → Go struct

### The json.Unmarshal Function

```go
func Unmarshal(data []byte, v interface{}) error
```

**Parameters:**
- `data []byte` - JSON bytes to parse
- `v interface{}` - Pointer to struct to fill

**Returns:**
- `error` - nil if successful, error if parsing fails

### Our Code

```go
var request CreateJobRequest
if err := json.Unmarshal(bodyBytes, &request); err != nil {
    ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
    return
}
```

### How It Works

**Step 1: Define Target Struct**
```go
type CreateJobRequest struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

**Step 2: Create Variable**
```go
var request CreateJobRequest  // Empty struct
```

**Step 3: Unmarshal**
```go
json.Unmarshal(bodyBytes, &request)
// request is now filled with data from JSON
```

**What happens:**
1. `json.Unmarshal` reads the JSON bytes
2. Matches JSON fields to struct fields using `json:"..."` tags
3. Converts JSON values to Go types
4. Fills the struct fields

### JSON Tags Explained

```go
Type    string          `json:"type"`
```

**Breaking it down:**
- `json:"type"` - JSON field name is "type"
- Maps JSON `"type"` to Go field `Type`
- Case-insensitive matching (but use tags for clarity)

**Why tags?**
- JSON uses `snake_case` or `camelCase`
- Go uses `PascalCase` for exported fields
- Tags bridge the gap

---

## Request Struct Design

### Our Request Struct

```go
type CreateJobRequest struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

### Design Principles

**1. Match the API Contract**
- Fields match the JSON request exactly
- Tag names match JSON keys

**2. Use Appropriate Types**
- `Type string` - Simple string
- `Payload json.RawMessage` - Opaque JSON

**3. Keep It Simple**
- Only fields needed for this endpoint
- Don't include fields from other endpoints

**4. Validation-Friendly**
- Easy to check for empty values
- Types make validation clear

### Why Separate Request Struct?

**Alternative:** Use domain struct directly

```go
// ❌ BAD: Mixing HTTP and domain
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    var job domain.Job  // Domain struct in HTTP handler
    json.Unmarshal(bodyBytes, &job)
}
```

**Problems:**
- HTTP layer knows about domain internals
- Can't have different request/response formats
- Harder to evolve independently

**Our approach:**
```go
// ✅ GOOD: Separate request struct
type CreateJobRequest struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

// Convert to domain model
job := domain.NewJob(request.Type, request.Payload)
```

**Benefits:**
- HTTP layer is independent
- Can change request format without changing domain
- Clear separation of concerns

---

## Error Handling During Parsing

### Common Parsing Errors

**1. Invalid JSON**
```json
{
  "type": "email",
  "payload": { invalid json }
}
```
- `json.Unmarshal` returns error
- Return `400 Bad Request`

**2. Missing Required Fields**
```json
{
  "payload": {...}
}
// Missing "type"
```
- Unmarshals successfully (type is empty string)
- Need to validate after unmarshaling

**3. Wrong Types**
```json
{
  "type": 123,  // Should be string
  "payload": {...}
}
```
- `json.Unmarshal` may succeed or fail depending on type
- Need validation

**4. Request Body Too Large**
- `MaxBytesReader` returns error
- Return `413 Request Entity Too Large`

### Our Error Handling

```go
// 1. Read body
bodyBytes, err := io.ReadAll(r.Body)
if err != nil {
    if strings.Contains(err.Error(), "request body too large") {
        ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
        return
    }
    ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
    return
}

// 2. Unmarshal JSON
var request CreateJobRequest
if err := json.Unmarshal(bodyBytes, &request); err != nil {
    ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
    return
}

// 3. Validate
if request.Type == "" {
    ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
    return
}
```

### Error Response Strategy

**Consistent format:**
```json
{
  "error": "Error message here"
}
```

**Appropriate status codes:**
- `400 Bad Request` - Invalid request format
- `413 Request Entity Too Large` - Body too big
- `500 Internal Server Error` - Server error

---

## Our Implementation

### Complete Flow

```go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Limit body size
    r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
    
    // 2. Read body
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        // Handle read errors
        return
    }
    
    // 3. Unmarshal JSON
    var request CreateJobRequest
    if err := json.Unmarshal(bodyBytes, &request); err != nil {
        ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
        return
    }
    
    // 4. Validate
    if request.Type == "" {
        ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
        return
    }
    
    // 5. Create domain object
    job := domain.NewJob(request.Type, request.Payload)
    
    // 6. Return response
    // ... (response code)
}
```

### Why This Order?

**1. Size limit first**
- Prevents reading huge bodies
- Security: prevents DoS attacks

**2. Read body**
- Get the data
- Handle read errors

**3. Parse JSON**
- Convert bytes to struct
- Handle parse errors

**4. Validate**
- Check business rules
- Handle validation errors

**5. Process**
- Create domain object
- Do business logic

**6. Respond**
- Write response
- Handle write errors

---

## Common Mistakes

### Mistake 1: Not Limiting Body Size

```go
// ❌ BAD: No size limit
bodyBytes, err := io.ReadAll(r.Body)
```

**Problem:** Client can send huge body, causing memory issues

**Fix:**
```go
// ✅ GOOD: Limit size
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
bodyBytes, err := io.ReadAll(r.Body)
```

### Mistake 2: Reading Body Twice

```go
// ❌ BAD: Can't read twice
body1, _ := io.ReadAll(r.Body)
body2, _ := io.ReadAll(r.Body)  // body2 is empty!
```

**Problem:** Request body can only be read once

**Fix:** Read once, reuse:
```go
// ✅ GOOD: Read once
bodyBytes, _ := io.ReadAll(r.Body)
// Use bodyBytes multiple times
```

### Mistake 3: Not Checking Unmarshal Errors

```go
// ❌ BAD: Ignoring errors
var request CreateJobRequest
json.Unmarshal(bodyBytes, &request)  // Error ignored!
// request might be partially filled or empty
```

**Fix:**
```go
// ✅ GOOD: Check errors
if err := json.Unmarshal(bodyBytes, &request); err != nil {
    // Handle error
    return
}
```

### Mistake 4: Wrong Pointer Type

```go
// ❌ BAD: Not a pointer
var request CreateJobRequest
json.Unmarshal(bodyBytes, request)  // Error! Need pointer
```

**Fix:**
```go
// ✅ GOOD: Use pointer
var request CreateJobRequest
json.Unmarshal(bodyBytes, &request)  // & = address of
```

### Mistake 5: Not Validating After Unmarshal

```go
// ❌ BAD: Assuming unmarshal means valid
var request CreateJobRequest
json.Unmarshal(bodyBytes, &request)
// What if type is empty? Or payload is invalid?
```

**Fix:**
```go
// ✅ GOOD: Validate after unmarshal
if request.Type == "" {
    ErrorResponse(w, "Job type is required", http.StatusBadRequest)
    return
}
```

---

## Key Takeaways

1. **Request body** = `io.ReadCloser`, can only read once
2. **`io.ReadAll`** = Reads entire body into `[]byte`
3. **`json.Unmarshal`** = Converts JSON bytes to Go struct
4. **JSON tags** = Map JSON fields to Go struct fields
5. **Always validate** after unmarshaling
6. **Limit body size** for security
7. **Separate request structs** from domain structs

---

## Next Steps

- Read [Request Validation](./04-request-validation.md) to see validation patterns
- Read [Error Response Centralization](./05-error-response-centralization.md) to see error handling
- Read [HTTP Handler Patterns](./11-http-handler-patterns.md) to see the complete handler pattern

