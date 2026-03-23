package api

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (h *LLMAPIHandler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "llm_api":
				args := d.RemainingArgs()
				if len(args) < 1 || len(args) > 2 {
					return d.ArgErr()
				}
				binding := Binding{API: args[0]}
				if len(args) == 2 {
					binding.Provider = args[1]
				}
				h.Bindings = append(h.Bindings, binding)
			default:
				return d.Errf("unknown directive: %s", d.Val())
			}
		}
	}
	return nil
}

func parseHandleLLMAPI(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var handler LLMAPIHandler
	if err := handler.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return &handler, nil
}
