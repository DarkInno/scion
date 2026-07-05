# Cache 缓存模块

通用 TTL + LRU 内存缓存，支持后台清理。

## 包含内容

- 类型安全的通用缓存
- TTL（存活时间）支持
- LRU（最近最少使用）淘汰
- 后台清理 goroutine
- 内存耗尽防护

## 快速复制

```bash
cp -r registry/cache/src/go/* yourproject/internal/cache/
```

## 使用方式

### 基本缓存

```go
// 创建缓存，5分钟 TTL，最多1000条目
c := cache.New[string, User](5*time.Minute, 1000)

// 设置值
c.Set("user:123", user)

// 获取值
user, ok := c.Get("user:123")

// 删除值
c.Delete("user:123")
```

### 自定义 TTL

```go
c.SetWithTTL("key", value, 10*time.Minute)
```

### 缓存统计

```go
stats := c.Stats()
fmt.Printf("命中: %d, 未命中: %d, 大小: %d", stats.Hits, stats.Misses, stats.Size)
```

## 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `ttl` | 条目存活时间 | 必需 |
| `maxEntries` | 缓存最大条目数 | 1000 |
| `cleanupInterval` | 后台清理间隔 | ttl/2 |

## 文件参考

| 文件 | 用途 |
|------|------|
| `memory.go` | 缓存实现 |
| `store.go` | Store 接口 |
| `pentest_test.go` | 安全测试 |

## 安全特性

- 最大条目数内存耗尽防护
- 达到限制时 LRU 淘汰
- 后台清理防止内存泄漏
- Goroutine 泄漏防护

## 测试

```bash
cd registry/cache/src/go
go test -v ./...
```
