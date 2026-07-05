# 模块概览

Scion 提供 11 个生产就绪、可复制粘贴的 Go 模块。每个模块自包含。模块默认仅使用标准库；安全例外会在 registry 中显式标记。

## 可用模块

| 模块 | 描述 | 安全特性 |
|------|------|---------|
| [Auth](/zh/modules/auth) | JWT 认证 + bcrypt | 限流，用户枚举防护，JTI |
| [CRUD](/zh/modules/crud) | 通用 CRUD + 分页 | 排序/过滤白名单，SQL注入防护 |
| [Middleware](/zh/modules/middleware) | Recovery、CORS、日志、超时 | CRLF 注入防护，请求体大小限制 |
| [RBAC](/zh/modules/rbac) | 基于角色的访问控制 | 通配符权限，循环检测 |
| [Rate Limit](/zh/modules/ratelimit) | 固定/滑动窗口、令牌桶 | 内存耗尽防护，LRU 淘汰 |
| [Validation](/zh/modules/validation) | 链式请求验证 | 正则DoS防护，空字节拒绝 |
| [File Upload](/zh/modules/file-upload) | 安全文件上传 | 魔数校验，路径遍历防护 |
| [Health](/zh/modules/health) | 存活/就绪探针 | SSRF 防护，CRLF 注入防护 |
| [Cache](/zh/modules/cache) | TTL + LRU 内存缓存 | 后台清理，最大条目限制 |
| [Pagination](/zh/modules/pagination) | 偏移/游标分页 | 游标 Base64 校验，最大限制强制 |
| [Mail](/zh/modules/mail) | SMTP 邮件 + 模板 | 头部注入防护，XSS 转义 |

## 快速复制

```bash
# 复制模块到你的项目
cp -r registry/<module>/src/go/* yourproject/internal/<module>/
```

## 模块结构

每个模块遵循以下结构：

```
registry/<module>/
├── src/go/
│   ├── go.mod              # module <name>, go 1.22
│   ├── config.go           # Options struct, Defaults(), FromEnv()
│   ├── handler.go          # HTTP handlers
│   ├── <core>.go           # 核心逻辑
│   ├── <core>_test.go      # 功能测试
│   └── pentest_test.go     # 渗透测试用例
├── README.md               # 人类可读的适配指南
└── __llms__.md             # AI可读的摘要 (~150 tokens)
```

## 测试

每个模块包含功能测试和渗透测试用例：

```bash
cd registry/<module>/src/go
go test -v ./...
```

## 依赖

模块默认仅使用 Go 标准库。显式例外（如 auth）会在 standalone 模式下复制自己的 `go.mod`。
