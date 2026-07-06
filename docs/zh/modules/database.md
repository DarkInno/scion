# Database 数据库模块

`database/sql` 设置、事务封装和安全 SQL 片段构建。

## 包含内容

- 显式连接池默认值
- 基于 CPU 和 IO 能力的动态连接池 sizing
- 基于 `DATABASE_*` 的 `FromEnv()`
- 带超时的启动 ping，错误不暴露 DSN
- `WithinTx`，在错误或 panic 时回滚
- 供 repository 使用的 `DBTX` 接口
- 基于白名单的 `WHERE` 和 `ORDER BY` 片段构建
- 每个 builder/query 最多 32 个过滤条件

## 快速复制

```bash
cp -r registry/database/src/go/* yourproject/internal/database/
```

## 使用方式

```go
opts := database.FromEnv()
db, err := database.Open(ctx, opts)
if err != nil {
    return err
}
defer db.Close()

err = database.WithinTx(ctx, db, nil, func(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?", true, 123)
    return err
})
```

## 安全 SQL 片段

```go
columns := database.ColumnMap{
    "email": "users.email",
    "created_at": "users.created_at",
}

where, args, err := database.WhereEqual(
    map[string]string{"email": "ada@example.com"},
    columns,
    database.DollarPlaceholder,
)
order, err := database.OrderBy("-created_at", columns)
```

## 动态连接池策略

`Defaults()` 默认使用基于 `GOMAXPROCS` 的 balanced 动态连接池。IO-heavy
服务可以显式使用更大的有界连接池：

```go
opts := database.ApplyPoolStrategy(database.FromEnv(), database.PoolStrategy{
    Profile:       database.PoolIOHeavy,
    CPUCores:      8,
    IOParallelism: 4,
    MaxOpenLimit:  96,
})
```

环境变量：

| 变量 | 用途 |
|------|------|
| `DATABASE_POOL_PROFILE` | `conservative`、`balanced` 或 `io-heavy` |
| `DATABASE_POOL_CPU_CORES` | sizing 公式使用的 CPU 核数 |
| `DATABASE_IO_PARALLELISM` | 额外独立 IO 能力 |
| `DATABASE_POOL_MAX_OPEN_LIMIT` | 推导出的最大连接数硬上限 |

## 安全特性

- 拒绝字符串输入中的 CRLF 和 null 字节
- 对 driver 名称、DSN、字段和值设置长度限制
- SQL 标识符必须来自白名单映射
- 过滤值只作为参数返回，不拼接进 SQL
- 限制生成的过滤条件数量，避免无界内存增长
- open 或 ping 错误不包含 DSN

## 测试

```bash
cd registry/database/src/go
go test -v ./...
```
