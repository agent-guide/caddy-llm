package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	llm "github.com/agent-guide/caddy-agent-gateway/llm"
	configstoreintf "github.com/agent-guide/caddy-agent-gateway/configstore/intf"
	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
)

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

	cfg, err := provider.DecodeStoredProviderConfig(tag, obj)
	if err != nil {
		return nil, "", err
	}
	prov, err := provider.NewProvider(cfg)
	if err != nil {
		return nil, "", err
	}

	r.mu.Lock()
	r.cache[ref] = cachedProviderEntry{cfgJSON: fingerprint, provider: prov, name: cfg.Name}
	r.mu.Unlock()

	return prov, cfg.Name, nil
}

var globalAgentGateway = NewAgentGateway()

var globalProvisionState struct {
	mu            sync.Mutex
	configuredApp *llm.App
}

func GlobalAgentGateway() *AgentGateway {
	globalProvisionState.mu.Lock()
	defer globalProvisionState.mu.Unlock()
	if globalAgentGateway == nil || !globalAgentGateway.configured {
		panic(errors.New("Global AgentGateway is not availabled or configured"))
	}
	return globalAgentGateway
}

// SetGlobalRoutes updates the static route table on the global gateway without
// requiring it to be fully configured. This is safe to call during Caddyfile
// parsing, before ConfigureGlobalAgentGateway has been invoked.
func SetGlobalRoutes(routes []Route) {
	globalAgentGateway.SetRoutes(routes)
}

func ResetGlobalAgentGateway() *AgentGateway {
	globalProvisionState.mu.Lock()
	defer globalProvisionState.mu.Unlock()
	globalAgentGateway.Reset()
	globalProvisionState.configuredApp = nil
	return globalAgentGateway
}

// ConfigureGlobalAgentGateway configures the global gateway dependencies from
// the app. Repeated calls for the same app are ignored.
func ConfigureGlobalAgentGateway(app *llm.App) (*AgentGateway, error) {
	if app == nil {
		return nil, fmt.Errorf("app is required")
	}

	globalProvisionState.mu.Lock()
	defer globalProvisionState.mu.Unlock()

	if globalProvisionState.configuredApp == app {
		return globalAgentGateway, nil
	}

	routeLoader, providerResolver, localAPIKeyStore, err := buildGatewayDependencies(app)
	if err != nil {
		return nil, err
	}

	globalAgentGateway.Configure(routeLoader, providerResolver, localAPIKeyStore, app.AuthManager(), nil)
	globalProvisionState.configuredApp = app
	return globalAgentGateway, nil
}

func buildGatewayDependencies(app *llm.App) (RouteLoader, ProviderResolver, configstoreintf.LocalAPIKeyStorer, error) {
	staticResolver := NewStaticProviderResolver(func(name string) (provider.Provider, bool) {
		return app.Provider(name)
	})

	if app.ConfigStore() == nil {
		return nil, staticResolver, nil, nil
	}

	localAPIKeyStore, err := app.ConfigStore().GetLocalAPIKeyStore(context.Background(), DecodeLocalAPIKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get local api key store: %w", err)
	}

	var dynamicResolver ProviderResolver
	providerStore := app.ConfigStore().GetProviderConfigStore()
	if providerStore != nil {
		dynamicResolver = newCachedDynamicResolver(providerStore)
	}

	var routeLoader RouteLoader
	routeStore, err := app.ConfigStore().GetRouteStore(context.Background(), DecodeRoute)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get route store: %w", err)
	}
	routeLoader = func(ctx context.Context, routeID string) (*Route, error) {
		item, err := routeStore.Get(ctx, routeID)
		if err != nil {
			return nil, err
		}
		route, ok := item.(*Route)
		if !ok || route == nil {
			return nil, fmt.Errorf("route %q has unexpected type %T", routeID, item)
		}
		return route, nil
	}

	return routeLoader, ChainProviderResolvers(dynamicResolver, staticResolver), localAPIKeyStore, nil
}
