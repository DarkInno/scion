# CRUD 增删改查模块

通用 CRUD 操作，支持分页、排序/过滤白名单和 SQL 注入防护。

## 包含内容

- 通用 CRUD handler（创建、读取、更新、删除、列表）
- 分页（偏移/限制）
- 排序和过滤白名单
- SQL 注入防护
- 输入验证

## 快速复制

```bash
cp -r registry/crud/src/go/* yourproject/internal/crud/
```

## 适配指南

### 1. 定义模型

```go
type Product struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Price     float64   `json:"price"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 2. 实现 Store 接口

```go
type Store[T any] interface {
    Create(ctx context.Context, entity *T) error
    GetByID(ctx context.Context, id string) (*T, error)
    Update(ctx context.Context, id string, entity *T) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, opts ListOptions) ([]T, int, error)
}
```

### 3. 配置

```go
handler := crud.NewHandler(store, crud.Config{
    MaxPageSize: 100,
    DefaultPageSize: 20,
    SortWhitelist: []string{"name", "created_at"},
    FilterWhitelist: []string{"name", "price"},
})
```

## 文件参考

| 文件 | 用途 |
|------|------|
| `config.go` | 配置选项 |
| `models.go` | 通用类型和接口 |
| `handlers.go` | HTTP handler |
| `routes.go` | 路由注册 |

## 安全特性

- 排序/过滤白名单防止任意字段访问
- 参数化 SQL 查询
- 所有字段输入验证
- 分页上限防止内存耗尽

## 测试

```bash
cd registry/crud/src/go
go test -v ./...
```
