# File Upload Module

Secure file upload handler with magic bytes validation and path traversal prevention.

## What's Included

- Secure file upload handling
- Magic bytes validation (not just extension)
- Path traversal prevention
- File size limits
- Rate limiting
- Storage abstraction

## Quick Copy

```bash
cp -r registry/file-upload/src/go/* yourproject/internal/fileupload/
```

## Usage

### Basic Upload

```go
handler := fileupload.NewHandler(fileupload.Config{
    MaxFileSize: 10 << 20, // 10 MB
    AllowedTypes: []string{"image/jpeg", "image/png", "application/pdf"},
    UploadDir: "./uploads",
})

http.Handle("/upload", handler)
```

### Custom Storage

```go
type S3Storage struct {
    // ...
}

func (s *S3Storage) Save(ctx context.Context, name string, reader io.Reader) error {
    // Upload to S3
}

handler := fileupload.NewHandler(fileupload.Config{
    Storage: &S3Storage{},
})
```

### Rate Limiting

```go
handler := fileupload.NewHandler(fileupload.Config{
    RateLimiter: ratelimit.NewFixedWindow(10, time.Minute),
})
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `MaxFileSize` | Maximum file size in bytes | 10 MB |
| `AllowedTypes` | Allowed MIME types | All |
| `UploadDir` | Local storage directory | ./uploads |
| `Storage` | Custom storage backend | Local |
| `RateLimiter` | Rate limiter instance | None |

## File Reference

| File | Purpose |
|------|---------|
| `handler.go` | HTTP upload handler |
| `config.go` | Configuration |
| `validate.go` | File validation |
| `storage.go` | Local storage implementation |

## Security Features

- Magic bytes validation (not just extension check)
- Path traversal prevention with `filepath.Base()`
- File size limit prevents large payload attacks
- Rate limiting prevents abuse

## Tests

```bash
cd registry/file-upload/src/go
go test -v ./...
```
