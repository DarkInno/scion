# Validation 验证模块

链式请求验证构建器，安全优先设计。

## 包含内容

- 链式验证构建器
- 常用验证规则
- 正则 DoS 防护（RE2 引擎）
- 空字节和 CRLF 拒绝
- Panic 恢复
- HTTP 中间件

## 快速复制

```bash
cp -r registry/validation/src/go/* yourproject/internal/validation/
```

## 使用方式

### 定义验证规则

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

rules := validation.For[CreateUserRequest]().
    Field("name").
        Required().
        MinLength(2).
        MaxLength(100).
        CRLF().
        NullByte().
    Field("email").
        Required().
        Email().
        MaxLength(255).
    Field("age").
        Required().
        Min(0).
        Max(150)
```

### 验证请求

```go
handler := func(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    if errs := rules.Validate(req); len(errs) > 0 {
        // 返回验证错误
        w.WriteHeader(http.StatusBadRequest)
        _ = json.NewEncoder(w).Encode(map[string]any{"errors": errs})
        return
    }
    
    // 处理有效请求
}
```

### 使用中间件

```go
handler := validation.Middleware(rules)(handler)
```

## 可用规则

| 规则 | 描述 |
|------|------|
| `Required()` | 字段不能为空 |
| `MinLength(n)` | 最小字符串长度 |
| `MaxLength(n)` | 最大字符串长度 |
| `Min(n)` | 最小数值 |
| `Max(n)` | 最大数值 |
| `Email()` | 有效邮箱格式 |
| `URL()` | 有效 URL 格式 |
| `Pattern(regex)` | 正则匹配（RE2 引擎） |
| `In(values...)` | 值在允许列表中 |
| `CRLF()` | 拒绝 CRLF 注入 |
| `NullByte()` | 拒绝空字节 |

## 文件参考

| 文件 | 用途 |
|------|------|
| `validator.go` | 核心验证逻辑 |
| `rules.go` | 验证规则 |
| `errors.go` | 错误类型 |
| `middleware.go` | HTTP 中间件 |

## 安全特性

- 正则 DoS 防护（RE2 引擎，无回溯）
- 空字节拒绝
- CRLF 注入防护
- 验证中的 panic 恢复

## 测试

```bash
cd registry/validation/src/go
go test -v ./...
```
