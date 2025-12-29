# Understanding Metrics Endpoint Design

## Table of Contents

1. [Why Expose Metrics via HTTP?](#why-expose-metrics-via-http)
2. [Endpoint Design](#endpoint-design)
3. [Response Format](#response-format)
4. [Handler Separation](#handler-separation)
5. [Error Handling](#error-handling)
6. [Common Mistakes](#common-mistakes)

---

## Why Expose Metrics via HTTP?

### The Problem: Internal Metrics

**Without HTTP endpoint:**

- Metrics stored in memory
- No way to access metrics externally
- Can't monitor system health
- Can't integrate with monitoring tools

### The Solution: HTTP Endpoint

**With HTTP endpoint:**

- Metrics accessible via HTTP GET request
- Can be polled by monitoring tools
- Can be viewed in browser
- Can be integrated with Prometheus, Grafana, etc.

### Use Cases

1. **Health monitoring** - Check system health
2. **Alerting** - Alert when metrics exceed thresholds
3. **Dashboards** - Visualize metrics over time
4. **Debugging** - Inspect current system state

---

## Endpoint Design

### Our Endpoint

**Route:** `GET /metrics`

**Handler:** `MetricHandler.GetMetrics`

**Implementation:**

```go
mux.HandleFunc("GET /metrics", metricHandler.GetMetrics)
```

### Why GET?

**GET is appropriate because:**

- Reading data (not modifying)
- Idempotent (same request = same response)
- Can be cached
- Standard for metrics endpoints

### HTTP Status Codes

**Success:** `200 OK`

```go
w.WriteHeader(http.StatusOK)
```

**Error:** `500 Internal Server Error`

```go
ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
```

---

## Response Format

### JSON Format

**Our response:**

```json
{
  "total_jobs_created": 120,
  "jobs_completed": 110,
  "jobs_failed": 5,
  "jobs_retried": 10,
  "jobs_in_progress": 2
}
```

### Field Naming

**JSON field names:** snake_case

```go
type MetricResponse struct {
    TotalJobsCreated int `json:"total_jobs_created"`
    JobsCompleted    int `json:"jobs_completed"`
    JobsFailed       int `json:"jobs_failed"`
    JobsRetried      int `json:"jobs_retried"`
    JobsInProgress   int `json:"jobs_in_progress"`
}
```

**Why snake_case?**

- Consistent with log field naming
- Standard for JSON APIs
- Easy to read

### Content-Type Header

**Set JSON content type:**

```go
w.Header().Set("Content-Type", "application/json")
```

**Why?**

- Tells client how to parse response
- Standard HTTP practice
- Required for JSON APIs

---

## Handler Separation

### Separation of Concerns

**Handler responsibility:**

- Handle HTTP request/response
- Call metric store
- Format response
- Handle errors

**Metric store responsibility:**

- Store metrics
- Update metrics
- Return metrics data

### Our Implementation

**Handler (HTTP layer):**

```go
type MetricHandler struct {
    metricStore store.MetricStore
    logger      *slog.Logger
}

func (h *MetricHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    // Get metrics from store
    metrics, err := h.metricStore.GetMetrics(r.Context())
    if err != nil {
        ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
        return
    }
    
    // Format response
    response := MetricResponse{
        TotalJobsCreated: metrics.TotalJobsCreated,
        JobsCompleted:    metrics.JobsCompleted,
        JobsFailed:       metrics.JobsFailed,
        JobsRetried:      metrics.JobsRetried,
        JobsInProgress:   metrics.JobsInProgress,
    }
    
    // Send response
    // ...
}
```

**Store (business logic):**

```go
func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    m := *s.metrics
    return &m, nil
}
```

**Key point:** Handler doesn't know how metrics are stored, store doesn't know about HTTP.

---

## Error Handling

### Error Cases

**1. Store error:**

```go
metrics, err := h.metricStore.GetMetrics(r.Context())
if err != nil {
    ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
    return
}
```

**2. JSON marshal error:**

```go
responseBytes, err := json.Marshal(response)
if err != nil {
    ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
    return
}
```

**3. Write error:**

```go
if _, err := w.Write(responseBytes); err != nil {
    h.logger.Error("Failed to write response", "error", err)
    return
}
```

### Error Response Format

**Consistent error format:**

```json
{
  "error": "Failed to get metrics"
}
```

**Why consistent?**

- Easy for clients to parse
- Standard error format
- Clear error messages

---

## Common Mistakes

### Mistake 1: Handler Updates Metrics Directly

```go
// ❌ BAD: Handler updates metrics
type MetricHandler struct {
    metrics *domain.Metric
}

func (h *MetricHandler) GetMetrics() {
    h.metrics.TotalJobsCreated++  // Handler updates metrics!
}
```

**Fix:** Use metric store

```go
// ✅ GOOD: Handler calls store
type MetricHandler struct {
    metricStore store.MetricStore
}

func (h *MetricHandler) GetMetrics() {
    metrics, _ := h.metricStore.GetMetrics(ctx)
    // Handler only reads, doesn't update
}
```

### Mistake 2: Missing Content-Type Header

```go
// ❌ BAD: No content type
w.WriteHeader(http.StatusOK)
w.Write(responseBytes)
```

**Fix:** Set content type

```go
// ✅ GOOD: Set content type
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write(responseBytes)
```

### Mistake 3: Not Handling Errors

```go
// ❌ BAD: Ignore errors
metrics, _ := h.metricStore.GetMetrics(ctx)
json.Marshal(metrics)
```

**Fix:** Handle errors

```go
// ✅ GOOD: Handle errors
metrics, err := h.metricStore.GetMetrics(ctx)
if err != nil {
    ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
    return
}
```

### Mistake 4: Inconsistent Field Naming

```go
// ❌ BAD: Mixed naming
type MetricResponse struct {
    TotalJobsCreated int `json:"totalJobsCreated"`  // camelCase
    JobsCompleted    int `json:"jobs_completed"`     // snake_case
}
```

**Fix:** Consistent naming

```go
// ✅ GOOD: Consistent snake_case
type MetricResponse struct {
    TotalJobsCreated int `json:"total_jobs_created"`
    JobsCompleted    int `json:"jobs_completed"`
}
```

### Mistake 5: Not Logging Write Errors

```go
// ❌ BAD: Silent failure
if _, err := w.Write(responseBytes); err != nil {
    return  // No logging!
}
```

**Fix:** Log errors

```go
// ✅ GOOD: Log errors
if _, err := w.Write(responseBytes); err != nil {
    h.logger.Error("Failed to write response", "error", err)
    return
}
```

---

## Key Takeaways

1. **HTTP endpoint** = Makes metrics accessible externally
2. **GET method** = Appropriate for reading data
3. **JSON format** = Standard, easy to parse
4. **Handler separation** = Handler handles HTTP, store handles metrics
5. **Error handling** = Handle all error cases
6. **Content-Type** = Always set for JSON responses
7. **Consistent naming** = Use snake_case for JSON fields

---

## Real-World Example

**Our metrics endpoint:**

```go
func (h *MetricHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    // Get metrics from store
    metrics, err := h.metricStore.GetMetrics(r.Context())
    if err != nil {
        ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
        return
    }
    
    // Format response
    response := MetricResponse{
        TotalJobsCreated: metrics.TotalJobsCreated,
        JobsCompleted:    metrics.JobsCompleted,
        JobsFailed:       metrics.JobsFailed,
        JobsRetried:      metrics.JobsRetried,
        JobsInProgress:   metrics.JobsInProgress,
    }
    
    // Marshal to JSON
    responseBytes, err := json.Marshal(response)
    if err != nil {
        ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
        return
    }
    
    // Send response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    
    if _, err := w.Write(responseBytes); err != nil {
        h.logger.Error("Failed to write response", "error", err)
        return
    }
}
```

**Response:**

```json
{
  "total_jobs_created": 120,
  "jobs_completed": 110,
  "jobs_failed": 5,
  "jobs_retried": 10,
  "jobs_in_progress": 2
}
```

---

## Next Steps

- Read [Metrics Collection and Storage](./02-metrics-collection-storage.md) to understand what metrics we track
- Read [Concurrency-Safe Metrics](./04-concurrency-safe-metrics.md) to see how metrics are protected
- Read [Dependency Injection for Observability](./03-dependency-injection-observability.md) to see how we wire the handler

