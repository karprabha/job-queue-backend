# Understanding Time Handling in Go

## Table of Contents

1. [time.Time Type](#timetime-type)
2. [UTC vs Local Time](#utc-vs-local-time)
3. [Why Always Use UTC?](#why-always-use-utc)
4. [Time Formatting](#time-formatting)
5. [Our Implementation](#our-implementation)
6. [Common Mistakes](#common-mistakes)

---

## time.Time Type

### What is time.Time?

`time.Time` is Go's standard type for representing time.

**Characteristics:**

- Represents a point in time
- Timezone-aware
- Rich API (formatting, comparison, etc.)
- Nanosecond precision

### Creating time.Time

```go
import "time"

// Current time
now := time.Now()

// UTC time
utcNow := time.Now().UTC()

// Specific time
specific := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
```

---

## UTC vs Local Time

### UTC (Coordinated Universal Time)

**What:** Standard time, no timezone offset

**Example:** `2024-01-01T12:00:00Z`

**Why use UTC:**

- Consistent across servers
- No timezone confusion
- Standard for APIs
- Easy to convert to any timezone

### Local Time

**What:** Time in server's timezone

**Example:** `2024-01-01T12:00:00-05:00` (EST)

**Problems:**

- Varies by server location
- Timezone confusion
- Hard to compare
- Not standard for APIs

---

## Why Always Use UTC?

### The Problem with Local Time

**Server in New York:**

```go
createdAt := time.Now()  // EST time
// 2024-01-01 12:00:00 EST
```

**Server in London:**

```go
createdAt := time.Now()  // GMT time
// 2024-01-01 12:00:00 GMT
```

**Same moment, different times!**

### The Solution: UTC

**All servers:**

```go
createdAt := time.Now().UTC()  // UTC time
// 2024-01-01 17:00:00 UTC (same everywhere!)
```

**Benefits:**

- Consistent across servers
- No timezone confusion
- Standard for APIs
- Easy to convert client-side

---

## Time Formatting

### RFC3339 Format

**Standard:** ISO 8601 / RFC 3339

**Format:** `2006-01-02T15:04:05Z07:00`

**Example:** `2024-01-01T12:00:00Z`

### Our Formatting

```go
CreatedAt: job.CreatedAt.Format(time.RFC3339)
```

**What this does:**

- Converts `time.Time` to string
- Uses RFC3339 format
- Standard for JSON APIs

**Result:**

```json
{
  "created_at": "2024-01-01T12:00:00Z"
}
```

### Why RFC3339?

**Benefits:**

- Standard format
- Easy to parse
- Human-readable
- Works with JavaScript `Date()`

---

## Our Implementation

### Domain Layer

```go
type Job struct {
    CreatedAt time.Time  // Store as time.Time
}

func NewJob(...) *Job {
    return &Job{
        CreatedAt: time.Now().UTC(),  // Always UTC!
    }
}
```

**Why time.Time?**

- Rich API
- Timezone-aware
- Standard Go type

**Why UTC?**

- Consistent
- Standard for APIs
- No timezone confusion

### HTTP Layer

```go
type CreateJobResponse struct {
    CreatedAt string `json:"created_at"`  // String for JSON
}

response := CreateJobResponse{
    CreatedAt: job.CreatedAt.Format(time.RFC3339),  // Format to string
}
```

**Why string in response?**

- JSON needs strings
- RFC3339 is standard
- Easy to parse client-side

---

## Common Mistakes

### Mistake 1: Using Local Time

```go
// ❌ BAD: Local time
CreatedAt: time.Now()  // Varies by server!
```

**Fix:**

```go
// ✅ GOOD: UTC time
CreatedAt: time.Now().UTC()
```

### Mistake 2: Wrong Format

```go
// ❌ BAD: Custom format
CreatedAt: time.Now().Format("2006-01-02 15:04:05")  // Not standard
```

**Fix:**

```go
// ✅ GOOD: RFC3339
CreatedAt: time.Now().Format(time.RFC3339)
```

### Mistake 3: Storing as String

```go
// ❌ BAD: String in domain
type Job struct {
    CreatedAt string  // Can't do time operations!
}
```

**Fix:**

```go
// ✅ GOOD: time.Time in domain
type Job struct {
    CreatedAt time.Time  // Can do all time operations
}
```

---

## Key Takeaways

1. **time.Time** = Go's standard time type
2. **Always UTC** = Consistent across servers
3. **RFC3339** = Standard format for APIs
4. **Store as time.Time** = Rich API, timezone-aware
5. **Format for JSON** = Convert to string in responses

---

## Next Steps

- Read [Domain Modeling](./01-domain-modeling.md) to see time in domain model
- Read [HTTP Handler Patterns](./11-http-handler-patterns.md) to see time formatting in responses
