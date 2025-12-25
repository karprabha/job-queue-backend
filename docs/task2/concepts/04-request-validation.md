# Understanding Request Validation

## Table of Contents

1. [Why Validate Requests?](#why-validate-requests)
2. [Validation Patterns in Go](#validation-patterns-in-go)
3. [Our Validation Strategy](#our-validation-strategy)
4. [Validation vs Business Logic](#validation-vs-business-logic)
5. [Error Messages and UX](#error-messages-and-ux)
6. [Common Mistakes](#common-mistakes)

---

## Why Validate Requests?

### The Problem: Trust No One

**Never trust client input!**

Clients can send:
- Invalid data
- Malformed JSON
- Missing required fields
- Wrong data types
- Malicious data

**Without validation:**
- Server crashes (panics)
- Data corruption
- Security vulnerabilities
- Poor user experience

### The Solution: Validate Early

**Validate at the boundary:**
- HTTP layer (handlers) = First line of defense
- Validate before domain logic
- Fail fast with clear errors

---

## Validation Patterns in Go

### Pattern 1: Check After Unmarshal

```go
var request CreateJobRequest
if err := json.Unmarshal(bodyBytes, &request); err != nil {
    // JSON parsing failed
    return
}

// Now validate the struct
if request.Type == "" {
    // Validation failed
    return
}
```

**Why this order?**
1. Parse JSON first (might fail)
2. Then validate parsed data
3. Fail fast at each step

### Pattern 2: Empty String Check

```go
if request.Type == "" {
    ErrorResponse(w, "Job type is required", http.StatusBadRequest)
    return
}
```

**Why check empty strings?**
- Go's zero value for string is `""`
- Unmarshal sets missing fields to zero value
- Need to explicitly check

### Pattern 3: Type Validation

```go
// JSON unmarshal handles type validation
// If type is wrong, unmarshal fails
var request CreateJobRequest
if err := json.Unmarshal(bodyBytes, &request); err != nil {
    // Type mismatch = parse error
    ErrorResponse(w, "Invalid request format", http.StatusBadRequest)
    return
}
```

---

## Our Validation Strategy

### Step 1: Size Limit

```go
r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB max
```

**Why first?**
- Prevents reading huge bodies
- Security: prevents DoS attacks
- Fail fast before processing

### Step 2: Read Body

```go
bodyBytes, err := io.ReadAll(r.Body)
if err != nil {
    // Handle read errors (including size limit)
    return
}
```

### Step 3: Parse JSON

```go
var request CreateJobRequest
if err := json.Unmarshal(bodyBytes, &request); err != nil {
    ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
    return
}
```

**What this validates:**
- Valid JSON syntax
- Correct field types
- Required fields present (but might be empty!)

### Step 4: Business Validation

```go
if request.Type == "" {
    ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
    return
}
```

**What this validates:**
- Required fields not empty
- Business rules
- Data constraints

---

## Validation vs Business Logic

### Validation (HTTP Layer)

**What:** Check if request is valid
- Format is correct
- Required fields present
- Types are correct
- Size is acceptable

**Where:** HTTP handlers
**When:** Before creating domain objects

### Business Logic (Domain Layer)

**What:** Process valid data
- Create jobs
- Calculate values
- Apply business rules

**Where:** Domain layer
**When:** After validation

### Our Approach

**HTTP Layer (Validation):**
```go
// Validate request format
if request.Type == "" {
    ErrorResponse(w, "Job type is required", http.StatusBadRequest)
    return
}
```

**Domain Layer (Business Logic):**
```go
// Process valid data
job := domain.NewJob(request.Type, request.Payload)
```

**Separation:**
- HTTP = Validation
- Domain = Business logic
- Clear boundaries

---

## Error Messages and UX

### Good Error Messages

**Characteristics:**
- **Clear** - User understands the problem
- **Specific** - Points to the issue
- **Actionable** - User knows how to fix

**Examples:**
```go
// ✅ GOOD: Clear and specific
ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)

// ❌ BAD: Vague
ErrorResponse(w, "Invalid request", http.StatusBadRequest)

// ❌ BAD: Too technical
ErrorResponse(w, "Field 'type' is nil or empty string", http.StatusBadRequest)
```

### Our Error Messages

```go
// Size error
ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)

// Parse error
ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)

// Validation error
ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
```

**Why these work:**
- Clear what went wrong
- User knows what to fix
- Not too technical

---

## Common Mistakes

### Mistake 1: Not Validating

```go
// ❌ BAD: No validation
var request CreateJobRequest
json.Unmarshal(bodyBytes, &request)
job := domain.NewJob(request.Type, request.Payload)  // What if type is empty?
```

**Fix:**
```go
// ✅ GOOD: Validate
if request.Type == "" {
    ErrorResponse(w, "Job type is required", http.StatusBadRequest)
    return
}
```

### Mistake 2: Validating in Domain

```go
// ❌ BAD: Validation in domain
func NewJob(jobType string, payload json.RawMessage) (*Job, error) {
    if jobType == "" {
        return nil, errors.New("type required")  // Domain shouldn't validate HTTP concerns
    }
}
```

**Fix:**
```go
// ✅ GOOD: Validate in HTTP layer
if request.Type == "" {
    ErrorResponse(w, "Job type is required", http.StatusBadRequest)
    return
}
job := domain.NewJob(request.Type, request.Payload)
```

### Mistake 3: Vague Error Messages

```go
// ❌ BAD: Too vague
ErrorResponse(w, "Error", http.StatusBadRequest)
```

**Fix:**
```go
// ✅ GOOD: Specific
ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
```

### Mistake 4: Wrong Status Code

```go
// ❌ BAD: Using 500 for validation errors
ErrorResponse(w, "Type required", http.StatusInternalServerError)
```

**Fix:**
```go
// ✅ GOOD: Use 400 for validation errors
ErrorResponse(w, "Type required", http.StatusBadRequest)
```

---

## Key Takeaways

1. **Validate early** = Check at HTTP boundary
2. **Fail fast** = Return errors immediately
3. **Clear messages** = User-friendly error messages
4. **Right status codes** = 4xx for validation errors
5. **Separate concerns** = Validation in HTTP, logic in domain

---

## Next Steps

- Read [HTTP Request Parsing](./03-http-request-parsing.md) to see parsing before validation
- Read [Error Response Centralization](./05-error-response-centralization.md) to see error handling
- Read [HTTP Status Codes](./06-http-status-codes.md) to see status codes for validation errors

