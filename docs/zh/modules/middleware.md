# Middleware 中间件模块

Go Web 应用的 HTTP 中间件集合。

## 包含内容

- **Recovery** — panic 恢复 + 日志
- **CORS** — 跨域资源共享
- **Logging** — 请求/响应日志
- **Timeout** — 请求超时
- **Request ID** — 唯一请求标识
- **Body Limit** — 请求体大小限制
- **Proxy** — 可信代理处理
- **Debug** — 调试模式检测

## 快速复制

```bash
cp -r registry/middleware/src/go/* yourproject/internal/middleware/
```

## 使用方式

### 链式中间件

```go
handler := middleware.Chain(
    middleware.Recovery(),
    middleware.CORS(middleware.CORSConfig{
        AllowOrigins: []string{"https://example.com"},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    }),
    middleware.Logging(),
    middleware.Timeout(30 * time.Second),
    middleware.BodyLimit(10 << 20), // 10 MB
)(yourHandler)
```

### 单独使用

```go
// Recovery
http.Handle("/api", middleware.Recovery()(handler))

// CORS
http.Handle("/api", middleware.CORS(config)(handler))

// Logging
http.Handle("/api", middleware.Logging()(handler))
```

## 配置

### CORS

```go
config := middleware.CORSConfig{
    AllowOrigins: []string{"https://example.com"},
    AllowMethods: []string{"GET", "POST"},
    AllowHeaders: []string{"Content-Type", "Authorization"},
    MaxAge: 86400,
}
```

### 超时

```go
handler := middleware.Timeout(30 * time.Second)(handler)
```

## 文件参考

| 文件 | 用途 |
|------|------|
| `recovery.go` | Panic 恢复中间件 |
| `cors.go` | CORS 中间件 |
| `logging.go` | 请求日志 |
| `timeout.go` | 请求超时 |
| `requestid.go` | 请求 ID 生成 |
| `bodylimit.go` | 请求体大小限制 |
| `chain.go` | 中间件链 |
| `config.go` | 配置 |

## 安全特性

- 头部 CRLF 注入防护
- 可信代理验证
- 请求体大小限制防止大载荷攻击

## 测试

```bash
cd registry/middleware/src/go
go test -v ./...
```
