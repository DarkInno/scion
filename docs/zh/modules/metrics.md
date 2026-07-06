# Metrics 指标模块

面向 `net/http` 服务的 Prometheus HTTP 指标模块。

## 功能

- 独立 Prometheus registry
- 请求计数、耗时直方图、in-flight gauge
- `/metrics` 抓取 handler
- `func(http.Handler) http.Handler` 中间件
- 可选 Go runtime/process collectors

## 使用

```go
m, err := metrics.New()
if err != nil {
    return err
}
_ = m.RegisterDefaults()

http.Handle("/metrics", m.Handler())
http.Handle("/users/", m.Middleware("/users/{id}")(usersHandler))
```

传入路由模板，不要传原始 URL。

## 安全

- 拒绝 label 中的 CRLF 和 null 字节
- 归一化原始 URL 和超长 label
- 限制 route label 基数，溢出进入 `route="overflow"`

## 复制

`metrics` 使用 Prometheus，需要 standalone 复制：

```bash
scion add metrics --standalone --to internal/metrics
```
