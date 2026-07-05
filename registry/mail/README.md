# Mail Module

SMTP email sender with message validation, templates, attachments, and async sending.

## What's Included

- SMTP sender using the Go standard library
- Plain text and HTML body support
- HTML template rendering
- Attachment path sanitization
- Header injection protection
- Optional async queue with workers

## Quick Copy

```bash
cp -r registry/mail/src/go/*.go yourproject/internal/mail/
```

Or with the Scion CLI:

```bash
scion add mail --to internal/mail
```

## Usage

```go
sender, err := mail.NewSender(mail.Options{
	Host: "smtp.example.com",
	Port: 587,
	From: "noreply@example.com",
})
if err != nil {
	return err
}
defer sender.Close()

err = sender.Send(&mail.Message{
	To:      []string{"user@example.com"},
	Subject: "Welcome",
	Body:    "Welcome!",
})
```

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Options, defaults, environment loading |
| `message.go` | Message and attachment validation |
| `sender.go` | SMTP and async queue implementation |
| `template.go` | HTML template rendering |
| `pentest_test.go` | Header and attachment abuse tests |

## Tests

```bash
cd registry/mail/src/go
go test -v ./...
```
