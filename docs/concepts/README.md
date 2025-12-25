# Go Concepts Explained

This directory contains detailed explanations of Go concepts used in this project, written for beginners.

## 📚 Concepts Covered

### 1. [Context](./01-context.md)

- What is context and why it exists
- `context.Background()` explained
- `context.WithTimeout()` deep dive
- The cancel function and `defer cancel()`
- How context works internally
- Real-world example: Server shutdown

### 2. [Goroutines and Channels](./02-goroutines-channels.md)

- Why goroutines exist
- What is a goroutine?
- How to create goroutines
- What is a channel?
- Channel operations (send/receive)
- Blocking and non-blocking
- Real example: Server in goroutine
- Signal channel deep dive

### 3. [HTTP Server](./03-http-server.md)

- HTTP server basics
- `http.Server` vs `http.ListenAndServe`
- HTTP handlers explained
- Request and response
- Health check handler breakdown
- Error handling in HTTP
- Server shutdown process

### 4. [Signal Handling](./04-signal-handling.md)

- What are OS signals?
- Why we need signal handling
- Common signals (SIGINT, SIGTERM, SIGKILL)
- Go's signal package
- Our signal handling code explained
- How `signal.Notify` works
- Complete shutdown flow

### 5. [Error Handling](./05-error-handling.md)

- Go's error philosophy
- What are errors in Go?
- Error handling patterns
- Error handling in our code
- Common mistakes
- Best practices

### 6. [JSON Encoding/Decoding](./06-json-encoding-decoding.md)

- Marshal/Unmarshal vs Encoder/Decoder
- When to use each approach
- Our handler implementation explained
- Buffer approach for error handling
- Performance considerations

### 7. [Project Structure](./07-project-structure.md)

- Standard Go project layout
- Our project structure analysis
- Is our structure idiomatic?
- Package naming conventions
- The `internal/` package
- The `cmd/` directory

### 8. [Context in Handlers](./08-context-in-handlers.md)

- Do we need context in handlers?
- What is request context?
- How to use context in handlers
- Context cancellation
- Should our handler use context?
- Best practices

## 🎯 How to Use This

These documents are designed to be read **in order** if you're new to Go. Each concept builds on previous ones.

**Recommended reading order:**

1. Start with [Goroutines and Channels](./02-goroutines-channels.md) - Foundation for concurrency
2. Then [Context](./01-context.md) - How to control goroutines
3. Then [Signal Handling](./04-signal-handling.md) - How OS signals work
4. Then [HTTP Server](./03-http-server.md) - How web servers work
5. Finally [Error Handling](./05-error-handling.md) - How to handle errors properly

Or read them as you encounter concepts in the code!

## 💡 Learning Approach

Each document:

- Explains **why** things exist (not just what they do)
- Breaks down code **line by line**
- Uses **analogies** and **mental models**
- Shows **common mistakes** to avoid
- Provides **real examples** from our codebase

## 🔗 Related Resources

- [Go Official Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)

## 📝 Contributing

If you find something unclear or want to add explanations, feel free to update these documents!
