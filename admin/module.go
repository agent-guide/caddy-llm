package admin

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"github.com/agent-guide/caddy-agent-gateway/gateway"
)

func init() {
	caddy.RegisterModule(LLMAdminHandler{})
	httpcaddyfile.RegisterHandlerDirective("handle_llm_admin", parseHandleLLMAdmin)
}

// LLMAdminHandler is the Caddy HTTP middleware for the LLM Admin API.
type LLMAdminHandler struct {
	handler           *Handler
	AdminUsername     string `json:"admin_username,omitempty"`
	AdminPasswordHash string `json:"admin_password_hash,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (LLMAdminHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.llm_admin",
		New: func() caddy.Module { return new(LLMAdminHandler) },
	}
}

// Provision sets up the handler.
func (h *LLMAdminHandler) Provision(ctx caddy.Context) error {
	app, err := gateway.GetApp(ctx)
	if err != nil {
		return fmt.Errorf("handle_llm_admin: get agent_gateway app: %w", err)
	}
	h.handler = NewHandler(app.AuthManager(), app.ConfigStore(), ctx.Logger(h), h.AdminUsername, h.AdminPasswordHash)
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (h LLMAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	h.handler.ServeHTTP(w, r)
	return nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
//
//	handle_llm_admin {
//	    admin_user     <username>
//	    admin_password_hash  <bcrypt-hash>
//	}
func (h *LLMAdminHandler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "admin_user":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.AdminUsername = d.Val()
			case "admin_password_hash":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.AdminPasswordHash = d.Val()
			default:
				return d.Errf("unrecognized handle_llm_admin option: %s", d.Val())
			}
		}
	}
	return nil
}

func parseHandleLLMAdmin(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var handler LLMAdminHandler
	if err := handler.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return &handler, nil
}

var (
	_ caddy.Module                = (*LLMAdminHandler)(nil)
	_ caddy.Provisioner           = (*LLMAdminHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*LLMAdminHandler)(nil)
	_ caddyfile.Unmarshaler       = (*LLMAdminHandler)(nil)
)
