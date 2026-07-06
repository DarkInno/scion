# Scion

> Graft backend patterns into your project. Copy source, not dependencies.

[English](README.md) | [中文](README_zh.md)

Scion is a copy-paste code library for Go backend development. It ships production-oriented modules as source templates, so you can copy them into your project, adapt them, and own every line.

## Quick Start

Install the CLI with Go 1.22 or newer:

```bash
go install github.com/DarkInno/scion/cmd/scion@latest
```

For reproducible installs, pin a release:

```bash
go install github.com/DarkInno/scion/cmd/scion@v0.1.3
```

Make sure your Go bin directory is on `PATH`, then verify the install:

```bash
scion version
scion list
```

Copy a standard-library-only module into your project:

```bash
scion add cache --dry-run
scion add cache
scion diff cache
```

When `--to` or `--target` is omitted, Scion uses the module's default target, such as `internal/cache`. You can still override it with `--to <dir>` or `--target <dir>`.

Scion copies source files and writes `.scion-module.json` metadata for later comparison. It never edits your project's `go.mod` automatically. Modules marked `stdlibOnly=false`, such as `auth`, require explicit standalone mode:

```bash
scion add auth --standalone
```

## Binary Downloads

Prebuilt binaries are available on the [GitHub Releases](https://github.com/DarkInno/scion/releases) page for macOS, Linux, and Windows on amd64 and arm64.

Verify a downloaded asset with `SHA256SUMS`:

```bash
sha256sum -c SHA256SUMS
```

On Windows PowerShell:

```powershell
Get-FileHash .\scion_v0.1.3_windows_amd64.zip -Algorithm SHA256
```

## Why Copy-Paste?

Backend modules such as auth, CRUD, file upload, and rate limiting share most of their structure across projects, but the last mile is usually project-specific:

- You need to customize business logic inside the module.
- You want to own the code instead of being locked to upstream APIs.
- AI coding assistants work better with source they can read and edit directly.
- Scion avoids dependency sprawl by default; security-sensitive exceptions are declared explicitly.

## Available Modules

| Module | Description | Security Features |
|--------|-------------|-------------------|
| [auth](registry/auth/) | JWT email/password auth + bcrypt | Rate limiting, user enumeration prevention, JTI, aud/iss validation |
| [crud](registry/crud/) | Generic CRUD with pagination | Sort/filter whitelist, SQL injection prevention, pagination ceiling |
| [database](registry/database/) | `database/sql` setup + transactions | DSN-safe errors, whitelisted SQL fragments, parameterized values |
| [middleware](registry/middleware/) | Recovery, CORS, logging, timeout, etc. | CRLF injection prevention, trusted proxy, body size limit |
| [rbac](registry/rbac/) | Role-based access control | Wildcard permissions, cycle detection, hierarchy inheritance |
| [ratelimit](registry/ratelimit/) | Fixed window / sliding window / token bucket | Memory exhaustion protection, LRU eviction, key length limit |
| [validation](registry/validation/) | Chainable request validation builder | Regex DoS prevention, null byte/CRLF rejection, panic recovery |
| [file-upload](registry/file-upload/) | Secure file upload handler | Magic bytes validation, path traversal prevention, size limit, rate limiting |
| [health](registry/health/) | Liveness/readiness probes | SSRF protection, CRLF injection prevention |
| [cache](registry/cache/) | Generic TTL + LRU cache | Background cleanup, goroutine leak prevention, max entries limit |
| [pagination](registry/pagination/) | Offset/limit + cursor pagination | Cursor base64 validation, negative offset clamp, max limit enforcement |
| [mail](registry/mail/) | SMTP email with templates | Header injection prevention, XSS escaping, attachment sanitization, async queue |

## CLI Commands

```bash
scion list [--json]
scion info <module> [--json]
scion add <module> [--to <dir>] [--dry-run] [--force] [--standalone]
scion diff <module> [--target <dir>] [--json]
scion doctor [--strict] [--json]
scion version [--json]
```

Use `scion help <command>` for command-specific examples.

## Project Structure

```text
scion/
|-- cmd/scion/              # CLI entrypoint
|-- internal/               # CLI implementation, bundle reader, doctor checks
|-- internal/bundle/        # Embedded registry bundle generated from registry/
|-- registry/
|   |-- index.json          # Machine-readable module index
|   |-- auth/               # Authentication module
|   |-- cache/              # TTL + LRU cache
|   |-- crud/               # CRUD operations module
|   |-- database/           # database/sql helpers
|   |-- file-upload/        # File upload handler
|   |-- health/             # Health check probes
|   |-- mail/               # SMTP email sender
|   |-- middleware/         # HTTP middleware collection
|   |-- pagination/         # Pagination utilities
|   |-- ratelimit/          # Rate limiting algorithms
|   |-- rbac/               # Role-based access control
|   `-- validation/         # Request validation builder
|-- docs/                   # VitePress documentation
|-- AGENTS.md               # AI coding agent instructions
|-- CONTRIBUTING.md         # Contribution guide
`-- LICENSE                 # MIT
```

## Development

```bash
git clone https://github.com/DarkInno/scion.git
cd scion

# Regenerate the embedded CLI bundle after registry changes
go run ./internal/cmd/build-bundle

# Test and vet the root CLI
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...

# Run strict registry checks
go run ./cmd/scion doctor --strict
```

Run tests for all registry modules in PowerShell:

```powershell
$modules = @('middleware','auth','crud','database','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }
```

## Releasing

Releases are created from semantic version tags:

```bash
git tag -a v0.1.3 -m "v0.1.3"
git push origin v0.1.3
```

The release workflow verifies the CLI, rebuilds the embedded bundle check, cross-compiles binaries, generates `SHA256SUMS`, and publishes GitHub Release assets.

## License

[MIT](LICENSE)
