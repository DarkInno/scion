# Getting Started

## 1. Install the CLI

Scion requires Go 1.22 or newer.

```bash
go install github.com/DarkInno/scion/cmd/scion@latest
```

For reproducible installs, pin a release:

```bash
go install github.com/DarkInno/scion/cmd/scion@v0.1.2
```

Make sure your Go bin directory is on `PATH`, then verify the install:

```bash
scion version
scion list
```

## 2. Choose a Module

Browse the [Modules](/modules/) section or run:

```bash
scion list
scion info cache
```

Modules marked `stdlibOnly=true` can be copied directly into an existing project. Modules marked `stdlibOnly=false`, such as `auth`, require `--standalone` so their `go.mod` and `go.sum` are copied explicitly.

## 3. Copy Source Into Your Project

Preview the files first:

```bash
scion add cache --to internal/cache --dry-run
```

Copy the module:

```bash
scion add cache --to internal/cache
```

Scion writes `.scion-module.json` metadata in the target directory. This file records the copied module, registry version, source hashes, and whether standalone mode was used.

## 4. Compare Later

After you adapt the copied source, you can compare it against Scion's embedded template:

```bash
scion diff cache --target internal/cache
```

`diff` only reports differences. It never merges or overwrites your local changes.

## Standalone Modules

The `auth` module intentionally uses mature security dependencies for JWT and bcrypt. Copy it with standalone mode:

```bash
scion add auth --standalone --to internal/auth
```

Scion still does not edit your project-level `go.mod`; standalone mode only copies the module's own `go.mod` and `go.sum` into the target.

## Binary Downloads

You can also download prebuilt binaries from [GitHub Releases](https://github.com/DarkInno/scion/releases).

Verify downloads with `SHA256SUMS`:

```bash
sha256sum -c SHA256SUMS
```

Windows PowerShell:

```powershell
Get-FileHash .\scion_v0.1.2_windows_amd64.zip -Algorithm SHA256
```

## Manual Copy

Manual copy still works when you are working inside the Scion repository:

```bash
cp -r registry/cache/src/go/*.go yourproject/internal/cache/
```

The CLI is preferred because it validates paths, records metadata, supports dry-runs, and enables future `scion diff` checks.

## Available Modules

| Module | Description |
|--------|-------------|
| [Auth](/modules/auth) | JWT authentication with bcrypt |
| [CRUD](/modules/crud) | Generic CRUD with pagination |
| [Middleware](/modules/middleware) | Recovery, CORS, logging, timeout |
| [RBAC](/modules/rbac) | Role-based access control |
| [Rate Limit](/modules/ratelimit) | Fixed/sliding window, token bucket |
| [Validation](/modules/validation) | Chainable request validation |
| [File Upload](/modules/file-upload) | Secure file upload handler |
| [Health](/modules/health) | Liveness/readiness probes |
| [Cache](/modules/cache) | TTL + LRU in-memory cache |
| [Pagination](/modules/pagination) | Offset/cursor pagination |
| [Mail](/modules/mail) | SMTP email with templates |

## Module Structure

Each registry module follows this structure:

```text
registry/<module>/
|-- README.md               # Human-readable adaptation guide
|-- __llms__.md             # AI-readable summary
`-- src/go/
    |-- go.mod              # module <name>, go 1.22
    |-- config.go           # Options struct, Defaults(), FromEnv()
    |-- *_test.go           # Functional tests
    `-- pentest_test.go     # Attack-scenario tests
```

## Next Steps

- Read [Why Copy-Paste?](/guide/why-copy-paste) to understand the philosophy.
- Check [Security Design](/guide/security) for security guarantees and constraints.
- Browse [Modules](/modules/) to find the template you need.
