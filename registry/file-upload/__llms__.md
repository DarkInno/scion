# File Upload

Zero-dependency Go file upload module. Copy `src/go/*.go` into `internal/fileupload`. Uses `net/http`, validates magic bytes instead of trusting extensions or Content-Type, generates server-side filenames, and stores through `Storage`. `LocalStorage` rejects separators, `..`, CRLF, and null bytes, then verifies the resolved path stays inside the root. Defaults allow common images and PDF with a 10 MiB limit.
