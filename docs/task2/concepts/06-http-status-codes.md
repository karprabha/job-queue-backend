# Understanding HTTP Status Codes

## Table of Contents

1. [What Are HTTP Status Codes?](#what-are-http-status-codes)
2. [Status Code Categories](#status-code-categories)
3. [Common Status Codes](#common-status-codes)
4. [When to Use Which Code](#when-to-use-which-code)
5. [Our Usage in Task 2](#our-usage-in-task-2)
6. [Status Code Best Practices](#status-code-best-practices)
7. [Common Mistakes](#common-mistakes)

---

## What Are HTTP Status Codes?

### The Basics

HTTP status codes are **3-digit numbers** that indicate the result of an HTTP request.

**Format:**
- First digit = Category (1xx, 2xx, 3xx, 4xx, 5xx)
- Last two digits = Specific code

**Example:**
- `200` = Success
- `400` = Bad Request
- `404` = Not Found
- `500` = Internal Server Error

### Why They Matter

**For clients:**
- Know if request succeeded or failed
- Know what type of error occurred
- Can handle errors appropriately

**For developers:**
- Debug issues
- Monitor API health
- Understand API behavior

---

## Status Code Categories

### 1xx - Informational

**Rarely used in APIs**
- `100 Continue` - Request received, continue
- `101 Switching Protocols` - Protocol upgrade

**When to use:** Almost never in REST APIs

### 2xx - Success

**Request was successful**

- `200 OK` - Generic success
- `201 Created` - Resource created (our case!)
- `204 No Content` - Success, no body

**When to use:**
- `200` - GET, PUT, PATCH success
- `201` - POST success (resource created)
- `204` - DELETE success (no content to return)

### 3xx - Redirection

**Request needs to go elsewhere**

- `301 Moved Permanently`
- `302 Found` (temporary redirect)
- `304 Not Modified` (caching)

**When to use:** Rarely in APIs (more for web pages)

### 4xx - Client Error

**Client made a mistake**

- `400 Bad Request` - Invalid request format
- `401 Unauthorized` - Not authenticated
- `403 Forbidden` - Not authorized
- `404 Not Found` - Resource doesn't exist
- `413 Request Entity Too Large` - Body too big
- `422 Unprocessable Entity` - Valid format, invalid data

**When to use:**
- `400` - Malformed request, missing fields
- `401` - Need to authenticate
- `403` - Authenticated but not allowed
- `404` - Resource not found
- `413` - Body exceeds size limit
- `422` - Valid JSON but invalid business rules

### 5xx - Server Error

**Server made a mistake**

- `500 Internal Server Error` - Generic server error
- `502 Bad Gateway` - Upstream server error
- `503 Service Unavailable` - Server overloaded

**When to use:**
- `500` - Unexpected server error
- `502` - Gateway/proxy error
- `503` - Server temporarily unavailable

---

## Common Status Codes

### 200 OK

**Meaning:** Request succeeded

**When to use:**
- GET request successful
- PUT/PATCH update successful
- Any successful operation that doesn't create a resource

**Example:**
```go
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(data)
```

### 201 Created

**Meaning:** Resource was created

**When to use:**
- POST request that creates a resource
- Resource has a URL (Location header)

**Example:**
```go
w.Header().Set("Location", "/jobs/123")
w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(createdResource)
```

**Our usage:**
```go
w.WriteHeader(http.StatusCreated)  // Job was created!
```

### 400 Bad Request

**Meaning:** Client sent invalid request

**When to use:**
- Invalid JSON
- Missing required fields
- Wrong data types
- Malformed request

**Example:**
```go
ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
ErrorResponse(w, "Job type is required", http.StatusBadRequest)
```

### 413 Request Entity Too Large

**Meaning:** Request body is too large

**When to use:**
- Body exceeds size limit
- File upload too large

**Example:**
```go
if strings.Contains(err.Error(), "request body too large") {
    ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
}
```

### 500 Internal Server Error

**Meaning:** Server encountered an error

**When to use:**
- Unexpected server error
- Database connection failed
- Can't read request body
- Can't marshal response

**Example:**
```go
ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
```

---

## When to Use Which Code

### Decision Tree

**Request succeeded?**
- ✅ Yes → Use 2xx
  - Created resource? → `201 Created`
  - Just success? → `200 OK`
  - No content? → `204 No Content`

**Request failed?**
- ❌ Yes → Client error or server error?

**Client error (4xx):**
- Invalid format? → `400 Bad Request`
- Too large? → `413 Request Entity Too Large`
- Not found? → `404 Not Found`
- Not authenticated? → `401 Unauthorized`
- Not authorized? → `403 Forbidden`
- Valid format, invalid data? → `422 Unprocessable Entity`

**Server error (5xx):**
- Unexpected error? → `500 Internal Server Error`
- Service unavailable? → `503 Service Unavailable`

---

## Our Usage in Task 2

### Status Codes We Use

**201 Created**
```go
w.WriteHeader(http.StatusCreated)  // Job created successfully
```

**400 Bad Request**
```go
ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
ErrorResponse(w, "Job type is required", http.StatusBadRequest)
```

**413 Request Entity Too Large**
```go
ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
```

**500 Internal Server Error**
```go
ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
```

### Why These Codes?

**201 Created:**
- POST `/jobs` creates a job
- Job is a new resource
- Standard REST practice

**400 Bad Request:**
- Invalid JSON = malformed request
- Missing type = invalid request
- Client needs to fix the request

**413 Request Entity Too Large:**
- Body exceeds 1MB limit
- Client sent too much data
- Specific error for size issues

**500 Internal Server Error:**
- Can't read body = server problem
- Can't marshal = server problem
- Unexpected errors = server problem

---

## Status Code Best Practices

### 1. Be Specific

**❌ Bad:**
```go
ErrorResponse(w, "Error", http.StatusBadRequest)  // Too generic
```

**✅ Good:**
```go
ErrorResponse(w, "Job type is required", http.StatusBadRequest)  // Specific
```

### 2. Use Appropriate Codes

**❌ Bad:**
```go
// Using 500 for client errors
ErrorResponse(w, "Invalid JSON", http.StatusInternalServerError)  // Wrong!
```

**✅ Good:**
```go
// Use 400 for client errors
ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)  // Correct
```

### 3. Match Semantics

**❌ Bad:**
```go
// Using 200 for creation
w.WriteHeader(http.StatusOK)  // Should be 201!
```

**✅ Good:**
```go
// Use 201 for creation
w.WriteHeader(http.StatusCreated)  // Correct
```

### 4. Consistent Error Format

**❌ Bad:**
```go
// Different error formats
http.Error(w, "Error", 400)           // Plain text
ErrorResponse(w, "Error", 400)        // JSON
```

**✅ Good:**
```go
// Always use ErrorResponse
ErrorResponse(w, "Error", http.StatusBadRequest)  // Consistent
```

---

## Common Mistakes

### Mistake 1: Using 200 for Everything

```go
// ❌ BAD: Using 200 for creation
w.WriteHeader(http.StatusOK)  // Should be 201!
```

**Fix:**
```go
// ✅ GOOD: Use 201 for creation
w.WriteHeader(http.StatusCreated)
```

### Mistake 2: Using 500 for Client Errors

```go
// ❌ BAD: Client error but using 500
ErrorResponse(w, "Invalid JSON", http.StatusInternalServerError)
```

**Fix:**
```go
// ✅ GOOD: Use 400 for client errors
ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
```

### Mistake 3: Not Being Specific

```go
// ❌ BAD: Generic error
ErrorResponse(w, "Error", http.StatusBadRequest)
```

**Fix:**
```go
// ✅ GOOD: Specific error
ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
```

### Mistake 4: Wrong Code for Size Errors

```go
// ❌ BAD: Using 400 for size error
ErrorResponse(w, "Body too large", http.StatusBadRequest)
```

**Fix:**
```go
// ✅ GOOD: Use 413 for size errors
ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
```

---

## Key Takeaways

1. **Status codes** = 3-digit numbers indicating request result
2. **2xx** = Success (200 OK, 201 Created)
3. **4xx** = Client error (400 Bad Request, 413 Too Large)
4. **5xx** = Server error (500 Internal Server Error)
5. **201 Created** = Use for POST that creates resources
6. **400 Bad Request** = Invalid request format
7. **413 Request Entity Too Large** = Body too big
8. **500 Internal Server Error** = Unexpected server error
9. **Be specific** = Use appropriate codes, specific messages

---

## Next Steps

- Read [Error Response Centralization](./05-error-response-centralization.md) to see how we use status codes
- Read [HTTP Handler Patterns](./11-http-handler-patterns.md) to see status codes in context

