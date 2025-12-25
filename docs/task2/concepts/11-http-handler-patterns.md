# Understanding HTTP Handler Patterns

## Table of Contents

1. [Handler Function Signature](#handler-function-signature)
2. [Request/Response Flow](#requestresponse-flow)
3. [Handler Structure](#handler-structure)
4. [Response Writing Patterns](#response-writing-patterns)
5. [Error Handling in Handlers](#error-handling-in-handlers)
6. [Our Complete Handler](#our-complete-handler)
7. [Best Practices](#best-practices)
8. [Common Mistakes](#common-mistakes)

---

## Handler Function Signature

### Standard Signature

```go
func HandlerName(w http.ResponseWriter, r *http.Request)
```

**Parameters:**

- `w http.ResponseWriter` - Write response
- `r *http.Request` - Read request

**Why this signature?**

- Standard Go HTTP handler interface
- Matches `http.HandlerFunc` type
- Used by `http.ServeMux`

### Our Handler

```go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // Handler implementation
}
```

---

## Request/Response Flow

### The Flow

```
1. Read Request
   ↓
2. Parse Request
   ↓
3. Validate Request
   ↓
4. Process (Domain Logic)
   ↓
5. Format Response
   ↓
6. Write Response
```

### Our Implementation

```go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Limit body size
    r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

    // 2. Read body
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        // Handle error
        return
    }

    // 3. Parse JSON
    var request CreateJobRequest
    if err := json.Unmarshal(bodyBytes, &request); err != nil {
        ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
        return
    }

    // 4. Validate
    if request.Type == "" {
        ErrorResponse(w, "Job type is required", http.StatusBadRequest)
        return
    }

    // 5. Process (Domain)
    job := domain.NewJob(request.Type, request.Payload)

    // 6. Format response
    response := CreateJobResponse{
        ID:        job.ID,
        Type:      job.Type,
        Status:    string(job.Status),
        CreatedAt: job.CreatedAt.Format(time.RFC3339),
    }

    // 7. Write response
    responseBytes, _ := json.Marshal(response)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    w.Write(responseBytes)
}
```

---

## Handler Structure

### Typical Structure

**1. Setup/Validation**

- Size limits
- Read request
- Parse request

**2. Business Logic**

- Call domain functions
- Process data

**3. Response**

- Format response
- Write response

### Our Structure

```go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // === SETUP ===
    r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
    bodyBytes, err := io.ReadAll(r.Body)
    // ... error handling

    // === PARSE ===
    var request CreateJobRequest
    json.Unmarshal(bodyBytes, &request)
    // ... error handling

    // === VALIDATE ===
    if request.Type == "" {
        // ... error handling
    }

    // === PROCESS ===
    job := domain.NewJob(request.Type, request.Payload)

    // === RESPONSE ===
    response := CreateJobResponse{...}
    // ... marshal and write
}
```

---

## Response Writing Patterns

### Pattern 1: Set Headers First

```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
w.Write(responseBytes)
```

**Why this order?**

- Headers must be set before `WriteHeader`
- `WriteHeader` sends headers
- `Write` sends body

### Pattern 2: Marshal Then Write

```go
responseBytes, err := json.Marshal(response)
if err != nil {
    // Handle error
    return
}
w.Write(responseBytes)
```

**Why marshal first?**

- Check for marshal errors
- Can handle errors before writing headers
- Cleaner error handling

### Our Pattern

```go
// Marshal
responseBytes, err := json.Marshal(response)
if err != nil {
    ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
    return
}

// Set headers
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)

// Write
if _, err := w.Write(responseBytes); err != nil {
    return  // Headers already written, can't send error
}
```

---

## Error Handling in Handlers

### Early Returns

**Pattern:** Check errors early, return immediately

```go
if err != nil {
    ErrorResponse(w, "Error message", statusCode)
    return  // Exit early
}
```

**Why early returns?**

- Avoids nested if statements
- Clear error handling
- Fail fast

### Our Error Handling

```go
// Read error
if err != nil {
    if strings.Contains(err.Error(), "request body too large") {
        ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
        return
    }
    ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
    return
}

// Parse error
if err := json.Unmarshal(bodyBytes, &request); err != nil {
    ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
    return
}

// Validation error
if request.Type == "" {
    ErrorResponse(w, "Job type is required", http.StatusBadRequest)
    return
}
```

---

## Our Complete Handler

### Full Implementation

```go
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Limit body size
    r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

    // 2. Read body
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        if strings.Contains(err.Error(), "request body too large") {
            ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
            return
        }
        ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
        return
    }

    // 3. Parse JSON
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

    // 5. Create job (Domain)
    job := domain.NewJob(request.Type, request.Payload)

    // 6. Format response
    response := CreateJobResponse{
        ID:        job.ID,
        Type:      job.Type,
        Status:    string(job.Status),
        CreatedAt: job.CreatedAt.Format(time.RFC3339),
    }

    // 7. Marshal response
    responseBytes, err := json.Marshal(response)
    if err != nil {
        ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
        return
    }

    // 8. Write response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    if _, err := w.Write(responseBytes); err != nil {
        return  // Headers written, can't send error
    }
}
```

### Why This Structure?

**1. Security first** - Size limit
**2. Read early** - Get data
**3. Parse early** - Convert to struct
**4. Validate early** - Check rules
**5. Process** - Domain logic
**6. Format** - Response struct
**7. Marshal** - JSON bytes
**8. Write** - Send response

---

## Best Practices

### 1. Fail Fast

```go
// ✅ GOOD: Check errors early
if err != nil {
    return
}
```

### 2. Consistent Error Handling

```go
// ✅ GOOD: Use ErrorResponse everywhere
ErrorResponse(w, "Error", http.StatusBadRequest)
```

### 3. Appropriate Status Codes

```go
// ✅ GOOD: Right status codes
ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)  // 4xx for client errors
ErrorResponse(w, "Server error", http.StatusInternalServerError)  // 5xx for server errors
```

### 4. Clear Error Messages

```go
// ✅ GOOD: Specific messages
ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
```

### 5. Separate Concerns

```go
// ✅ GOOD: HTTP layer thin, domain layer does work
job := domain.NewJob(request.Type, request.Payload)
```

---

## Common Mistakes

### Mistake 1: Not Checking Errors

```go
// ❌ BAD: Ignoring errors
bodyBytes, _ := io.ReadAll(r.Body)
json.Unmarshal(bodyBytes, &request)
```

**Fix:**

```go
// ✅ GOOD: Check all errors
bodyBytes, err := io.ReadAll(r.Body)
if err != nil {
    return
}
```

### Mistake 2: Wrong Header Order

```go
// ❌ BAD: WriteHeader before headers
w.WriteHeader(200)
w.Header().Set("Content-Type", "application/json")
```

**Fix:**

```go
// ✅ GOOD: Headers before WriteHeader
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(200)
```

### Mistake 3: Business Logic in Handler

```go
// ❌ BAD: Logic in handler
func CreateJobHandler(...) {
    id := uuid.New().String()  // Should be in domain!
}
```

**Fix:**

```go
// ✅ GOOD: Logic in domain
job := domain.NewJob(request.Type, request.Payload)
```

---

## Key Takeaways

1. **Handler signature** = `func(w ResponseWriter, r *Request)`
2. **Flow** = Read → Parse → Validate → Process → Respond
3. **Early returns** = Fail fast with clear errors
4. **Header order** = Set headers before WriteHeader
5. **Separate concerns** = HTTP thin, domain does work
6. **Consistent errors** = Use ErrorResponse everywhere

---

## Next Steps

- Review all concepts to see how they fit together
- Read [Domain Separation](./10-domain-separation.md) to understand architecture
- Read [Error Response Centralization](./05-error-response-centralization.md) for error handling
