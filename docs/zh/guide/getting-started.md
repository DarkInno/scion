# 快速开始

## 1. 安装 CLI

Scion 需要 Go 1.22 或更新版本。

```bash
go install github.com/DarkInno/scion/cmd/scion@latest
```

如果需要固定版本：

```bash
go install github.com/DarkInno/scion/cmd/scion@v0.1.3
```

确认 Go bin 目录已加入 `PATH`，然后验证安装：

```bash
scion version
scion list
```

## 2. 选择模块

可以浏览 [模块](/zh/modules/) 页面，也可以运行：

```bash
scion list
scion info cache
```

标记为 `stdlibOnly=true` 的模块可以直接复制进现有项目。标记为 `stdlibOnly=false` 的模块，例如 `auth`，需要使用 `--standalone`，让它的 `go.mod` 和 `go.sum` 被显式复制。

## 3. 复制源码到项目

先预览将要复制的文件：

```bash
scion add cache --dry-run
```

执行复制：

```bash
scion add cache
```

省略 `--to` 时，Scion 会使用模块默认目标目录，例如 `cache` 的默认目标是 `internal/cache`。你可以用 `--to <dir>` 覆盖。

Scion 会在目标目录写入 `.scion-module.json`。这个文件记录模块名、registry 版本、源文件 hash，以及是否使用 standalone 模式。

## 4. 后续对比

当你改造过复制进项目的源码后，可以和 Scion 内置模板对比：

```bash
scion diff cache
```

省略 `--target` 时，`diff` 会使用模块默认目标目录。它只报告差异，不会自动合并，也不会覆盖你的本地改动。

## Standalone 模块

`auth` 模块为了 JWT 和 bcrypt 使用了成熟安全依赖。复制它时需要 standalone 模式：

```bash
scion add auth --standalone
```

Scion 仍然不会修改你的项目级 `go.mod`；standalone 模式只会把该模块自己的 `go.mod` 和 `go.sum` 复制到目标目录。

## 二进制下载

你也可以从 [GitHub Releases](https://github.com/DarkInno/scion/releases) 下载预编译二进制。

使用 `SHA256SUMS` 校验下载文件：

```bash
sha256sum -c SHA256SUMS
```

Windows PowerShell：

```powershell
Get-FileHash .\scion_v0.1.3_windows_amd64.zip -Algorithm SHA256
```

## 手动复制

如果你正在 Scion 仓库内部工作，仍然可以手动复制：

```bash
cp -r registry/cache/src/go/*.go yourproject/internal/cache/
```

推荐优先使用 CLI，因为它会验证路径、记录元数据、支持 dry-run，并允许之后使用 `scion diff`。

## 可用模块

| 模块 | 描述 |
|------|------|
| [Auth](/zh/modules/auth) | JWT 认证 + bcrypt |
| [CRUD](/zh/modules/crud) | 泛型 CRUD + 分页 |
| [Middleware](/zh/modules/middleware) | Recovery、CORS、日志、超时 |
| [RBAC](/zh/modules/rbac) | 基于角色的访问控制 |
| [Rate Limit](/zh/modules/ratelimit) | 固定窗口、滑动窗口、令牌桶 |
| [Validation](/zh/modules/validation) | 链式请求校验 |
| [File Upload](/zh/modules/file-upload) | 安全文件上传 |
| [Health](/zh/modules/health) | 存活/就绪探针 |
| [Cache](/zh/modules/cache) | TTL + LRU 内存缓存 |
| [Pagination](/zh/modules/pagination) | Offset/cursor 分页 |
| [Mail](/zh/modules/mail) | SMTP 邮件 + 模板 |

## 模块结构

每个 registry 模块遵循以下结构：

```text
registry/<module>/
|-- README.md               # 面向人的适配指南
|-- __llms__.md             # 面向 AI 的摘要
`-- src/go/
    |-- go.mod              # module <name>, go 1.22
    |-- config.go           # Options、Defaults()、FromEnv()
    |-- *_test.go           # 功能测试
    `-- pentest_test.go     # 攻击场景测试
```

## 下一步

- 阅读 [为什么复制粘贴？](/zh/guide/why-copy-paste) 了解设计理念。
- 查看 [安全设计](/zh/guide/security) 了解安全边界和约束。
- 浏览 [模块](/zh/modules/) 找到你需要的模板。
