package gateway

import (
	"context"
	"fmt"

	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
)

// ProviderLookup resolves a statically provisioned provider instance by reference.
type ProviderLookup func(name string) (provider.Provider, bool)

// ProviderResolver resolves a provider reference into an executable provider instance.
type ProviderResolver interface {
	ResolveProvider(ctx context.Context, ref string) (provider.Provider, string, error)
}

// ProviderResolverFunc adapts a function into ProviderResolver.
type ProviderResolverFunc func(ctx context.Context, ref string) (provider.Provider, string, error)

func (f ProviderResolverFunc) ResolveProvider(ctx context.Context, ref string) (provider.Provider, string, error) {
	return f(ctx, ref)
}

// NewStaticProviderResolver wraps a static provider lookup as a ProviderResolver.
func NewStaticProviderResolver(lookup ProviderLookup) ProviderResolver {
	if lookup == nil {
		return nil
	}
	return ProviderResolverFunc(func(_ context.Context, ref string) (provider.Provider, string, error) {
		prov, ok := lookup(ref)
		if !ok {
			return nil, "", fmt.Errorf("provider %q is not configured", ref)
		}
		return prov, ref, nil
	})
}

// ChainProviderResolvers tries resolvers in order until one succeeds.
func ChainProviderResolvers(resolvers ...ProviderResolver) ProviderResolver {
	filtered := make([]ProviderResolver, 0, len(resolvers))
	for _, resolver := range resolvers {
		if resolver != nil {
			filtered = append(filtered, resolver)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return ProviderResolverFunc(func(ctx context.Context, ref string) (provider.Provider, string, error) {
		var lastErr error
		for _, resolver := range filtered {
			prov, name, err := resolver.ResolveProvider(ctx, ref)
			if err == nil && prov != nil {
				return prov, name, nil
			}
			if err != nil {
				lastErr = err
			}
		}
		if lastErr != nil {
			return nil, "", lastErr
		}
		return nil, "", fmt.Errorf("provider %q is not configured", ref)
	})
}
