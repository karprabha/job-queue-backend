# JSON Encoding and Decoding in Go

## Table of Contents

1. [Overview: Two Approaches](#overview-two-approaches)
2. [json.Marshal / json.Unmarshal](#jsonmarshal--jsonunmarshal)
3. [json.Encoder / json.Decoder](#jsonencoder--jsondecoder)
4. [Detailed Comparison](#detailed-comparison)
5. [When to Use Each](#when-to-use-each)
6. [Our Handler Implementation](#our-handler-implementation)
7. [Common Patterns](#common-patterns)
8. [Performance Considerations](#performance-considerations)

---

## Overview: Two Approaches

Go provides **two main ways** to work with JSON:

### 1. Marshal/Unmarshal (Memory-Based)

- Convert Go value ↔ JSON bytes in memory
- Returns `[]byte` (byte slice)
- Good for: Small data, simple cases

### 2. Encoder/Decoder (Stream-Based)

- Convert Go value ↔ JSON directly to/from `io.Writer`/`io.Reader`
- Streams data (no intermediate memory)
- Good for: Large data, HTTP responses, files

---

## json.Marshal / json.Unmarshal

### json.Marshal

**What it does:**

- Takes a Go value
- Converts it to JSON
- Returns JSON as `[]byte` (byte slice)
- Stores everything in memory first

**Function signature:**

```go
func Marshal(v interface{}) ([]byte, error)
```

**Example:**

```go
type Person struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

person := Person{Name: "Alice", Age: 30}
jsonBytes, err := json.Marshal(person)
if err != nil {
    // Handle error
}

// jsonBytes is []byte containing: {"name":"Alice","age":30}
fmt.Println(string(jsonBytes))
```

**What happens internally:**

1. Go creates a JSON representation in memory
2. Allocates a byte slice
3. Writes JSON into the byte slice
4. Returns the byte slice

**Memory usage:**

- Entire JSON is in memory at once
- For large objects, this uses more memory

### json.Unmarshal

**What it does:**

- Takes JSON `[]byte`
- Converts it to a Go value
- Fills the Go struct/map

**Function signature:**

```go
func Unmarshal(data []byte, v interface{}) error
```

**Example:**

```go
jsonStr := `{"name":"Alice","age":30}`
jsonBytes := []byte(jsonStr)

var person Person
err := json.Unmarshal(jsonBytes, &person)
if err != nil {
    // Handle error
}

// person.Name = "Alice"
// person.Age = 30
```

**What happens internally:**

1. Takes the entire JSON byte slice
2. Parses it completely
3. Fills the Go struct
4. Returns error if JSON is invalid

**Memory usage:**

- Entire JSON must be in memory
- Entire Go struct is created at once

---

## json.Encoder / json.Decoder

### json.Encoder

**What it does:**

- Takes an `io.Writer` (like `http.ResponseWriter`, file, buffer)
- Writes JSON **directly** to the writer
- Streams data (no intermediate memory for JSON)

**Creating an encoder:**

```go
encoder := json.NewEncoder(writer)
```

**Function signature:**

```go
func (enc *Encoder) Encode(v interface{}) error
```

**Example:**

```go
person := Person{Name: "Alice", Age: 30}

// Write directly to HTTP response
encoder := json.NewEncoder(w)
encoder.Encode(person)
// JSON is written directly to w (http.ResponseWriter)
```

**What happens internally:**

1. Encoder writes JSON directly to the writer
2. No intermediate byte slice for JSON
3. Data flows: Go value → JSON → Writer
4. More memory efficient for large data

**Key point:** Encoder writes **directly** to the destination. No intermediate storage.

### json.Decoder

**What it does:**

- Takes an `io.Reader` (like `http.Request.Body`, file)
- Reads JSON **directly** from the reader
- Streams data (can process as it reads)

**Creating a decoder:**

```go
decoder := json.NewDecoder(reader)
```

**Function signature:**

```go
func (dec *Decoder) Decode(v interface{}) error
```

**Example:**

```go
var person Person

// Read directly from HTTP request body
decoder := json.NewDecoder(r.Body)
err := decoder.Decode(&person)
if err != nil {
    // Handle error
}

// person is now filled from JSON in request body
```

**What happens internally:**

1. Decoder reads JSON from the reader
2. Parses it as it reads (streaming)
3. Fills the Go struct
4. More memory efficient for large JSON

---

## Detailed Comparison

### Memory Usage

| Method      | Memory Pattern        | Best For                   |
| ----------- | --------------------- | -------------------------- |
| `Marshal`   | Entire JSON in memory | Small data                 |
| `Unmarshal` | Entire JSON in memory | Small data                 |
| `Encoder`   | Streams to writer     | Large data, HTTP responses |
| `Decoder`   | Streams from reader   | Large data, HTTP requests  |

### Code Patterns

**Marshal pattern:**

```go
// 1. Convert to bytes
jsonBytes, err := json.Marshal(data)
if err != nil {
    return err
}

// 2. Write bytes
w.Header().Set("Content-Type", "application/json")
w.Write(jsonBytes)
```

**Encoder pattern:**

```go
// 1. Set headers
w.Header().Set("Content-Type", "application/json")

// 2. Encode directly
json.NewEncoder(w).Encode(data)
```

### Error Handling

**Marshal:**

- Error happens **before** writing
- Can check error, then write
- Can set error status code easily

**Encoder:**

- Error happens **during** writing
- If error occurs, headers/status may already be written
- Harder to handle errors after writing starts

**This is why we use buffer approach!**

---

## When to Use Each

### Use Marshal/Unmarshal When:

✅ **Small data structures**

- Simple responses
- Configuration files
- Small API payloads

✅ **You need the JSON bytes**

- Storing in database
- Comparing JSON strings
- Manipulating JSON before sending

✅ **Simple error handling**

- Want to check encoding before writing
- Need to validate JSON before use

**Example:**

```go
// Small health check response
response := HealthCheckResponse{Status: "ok"}
jsonBytes, err := json.Marshal(response)
if err != nil {
    return err  // Can handle before writing
}
w.Write(jsonBytes)
```

### Use Encoder/Decoder When:

✅ **Large data**

- Streaming large objects
- Large file processing
- Big API responses

✅ **Direct I/O**

- Writing to HTTP response
- Reading from HTTP request
- File I/O

✅ **Memory efficiency**

- Don't want entire JSON in memory
- Processing as you go

**Example:**

```go
// Large list of items
w.Header().Set("Content-Type", "application/json")
encoder := json.NewEncoder(w)
for _, item := range largeList {
    encoder.Encode(item)  // Stream each item
}
```

### Use Buffer + Encoder When:

✅ **Need error handling before writing**

- Want to check encoding errors
- Need to set error status if encoding fails
- Our current handler pattern!

**Example (our handler):**

```go
// Encode to buffer first
buffer := bytes.NewBuffer(nil)
encoder := json.NewEncoder(buffer)
err := encoder.Encode(data)
if err != nil {
    http.Error(w, "Encoding failed", 500)  // Can still set error status
    return
}

// Now write to response
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(200)
w.Write(buffer.Bytes())
```

---

## Our Handler Implementation

### Current Code

```go
buffer := bytes.NewBuffer(nil)
encoder := json.NewEncoder(buffer)
encoder.SetIndent("", "  ")
err := encoder.Encode(responseData)
if err != nil {
    http.Error(w, "Failed to encode response", http.StatusInternalServerError)
    return
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write(buffer.Bytes())
```

### Why This Approach?

**Problem with direct encoding:**

```go
// ❌ Problematic
w.Header().Set("Content-Type", "application/json")
err := json.NewEncoder(w).Encode(data)
if err != nil {
    http.Error(w, "Error", 500)  // Too late! Headers already written
}
```

**Why it's problematic:**

- `Encode()` writes to `w` immediately
- This triggers `WriteHeader(200)` automatically
- If encoding fails mid-write, you can't change status to 500
- Response might be partially written

**Our solution (buffer approach):**

```go
// ✅ Correct
buffer := bytes.NewBuffer(nil)  // Temporary storage
encoder := json.NewEncoder(buffer)  // Encode to buffer, not response
err := encoder.Encode(data)  // Error happens here, before writing to response
if err != nil {
    http.Error(w, "Error", 500)  // Can still set error status!
    return
}
// Only write to response if encoding succeeded
w.Write(buffer.Bytes())
```

**Benefits:**

1. **Error handling works** - Can set error status if encoding fails
2. **No partial writes** - Either all JSON or error response
3. **Control over headers** - Set headers after knowing encoding succeeded
4. **Safe** - No risk of corrupted responses

### About SetIndent

```go
encoder.SetIndent("", "  ")
```

**What this does:**

- Makes JSON "pretty" (formatted with indentation)
- First parameter: prefix for each line (empty = no prefix)
- Second parameter: indentation string (`"  "` = 2 spaces)

**Output:**

```json
{
  "status": "ok"
}
```

**Instead of:**

```json
{ "status": "ok" }
```

**Trade-off:**

- **Pro:** Human-readable (good for debugging)
- **Con:** Larger response size (more bytes)
- **Con:** Slightly slower encoding

**For production:** Consider removing `SetIndent` to save bandwidth.

---

## Common Patterns

### Pattern 1: Simple Response (Marshal)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data := map[string]string{"status": "ok"}

    jsonBytes, err := json.Marshal(data)
    if err != nil {
        http.Error(w, "Encoding failed", 500)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(jsonBytes)
}
```

### Pattern 2: Direct Encoding (Simple Case)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data := map[string]string{"status": "ok"}

    w.Header().Set("Content-Type", "application/json")
    // Accept risk: if encoding fails, can't change status
    json.NewEncoder(w).Encode(data)
}
```

### Pattern 3: Buffer Approach (Our Pattern - Safest)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data := map[string]string{"status": "ok"}

    var buf bytes.Buffer
    if err := json.NewEncoder(&buf).Encode(data); err != nil {
        http.Error(w, "Encoding failed", 500)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)
    w.Write(buf.Bytes())
}
```

### Pattern 4: Reading Request Body (Decoder)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    var requestData MyStruct

    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&requestData); err != nil {
        http.Error(w, "Invalid JSON", 400)
        return
    }

    // Use requestData...
}
```

### Pattern 5: Reading Request Body (Unmarshal)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Read failed", 500)
        return
    }

    var requestData MyStruct
    if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
        http.Error(w, "Invalid JSON", 400)
        return
    }

    // Use requestData...
}
```

---

## Performance Considerations

### Memory

**Marshal:**

- Entire JSON in memory: `O(n)` where n = JSON size
- Plus Go struct in memory
- **Total: 2x memory** (struct + JSON bytes)

**Encoder (direct):**

- Only Go struct in memory
- JSON streams directly to writer
- **Total: 1x memory** (just struct)

**Encoder (buffer):**

- Go struct in memory
- JSON in buffer
- **Total: 2x memory** (struct + buffer)
- But: Allows error handling!

### Speed

**Marshal:**

- Fast for small data
- Slower for large data (must allocate large slice)

**Encoder:**

- Consistent performance
- Better for large data (streaming)

### For Our Use Case

**Health check response:**

- Tiny JSON: `{"status":"ok"}`
- Performance difference: **Negligible**
- **Buffer approach is still better** because:
  - Proper error handling
  - No risk of partial writes
  - Code is safer

**Recommendation:** Keep buffer approach for safety, even for small responses.

---

## Key Takeaways

1. **Marshal/Unmarshal** - Memory-based, good for small data
2. **Encoder/Decoder** - Stream-based, good for large data
3. **Buffer + Encoder** - Best for HTTP handlers (error handling)
4. **Direct Encoder** - Risky (can't handle errors after writing starts)
5. **Choose based on:** Data size, error handling needs, memory constraints

---

## Common Mistakes

❌ **Direct encoding without error handling**

```go
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(data)  // What if this fails?
```

✅ **Buffer approach with error handling**

```go
var buf bytes.Buffer
if err := json.NewEncoder(&buf).Encode(data); err != nil {
    http.Error(w, "Encoding failed", 500)
    return
}
w.Header().Set("Content-Type", "application/json")
w.Write(buf.Bytes())
```

❌ **Using Marshal for large data**

```go
jsonBytes, err := json.Marshal(hugeData)  // Uses lots of memory!
```

✅ **Use Encoder for large data**

```go
json.NewEncoder(w).Encode(hugeData)  // Streams, uses less memory
```

---

## Next Steps

- Review [HTTP Server](./03-http-server.md) - How handlers work
- Understand [Error Handling](./05-error-handling.md) - Error patterns
- Learn about [Context](./01-context.md) - Request context
