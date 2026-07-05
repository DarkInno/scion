# Getting Started

## 1. Choose a Pattern

Browse `registry/index.json` or the `registry/` directory to find the pattern you need.

## 2. Copy It

```bash
# Example: copy the auth module
cp -r registry/auth/src/* src/auth/
cp -r registry/auth/examples/fastapi/* src/auth/
```

## 3. Adapt It

Read `registry/auth/README.md` for the adaptation checklist:
- Update database models
- Set environment variables
- Adjust route prefixes

## 4. Run It

Each example in `examples/<framework>/` is a minimal runnable project. Use it as a reference.
