# Mail

Zero-dependency Go SMTP mail module. Copy `src/go/*.go` into `internal/mail`. Configure with `Options`, `Defaults`, or `FromEnv`; use `NewSender`, `Send`, optional `SendAsync`, and `Close`. Validates addresses, sender, subject, text/html body sizes, attachment count/size/name/path, and rejects CRLF or null bytes to prevent header injection and traversal.
