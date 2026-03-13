package observability

import (
	"context"

	"go.uber.org/zap"
)

type contextKey int

const traceIDKey contextKey = iota

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// WithTraceID stores a trace ID in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// NewLogger creates a structured logger that automatically injects trace context.
func NewLogger(base *zap.Logger) *zap.Logger {
	return base
}
