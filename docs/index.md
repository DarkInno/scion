---
layout: home

hero:
  name: Scion
  text: Copy-Paste Go Backend Modules
  tagline: Explicit dependencies, security-first, AI-friendly. Copy production-ready code into your project.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View Modules
      link: /modules/
    - theme: alt
      text: GitHub
      link: https://github.com/DarkInno/scion

features:
  - icon: 🔐
    title: Auth
    details: JWT authentication with bcrypt, rate limiting, and user enumeration prevention.
    link: /modules/auth
  - icon: 📦
    title: CRUD
    details: Generic CRUD operations with pagination, sort/filter whitelist, SQL injection prevention.
    link: /modules/crud
  - icon: SQL
    title: Database
    details: database/sql setup, transactions, and whitelisted query fragments.
    link: /modules/database
  - icon: 🛡️
    title: Middleware
    details: Recovery, CORS, logging, timeout, request ID, body size limit.
    link: /modules/middleware
  - icon: 👥
    title: RBAC
    details: Role-based access control with wildcard permissions and hierarchy inheritance.
    link: /modules/rbac
  - icon: ⏱️
    title: Rate Limit
    details: Fixed window, sliding window, and token bucket algorithms with LRU eviction.
    link: /modules/ratelimit
  - icon: ✅
    title: Validation
    details: Chainable request validation with regex DoS prevention and panic recovery.
    link: /modules/validation
  - icon: 📁
    title: File Upload
    details: Secure file upload with magic bytes validation and path traversal prevention.
    link: /modules/file-upload
  - icon: 💚
    title: Health
    details: Liveness/readiness probes with SSRF protection.
    link: /modules/health
  - icon: 💾
    title: Cache
    details: Generic TTL + LRU in-memory cache with background cleanup.
    link: /modules/cache
  - icon: 📄
    title: Pagination
    details: Offset/limit and cursor pagination with base64 validation.
    link: /modules/pagination
  - icon: 📧
    title: Mail
    details: SMTP email with templates, header injection prevention, and async queue.
    link: /modules/mail
  - icon: SQL
    title: Migrations
    details: SQL migration runner with checksums, transactions, and safe file validation.
    link: /modules/migrations
  - icon: 📈
    title: Metrics
    details: Prometheus HTTP metrics with route cardinality limits and label sanitization.
    link: /modules/metrics
  - icon: ⚠️
    title: Problem Details
    details: RFC 9457 API error responses with panic recovery and safe details.
    link: /modules/problem
---

## Quick Start

```bash
# Copy a module into your project
cp -r registry/auth/src/go/* yourproject/internal/auth/

# Adapt the configuration
# Edit config.go: set JWT secret, database URL, etc.

# Implement the store interface
# type UserStore interface { ... }

# Wire up routes
# See registry/auth/examples/gin/main.go
```

## Why Copy-Paste?

Backend modules share 80% of their skeleton across projects. Instead of installing a framework, copy pre-built, production-ready modules and own every line of code.

- **Code ownership** — every line is yours after copying
- **Explicit dependencies** — standard library by default; security exceptions are declared
- **Security-first** — input validation, rate limiting, injection prevention built in
- **AI-friendly** — `__llms__.md` files let AI assistants understand modules quickly
