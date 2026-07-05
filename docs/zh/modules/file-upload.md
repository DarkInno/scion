# File Upload 文件上传模块

安全文件上传处理器，支持魔数校验和路径遍历防护。

## 包含内容

- 安全文件上传处理
- 魔数校验（不仅检查扩展名）
- 路径遍历防护
- 文件大小限制
- 限流
- 存储抽象

## 快速复制

```bash
cp -r registry/file-upload/src/go/* yourproject/internal/fileupload/
```

## 使用方式

### 基本上传

```go
handler := fileupload.NewHandler(fileupload.Config{
    MaxFileSize: 10 << 20, // 10 MB
    AllowedTypes: []string{"image/jpeg", "image/png", "application/pdf"},
    UploadDir: "./uploads",
})

http.Handle("/upload", handler)
```

### 自定义存储

```go
type S3Storage struct {
    // ...
}

func (s *S3Storage) Save(ctx context.Context, name string, reader io.Reader) error {
    // 上传到 S3
}

handler := fileupload.NewHandler(fileupload.Config{
    Storage: &S3Storage{},
})
```

### 限流

```go
handler := fileupload.NewHandler(fileupload.Config{
    RateLimiter: ratelimit.NewFixedWindow(10, time.Minute),
})
```

## 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `MaxFileSize` | 最大文件大小（字节） | 10 MB |
| `AllowedTypes` | 允许的 MIME 类型 | 所有 |
| `UploadDir` | 本地存储目录 | ./uploads |
| `Storage` | 自定义存储后端 | 本地 |
| `RateLimiter` | 限流器实例 | 无 |

## 文件参考

| 文件 | 用途 |
|------|------|
| `handler.go` | HTTP 上传处理器 |
| `config.go` | 配置 |
| `validate.go` | 文件验证 |
| `storage.go` | 本地存储实现 |

## 安全特性

- 魔数校验（不仅检查扩展名）
- 使用 `filepath.Base()` 防止路径遍历
- 文件大小限制防止大载荷攻击
- 限流防止滥用

## 测试

```bash
cd registry/file-upload/src/go
go test -v ./...
```
