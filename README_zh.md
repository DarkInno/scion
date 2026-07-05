# Scion

> 把后端常用模式嫁接到你的项目里：复制源码，而不是引入依赖。

[English](README.md) | [中文](README_zh.md)

Scion 是面向 Go 后端开发的复制粘贴式源码模板库。它把常见后端模块以源码模板的形式发布，你可以把模块复制进自己的项目，按业务改造，并最终拥有这些代码。

## 快速开始

需要 Go 1.22 或更新版本：

```bash
go install github.com/DarkInno/scion/cmd/scion@latest
```

如果需要固定版本：

```bash
go install github.com/DarkInno/scion/cmd/scion@v0.1.2
```

确认 Go bin 目录已加入 `PATH`，然后验证安装：

```bash
scion version
scion list
```

复制一个仅使用标准库的模块：

```bash
scion add cache --to internal/cache --dry-run
scion add cache --to internal/cache
scion diff cache --target internal/cache
```

Scion CLI 会复制源码，并写入 `.scion-module.json` 供后续对比使用。它不会自动修改你的 `go.mod`。标记为 `stdlibOnly=false` 的模块，例如 `auth`，需要显式使用 standalone 模式：

```bash
scion add auth --standalone --to internal/auth
```

## 二进制下载

预编译二进制可以在 [GitHub Releases](https://github.com/DarkInno/scion/releases) 下载，支持 macOS、Linux、Windows 的 amd64 和 arm64。

使用 `SHA256SUMS` 校验下载文件：

```bash
sha256sum -c SHA256SUMS
```

Windows PowerShell：

```powershell
Get-FileHash .\scion_v0.1.2_windows_amd64.zip -Algorithm SHA256
```

## 为什么是复制粘贴？

认证、CRUD、文件上传、限流等后端模块在不同项目里大部分结构相同，但最后一段总要贴合具体业务：

- 你需要深入模块内部修改业务逻辑。
- 你想拥有源码，而不是被上游包的 API 设计锁住。
- AI 编码助手更擅长处理能直接读取和编辑的源码。
- Scion 默认避免依赖膨胀；安全相关例外会在模块元数据中明确标记。

## 可用模块

| 模块 | 描述 | 安全特性 |
|------|------|----------|
| [auth](registry/auth/) | JWT 邮箱/密码认证 + bcrypt | 限流、用户枚举防护、JTI、aud/iss 校验 |
| [crud](registry/crud/) | 泛型 CRUD + 分页 | 排序/过滤白名单、SQL 注入防护、分页上限 |
| [middleware](registry/middleware/) | Recovery、CORS、日志、超时等 | CRLF 注入防护、可信代理、请求体大小限制 |
| [rbac](registry/rbac/) | 基于角色的访问控制 | 通配符权限、循环检测、层级继承 |
| [ratelimit](registry/ratelimit/) | 固定窗口 / 滑动窗口 / 令牌桶 | 内存耗尽防护、LRU 驱逐、key 长度限制 |
| [validation](registry/validation/) | 链式请求校验构建器 | 正则 DoS 防护、拒绝 null 字节/CRLF、panic 恢复 |
| [file-upload](registry/file-upload/) | 安全文件上传处理器 | Magic bytes 校验、路径穿越防护、大小限制、限流 |
| [health](registry/health/) | 存活/就绪探针 | SSRF 防护、CRLF 注入防护 |
| [cache](registry/cache/) | 泛型 TTL + LRU 缓存 | 后台清理、goroutine 泄漏防护、最大条目数限制 |
| [pagination](registry/pagination/) | Offset/limit + cursor 分页 | cursor base64 校验、负 offset 归零、最大 limit 限制 |
| [mail](registry/mail/) | SMTP 邮件 + 模板 | 邮件头注入防护、XSS 转义、附件净化、异步队列 |

## CLI 命令

```bash
scion list [--json]
scion info <module> [--json]
scion add <module> --to <dir> [--dry-run] [--force] [--standalone]
scion diff <module> --target <dir> [--json]
scion doctor [--strict] [--json]
scion version [--json]
```

## 项目结构

```text
scion/
|-- cmd/scion/              # CLI 入口
|-- internal/               # CLI 实现、bundle 读取、doctor 检查
|-- internal/bundle/        # 从 registry/ 生成的内置模板包
|-- registry/
|   |-- index.json          # 机器可读模块索引
|   |-- auth/               # 认证模块
|   |-- cache/              # TTL + LRU 缓存
|   |-- crud/               # CRUD 模块
|   |-- file-upload/        # 文件上传模块
|   |-- health/             # 健康检查模块
|   |-- mail/               # SMTP 邮件模块
|   |-- middleware/         # HTTP 中间件集合
|   |-- pagination/         # 分页工具
|   |-- ratelimit/          # 限流算法
|   |-- rbac/               # 角色权限控制
|   `-- validation/         # 请求校验构建器
|-- docs/                   # VitePress 文档
|-- AGENTS.md               # AI 编码代理说明
|-- CONTRIBUTING.md         # 贡献指南
`-- LICENSE                 # MIT
```

## 开发

```bash
git clone https://github.com/DarkInno/scion.git
cd scion

# registry 改动后重新生成 CLI 内置 bundle
go run ./internal/cmd/build-bundle

# 测试和静态检查 CLI
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...

# 严格检查 registry
go run ./cmd/scion doctor --strict
```

PowerShell 中运行所有 registry 模块测试：

```powershell
$modules = @('middleware','auth','crud','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }
```

## 发布

通过语义化版本 tag 发布：

```bash
git tag -a v0.1.2 -m "v0.1.2"
git push origin v0.1.2
```

Release workflow 会验证 CLI、检查内置 bundle、交叉编译二进制、生成 `SHA256SUMS`，并发布 GitHub Release 资产。

## 许可证

[MIT](LICENSE)
