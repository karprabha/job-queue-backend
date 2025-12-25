# Go Concepts Explained - Task 3

This directory contains detailed explanations of Go concepts used in Task 3, written for beginners learning Go.

## 📚 Concepts Covered

### 1. [Dependency Injection](./01-dependency-injection.md)

- What is dependency injection?
- Why inject dependencies instead of creating them?
- Constructor functions for DI
- Real example: Our job handler
- Dependency injection patterns
- Common mistakes

### 2. [Handler Struct Pattern](./02-handler-struct-pattern.md)

- Function handlers vs struct handlers
- Why use struct handlers?
- Method receivers explained
- Real example: Our refactoring
- When to use struct handlers
- Common patterns

### 3. [In-Memory Storage with Maps](./03-in-memory-storage.md)

- What is in-memory storage?
- Why use maps for storage?
- Map basics in Go
- Our in-memory store implementation
- Map operations explained
- Converting map to slice
- Sorting results

### 4. [Concurrency Safety with Mutexes](./04-concurrency-safety.md)

- What is concurrency?
- The problem: Race conditions
- What is a mutex?
- How mutexes work
- Using mutexes in our store
- Lock and unlock patterns
- Common mistakes

### 5. [RWMutex vs Mutex](./05-rwmutex-vs-mutex.md)

- The problem: Read vs write operations
- What is RWMutex?
- Mutex vs RWMutex comparison
- How RWMutex works
- Our implementation: Why RWMutex?
- Performance considerations
- When to use which?

### 6. [Context in Storage Layer](./06-context-in-storage.md)

- Why context in storage?
- Context for cancellation
- Our implementation
- Context check before lock
- Context check after lock
- When context cancellation matters
- Best practices

### 7. [Interface Design for Storage](./07-interface-design.md)

- What is an interface?
- Why use interfaces for storage?
- Our JobStore interface
- Interface vs concrete type
- Dependency injection with interfaces
- Testing with interfaces
- Future implementations

## 🎯 How to Use This

These documents are designed to be read **in order** if you're new to Go. Each concept builds on previous ones.

**Recommended reading order:**

1. Start with [Dependency Injection](./01-dependency-injection.md) - Foundation for how dependencies work
2. Then [Handler Struct Pattern](./02-handler-struct-pattern.md) - How struct handlers enable DI
3. Then [Interface Design](./07-interface-design.md) - Why interfaces matter for DI
4. Then [In-Memory Storage](./03-in-memory-storage.md) - How we store data
5. Then [Concurrency Safety](./04-concurrency-safety.md) - Why we need mutexes
6. Then [RWMutex vs Mutex](./05-rwmutex-vs-mutex.md) - Why we chose RWMutex
7. Finally [Context in Storage Layer](./06-context-in-storage.md) - How context flows through layers

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

### Task 1 Concepts

- [Context in Go](../task1/concepts/01-context.md)
- [Goroutines and Channels](../task1/concepts/02-goroutines-channels.md)
- [Context in Handlers](../task1/concepts/08-context-in-handlers.md)

### Task 2 Concepts

- [Domain Modeling](../task2/concepts/01-domain-modeling.md)
- [Domain Separation](../task2/concepts/10-domain-separation.md)
- [HTTP Handler Patterns](../task2/concepts/11-http-handler-patterns.md)

### External Resources

- [Go Official Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)

## 📝 Key Concepts Summary

### Dependency Injection

- Dependencies come from outside, not created inside
- Constructor functions: `NewHandler(dep)`
- Enables testing and flexibility

### Struct Handlers

- Methods on structs that hold dependencies
- Enable dependency injection
- Better than function handlers when you need state

### In-Memory Storage

- Maps for key-value storage
- Fast but temporary
- Must convert to slice for ordered results

### Concurrency Safety

- Mutexes prevent race conditions
- Always use `defer Unlock()`
- Protect all shared data access

### RWMutex

- Allows concurrent reads
- Exclusive writes
- Better for read-heavy workloads

### Context in Storage

- Check context before acquiring lock
- Respect cancellation
- Don't waste resources on canceled requests

### Interfaces

- Define contracts, not implementations
- Enable flexibility and testing
- Keep interfaces small

## 🎓 What You'll Learn

After reading these documents, you'll understand:

- ✅ How to structure code with dependency injection
- ✅ When to use struct handlers vs function handlers
- ✅ How to safely store data in memory
- ✅ How to protect shared data from race conditions
- ✅ When to use RWMutex vs regular Mutex
- ✅ How context flows through application layers
- ✅ How interfaces enable flexibility and testing

## 🚀 Next Steps

After Task 3, you'll be ready for:

- Background workers (processing jobs)
- Database persistence
- More complex concurrency patterns
- Advanced error handling

## 📝 Contributing

If you find something unclear or want to add explanations, feel free to update these documents!
