package middleware

import (
	"context"
	"testing"
)

func TestContextKeysDoNotCollide(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, requestIDKey, "request")
	ctx = context.WithValue(ctx, clientIPKey, "client")
	ctx = context.WithValue(ctx, traceIDKey, "trace")
	if ctx.Value(requestIDKey) == ctx.Value(clientIPKey) {
		t.Fatal("request and client IP keys collided")
	}
	if ctx.Value(traceIDKey) != "trace" {
		t.Fatalf("trace key lookup failed: %v", ctx.Value(traceIDKey))
	}
}
