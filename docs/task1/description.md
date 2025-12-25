# 🧱 Project: **Job Queue Backend (Go)**

You are building a **production-grade backend service** that:

- accepts jobs via HTTP
- stores them
- processes them asynchronously
- exposes status & metrics

This project will cover **almost all core Go backend concepts** over time.

---

# 📌 RULES (very important)

1. ❌ **Do not ask AI to write code**
2. ✅ You may ask AI (or me) to:

   - review code
   - explain concepts
   - point out non-idiomatic Go

3. ❌ No frameworks initially (Gin, Fiber, Echo)
4. ✅ Standard library first
5. ❌ No “perfect code” mindset — ship ugly, then improve

---

# 🟢 TASK 1 — Service Skeleton + Health Check

### 🎯 Goal

Create a **minimal but production-minded Go HTTP service**.

---

## Task Description

Implement a Go HTTP server with a single endpoint:

```
GET /health
```

### Expected Response

- HTTP status: `200`
- JSON body:

```json
{
  "status": "ok"
}
```

---

## Constraints (non-negotiable)

### 1️⃣ Project structure

You **must not** put everything in `main.go`.

Expected minimum structure (you can add more):

```
jobqueue/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   └── http/
│       └── handler.go
├── go.mod
```

---

### 2️⃣ Server behavior

- Port must be configurable via environment variable
- Default port: `8080`
- Use `net/http`
- Use `http.Server` (not `ListenAndServe` directly)

---

### 3️⃣ Graceful shutdown

- Handle `SIGINT` / `SIGTERM`
- Shutdown server with timeout
- Do **not** ignore context

---

### 4️⃣ JSON handling

- Proper `Content-Type`
- Use `encoding/json`
- No `fmt.Fprintf` hacks

---

### 5️⃣ Error handling

- No panics
- No ignored errors
- Errors must be **handled or returned**

---

## Explicit Non-Goals (do NOT implement)

- Logging framework
- Middleware
- Routing framework
- Docker
- Tests (for now)

---

## What I will review in your PR

I will **block the PR** if I see:

- God `main.go`
- Global variables
- `log.Fatal` everywhere
- No context usage
- Hardcoded port
- Non-idiomatic naming

I will comment on:

- Project layout
- Handler design
- Shutdown logic
- Error handling style
- Go idioms you missed

---

## Deliverable

1. Create a GitHub repo
2. Implement the task
3. Open a PR:

   - `main` ← `feature/health-endpoint`

4. In PR description, answer:

   - What felt confusing?
   - What felt ugly?
   - What are you unsure about?

---

## Definition of “Done”

- `curl localhost:8080/health` works
- Server shuts down cleanly with `Ctrl+C`
- Code compiles with `go build ./...`

---

When you’re done:

- Paste **repo link**
- Paste **PR link**

I’ll review it like a senior Go backend engineer and ask for changes.

🚀
Take your time. Write bad code. That’s the point.
