# Health 健康检查模块

存活和就绪探针，支持 SSRF 防护。

## 包含内容

- 存活探针（服务是否存活？）
- 就绪探针（服务是否准备好接受流量？）
- 自定义健康检查
- SSRF 防护（私有 IP 拒绝）

## 快速复制

```bash
cp -r registry/health/src/go/* yourproject/internal/health/
```

## 使用方式

### 基本设置

```go
checker := health.NewChecker()

// 添加就绪检查
checker.AddCheck("database", func(ctx context.Context) error {
    return db.PingContext(ctx)
})

checker.AddCheck("redis", func(ctx context.Context) error {
    return redis.Ping(ctx).Err()
})

// 注册 handler
http.Handle("/healthz", checker.LivenessHandler())
http.Handle("/readyz", checker.ReadinessHandler())
```

### 自定义检查

```go
checker.AddCheck("external-api", func(ctx context.Context) error {
    resp, err := http.Get("https://api.example.com/health")
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        return fmt.Errorf("status: %d", resp.StatusCode)
    }
    return nil
})
```

## 端点

| 端点 | 用途 | 响应 |
|------|------|------|
| `/healthz` | 存活探针 | 存活时返回 200 OK |
| `/readyz` | 就绪探针 | 所有检查通过时返回 200 OK |

## 配置

```go
checker := health.NewChecker(health.Config{
    Timeout: 5 * time.Second,
})
```

## 文件参考

| 文件 | 用途 |
|------|------|
| `checker.go` | 健康检查管理器 |
| `handler.go` | HTTP handler |
| `checks.go` | 内置检查 |

## 安全特性

- SSRF 防护（HTTP 检查中拒绝私有 IP）
- CRLF 注入防护
- 所有检查超时

## 测试

```bash
cd registry/health/src/go
go test -v ./...
```
