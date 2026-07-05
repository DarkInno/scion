# File Upload Module

Secure file upload handling with magic-byte validation and storage adapters.

## What's Included

- Standard `net/http` upload handler
- File size limits
- Magic-byte MIME detection
- Local and in-memory storage implementations
- Generated server-side file names
- Path traversal, CRLF, and null-byte protection
- Per-client in-memory rate limiting

## Quick Copy

```bash
cp -r registry/file-upload/src/go/*.go yourproject/internal/fileupload/
```

Or with the Scion CLI:

```bash
scion add file-upload --to internal/fileupload
```

## Usage

```go
opts := fileupload.Defaults()
opts.UploadDir = "./uploads"
opts.AllowedTypes = []string{"image/png", "image/jpeg"}

handler, err := fileupload.NewHandler(opts)
if err != nil {
	return err
}
http.Handle("/upload", handler)
```

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Defaults, environment loading, generated names |
| `handler.go` | HTTP upload handler and rate limiting |
| `storage.go` | Storage interface, local disk and memory storage |
| `validate.go` | Magic-byte detection and filename sanitization |
| `pentest_test.go` | Attack-scenario tests |

## Tests

```bash
cd registry/file-upload/src/go
go test -v ./...
```
