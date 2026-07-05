# Scion

> 将后端模式嫁接到你的项目中。复制粘贴，而非安装依赖。

[English](README.md) | [中文](README_zh.md)

Scion 是一个面向 Go 后端开发的复制粘贴代码库。你不需要安装框架或拉取依赖，只需将预构建的生产级模块复制到你的项目中，每一行代码都归你所有。

## 为什么要复制粘贴？

后端模块（认证、CRUD、文件上传、限流）在不同项目中 80% 的骨架是相同的，但剩下的 20% 差异使得 npm/go 包变得别扭：

- 你需要在模块深处自定义业务逻辑
- 你想拥有代码的所有权，而不是被上游版本锁定
- 你的 AI 编码助手在能直接阅读和修改代码时表现更好
- 没有依赖地狱 — 零外部依赖，仅使用 Go 标准库

## 快速开始

```bash
# 1. 将模块复制到你的项目中
cp -r registry/auth/src/go/* yourproject/internal/auth/

# 2. 适配配置
#    编辑 config.go：设置 JWT 密钥、数据库 URL 等

# 3. 实现 Store 接口
#    type UserStore interface { ... }  // 你的数据库层

# 4. 注册路由
#    参考 registry/auth/examples/gin/main.go
```

## 可用模块

| 模块 | 描述 | 安全特性 |
|------|------|---------|
| [auth](registry/auth/) | JWT 邮箱/密码认证 + bcrypt | 限流、用户枚举防护、JTI、aud/iss 验证 |
| [crud](registry/crud/) | 泛型 CRUD + 分页 | 排序/过滤白名单、SQL 注入防护、分页上限 |
| [middleware](registry/middleware/) | Recovery、CORS、日志、超时等 | CRLF 注入防护、可信代理、请求体大小限制 |
| [rbac](registry/rbac/) | 基于角色的访问控制 | 通配符权限、循环检测、层级继承 |
| [ratelimit](registry/ratelimit/) | 固定窗口/滑动窗口/令牌桶 | 内存耗尽防护、LRU 驱逐、key 长度限制 |
| [validation](registry/validation/) | 链式请求验证构建器 | 正则 DoS 防护（RE2）、null 字节/CRLF 拒绝、panic 恢复 |
| [file-upload](registry/file-upload/) | 安全文件上传处理器 | magic bytes 验证、路径穿越防护、大小限制、限流 |
| [health](registry/health/) | 存活/就绪探针 | SSRF 防护（拒绝内网 IP）、CRLF 注入防护 |
| [cache](registry/cache/) | 泛型 TTL + LRU 缓存 | 后台清理、goroutine 泄漏防护、最大条目数限制 |
| [pagination](registry/pagination/) | Offset/limit + cursor 分页 | cursor base64 验证、负数 offset 归零、最大 limit 限制 |
| [mail](registry/mail/) | SMTP 邮件 + 模板 | 邮件头注入防护、XSS 转义、附件净化、异步队列 |

## 项目结构

```
scion/
├── registry/
│   ├── index.json              # 机器可读的模块索引
│   ├── auth/                   # 认证模块
│   │   ├── __llms__.md         # AI 可读摘要（约 150 token）
│   │   ├── README.md           # 人类可读适配指南
│   │   ├── src/go/             # Go 源代码
│   │   └── examples/gin/       # 最小可运行示例
│   ├── crud/                   # CRUD 操作模块
│   ├── middleware/             # HTTP 中间件集合
│   ├── rbac/                   # 角色权限控制
│   ├── ratelimit/              # 限流算法
│   ├── validation/             # 请求验证构建器
│   ├── file-upload/            # 文件上传处理器
│   ├── health/                 # 健康检查探针
│   ├── cache/                  # 内存缓存
│   ├── pagination/             # 分页工具
│   └── mail/                   # 邮件发送
├── docs/
│   └── getting-started.md      # 如何使用 Scion
├── AGENTS.md                   # AI 编码代理指令（英文）
├── AGENTS_zh.md                # AI 编码代理指令（中文）
├── CONTRIBUTING.md             # 贡献指南
├── LICENSE                     # MIT
└── llms.txt                    # LLM 友好的项目摘要
```

## 设计原则

1. **代码所有权** — 复制后每一行代码都是你的，没有上游锁定。
2. **自包含** — 每个模块独立工作，零外部依赖。
3. **框架无关** — 使用 Go 标准 `net/http`，可适配 Gin/Echo 等。
4. **安全优先** — 内置输入验证、限流、注入防护。
5. **AI 友好** — `__llms__.md` 文件让 AI 助手在约 200 token 内理解模块。
6. **经过测试** — 每个模块包含功能测试和渗透测试用例。

## 开发

```bash
# 克隆仓库
git clone https://github.com/your-org/scion.git
cd scion

# 运行单个模块的测试
cd registry/auth/src/go && go test -v ./...

# 运行所有模块的测试（PowerShell）
$modules = @('middleware','auth','crud','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }

# 格式化代码
cd registry/auth/src/go && gofmt -w .
```

## 贡献

欢迎贡献！请阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 了解添加新模块的指南。

## 许可证

[MIT](LICENSE) — Copyright (c) 2026 Scion Contributors
