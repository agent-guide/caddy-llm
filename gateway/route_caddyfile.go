package gateway

import (
	"strconv"
	"strings"

	routepkg "github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// ParseRouteSegment parses a route declaration from the current directive or subdirective.
// The dispenser must already be positioned on the route directive token.
func ParseRouteSegment(d *caddyfile.Dispenser) (routepkg.Route, error) {
	seg := d.NewFromNextSegment()
	if !seg.Next() {
		return routepkg.Route{}, d.Err("expected route directive")
	}

	args := seg.RemainingArgsRaw()
	if len(args) != 1 {
		return routepkg.Route{}, seg.ArgErr()
	}

	routeID := strings.Trim(args[0], "\"`")
	route := routepkg.Route{
		ID:   routeID,
		Name: routeID,
		Policy: routepkg.RoutePolicy{
			Selection: routepkg.SelectionPolicy{Strategy: routepkg.RouteSelectionStrategyAuto},
		},
	}

	for seg.NextBlock(0) {
		name := seg.Val()
		args := seg.RemainingArgsRaw()
		switch name {
		case "route_name":
			if len(args) != 1 {
				return routepkg.Route{}, seg.ArgErr()
			}
			route.Name = strings.Trim(args[0], "\"`")
		case "require_local_api_key":
			if len(args) == 0 {
				route.Policy.Auth.RequireLocalAPIKey = true
				continue
			}
			if len(args) != 1 {
				return routepkg.Route{}, seg.ArgErr()
			}
			v, err := strconv.ParseBool(strings.Trim(args[0], "\"`"))
			if err != nil {
				return routepkg.Route{}, seg.Errf("invalid require_local_api_key value: %s", args[0])
			}
			route.Policy.Auth.RequireLocalAPIKey = v
		case "allowed_model":
			if len(args) == 0 {
				return routepkg.Route{}, seg.ArgErr()
			}
			for _, arg := range args {
				route.Policy.AllowedModels = append(route.Policy.AllowedModels, strings.Trim(arg, "\"`"))
			}
		case "target":
			if len(args) == 0 || len(args) > 2 {
				return routepkg.Route{}, seg.ArgErr()
			}
			target := routepkg.RouteTarget{
				ProviderRef: strings.Trim(args[0], "\"`"),
				Mode:        routepkg.TargetModeWeighted,
				Weight:      1,
			}
			if len(args) == 2 {
				weight, err := strconv.Atoi(strings.Trim(args[1], "\"`"))
				if err != nil {
					return routepkg.Route{}, seg.Errf("invalid target weight: %s", args[1])
				}
				target.Weight = weight
			}
			route.Targets = append(route.Targets, target)
		default:
			return routepkg.Route{}, seg.Errf("unknown subdirective: %s", name)
		}
	}

	route.Policy.Defaults()
	return route, nil
}
