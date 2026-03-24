package api

import (
	"fmt"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("handle_llm_api", parseHandleLLMAPI)
}

func parseHandleLLMAPI(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	d := h.Dispenser
	if !d.Next() {
		return nil, d.Err("expected directive name")
	}
	if !d.NextArg() {
		return nil, d.Err("expected llm api name")
	}
	apiName := d.Val()
	if d.NextArg() {
		return nil, d.ArgErr()
	}

	moduleID := "http.handlers.llm_api." + apiName
	unm, err := caddyfile.UnmarshalModule(d, moduleID)
	if err != nil {
		return nil, err
	}

	handler, ok := unm.(caddyhttp.MiddlewareHandler)
	if !ok {
		return nil, fmt.Errorf("%s does not implement caddyhttp.MiddlewareHandler", moduleID)
	}
	return handler, nil
}
