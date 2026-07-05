# Contributing

## Adding a New Pattern

1. Create `registry/<pattern-name>/` with the following structure:
   - `__llms__.md` — AI-readable summary (100-200 tokens)
   - `README.md` — Human-readable adaptation guide + security checklist
   - `src/go/` — Go source code
   - `src/python/` — Python source code (optional)
   - `examples/gin/` — Go runnable example
   - `examples/fastapi/` — Python runnable example (optional)

2. Update `registry/index.json` with the new pattern entry.

3. Ensure all code follows:
   - Go 1.22+ with generics and standard `net/http`
   - Python 3.12+ with type annotations (if applicable)
   - Configuration via environment variables only
   - Self-contained: each module works independently after copying
   - Comments are primary documentation
