# Pagination 分页模块

偏移/限制和游标分页，安全优先设计。

## 包含内容

- **偏移分页** — 传统基于页码
- **游标分页** — 大数据集高效
- 游标 Base64 校验
- 负偏移钳制
- 最大限制强制

## 快速复制

```bash
cp -r registry/pagination/src/go/* yourproject/internal/pagination/
```

## 使用方式

### 偏移分页

```go
handler := pagination.OffsetHandler(pagination.OffsetConfig{
    DefaultLimit: 20,
    MaxLimit: 100,
})

// 在 handler 中
func listUsers(w http.ResponseWriter, r *http.Request) {
    params := pagination.GetOffsetParams(r)
    // params.Page, params.Limit
    
    users, total, err := store.List(r.Context(), params)
    // 返回分页响应
}
```

### 游标分页

```go
handler := pagination.CursorHandler(pagination.CursorConfig{
    DefaultLimit: 20,
    MaxLimit: 100,
})

// 在 handler 中
func listUsers(w http.ResponseWriter, r *http.Request) {
    params := pagination.GetCursorParams(r)
    // params.Cursor, params.Limit
    
    users, nextCursor, err := store.List(r.Context(), params)
    // 返回带 next_cursor 的分页响应
}
```

## 响应格式

### 偏移

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

### 游标

```json
{
  "data": [...],
  "pagination": {
    "next_cursor": "eyJpZCI6MTIzfQ==",
    "has_more": true
  }
}
```

## 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `DefaultLimit` | 默认页大小 | 20 |
| `MaxLimit` | 最大页大小 | 100 |

## 文件参考

| 文件 | 用途 |
|------|------|
| `offset.go` | 偏移分页 |
| `cursor.go` | 游标分页 |
| `config.go` | 配置 |
| `response.go` | 响应类型 |
| `middleware.go` | HTTP 中间件 |

## 安全特性

- 游标 Base64 校验
- 负偏移钳制
- 最大限制强制
- 输入验证

## 测试

```bash
cd registry/pagination/src/go
go test -v ./...
```
