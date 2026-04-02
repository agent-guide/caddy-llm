package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	configstoreintf "github.com/agent-guide/caddy-agent-gateway/configstore/intf"
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

// cachedProviderEntry holds a cached provider instance and the config fingerprint
// used to detect config changes.
type cachedProviderEntry struct {
	cfgJSON  string
	provider provider.Provider
	name     string
}

// cachedDynamicResolver wraps a ProviderConfigStorer and caches provider instances
// by their serialized config. A config change (different JSON fingerprint) causes
// the cached instance to be replaced, avoiding per-request provider construction.
type cachedDynamicResolver struct {
	mu    sync.RWMutex
	store configstoreintf.ProviderConfigStorer
	cache map[string]cachedProviderEntry
}

func newCachedDynamicResolver(store configstoreintf.ProviderConfigStorer) *cachedDynamicResolver {
	return &cachedDynamicResolver{
		store: store,
		cache: make(map[string]cachedProviderEntry),
	}
}

func (r *cachedDynamicResolver) ResolveProvider(ctx context.Context, ref string) (provider.Provider, string, error) {
	tag, obj, err := r.store.Get(ctx, ref)
	if err != nil {
		return nil, "", err
	}

	cfgJSON, err := json.Marshal(obj)
	if err != nil {
		return nil, "", fmt.Errorf("fingerprint provider config %q: %w", ref, err)
	}
	fingerprint := string(cfgJSON)

	r.mu.RLock()
	entry, ok := r.cache[ref]
	r.mu.RUnlock()
	if ok && entry.cfgJSON == fingerprint {
		return entry.provider, entry.name, nil
	}

	resolvedCfg, err := provider.NormalizeStoredProviderConfig(tag, obj)
	if err != nil {
		return nil, "", fmt.Errorf("normalize provider config %q: %w", ref, err)
	}

	prov, err := provider.NewProvider(resolvedCfg)
	if err != nil {
		return nil, "", err
	}

	r.mu.Lock()
	r.cache[ref] = cachedProviderEntry{cfgJSON: fingerprint, provider: prov, name: resolvedCfg.ProviderName}
	r.mu.Unlock()

	return prov, resolvedCfg.ProviderName, nil
}
