# Health

Zero-dependency Go health probe module. Copy `src/go/*.go` into `internal/health`. Provides `HealthChecker`, `HealthHandler`, `NewCustomCheck`, `NewTCPCheck`, and `NewHTTPCheck`. HTTP checks block localhost, loopback, private, link-local, unspecified, and multicast IPs unless explicitly allow-listed; dial-time validation reduces DNS rebinding risk. Handlers emit JSON and ignore encoder errors with `_ =`.
