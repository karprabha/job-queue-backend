# Understanding json.RawMessage - Opaque JSON Payloads

## Table of Contents

1. [What is json.RawMessage?](#what-is-jsonrawmessage)
2. [Why Use Opaque JSON?](#why-use-opaque-json)
3. [Opaque vs Concrete Types](#opaque-vs-concrete-types)
4. [How json.RawMessage Works](#how-jsonrawmessage-works)
5. [When to Use json.RawMessage](#when-to-use-jsonrawmessage)
6. [Real Example: Job Payload](#real-example-job-payload)
7. [Common Mistakes](#common-mistakes)

---

## What is json.RawMessage?

### The Definition

`json.RawMessage` is a type in Go's `encoding/json` package defined as:

```go
type RawMessage []byte
```

**That's it!** It's just a byte slice (`[]byte`) with special JSON marshaling/unmarshaling behavior.

### What Makes It Special?

**Normal `[]byte`:**
- When marshaled to JSON, becomes a base64-encoded string
- When unmarshaled, expects base64 string

**`json.RawMessage`:**
- When marshaled to JSON, writes the raw bytes as-is
- When unmarshaled, stores the raw JSON bytes
- **Preserves the original JSON structure**

### The Key Insight

`json.RawMessage` is like a **"JSON container"** - it holds JSON without interpreting it.

---

## Why Use Opaque JSON?

### The Problem: Concrete Types

Imagine if we defined a concrete struct for the payload:

```go
// ❌ BAD: Too specific
type EmailPayload struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
}

type Job struct {
    Payload EmailPayload  // Only works for emails!
}
```

**Problems:**
1. **Not flexible** - What if we add SMS jobs? Need a new struct
2. **Domain knows too much** - Domain shouldn't care about email structure
3. **Tight coupling** - Adding new job types requires code changes
4. **Breaks abstraction** - Domain layer knows about HTTP request structure

### The Solution: Opaque JSON

```go
// ✅ GOOD: Opaque and flexible
type Job struct {
    Payload json.RawMessage  // Can store any JSON!
}
```

**Benefits:**
1. **Flexible** - Can store any JSON structure
2. **Domain doesn't care** - Domain just stores it, doesn't interpret it
3. **Loose coupling** - New job types don't require domain changes
4. **Preserves abstraction** - Domain is independent of request structure

### Real-World Analogy

Think of it like a **mailbox**:
- **Concrete type** = A mailbox that only accepts letters (too specific!)
- **Opaque JSON** = A mailbox that accepts any mail (flexible!)

The domain (mailbox) doesn't need to know what's inside - it just stores it.

---

## Opaque vs Concrete Types

### Concrete Types (What We Avoided)

```go
// Concrete: Domain knows the structure
type EmailPayload struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
}

type Job struct {
    Payload EmailPayload  // Domain knows it's an email!
}
```

**When to use:**
- When the structure is **always the same**
- When you need **type safety** in the domain
- When you need to **validate structure** in the domain

**Trade-offs:**
- ✅ Type-safe
- ✅ Can validate in domain
- ❌ Not flexible
- ❌ Domain knows too much

### Opaque Types (What We Used)

```go
// Opaque: Domain doesn't know the structure
type Job struct {
    Payload json.RawMessage  // Domain doesn't care what's inside!
}
```

**When to use:**
- When the structure **varies by type**
- When you want **flexibility**
- When validation happens **outside domain** (in handlers)

**Trade-offs:**
- ✅ Flexible
- ✅ Domain stays simple
- ✅ Easy to extend
- ❌ Less type-safe
- ❌ Can't validate in domain

### Our Choice: Opaque

**Why?**
- Task says "payload (opaque JSON)"
- We need to support multiple job types
- Domain shouldn't know about email/SMS/etc. structure
- Validation happens in HTTP layer (appropriate)

---

## How json.RawMessage Works

### Under the Hood

`json.RawMessage` is just `[]byte`, but with custom JSON methods:

```go
// Simplified version of what json.RawMessage does
type RawMessage []byte

func (m RawMessage) MarshalJSON() ([]byte, error) {
    // If m is valid JSON, return it as-is
    // Otherwise, return error
    return m, nil
}

func (m *RawMessage) UnmarshalJSON(data []byte) error {
    // Store the raw bytes without parsing
    *m = data
    return nil
}
```

### Unmarshaling Behavior

**When you unmarshal JSON into `json.RawMessage`:**

```go
var request CreateJobRequest
json.Unmarshal([]byte(`{"type":"email","payload":{"to":"user@example.com"}}`), &request)

// request.Payload now contains: []byte(`{"to":"user@example.com"}`)
// It's the raw JSON bytes, not parsed!
```

**Key point:** The JSON is stored as-is, not parsed into a struct.

### Marshaling Behavior

**When you marshal `json.RawMessage` to JSON:**

```go
job := &Job{
    Payload: json.RawMessage(`{"to":"user@example.com"}`),
}

json.Marshal(job)
// Result: {"payload":{"to":"user@example.com"}}
// The raw JSON is written directly!
```

**Key point:** The raw bytes are written directly to JSON output.

---

## When to Use json.RawMessage

### Use Cases

**1. Opaque Payloads (Our Case)**
```go
type Job struct {
    Payload json.RawMessage  // Structure varies by job type
}
```

**2. Delayed Parsing**
```go
// Parse later when you know the type
type Message struct {
    Type    string          `json:"type"`
    Data    json.RawMessage `json:"data"`  // Parse based on type
}
```

**3. Preserving JSON Structure**
```go
// Need to preserve exact JSON formatting
type Document struct {
    Metadata map[string]interface{} `json:"metadata"`
    Content  json.RawMessage         `json:"content"`  // Preserve formatting
}
```

**4. Passing Through JSON**
```go
// Proxy/API gateway that forwards JSON
type ProxyRequest struct {
    Headers map[string]string `json:"headers"`
    Body    json.RawMessage   `json:"body"`  // Forward as-is
}
```

### When NOT to Use

**1. When Structure is Always the Same**
```go
// ❌ Don't use RawMessage if structure never changes
type User struct {
    Email json.RawMessage  // Bad! Email is always a string
}

// ✅ Use concrete type
type User struct {
    Email string  // Good!
}
```

**2. When You Need Type Safety**
```go
// ❌ Can't validate structure with RawMessage
type Config struct {
    Settings json.RawMessage  // Can't validate at compile time
}

// ✅ Use struct for type safety
type Config struct {
    Settings SettingsStruct  // Can validate
}
```

---

## Real Example: Job Payload

### Our Implementation

```go
// Domain model
type Job struct {
    Payload json.RawMessage  // Opaque - domain doesn't care
}

// HTTP request
type CreateJobRequest struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`  // Accept any JSON
}
```

### How It Works

**Step 1: Client Sends Request**
```json
{
  "type": "email",
  "payload": {
    "to": "user@example.com",
    "subject": "Hello"
  }
}
```

**Step 2: Handler Unmarshals**
```go
var request CreateJobRequest
json.Unmarshal(bodyBytes, &request)

// request.Payload now contains: []byte(`{"to":"user@example.com","subject":"Hello"}`)
```

**Step 3: Create Job**
```go
job := domain.NewJob(request.Type, request.Payload)

// job.Payload contains the same raw bytes
```

**Step 4: Store/Return**
```go
// When marshaling job to JSON:
json.Marshal(job)
// Result: {"payload":{"to":"user@example.com","subject":"Hello"}}
// Original structure preserved!
```

### Why This Works

1. **HTTP layer** validates that payload is valid JSON
2. **Domain layer** just stores it (doesn't care about structure)
3. **Future processing** can parse based on job type
4. **Flexible** - can add new job types without domain changes

---

## Common Mistakes

### Mistake 1: Trying to Access Fields

```go
// ❌ BAD: Can't access fields directly
job.Payload.To  // Error! RawMessage doesn't have fields
```

**Fix:** Parse when you need to:
```go
// ✅ GOOD: Parse when needed
var emailPayload EmailPayload
json.Unmarshal(job.Payload, &emailPayload)
// Now you can access emailPayload.To
```

### Mistake 2: Validating in Domain

```go
// ❌ BAD: Domain shouldn't validate structure
func NewJob(payload json.RawMessage) *Job {
    var email EmailPayload
    if err := json.Unmarshal(payload, &email); err != nil {
        // Domain is validating structure - wrong layer!
    }
}
```

**Fix:** Validate in HTTP layer:
```go
// ✅ GOOD: Validate in handler
var emailPayload EmailPayload
if err := json.Unmarshal(request.Payload, &emailPayload); err != nil {
    ErrorResponse(w, "Invalid email payload", http.StatusBadRequest)
    return
}
```

### Mistake 3: Converting to String

```go
// ❌ BAD: Converting to string loses type
payloadStr := string(job.Payload)  // Now it's just a string
```

**Fix:** Keep as `json.RawMessage`:
```go
// ✅ GOOD: Keep as RawMessage
// It's already the right type for JSON operations
```

### Mistake 4: Using map[string]interface{}

```go
// ❌ BAD: Loses JSON structure
type Job struct {
    Payload map[string]interface{}  // Structure is lost
}
```

**Problems:**
- Numbers become `float64` (loses precision)
- Order is not preserved
- Type information is lost

**Fix:** Use `json.RawMessage`:
```go
// ✅ GOOD: Preserves structure
type Job struct {
    Payload json.RawMessage  // Original JSON preserved
}
```

### Mistake 5: Not Validating at All

```go
// ❌ BAD: Accepting invalid JSON
job := domain.NewJob("email", json.RawMessage("not json"))
```

**Fix:** Validate in handler:
```go
// ✅ GOOD: Validate before creating job
var payloadCheck interface{}
if err := json.Unmarshal(request.Payload, &payloadCheck); err != nil {
    ErrorResponse(w, "Payload must be valid JSON", http.StatusBadRequest)
    return
}
```

---

## Key Takeaways

1. **`json.RawMessage`** = Opaque JSON container (just `[]byte` with special behavior)
2. **Opaque JSON** = Domain doesn't know/care about structure
3. **Use when** structure varies or you need flexibility
4. **Don't use when** structure is always the same
5. **Preserves** original JSON structure
6. **Validate** in HTTP layer, not domain layer

---

## Next Steps

- Read [HTTP Request Parsing](./03-http-request-parsing.md) to see how we read and parse requests
- Read [Request Validation](./04-request-validation.md) to see how we validate opaque payloads
- Read [Domain Modeling](./01-domain-modeling.md) to understand why we use opaque types

