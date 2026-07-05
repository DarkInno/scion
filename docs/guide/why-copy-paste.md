# Why Copy-Paste?

## The Problem

Backend modules (auth, CRUD, file upload, rate limiting) share 80% of their skeleton across projects, but the remaining 20% differs in ways that make packages awkward:

- You need to customize business logic deep inside the module
- You want to own the code, not be locked to upstream versions
- Your AI coding assistant works better with code it can read and modify directly
- No dependency hell — standard library by default, with declared security exceptions

## The Solution

Scion provides copy-paste code modules. Instead of installing a framework or pulling a dependency, you copy pre-built, production-ready modules into your project and own every line of code.

## Benefits

### Code Ownership

Every line is yours after copying. No upstream lock-in. No version conflicts. No waiting for maintainers to merge your PR.

### Explicit Dependencies

Each module uses only the Go standard library. No `go.sum` bloat. No transitive dependency vulnerabilities.

### Security-First

Input validation, rate limiting, injection prevention — built in from day one. Every module includes penetration test cases.

### AI-Friendly

`__llms__.md` files let AI assistants understand modules in ~200 tokens. Your AI coding assistant can read, modify, and extend the code directly.

### Framework-Agnostic

Uses Go standard `net/http`. Adaptable to Gin, Echo, Fiber, or any framework.

### Tested

Every module includes functional tests and penetration test cases. Run `go test -v ./...` to verify.

## Comparison

| Approach | Pros | Cons |
|----------|------|------|
| **Package (npm/go)** | Easy to install, auto-updates | Version lock, dependency hell, hard to customize |
| **Framework (Gin/Echo)** | Consistent API, community | Lock-in, bloat, learning curve |
| **Copy-paste (Scion)** | Full ownership, zero deps, customizable | Manual updates, more initial setup |

## When to Use Scion

- You need a production-ready module fast
- You want to own every line of code
- You need deep customization
- You're building with AI coding assistants
- You want no hidden transitive dependencies

## When NOT to Use Scion

- You prefer framework conventions over code ownership
- You need auto-updates from upstream
- You're building a quick prototype and don't care about dependencies
