package api

import (
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("handle_llm_api", parseHandleLLMAPI)
}

// ParseHandleLLMAPIForTest exposes the parser to external tests.
func ParseHandleLLMAPIForTest(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	return parseHandleLLMAPI(h)
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
