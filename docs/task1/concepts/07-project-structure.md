# Go Project Structure and Idioms

## Table of Contents
1. [Standard Go Project Layout](#standard-go-project-layout)
2. [Our Project Structure](#our-project-structure)
3. [Is Our Structure Idiomatic?](#is-our-structure-idiomatic)
4. [Package Naming Conventions](#package-naming-conventions)
5. [The `internal/` Package](#the-internal-package)
6. [The `cmd/` Directory](#the-cmd-directory)
7. [Common Go Project Patterns](#common-go-project-patterns)
8. [Improvements and Best Practices](#improvements-and-best-practices)

---

## Standard Go Project Layout

### The Standard Layout (Unofficial but Widely Used)

```
project-root/
├── cmd/              # Main applications
│   └── appname/
│       └── main.go
├── internal/         # Private application code
│   └── pkgname/
│       └── ...
├── pkg/              # Public library code (if applicable)
│   └── pkgname/
│       └── ...
├── api/              # API definitions (if applicable)
├── web/              # Web assets (if applicable)
├── configs/          # Configuration files
├── scripts/          # Build/deployment scripts
├── docs/             # Documentation
├── test/             # Additional test files
├── go.mod            # Go module definition
└── README.md         # Project documentation
```

### Key Directories Explained

**`cmd/`** - Command-line applications
- Each subdirectory is a separate executable
- Contains `main.go` files
- Example: `cmd/server/main.go`, `cmd/cli/main.go`

**`internal/`** - Private application code
- Cannot be imported by other modules
- Only accessible within this module
- Prevents external dependencies on internal code

**`pkg/`** - Public library code (optional)
- Code that can be imported by other projects
- Only include if you're building a library
- Not needed for applications

**`api/`** - API definitions (optional)
- OpenAPI/Swagger specs
- Protocol buffer definitions
- API contracts

---

## Our Project Structure

### Current Structure

```
job-queue-backend/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   └── http/
│       └── handler.go       # HTTP handlers
├── docs/
│   ├── concepts/            # Concept documentation
│   └── learnings.md        # Learning notes
├── go.mod                  # Go module
└── (README.md)             # Project readme (if exists)
```

### Analysis

✅ **Good:**
- Uses `cmd/` for main application
- Uses `internal/` for private code
- Separates HTTP handlers from main
- Has documentation directory

✅ **Idiomatic:**
- Follows standard Go layout
- Clear separation of concerns
- Package naming is correct

---

## Is Our Structure Idiomatic?

### ✅ Yes! Here's Why:

**1. Uses `cmd/` directory**
- ✅ Standard location for main applications
- ✅ Clear entry point
- ✅ Allows multiple commands if needed later

**2. Uses `internal/` package**
- ✅ Prevents external imports
- ✅ Clear that this is private code
- ✅ Follows Go best practices

**3. Package organization**
- ✅ `internal/http` - Clear domain separation
- ✅ Handlers separate from main
- ✅ Easy to test handlers independently

**4. Module structure**
- ✅ `go.mod` at root
- ✅ Module path follows GitHub convention
- ✅ Clear module name

### What Makes It Idiomatic?

**Go idioms we're following:**

1. **One main per directory** - `cmd/server/main.go`
2. **Internal packages** - Code that shouldn't be imported externally
3. **Domain-based organization** - `internal/http` for HTTP-related code
4. **Clear entry points** - `cmd/` makes it obvious where to start

---

## Package Naming Conventions

### Rules

1. **Lowercase only** - No uppercase letters
2. **Short names** - Prefer `http` over `httphandlers`
3. **No underscores** - Use `httpclient` not `http_client`
4. **No hyphens** - Use `httpclient` not `http-client`
5. **Singular when possible** - `handler` not `handlers`

### Our Packages

```go
package main        // ✅ cmd/server/main.go - special case
package http        // ✅ internal/http/handler.go - good!
```

**Why `package http`?**
- Short and clear
- Matches standard library package name (but in different namespace)
- Clear what it contains

**Alternative (also valid):**
```go
package handlers   // Also acceptable
package api        // Also acceptable
```

### Package Import Alias

```go
import (
    internalhttp "github.com/karprabha/job-queue-backend/internal/http"
)
```

**Why the alias?**
- Our package is named `http`
- Standard library also has `http` package
- Alias prevents naming conflict
- `internalhttp` makes it clear it's our internal package

**Without alias (wouldn't work):**
```go
import (
    "net/http"
    "github.com/karprabha/job-queue-backend/internal/http"  // ❌ Conflict!
)
```

---

## The `internal/` Package

### What It Does

The `internal/` directory is a **special Go feature**:

- Code in `internal/` can only be imported by:
  - The same module
  - Packages at the same level or deeper
- Code in `internal/` **cannot** be imported by:
  - Other Go modules
  - External projects

### Why Use It?

**1. Encapsulation**
```go
// internal/http/handler.go
package http

func HealthCheckHandler(...) { ... }  // Private to this module
```

**External project cannot do:**
```go
import "github.com/karprabha/job-queue-backend/internal/http"  // ❌ Not allowed!
```

**2. Clear Intent**
- Signals: "This is internal, don't depend on it"
- Prevents breaking changes from affecting external users
- Allows refactoring without worrying about external dependencies

**3. Prevents Accidental Imports**
- Go compiler enforces this
- Can't accidentally create external dependencies

### When to Use `internal/` vs `pkg/`

**Use `internal/` when:**
- ✅ Building an application (not a library)
- ✅ Code is private to your project
- ✅ You don't want others to import it

**Use `pkg/` when:**
- ✅ Building a library for others to use
- ✅ Code should be importable
- ✅ You want to maintain a public API

**Our case:** We're building an application → `internal/` is correct ✅

---

## The `cmd/` Directory

### Purpose

The `cmd/` directory contains **main applications** (executables).

### Structure

```
cmd/
├── server/          # Server application
│   └── main.go
├── cli/             # CLI tool (if you had one)
│   └── main.go
└── worker/          # Worker process (if you had one)
    └── main.go
```

### Why Separate Directories?

**Each subdirectory = one executable**

- `go build ./cmd/server` → builds server binary
- `go build ./cmd/cli` → builds CLI binary
- Each can have different dependencies
- Each can have different build tags

### Our Structure

```
cmd/
└── server/
    └── main.go
```

**This means:**
- One executable: the server
- Clear entry point
- Easy to build: `go build ./cmd/server`

**To build:**
```bash
go build -o bin/server ./cmd/server
```

**To run:**
```bash
go run ./cmd/server
```

---

## Common Go Project Patterns

### Pattern 1: Layered Architecture

```
internal/
├── http/          # HTTP layer (handlers)
├── service/       # Business logic
├── repository/    # Data access
└── model/         # Data models
```

**Our current structure:**
- We have `internal/http` ✅
- Could add `internal/service` for business logic
- Could add `internal/repository` for data access

### Pattern 2: Domain-Driven Design

```
internal/
├── user/         # User domain
│   ├── handler.go
│   ├── service.go
│   └── repository.go
├── order/         # Order domain
│   ├── handler.go
│   ├── service.go
│   └── repository.go
```

**When to use:**
- Larger applications
- Multiple domains/bounded contexts
- Team wants domain separation

### Pattern 3: Clean Architecture

```
internal/
├── handler/      # HTTP handlers
├── usecase/      # Business use cases
├── repository/   # Data repositories
└── entity/       # Domain entities
```

**When to use:**
- Complex business logic
- Need strict separation
- Multiple interfaces (HTTP, gRPC, CLI)

### Our Current Pattern

We're using a **simple layered pattern**:
- `cmd/server` - Entry point
- `internal/http` - HTTP handlers

**This is perfect for:**
- ✅ Small to medium applications
- ✅ Getting started
- ✅ Clear and simple

**Could evolve to:**
- Add `internal/service` when business logic grows
- Add `internal/repository` when you add a database
- Add `internal/model` when you have data models

---

## Improvements and Best Practices

### Current Structure: ✅ Good!

Our structure is already idiomatic. Here are some **optional** improvements as the project grows:

### 1. Add Service Layer (When Needed)

**When:** Business logic grows beyond handlers

```
internal/
├── http/
│   └── handler.go
└── service/
    └── health.go      # Business logic for health checks
```

**Why:**
- Separates HTTP concerns from business logic
- Makes handlers thin (just HTTP stuff)
- Easier to test business logic

### 2. Add Configuration Package

**When:** Configuration becomes complex

```
internal/
└── config/
    └── config.go      # Configuration loading
```

**Why:**
- Centralizes configuration
- Type-safe config
- Environment variable handling

### 3. Add Models Package

**When:** You have data models

```
internal/
└── model/
    └── health.go      # HealthCheckResponse, etc.
```

**Or keep in domain packages:**
```
internal/
└── http/
    └── model.go       # HTTP-specific models
```

**Both are valid!** Choose based on what makes sense.

### 4. Add Tests

**Standard Go test structure:**
```
internal/
└── http/
    ├── handler.go
    └── handler_test.go    # Tests in same package
```

**Or separate test package:**
```
internal/
└── http/
    └── handler.go
internal/
└── http_test/            # External test package
    └── handler_test.go
```

**Go convention:** `*_test.go` files in the same package.

### 5. Add Build Scripts

```
scripts/
├── build.sh
├── test.sh
└── deploy.sh
```

**Why:**
- Standardizes builds
- Makes CI/CD easier
- Documents build process

---

## Key Takeaways

1. **Our structure is idiomatic** ✅
2. **`cmd/` for executables** - Standard practice
3. **`internal/` for private code** - Prevents external dependencies
4. **Package naming** - Lowercase, short, clear
5. **Can evolve** - Add layers as project grows

---

## Common Anti-Patterns to Avoid

❌ **God package**
```
internal/
└── main.go  # Everything in one file
```

✅ **Separate by concern**
```
internal/
├── http/
└── service/
```

❌ **Public packages for app code**
```
pkg/
└── http/    # ❌ Should be internal/ if it's app-specific
```

✅ **Use internal for app code**
```
internal/
└── http/    # ✅ Private to your app
```

❌ **Deep nesting**
```
internal/
└── a/
    └── b/
        └── c/
            └── d/    # Too deep!
```

✅ **Flat structure**
```
internal/
├── http/
├── service/
└── repository/    # Clear and flat
```

---

## Comparison with Other Languages

### Java/Maven
```
src/
└── main/
    └── java/
        └── com/company/app/
```

**Go is simpler:**
- No deep package hierarchies needed
- Flat structure is preferred
- Domain-based, not company-based

### Node.js
```
src/
├── routes/
├── controllers/
└── models/
```

**Go is similar but:**
- Uses `internal/` instead of `src/`
- Package names match directory names
- Clearer module boundaries

---

## Next Steps

- Review [HTTP Server](./03-http-server.md) - How handlers fit in structure
- Understand [Error Handling](./05-error-handling.md) - Error patterns in structure
- Learn about [Context](./01-context.md) - How context flows through structure

