# Problem Details 错误响应模块

面向 `net/http` 的 RFC 9457 风格 API 错误响应。

## 功能

- `Problem` 和 `InvalidParam` JSON 类型
- `application/problem+json` 写入器
- 支持返回 `error` 的 handler 适配器
- panic 恢复中间件
- 可选安全 request ID 扩展

## 使用

```go
http.Handle("/users", problem.Handler(func(w http.ResponseWriter, r *http.Request) error {
    return problem.Error(http.StatusNotFound, "User not found", "no user matched the request")
}))
```

验证错误：

```go
problem.Write(w, r, problem.Validation([]problem.InvalidParam{
    {Detail: "must be a valid email", Pointer: "#/email"},
}))
```

## 安全

- 拒绝响应字段中的 CRLF 和 null 字节
- 截断过长 detail
- 未知内部错误统一隐藏为通用 500
- 限制 validation error 数量

## 复制

```bash
scion add problem --to internal/problem
```
