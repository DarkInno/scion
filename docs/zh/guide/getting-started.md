# 快速开始

## 1. 选择模块

浏览 [模块](/zh/modules/) 部分或 `registry/` 目录找到你需要的模块。

## 2. 复制模块

```bash
# 示例：复制 auth 模块
cp -r registry/auth/src/go/* yourproject/internal/auth/
```

## 3. 适配模块

每个模块的 `README.md` 包含适配清单：

1. **数据库层** — 实现 store 接口
2. **配置** — 设置环境变量
3. **路由** — 根据需要调整前缀

## 4. 运行测试

```bash
# 运行测试
cd yourproject/internal/auth
go test -v ./...
```

## 可用模块

| 模块 | 描述 |
|------|------|
| [Auth](/zh/modules/auth) | JWT 认证 + bcrypt |
| [CRUD](/zh/modules/crud) | 通用 CRUD + 分页 |
| [Middleware](/zh/modules/middleware) | Recovery、CORS、日志、超时 |
| [RBAC](/zh/modules/rbac) | 基于角色的访问控制 |
| [Rate Limit](/zh/modules/ratelimit) | 固定/滑动窗口、令牌桶 |
| [Validation](/zh/modules/validation) | 链式请求验证 |
| [File Upload](/zh/modules/file-upload) | 安全文件上传 |
| [Health](/zh/modules/health) | 存活/就绪探针 |
| [Cache](/zh/modules/cache) | TTL + LRU 内存缓存 |
| [Pagination](/zh/modules/pagination) | 偏移/游标分页 |
| [Mail](/zh/modules/mail) | SMTP 邮件 + 模板 |

## 项目结构

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

## 下一步

- 阅读 [为什么复制粘贴？](/zh/guide/why-copy-paste) 了解设计理念
- 查看 [安全设计](/zh/guide/security) 了解安全最佳实践
- 浏览 [模块](/zh/modules/) 找到你需要的模块
