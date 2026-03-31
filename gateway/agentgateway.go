package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/manager"
	configstoreintf "github.com/agent-guide/caddy-agent-gateway/configstore/intf"
	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
)

type AgentGateway struct {
	mu sync.RWMutex

	routes     map[string]Route
	configured bool

	RouteLoader      RouteLoader
	ProviderResolver ProviderResolver
	LocalAPIKeyStore configstoreintf.LocalAPIKeyStorer
	AuthManager      *manager.Manager
	Selector         RouteSelector
}

func NewAgentGateway() *AgentGateway {
	return &AgentGateway{
		routes:     map[string]Route{},
		configured: false,
	}
}

func (g *AgentGateway) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.routes = map[string]Route{}
	g.configured = false
	g.RouteLoader = nil
	g.ProviderResolver = nil
	g.LocalAPIKeyStore = nil
	g.AuthManager = nil
	g.Selector = nil
}

func (g *AgentGateway) Configure(routeLoader RouteLoader, providerResolver ProviderResolver, localAPIKeyStore configstoreintf.LocalAPIKeyStorer, authMgr *manager.Manager, selector RouteSelector) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.RouteLoader = routeLoader
	g.ProviderResolver = providerResolver
	g.LocalAPIKeyStore = localAPIKeyStore
	g.AuthManager = authMgr
	g.Selector = selector
	g.configured = true
}

func (g *AgentGateway) SetRoutes(routes []Route) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.routes = make(map[string]Route, len(routes))
	for _, route := range routes {
		if route.ID == "" {
			continue
		}
		route.Policy.Defaults()
		g.routes[route.ID] = route
	}
}

func (g *AgentGateway) EnsureRoute(route Route) {
	if route.ID == "" {
		return
	}

	route.Policy.Defaults()

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.routes == nil {
		g.routes = map[string]Route{}
	}
	g.routes[route.ID] = route
}

func (g *AgentGateway) Routes() []Route {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out := make([]Route, 0, len(g.routes))
	for _, route := range g.routes {
		out = append(out, route)
	}
	return out
}

func (g *AgentGateway) Route(routeID string) (Route, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	route, ok := g.routes[routeID]
	return route, ok
}

func (g *AgentGateway) ValidateRoute(ctx context.Context, routeID string) error {
	if routeID == "" {
		return fmt.Errorf("route_id is required")
	}

	route, ok := g.Route(routeID)
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
		route = *loaded
	}

	resolver := g.providerResolver()
	if resolver == nil {
		return fmt.Errorf("provider resolver is not configured")
	}

	for _, target := range route.Targets {
		if _, _, err := resolver.ResolveProvider(ctx, target.ProviderRef); err != nil {
			return fmt.Errorf("provider %q is not configured", target.ProviderRef)
		}
	}
	return nil
}

func (g *AgentGateway) ResolveProvider(ctx context.Context, routeID string, req ResolveRequest) (*ResolvedRoute, error) {
	route, err := g.resolveRoute(ctx, routeID)
	if err != nil {
		return nil, err
	}

	localKey, err := g.resolveLocalAPIKey(ctx, req.HTTPRequest, route)
	if err != nil {
		return nil, err
	}
	if err := validateRequestPolicy(route, localKey, req); err != nil {
		return nil, err
	}

	selector := g.selector()
	target, err := selector.SelectTarget(route, req)
	if err != nil {
		return nil, err
	}

	resolver := g.providerResolver()
	if resolver == nil {
		return nil, &HTTPError{status: http.StatusServiceUnavailable, msg: "provider resolver is not configured"}
	}
	prov, providerName, err := resolver.ResolveProvider(ctx, target.ProviderRef)
	if err != nil || prov == nil {
		return nil, &HTTPError{status: http.StatusBadGateway, msg: fmt.Sprintf("route target provider %q is not configured", target.ProviderRef)}
	}
	if providerName == "" {
		providerName = target.ProviderRef
	}
	prov = g.wrapProvider(prov, providerName)

	return &ResolvedRoute{
		Route:        route,
		LocalAPIKey:  localKey,
		ProviderName: providerName,
		Provider:     prov,
	}, nil
}

func (g *AgentGateway) resolveRoute(ctx context.Context, routeID string) (Route, error) {
	if routeID == "" {
		return Route{}, &HTTPError{status: http.StatusServiceUnavailable, msg: "route id is not configured"}
	}

	g.mu.RLock()
	route, ok := g.routes[routeID]
	loader := g.RouteLoader
	g.mu.RUnlock()

	if loader != nil {
		latest, err := loader(ctx, routeID)
		if err != nil {
			return Route{}, &HTTPError{status: http.StatusServiceUnavailable, msg: fmt.Sprintf("route %q is unavailable", routeID)}
		}
		if latest != nil {
			latest.Policy.Defaults()
			g.EnsureRoute(*latest)
			return *latest, nil
		}
	}

	if !ok {
		return Route{}, &HTTPError{status: http.StatusServiceUnavailable, msg: fmt.Sprintf("route %q is not configured", routeID)}
	}
	route.Policy.Defaults()
	return route, nil
}

func (g *AgentGateway) resolveLocalAPIKey(ctx context.Context, httpReq *http.Request, route Route) (*LocalAPIKey, error) {
	rawKey := extractAPIKey(httpReq)
	if rawKey == "" {
		if route.Policy.Auth.RequireLocalAPIKey {
			return nil, &HTTPError{status: http.StatusUnauthorized, msg: "local api key is required"}
		}
		return nil, nil
	}

	g.mu.RLock()
	store := g.LocalAPIKeyStore
	g.mu.RUnlock()
	if store == nil {
		return nil, &HTTPError{status: http.StatusServiceUnavailable, msg: "local api key store is not configured"}
	}

	item, err := store.Get(ctx, rawKey)
	if err != nil {
		return nil, &HTTPError{status: http.StatusUnauthorized, msg: "invalid local api key"}
	}

	key, ok := item.(*LocalAPIKey)
	if !ok || key == nil {
		return nil, &HTTPError{status: http.StatusUnauthorized, msg: "invalid local api key"}
	}
	return ValidateLocalAPIKeyForRoute(route, key)
}

func (g *AgentGateway) providerResolver() ProviderResolver {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.ProviderResolver
}

func (g *AgentGateway) selector() RouteSelector {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selector == nil {
		return DefaultRouteSelector{}
	}
	return g.Selector
}

func (g *AgentGateway) wrapProvider(prov provider.Provider, providerName string) provider.Provider {
	g.mu.RLock()
	authMgr := g.AuthManager
	g.mu.RUnlock()
	return provider.WrapWithAuthManager(prov, providerName, authMgr)
}
