# Migrations 迁移模块

基于 `database/sql` 的标准库 SQL 迁移执行器。

## 功能

- 从 `fs.FS` 加载 `YYYYMMDDHHMMSS_name.up.sql` 和可选 `.down.sql`
- 在 `schema_migrations` 中记录 version、name、checksum、timestamp
- 默认使用事务执行迁移
- 支持 `Up`、`Down`、`Status`
- 提供带 panic 恢复的状态 HTTP handler

## 使用

```go
m, err := migrations.New(migrationFS, migrations.Options{
    Dir: "migrations",
})
if err != nil {
    return err
}
if err := m.Up(ctx, db); err != nil {
    return err
}
```

PostgreSQL 风格占位符使用 `migrations.DollarPlaceholder`。

## 安全

- 拒绝配置和文件名中的 CRLF / null 字节
- 拒绝迁移目录和文件名中的路径穿越
- 校验 schema history 表名
- 限制迁移数量和 SQL 文件大小

## 复制

```bash
scion add migrations --to internal/migrations
```
