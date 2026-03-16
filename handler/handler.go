package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"

	"github.com/agent-guide/caddy-llm/handler/admin"
	"github.com/agent-guide/caddy-llm/handler/llmapi"
	llm "github.com/agent-guide/caddy-llm/llm"
	"github.com/agent-guide/caddy-llm/llm/provider"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler is the main HTTP handler for the LLM gateway.
// It routes requests to the appropriate sub-handler (llmapi, admin, auth).
type Handler struct {
	// DefaultProvider is the provider to use when none is specified.
	DefaultProvider string `json:"default_provider,omitempty"`

	// ConfigDSN is the SQLite DSN for config storage.
	ConfigDSN string `json:"config_dsn,omitempty"`

	logger       *zap.Logger
	adminHandler *admin.Handler
	llmapiRouter *llmapi.Router
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

	app, err := llm.GetApp(ctx)
	if err != nil {
		return fmt.Errorf("llm handler: get llm app: %w", err)
	}
	h.adminHandler = admin.NewHandler(app.AuthManager())

	// Create a shared OpenAI-compatible provider.
	// The static APIKey is overridden per-request by the context credential.
	provConfig := provider.ProviderConfig{
		Name:    "openai",
		APIKey:  "placeholder",
		BaseURL: "https://api.openai.com/v1",
	}
	provConfig.Network.Defaults()
	prov, err := provider.NewProvider(provConfig)
	if err != nil {
		return fmt.Errorf("llm handler: create openai provider: %w", err)
	}
	h.llmapiRouter = llmapi.NewRouter(app.AuthManager(), prov)
	return nil
}

// Validate validates the handler configuration.
func (h *Handler) Validate() error {
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	switch {
	case strings.HasPrefix(r.URL.Path, "/admin/"):
		h.adminHandler.ServeHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/v1/"):
		h.llmapiRouter.ServeHTTP(w, r)
	default:
		return next.ServeHTTP(w, r)
	}
	return nil
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

// Interface guards
var (
	_ caddy.Module                = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddy.Validator             = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)
