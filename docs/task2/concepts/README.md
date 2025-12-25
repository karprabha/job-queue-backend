# Go Concepts Explained - Task 2

This directory contains detailed explanations of Go concepts used in Task 2, written for beginners learning Go.

## 📚 Concepts Covered

### 1. [Domain Modeling](./01-domain-modeling.md)

- What is domain modeling?
- Why separate domain from HTTP layer?
- Struct design principles
- Typed constants vs strings
- Constructor functions (`NewJob`)
- The `internal/domain` package pattern

### 2. [JSON RawMessage - Opaque Payloads](./02-json-rawmessage.md)

- What is `json.RawMessage`?
- Why use opaque JSON?
- Opaque vs concrete types
- How `json.RawMessage` works internally
- When to use `json.RawMessage` vs structs
- Real example: Job payload

### 3. [HTTP Request Parsing](./03-http-request-parsing.md)

- Reading request body with `io.ReadAll`
- Unmarshaling JSON requests
- Request struct design
- Content-Type validation
- Request body lifecycle
- Common pitfalls

### 4. [Request Validation](./04-request-validation.md)

- Why validate requests?
- Validation patterns in Go
- Empty string checks
- JSON validation
- Error messages and user experience
- Validation vs business logic

### 5. [Error Response Centralization](./05-error-response-centralization.md)

- Why centralize error responses?
- The `ErrorResponse` function design
- Error response format consistency
- HTTP status codes in errors
- Fallback error handling
- When headers are already written

### 6. [HTTP Status Codes](./06-http-status-codes.md)

- HTTP status code categories
- When to use which status code
- `201 Created` vs `200 OK`
- `400 Bad Request` vs `422 Unprocessable Entity`
- `413 Request Entity Too Large`
- Status code best practices

### 7. [Request Body Size Limiting](./07-request-body-size-limiting.md)

- Why limit request body size?
- `http.MaxBytesReader` explained
- Security implications
- Error detection patterns
- Choosing appropriate limits
- Real-world considerations

### 8. [UUID Generation](./08-uuid-generation.md)

- What is a UUID?
- Why use UUIDs for IDs?
- The `google/uuid` package
- UUID versions
- UUID vs auto-increment
- When to use UUIDs

### 9. [Time Handling in Go](./09-time-handling.md)

- `time.Time` type explained
- UTC vs local time
- Why always use UTC?
- Time formatting (RFC3339)
- Time zones and APIs
- Best practices

### 10. [Domain Separation](./10-domain-separation.md)

- What is domain separation?
- Why separate domain from HTTP?
- The `internal/domain` pattern
- Domain vs infrastructure
- Clean architecture principles
- Real example: Job domain

### 11. [HTTP Handler Patterns](./11-http-handler-patterns.md)

- Handler function signature
- Request/Response flow
- Handler structure and organization
- Response writing patterns
- Error handling in handlers
- Handler best practices

### 12. [Enhanced ServeMux](./12-enhanced-servemux.md)

- What is enhanced ServeMux?
- Method-specific routing (Go 1.22+)
- Before vs after refactoring
- Benefits of method-specific routing
- Our server refactoring
- Common mistakes

## 🎯 How to Use This

These documents are designed to be read **in order** if you're new to Go. Each concept builds on previous ones.

**Recommended reading order:**

1. Start with [Domain Modeling](./01-domain-modeling.md) - Foundation for organizing code
2. Then [Domain Separation](./10-domain-separation.md) - Why we structure code this way
3. Then [JSON RawMessage](./02-json-rawmessage.md) - Understanding opaque types
4. Then [HTTP Request Parsing](./03-http-request-parsing.md) - How to read requests
5. Then [Request Validation](./04-request-validation.md) - How to validate input
6. Then [Error Response Centralization](./05-error-response-centralization.md) - Consistent error handling
7. Then [HTTP Status Codes](./06-http-status-codes.md) - When to use which codes
8. Finally [HTTP Handler Patterns](./11-http-handler-patterns.md) - Putting it all together

Or read them as you encounter concepts in the code!

## 💡 Learning Approach

Each document:

- Explains **why** things exist (not just what they do)
- Breaks down code **line by line**
- Uses **analogies** and **mental models**
- Shows **common mistakes** to avoid
- Provides **real examples** from our codebase
- Explains **design decisions** and trade-offs

## 🔗 Related Resources

- [Go Official Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [HTTP Status Codes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status)
- [RFC 3339 (Date and Time)](https://datatracker.ietf.org/doc/html/rfc3339)
- [UUID Specification](https://tools.ietf.org/html/rfc4122)

## 📝 Contributing

If you find something unclear or want to add explanations, feel free to update these documents!
