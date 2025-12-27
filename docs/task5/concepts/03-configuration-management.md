# Configuration Management

## Table of Contents

1. [Why Configuration Management?](#why-configuration-management)
2. [Hardcoded vs Configurable](#hardcoded-vs-configurable)
3. [Our Configuration Approach](#our-configuration-approach)
4. [Environment Variables](#environment-variables)
5. [Configuration Struct](#configuration-struct)
6. [Default Values](#default-values)
7. [Error Handling in Config](#error-handling-in-config)
8. [Common Mistakes](#common-mistakes)

---

## Why Configuration Management?

### The Problem with Hardcoded Values

In Task 4, we had hardcoded values:

```go
const jobQueueCapacity = 100
port := "8080"
```

**Problems:**
- Can't change without recompiling
- Different environments need different values
- Testing requires code changes
- Not flexible

### The Solution: Configuration

**Configuration** = External values that control program behavior without code changes.

**Benefits:**
- Change behavior without recompiling
- Different values per environment (dev, staging, prod)
- Easy testing (just change config)
- Flexible and maintainable

---

## Hardcoded vs Configurable

### Hardcoded (Task 4)

```go
const jobQueueCapacity = 100
port := "8080"
workerCount := 1
```

**Characteristics:**
- Values in code
- Must recompile to change
- Same for all environments
- Simple but inflexible

### Configurable (Task 5)

```go
config := config.NewConfig()
// config.JobQueueCapacity = 100 (from env or default)
// config.Port = "8080" (from env or default)
// config.WorkerCount = 10 (from env or default)
```

**Characteristics:**
- Values from environment or defaults
- Change without recompiling
- Different per environment
- More code but flexible

---

## Our Configuration Approach

### The Config Package

We created a separate `config` package:

```go
package config

type Config struct {
    Port             string
    JobQueueCapacity int
    WorkerCount      int
}

func NewConfig() *Config {
    // Read from environment variables
    // Provide defaults if not set
    // Return configured struct
}
```

### Why Separate Package?

**Benefits:**
1. **Separation of concerns** - Config logic isolated
2. **Reusability** - Can be used by multiple packages
3. **Testability** - Easy to test config logic
4. **Maintainability** - Changes to config don't affect other code

---

## Environment Variables

### What Are Environment Variables?

**Environment variables** = Key-value pairs set in the operating system or process environment.

**Examples:**
```bash
export PORT=8080
export WORKER_COUNT=10
export JOB_QUEUE_CAPACITY=100
```

### Reading Environment Variables in Go

```go
port := os.Getenv("PORT")
```

**What it does:**
- Looks for environment variable named `PORT`
- Returns its value as string
- Returns empty string if not set

### Our Implementation

```go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"  // Default value
}
```

**Pattern:**
1. Try to read from environment
2. If empty, use default
3. This allows flexibility with sensible defaults

---

## Configuration Struct

### The Config Type

```go
type Config struct {
    Port             string
    JobQueueCapacity int
    WorkerCount      int
}
```

**Why a struct?**
- Groups related configuration
- Type-safe (each field has a type)
- Easy to pass around
- Self-documenting (fields show what's configurable)

### Using the Config

```go
config := config.NewConfig()

// Use config values
jobQueue := make(chan *domain.Job, config.JobQueueCapacity)
srv := &http.Server{
    Addr: ":" + config.Port,
}
for i := 0; i < config.WorkerCount; i++ {
    // Create workers
}
```

**Benefits:**
- All config in one place
- Type-safe access
- Easy to extend (just add fields)

---

## Default Values

### Why Defaults Matter

**Scenario:** User doesn't set `WORKER_COUNT` environment variable.

**Without defaults:**
```go
workerCount := os.Getenv("WORKER_COUNT")
// workerCount = "" (empty string)
// Can't use empty string as int!
```

**With defaults:**
```go
workerCount := os.Getenv("WORKER_COUNT")
if workerCount == "" {
    workerCount = "10"  // Sensible default
}
```

### Our Defaults

```go
func NewConfig() *Config {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"  // Default port
    }

    jobQueueCapacity := os.Getenv("JOB_QUEUE_CAPACITY")
    if jobQueueCapacity == "" {
        jobQueueCapacity = "100"  // Default capacity
    }

    workerCount := os.Getenv("WORKER_COUNT")
    if workerCount == "" {
        workerCount = "10"  // Default worker count
    }
    // ...
}
```

**Why these defaults?**
- **Port 8080:** Common HTTP port, safe default
- **Capacity 100:** Handles bursts, provides backpressure
- **Workers 10:** Good balance for most workloads

---

## Error Handling in Config

### The Problem

Environment variables are **strings**, but we need **integers**:

```go
workerCount := os.Getenv("WORKER_COUNT")  // Returns string "10"
// But we need int 10
```

### Converting Strings to Integers

```go
workerCountInt, err := strconv.Atoi(workerCount)
if err != nil {
    workerCountInt = 10  // Use default on error
}
```

**What `strconv.Atoi` does:**
- Converts string to integer
- Returns error if string is not a valid number
- Example: `"10"` → `10`, `"abc"` → error

### Our Error Handling Pattern

```go
workerCountInt, err := strconv.Atoi(workerCount)
if err != nil {
    workerCountInt = 10  // Fallback to default
}
```

**Why this pattern?**
- **Fail gracefully** - Don't crash on bad config
- **Use defaults** - Sensible fallback
- **Log if needed** - Could log the error (we don't, but could)

### Complete Example

```go
workerCount := os.Getenv("WORKER_COUNT")
if workerCount == "" {
    workerCount = "10"  // Default as string
}

workerCountInt, err := strconv.Atoi(workerCount)
if err != nil {
    // Invalid value (e.g., "abc"), use default
    workerCountInt = 10
}

// Now workerCountInt is guaranteed to be a valid int
```

---

## Common Mistakes

### Mistake 1: No Defaults

```go
// ❌ BAD: Crashes if env var not set
port := os.Getenv("PORT")
srv := &http.Server{
    Addr: ":" + port,  // port might be ""
}
```

**Fix:** Provide defaults
```go
// ✅ GOOD: Has defaults
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
```

### Mistake 2: Not Handling Conversion Errors

```go
// ❌ BAD: Crashes on invalid input
workerCount, _ := strconv.Atoi(os.Getenv("WORKER_COUNT"))
// If "abc" is set, workerCount = 0 (silent failure)
```

**Fix:** Handle errors
```go
// ✅ GOOD: Handles errors
workerCountStr := os.Getenv("WORKER_COUNT")
if workerCountStr == "" {
    workerCountStr = "10"
}
workerCount, err := strconv.Atoi(workerCountStr)
if err != nil {
    workerCount = 10  // Default on error
}
```

### Mistake 3: Hardcoding Values

```go
// ❌ BAD: Can't change without code
const workerCount = 10
```

**Fix:** Make configurable
```go
// ✅ GOOD: Configurable
config := config.NewConfig()
// config.WorkerCount from env or default
```

### Mistake 4: No Validation

```go
// ❌ BAD: Accepts invalid values
workerCount := -5  // Negative workers?!
```

**Fix:** Validate
```go
// ✅ GOOD: Validates
if workerCount < 1 {
    workerCount = 1  // Minimum 1 worker
}
if workerCount > 100 {
    workerCount = 100  // Maximum 100 workers
}
```

---

## Key Takeaways

1. **Configuration** = External values, no recompilation needed
2. **Environment variables** = Common way to provide config
3. **Defaults** = Sensible fallbacks when config not provided
4. **Error handling** = Graceful degradation on invalid config
5. **Config struct** = Groups related configuration
6. **Separate package** = Better organization and testability

---

## Real-World Analogy

Think of a car:

- **Hardcoded** = Car with fixed settings (can't change radio station)
- **Configurable** = Car with adjustable settings (can change radio, temperature, etc.)
- **Environment variables** = Settings stored in the car's memory
- **Defaults** = Factory settings if memory is cleared

---

## Next Steps

- Read [Proper Shutdown Order](./04-proper-shutdown-order.md) to see how config affects shutdown
- Read [WaitGroup with Multiple Goroutines](./05-waitgroup-multiple-goroutines.md) to understand how we track configured number of workers

