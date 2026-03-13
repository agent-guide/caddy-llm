package handler

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

// Handler is the main HTTP handler for the LLM gateway.
// It routes requests to the appropriate sub-handler (llmapi, admin, auth).
type Handler struct {
	// DefaultProvider is the provider to use when none is specified.
	DefaultProvider string `json:"default_provider,omitempty"`

	// ConfigDSN is the SQLite DSN for config storage.
	ConfigDSN string `json:"config_dsn,omitempty"`

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
// Note: directory is "handler/" but module ID is "http.handlers.llm".
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.llm",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up the handler.
func (h *Handler) Provision(ctx caddy.Context) error {
	h.logger = ctx.Logger(h)
	return nil
}

// Validate validates the handler configuration.
func (h *Handler) Validate() error {
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// TODO: route to llmapi / admin / auth sub-handlers
	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "default_provider":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.DefaultProvider = d.Val()
			case "config_dsn":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.ConfigDSN = d.Val()
			default:
				return d.Errf("unknown directive: %s", d.Val())
			}
		}
	}
	return nil
}
