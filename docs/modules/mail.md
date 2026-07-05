# Mail Module

SMTP email sender with templates and security features.

## What's Included

- SMTP email sending
- HTML templates
- Header injection prevention
- XSS escaping
- Attachment sanitization
- Async queue

## Quick Copy

```bash
cp -r registry/mail/src/go/* yourproject/internal/mail/
```

## Usage

### Basic Sending

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

### With Templates

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

### Async Queue

```go
sender := mail.NewSender(mail.Config{
    QueueSize: 100,
    Workers: 4,
})

// Non-blocking send
sender.SendAsync(mail.Message{...})
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `Host` | SMTP host | Required |
| `Port` | SMTP port | Required |
| `Username` | SMTP username | Required |
| `Password` | SMTP password | Required |
| `From` | Sender address | Required |
| `TemplateDir` | Template directory | None |
| `QueueSize` | Async queue size | 0 (sync) |
| `Workers` | Async workers | 1 |

## File Reference

| File | Purpose |
|------|---------|
| `sender.go` | Email sender |
| `config.go` | Configuration |
| `message.go` | Message types |
| `template.go` | Template engine |

## Security Features

- Header injection prevention
- XSS escaping in templates
- Attachment sanitization
- Input validation

## Tests

```bash
cd registry/mail/src/go
go test -v ./...
```
