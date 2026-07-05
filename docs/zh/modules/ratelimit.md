# Rate Limit 限流模块

限流算法，支持内存耗尽防护。

## 包含内容

- **固定窗口** — 简单计数器限流
- **滑动窗口** — 更平滑的限流
- **令牌桶** — 支持突发流量
- 内存耗尽防护 + LRU 淘汰
- HTTP 中间件

## 快速复制

```bash
cp -r registry/ratelimit/src/go/* yourproject/internal/ratelimit/
```

## 使用方式

### 固定窗口

```go
limiter := ratelimit.NewFixedWindow(100, time.Minute)
handler := ratelimit.Middleware(limiter)(handler)
```

### 滑动窗口

```go
limiter := ratelimit.NewSlidingWindow(100, time.Minute)
handler := ratelimit.Middleware(limiter)(handler)
```

### 令牌桶

```go
limiter := ratelimit.NewTokenBucket(100, 10) // 100 tokens/sec, burst 10
handler := ratelimit.Middleware(limiter)(handler)
```

### 自定义键函数

```go
handler := ratelimit.Middleware(limiter, ratelimit.WithKeyFunc(func(r *http.Request) string {
    return r.Header.Get("X-API-Key")
}))(handler)
```

## 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `maxRequests` | 窗口内最大请求数 | 必需 |
| `window` | 时间窗口 | 必需 |
| `burst` | 突发容量（令牌桶） | 必需 |
| `maxBuckets` | 最大跟踪键数 | 10000 |

## 文件参考

| 文件 | 用途 |
|------|------|
| `fixed_window.go` | 固定窗口算法 |
| `sliding_window.go` | 滑动窗口算法 |
| `token_bucket.go` | 令牌桶算法 |
| `store.go` | 内存存储 + LRU 淘汰 |
| `middleware.go` | HTTP 中间件 |

## 安全特性

- 最大桶数内存耗尽防护
- 达到限制时 LRU 淘汰
- 键长度限制防止滥用

## 测试

```bash
cd registry/ratelimit/src/go
go test -v ./...
```
