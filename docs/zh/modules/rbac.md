# RBAC 权限控制模块

基于角色的访问控制，支持通配符权限和层级继承。

## 包含内容

- 角色定义和管理
- 通配符权限分配
- 角色层级继承
- 循环检测
- HTTP 权限检查中间件

## 快速复制

```bash
cp -r registry/rbac/src/go/* yourproject/internal/rbac/
```

## 使用方式

### 定义角色和权限

```go
manager := rbac.NewManager()

// 定义角色
manager.AddRole(rbac.Role{
    ID: "admin",
    Permissions: []string{"*"},
})

manager.AddRole(rbac.Role{
    ID: "editor",
    Permissions: []string{"posts:*", "comments:read"},
})

manager.AddRole(rbac.Role{
    ID: "viewer",
    Permissions: []string{"*:read"},
})

// 设置层级（editor 继承 viewer）
manager.SetParent("editor", "viewer")
```

### 使用中间件

```go
// 要求特定权限
handler := rbac.Require("posts:write")(handler)

// 要求多个权限之一
handler := rbac.RequireAny("posts:write", "posts:delete")(handler)

// 要求所有权限
handler := rbac.RequireAll("posts:write", "comments:write")(handler)
```

### 设置用户角色

```go
// 在认证中间件中
ctx := rbac.WithRoles(ctx, []string{"editor"})
```

## 权限格式

权限使用 `resource:action` 格式，支持通配符：

- `posts:read` — 读取文章
- `posts:*` — 文章的所有操作
- `*:read` — 读取任何资源
- `*` — 完全访问

## 文件参考

| 文件 | 用途 |
|------|------|
| `model.go` | Role 和 Permission 类型 |
| `manager.go` | 角色/权限管理 |
| `middleware.go` | HTTP 中间件 |
| `context.go` | 上下文辅助函数 |

## 安全特性

- 通配符权限匹配
- 角色层级循环检测
- 层级继承

## 测试

```bash
cd registry/rbac/src/go
go test -v ./...
```
