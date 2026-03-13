package manager

import "context"

type skipPersistContextKey struct{}

// WithSkipPersist returns a derived context that disables persistence for
// Manager Register/Update calls. Useful for code paths that are reacting to
// store-watcher events where the store is already the source of truth and
// writing back would create a loop.
func WithSkipPersist(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, skipPersistContextKey{}, true)
}

func shouldSkipPersist(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v := ctx.Value(skipPersistContextKey{})
	enabled, ok := v.(bool)
	return ok && enabled
}
