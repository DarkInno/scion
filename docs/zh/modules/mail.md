# Mail 邮件模块

SMTP 邮件发送器，支持模板和安全特性。

## 包含内容

- SMTP 邮件发送
- HTML 模板
- 头部注入防护
- XSS 转义
- 附件清理
- 异步队列

## 快速复制

```bash
cp -r registry/mail/src/go/* yourproject/internal/mail/
```

## 使用方式

### 基本发送

```go
sender := mail.NewSender(mail.Config{
    Host:     "smtp.example.com",
    Port:     587,
    Username: "user@example.com",
    Password: "password",
    From:     "noreply@example.com",
})

err := sender.Send(mail.Message{
    To:      []string{"user@example.com"},
    Subject: "Welcome",
    Body:    "<h1>Welcome!</h1>",
    IsHTML:  true,
})
```

### 使用模板

```go
sender := mail.NewSender(mail.Config{
    TemplateDir: "./templates",
})

err := sender.SendTemplate(mail.Message{
    To:      []string{"user@example.com"},
    Subject: "Welcome",
}, "welcome.html", map[string]any{
    "Name": "John",
})
```

### 异步队列

```go
sender := mail.NewSender(mail.Config{
    QueueSize: 100,
    Workers: 4,
})

// 非阻塞发送
sender.SendAsync(mail.Message{...})
```

## 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `Host` | SMTP 主机 | 必需 |
| `Port` | SMTP 端口 | 必需 |
| `Username` | SMTP 用户名 | 必需 |
| `Password` | SMTP 密码 | 必需 |
| `From` | 发件人地址 | 必需 |
| `TemplateDir` | 模板目录 | 无 |
| `QueueSize` | 异步队列大小 | 0（同步） |
| `Workers` | 异步工作线程 | 1 |

## 文件参考

| 文件 | 用途 |
|------|------|
| `sender.go` | 邮件发送器 |
| `config.go` | 配置 |
| `message.go` | 消息类型 |
| `template.go` | 模板引擎 |

## 安全特性

- 头部注入防护
- 模板 XSS 转义
- 附件清理
- 输入验证

## 测试

```bash
cd registry/mail/src/go
go test -v ./...
```
