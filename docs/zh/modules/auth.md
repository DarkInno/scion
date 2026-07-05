# Auth 认证模块

基于 JWT 的邮箱/密码认证，使用 bcrypt 加密。

## 包含内容

- 用户注册和登录
- 密码哈希 (bcrypt，默认 cost 12，可配置 10-15)
- JWT 访问令牌生成和验证 (HS256, JTI, aud, iss, nbf)
- 路由保护中间件
- 限流钩子（邮箱 + IP），包含内存实现
- 用户枚举防护

## 快速复制

```bash
cp -r registry/auth/src/go/* yourproject/internal/auth/
```

## 适配指南

### 1. 数据库层

实现 `auth.UserStore` 接口：

```go
type UserStore interface {
    Create(ctx context.Context, user *User) error
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByID(ctx context.Context, id string) (*User, error)
}
```

### 2. 配置

设置环境变量：

| 变量 | 必需 | 默认值 | 描述 |
|------|------|--------|------|
| `JWT_SECRET` | 是 | - | 最少32字符，最多512 |
| `DB_URL` | 是 | - | 数据库连接字符串 |
| `TOKEN_EXPIRY` | 否 | 3600 | 令牌过期时间（秒），最大 604800 |
| `JWT_ISSUER` | 否 | Scion-auth | 用于 aud/iss 验证的签发者 |
| `BCRYPT_COST` | 否 | 12 | bcrypt cost (10-15) |

### 3. 限流

```go
rl := auth.NewMemoryRateLimiter(10, 15*time.Minute)
handler := auth.NewHandler(store, cfg).WithRateLimiter(rl)
```

### 4. 路由

默认前缀：`/api/v1/auth`

使用 `auth.RoutePrefix` 在注册路由前更改。

## 文件参考

| 文件 | 用途 |
|------|------|
| `config.go` | 环境变量加载和验证 |
| `models.go` | User 结构体，请求/响应类型 |
| `password.go` | bcrypt 哈希和验证 |
| `jwt.go` | 令牌生成和解析 |
| `handlers.go` | HTTP handler（注册、登录、me） |
| `middleware.go` | JWT Bearer 验证中间件 |
| `routes.go` | 路由注册 |
| `ratelimiter.go` | 内存滑动窗口限流器 |

## 安全清单

- [ ] `JWT_SECRET` 至少32个随机字符
- [ ] `BCRYPT_COST` 在 10-15 之间
- [ ] 邮箱地址在存储/查询前使用 `NormalizeEmail()` 标准化
- [ ] 配置了限流（邮箱 + IP）
- [ ] `JWT_ISSUER` 在所有服务间保持一致

## 测试

```bash
cd registry/auth/src/go
go test -v ./...
```

## 示例

参见 `registry/auth/examples/gin/` 获取最小可运行示例。
