# 贡献指南

欢迎贡献！以下是向 Scion 添加新模块的方法。

## 添加新模块

### 1. 创建模块结构

```
registry/<module-name>/
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

### 2. 遵循安全要求

每个模块必须实现：

- **CRLF 注入防护** — 在所有用户输入中拒绝 `\r\n`
- **空字节拒绝** — 在所有字符串输入中拒绝 `\x00`
- **长度限制** — 所有用户提供的字符串都有最大长度检查
- **内存耗尽防护** — 无界增长的 map/slice 必须有限制
- **不信任 X-Forwarded-For** — `ClientIP()` 仅返回 `r.RemoteAddr`
- **路径遍历防护** — 使用 `filepath.Base()` + 拒绝 `..`
- **参数化查询** — 永远不拼接用户输入到 SQL
- **Panic 恢复** — 所有 HTTP handler 必须从 panic 中恢复

### 3. 编写测试

每个源文件需要对应的 `_test.go`：

```bash
cd registry/<module>/src/go
go test -v ./...
```

### 4. 添加文档

- `README.md` — 人类可读的适配指南
- `__llms__.md` — AI可读的摘要 (~150 tokens)

### 5. 更新注册表索引

将你的模块添加到 `registry/index.json`。

## 代码标准

- Go 1.22+ 支持泛型
- 默认仅使用标准库；外部依赖必须有显式的 `stdlibOnly:false` 注册表声明，并说明安全或正确性理由
- `gofmt` 格式化
- `go vet` 必须通过
- 中间件签名：`func(http.Handler) http.Handler`
- 使用 `log/slog` 日志

## 测试

```bash
# 测试单个模块
cd registry/<module>/src/go && go test -v ./...

# 测试所有模块
$modules = @('middleware','auth','crud','database','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail','migrations','metrics','problem')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }
```

## Pull Request 流程

1. Fork 仓库
2. 创建功能分支
3. 按照上述指南添加模块
4. 运行所有测试
5. 提交 Pull Request

## 问题？

在 GitHub 上开 issue 或加入讨论。
