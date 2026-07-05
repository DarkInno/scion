# RBAC

Zero-dependency Go RBAC module. Copy `src/go/*.go` into `internal/rbac`. Uses explicit roles and permissions with wildcard matching, hierarchy inheritance, and cycle detection. Role IDs and permissions reject empty, oversized, CRLF, and null-byte values. HTTP middleware reads roles from request context; integrate it after authentication middleware that sets trusted roles.
