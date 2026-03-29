package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/agent-guide/caddy-llm/gateway"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

const caddyfileStateKey = "agent-guide/caddy-llm/api"

func init() {
	httpcaddyfile.RegisterHandlerDirective("handle_llm_api", parseHandleLLMAPI)
	httpcaddyfile.RegisterDirective("agent_gateway_route", parseAgentGatewayRoute)
}

type caddyfileState struct {
	declaredRoutes []*gateway.Route
}

// ParseHandleLLMAPIForTest exposes the parser to external tests.
func ParseHandleLLMAPIForTest(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	return parseHandleLLMAPI(h)
}

// ParseAgentGatewayRouteForTest exposes the route parser to external tests.
func ParseAgentGatewayRouteForTest(h httpcaddyfile.Helper) error {
	_, err := parseAgentGatewayRoute(h)
	return err
}

func parseHandleLLMAPI(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	d := h.Dispenser
	if !d.Next() {
		return nil, d.Err("expected directive name")
	}
	if !d.NextArg() {
		return nil, d.Err("expected llm api name")
	}
	apiName := strings.Trim(d.Val(), "\"`")
	if d.NextArg() {
		return nil, d.ArgErr()
	}

	var routeID string
	for d.NextBlock(0) {
		switch d.Val() {
		case "route_id":
			args := d.RemainingArgsRaw()
			if len(args) != 1 {
				return nil, d.ArgErr()
			}
			routeID = strings.Trim(args[0], "\"`")
		default:
			return nil, d.Errf("unknown subdirective: %s", d.Val())
		}
	}
	if routeID == "" {
		return nil, d.Err("route_id is required")
	}

	moduleID := "http.handlers.llm_api." + apiName
	mod, err := caddy.GetModule(moduleID)
	if err != nil {
		return nil, d.Errf("getting module named '%s': %v", moduleID, err)
	}
	inst := mod.New()

	if acceptor, ok := inst.(RouteIDAcceptor); ok {
		acceptor.SetRouteID(routeID)
	}

	handler, ok := inst.(caddyhttp.MiddlewareHandler)
	if !ok {
		return nil, fmt.Errorf("%s does not implement caddyhttp.MiddlewareHandler", moduleID)
	}
	return handler, nil
}

func parseAgentGatewayRoute(h httpcaddyfile.Helper) ([]httpcaddyfile.ConfigValue, error) {
	d := h.Dispenser
	if !d.Next() {
		return nil, d.Err("expected directive name")
	}
	if !d.NextArg() {
		return nil, d.Err("expected route id")
	}

	route := gateway.Route{
		ID:   strings.Trim(d.Val(), "\"`"),
		Name: strings.Trim(d.Val(), "\"`"),
		Policy: gateway.RoutePolicy{
			Selection: gateway.SelectionPolicy{Strategy: gateway.RouteSelectionStrategyAuto},
		},
	}
	if d.NextArg() {
		return nil, d.ArgErr()
	}

	for d.NextBlock(0) {
		name := d.Val()
		args := d.RemainingArgsRaw()
		switch name {
		case "route_name":
			if len(args) != 1 {
				return nil, d.ArgErr()
			}
			route.Name = strings.Trim(args[0], "\"`")
		case "require_local_api_key":
			if len(args) == 0 {
				route.Policy.Auth.RequireLocalAPIKey = true
				continue
			}
			if len(args) != 1 {
				return nil, d.ArgErr()
			}
			v, err := strconv.ParseBool(strings.Trim(args[0], "\"`"))
			if err != nil {
				return nil, d.Errf("invalid require_local_api_key value: %s", args[0])
			}
			route.Policy.Auth.RequireLocalAPIKey = v
		case "allowed_model":
			if len(args) == 0 {
				return nil, d.ArgErr()
			}
			for _, arg := range args {
				route.Policy.AllowedModels = append(route.Policy.AllowedModels, strings.Trim(arg, "\"`"))
			}
		case "target":
			if len(args) == 0 || len(args) > 2 {
				return nil, d.ArgErr()
			}
			target := gateway.RouteTarget{
				ProviderRef: strings.Trim(args[0], "\"`"),
				Mode:        gateway.TargetModeWeighted,
				Weight:      1,
			}
			if len(args) == 2 {
				weight, err := strconv.Atoi(strings.Trim(args[1], "\"`"))
				if err != nil {
					return nil, d.Errf("invalid target weight: %s", args[1])
				}
				target.Weight = weight
			}
			route.Targets = append(route.Targets, target)
		default:
			return nil, d.Errf("unknown subdirective: %s", name)
		}
	}

	route.Policy.Defaults()
	if err := registerDeclaredRoute(h, route); err != nil {
		return nil, err
	}
	return nil, nil
}

func registerDeclaredRoute(h httpcaddyfile.Helper, route gateway.Route) error {
	state := getCaddyfileState(h)
	for _, declared := range state.declaredRoutes {
		if declared.ID == route.ID {
			return fmt.Errorf("%s:%d: duplicate agent_gateway_route %q", h.File(), h.Line(), route.ID)
		}
	}
	state.declaredRoutes = append(state.declaredRoutes, &route)
	syncGlobalRoutes(state)
	return nil
}

func getCaddyfileState(h httpcaddyfile.Helper) *caddyfileState {
	if h.State == nil {
		h.State = make(map[string]any)
	}
	if state, ok := h.State[caddyfileStateKey].(*caddyfileState); ok && state != nil {
		return state
	}
	state := &caddyfileState{}
	h.State[caddyfileStateKey] = state
	return state
}

func syncGlobalRoutes(state *caddyfileState) {
	routes := make([]gateway.Route, 0, len(state.declaredRoutes))
	for _, declared := range state.declaredRoutes {
		routes = append(routes, *declared)
	}
	gateway.SetGlobalRoutes(routes)
}
