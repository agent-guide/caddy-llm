package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	configstoreintf "github.com/agent-guide/caddy-agent-gateway/configstore/intf"
	routepkg "github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"github.com/agent-guide/caddy-agent-gateway/llm/cliauth/manager"
	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
)

type AgentGateway struct {
	mu sync.RWMutex

	routes     map[string]routepkg.Route
	configured bool

	RouteLoader      routepkg.RouteLoader
	ProviderResolver ProviderResolver
	LocalAPIKeyStore configstoreintf.LocalAPIKeyStorer
	cliauthManager   *manager.Manager
	Selector         routepkg.RouteSelector
}

func NewAgentGateway() *AgentGateway {
	return &AgentGateway{
		routes:     map[string]routepkg.Route{},
		configured: false,
	}
}

func (g *AgentGateway) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.routes = map[string]routepkg.Route{}
	g.configured = false
	g.RouteLoader = nil
	g.ProviderResolver = nil
	g.LocalAPIKeyStore = nil
	g.cliauthManager = nil
	g.Selector = nil
}

func (g *AgentGateway) Configure(routeLoader routepkg.RouteLoader, providerResolver ProviderResolver, localAPIKeyStore configstoreintf.LocalAPIKeyStorer, cliauthMgr *manager.Manager, selector routepkg.RouteSelector) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.RouteLoader = routeLoader
	g.ProviderResolver = providerResolver
	g.LocalAPIKeyStore = localAPIKeyStore
	g.cliauthManager = cliauthMgr
	g.Selector = selector
	g.configured = true
}

func (g *AgentGateway) SetRoutes(routes []routepkg.Route) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.routes = make(map[string]routepkg.Route, len(routes))
	for _, r := range routes {
		if r.ID == "" {
			continue
		}
		r.Policy.Defaults()
		g.routes[r.ID] = r
	}
}

func (g *AgentGateway) EnsureRoute(r routepkg.Route) {
	if r.ID == "" {
		return
	}

	r.Policy.Defaults()

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.routes == nil {
		g.routes = map[string]routepkg.Route{}
	}
	g.routes[r.ID] = r
}

func (g *AgentGateway) Routes() []routepkg.Route {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out := make([]routepkg.Route, 0, len(g.routes))
	for _, r := range g.routes {
		out = append(out, r)
	}
	return out
}

func (g *AgentGateway) Route(routeID string) (routepkg.Route, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r, ok := g.routes[routeID]
	return r, ok
}

func (g *AgentGateway) ValidateRoute(ctx context.Context, routeID string) error {
	if routeID == "" {
		return fmt.Errorf("route_id is required")
	}

	r, ok := g.Route(routeID)
	if !ok {
		g.mu.RLock()
		routeLoader := g.RouteLoader
		g.mu.RUnlock()
		if routeLoader == nil {
			return fmt.Errorf("route %q is not configured", routeID)
		}
		loaded, err := routeLoader(ctx, routeID)
		if err != nil || loaded == nil {
			return fmt.Errorf("route %q is not configured", routeID)
		}
		r = *loaded
	}

	resolver := g.providerResolver()
	if resolver == nil {
		return fmt.Errorf("provider resolver is not configured")
	}

	for _, target := range r.Targets {
		if _, _, err := resolver.ResolveProvider(ctx, target.ProviderRef); err != nil {
			return fmt.Errorf("provider %q is not configured", target.ProviderRef)
		}
	}
	return nil
}

func (g *AgentGateway) ResolveProvider(ctx context.Context, routeID string, req routepkg.ResolveRequest) (*routepkg.ResolvedRoute, error) {
	r, err := g.resolveRoute(ctx, routeID)
	if err != nil {
		return nil, err
	}

	localKey, err := g.resolveLocalAPIKey(ctx, req.HTTPRequest, r)
	if err != nil {
		return nil, err
	}
	if err := routepkg.ValidateRequestPolicy(r, localKey, req); err != nil {
		return nil, err
	}

	selector := g.selector()
	target, err := selector.SelectTarget(r, req)
	if err != nil {
		return nil, err
	}

	resolver := g.providerResolver()
	if resolver == nil {
		return nil, routepkg.NewHTTPError(http.StatusServiceUnavailable, "provider resolver is not configured")
	}
	prov, providerName, err := resolver.ResolveProvider(ctx, target.ProviderRef)
	if err != nil || prov == nil {
		return nil, routepkg.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("route target provider %q is not configured", target.ProviderRef))
	}
	if providerName == "" {
		providerName = target.ProviderRef
	}
	prov = g.wrapProvider(prov, providerName)

	return &routepkg.ResolvedRoute{
		Route:        r,
		LocalAPIKey:  localKey,
		ProviderName: providerName,
		Provider:     prov,
	}, nil
}

func (g *AgentGateway) resolveRoute(ctx context.Context, routeID string) (routepkg.Route, error) {
	if routeID == "" {
		return routepkg.Route{}, routepkg.NewHTTPError(http.StatusServiceUnavailable, "route id is not configured")
	}

	g.mu.RLock()
	r, ok := g.routes[routeID]
	loader := g.RouteLoader
	g.mu.RUnlock()

	if loader != nil {
		latest, err := loader(ctx, routeID)
		if err != nil {
			return routepkg.Route{}, routepkg.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("route %q is unavailable", routeID))
		}
		if latest != nil {
			latest.Policy.Defaults()
			g.EnsureRoute(*latest)
			return *latest, nil
		}
	}

	if !ok {
		return routepkg.Route{}, routepkg.NewHTTPError(http.StatusServiceUnavailable, fmt.Sprintf("route %q is not configured", routeID))
	}
	r.Policy.Defaults()
	return r, nil
}

func (g *AgentGateway) resolveLocalAPIKey(ctx context.Context, httpReq *http.Request, r routepkg.Route) (*routepkg.LocalAPIKey, error) {
	rawKey := routepkg.ExtractAPIKey(httpReq)
	if rawKey == "" {
		if r.Policy.Auth.RequireLocalAPIKey {
			return nil, routepkg.NewHTTPError(http.StatusUnauthorized, "local api key is required")
		}
		return nil, nil
	}

	g.mu.RLock()
	store := g.LocalAPIKeyStore
	g.mu.RUnlock()
	if store == nil {
		return nil, routepkg.NewHTTPError(http.StatusServiceUnavailable, "local api key store is not configured")
	}

	item, err := store.Get(ctx, rawKey)
	if err != nil {
		return nil, routepkg.NewHTTPError(http.StatusUnauthorized, "invalid local api key")
	}

	key, ok := item.(*routepkg.LocalAPIKey)
	if !ok || key == nil {
		return nil, routepkg.NewHTTPError(http.StatusUnauthorized, "invalid local api key")
	}
	return routepkg.ValidateLocalAPIKeyForRoute(r, key)
}

func (g *AgentGateway) providerResolver() ProviderResolver {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.ProviderResolver
}

func (g *AgentGateway) selector() routepkg.RouteSelector {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selector == nil {
		return routepkg.DefaultRouteSelector{}
	}
	return g.Selector
}

func (g *AgentGateway) wrapProvider(prov provider.Provider, providerName string) provider.Provider {
	g.mu.RLock()
	cliauthMgr := g.cliauthManager
	g.mu.RUnlock()
	return provider.WrapWithAuthManager(prov, providerName, cliauthMgr)
}
