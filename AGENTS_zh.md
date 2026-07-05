# AGENTS_zh.md

> AI 编码代理（Codex、Claude、Cursor 等）在本项目上工作的指令文件。

[English](AGENTS.md) | [中文](AGENTS_zh.md)

## 项目描述

Scion 是一个面向 Go 后端开发的复制粘贴代码库。`registry/` 目录下有 11 个自包含模块，每个都是独立的 Go 包。模块默认仅使用标准库；安全敏感模块可以在 `registry/index.json` 中显式标记为 `stdlibOnly:false`，并以 standalone 模式复制。模块的设计目的是被复制到用户项目中并适配，而非作为依赖导入。

## 编码标准

- Go 1.22+，使用泛型
- 默认仅使用标准库；显式标记为 `stdlibOnly:false` 的模块可以使用成熟安全库
- `gofmt` 格式化是强制的
- `go vet` 必须零警告通过
- 中间件签名：`func(http.Handler) http.Handler`
- 所有 `json.NewEncoder(w).Encode()` 错误必须用 `_ =` 显式忽略
- `defer r.Body.Close()` 放在 `io.ReadAll` 之后，不是之前（Go http.Server 会自动关闭 body）
- 使用 `log/slog` 记录日志 — 禁止 `fmt.Println` 或 `log.Printf`

## 模块约定

每个模块位于 `registry/<模块名>/src/go/`，结构如下：

```
registry/<module>/
├── src/go/
│   ├── go.mod              # module <name>, go 1.22
│   ├── config.go           # Options 结构体、Defaults()、FromEnv()
│   ├── handler.go          # HTTP 处理器
│   ├── <core>.go           # 核心逻辑
│   ├── <core>_test.go      # 功能测试
│   └── pentest_test.go     # 渗透测试用例
├── README.md               # 人类可读适配指南
└── __llms__.md             # AI 可读摘要（约 150 token）
```

## 安全要求（不可妥协）

每个模块必须实现以下要求：

- **CRLF 注入防护** — 拒绝所有用户输入中的 `\r\n`（headers、URL、名称）
- **Null 字节拒绝** — 拒绝所有字符串输入中的 `\x00`
- **长度限制** — 所有用户提供的字符串都有最大长度检查
- **内存耗尽防护** — 无界增长的 map/slice 必须有 `maxBuckets` 或 `maxEntries` 限制 + LRU 驱逐
- **不信任 X-Forwarded-For** — `ClientIP()` 只返回 `r.RemoteAddr`；XFF 是客户端可控的，可伪造
- **路径穿越防护** — 所有文件操作使用 `filepath.Base()` + 拒绝 `..`
- **参数化查询** — 永远不将用户输入拼接到 SQL 中
- **Panic 恢复** — 所有 HTTP 处理器必须恢复 panic

## 测试要求

- 每个源文件都有对应的 `_test.go`
- 每个模块都有 `pentest_test.go`，包含攻击场景测试用例
- 提交前必须运行测试：

```bash
cd registry/<module>/src/go && go test -v -count=1 ./...
```

- 运行所有模块测试：

```bash
# PowerShell
$modules = @('middleware','auth','crud','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }
```

## 关键约束

- 不要给任何模块的 `go.mod` 添加外部依赖，除非该模块已在 `registry/index.json` 显式标记为 `stdlibOnly:false`，且依赖对安全或正确性有明确价值
- 不要在 HTTP 处理器中使用 `panic` — 返回错误
- 不要信任客户端 header（`Content-Type`、`X-Forwarded-For`、`X-Real-Ip`）
- 不要用 `strings.Split` 解析 header — 用 `strings.SplitN` 加限制
- 不要在未经用户明确同意的情况下修改 `go.mod` 添加依赖
- 配置必须存在于环境变量或带有 `FromEnv()` 的 `config.go` 中
- 所有模块必须框架无关（使用 `net/http`，不直接使用 Gin/Echo）

## AI 提示词模板

当你想让 AI 助手在本项目上工作时，复制以下提示词：

---

```
你正在开发 Scion，一个面向 Go 后端开发的复制粘贴代码库。

项目路径: <scion-项目路径>

架构:
- registry/ 下有 11 个模块 — 每个是独立的 Go 包
- 模块路径模式: registry/<模块名>/src/go/
- Go 1.22+，默认仅使用标准库，gofmt 强制

安全规则（不可妥协）:
1. 拒绝所有用户输入中的 CRLF (\r\n) 和 null 字节 (\x00)
2. 所有字符串有最大长度检查（根据上下文 128-1024 字符）
3. 无界增长的 map 必须有 maxBuckets/maxEntries + LRU 驱逐
4. ClientIP() 只返回 r.RemoteAddr — 永远不信任 X-Forwarded-For
5. 所有文件路径操作使用 filepath.Base() + 拒绝 ".."
6. 所有 HTTP 处理器必须恢复 panic
7. 所有 json.NewEncoder(w).Encode() 错误必须用 `_ =` 忽略

测试要求:
- 每个模块有 pentest_test.go 渗透测试用例
- 运行: cd registry/<模块名>/src/go && go test -v -count=1 ./...

任务: <在此描述你的任务>
```

---

## 常用命令

| 操作 | 命令 |
|------|------|
| 测试单个模块 | `cd registry/auth/src/go && go test -v ./...` |
| 测试所有模块 | 见上方 PowerShell 代码片段 |
| 格式化代码 | `cd registry/<模块名>/src/go && gofmt -w .` |
| 代码检查 | `cd registry/<模块名>/src/go && go vet ./...` |
| 覆盖率 | `cd registry/<模块名>/src/go && go test -cover ./...` |

## 目录结构

```
scion/
├── registry/                 # 11 个复制粘贴模块
│   ├── auth/                 # JWT 认证 + bcrypt
│   ├── crud/                 # 泛型 CRUD + 分页
│   ├── middleware/           # 9 个 HTTP 中间件
│   ├── rbac/                 # 角色权限控制
│   ├── ratelimit/            # 3 种限流算法
│   ├── validation/           # 链式验证构建器
│   ├── file-upload/          # 安全文件上传
│   ├── health/               # 存活/就绪探针
│   ├── cache/                # TTL + LRU 缓存
│   ├── pagination/           # Offset + cursor 分页
│   └── mail/                 # SMTP 邮件发送
├── docs/                     # 人类可读文档
├── AGENTS.md                 # 本文件（英文）
├── AGENTS_zh.md              # 本文件（中文）
├── CONTRIBUTING.md           # 贡献指南
└── LICENSE                   # MIT
```
