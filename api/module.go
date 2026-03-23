package api

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"

	"github.com/agent-guide/caddy-llm/api/llmapi"
	llm "github.com/agent-guide/caddy-llm/llm"
	"github.com/agent-guide/caddy-llm/llm/provider"
)

func init() {
	caddy.RegisterModule(LLMAPIHandler{})
	httpcaddyfile.RegisterHandlerDirective("handle_llm_api", parseHandleLLMAPI)
}

// LLMAPIHandler exposes one or more compatible LLM APIs under the HTTP app.
type LLMAPIHandler struct {
	LLMAPIs []string `json:"llm_apis,omitempty"`

	logger *zap.Logger
	router *llmapi.Router
}

// CaddyModule returns the Caddy module information.
func (LLMAPIHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.llm_api",
		New: func() caddy.Module { return new(LLMAPIHandler) },
	}
}

// Provision sets up the handler.
func (h *LLMAPIHandler) Provision(ctx caddy.Context) error {
	h.logger = ctx.Logger(h)

	app, err := llm.GetApp(ctx)
	if err != nil {
		return fmt.Errorf("llm_api: get llm app: %w", err)
	}

	apis := h.LLMAPIs
	if len(apis) == 0 {
		apis = []string{"openai"}
	}

	handlers := make([]llmapi.LLMApiHandler, 0, len(apis))
	for _, name := range apis {
		// Create the correct provider type for each API format.
		provConfig := provider.ProviderConfig{
			Name:   name,
			APIKey: "placeholder",
		}
		provConfig.Network.Defaults()
		prov, err := provider.NewProvider(provConfig)
		if err != nil {
			return fmt.Errorf("handle_llm_api: create %s provider: %w", name, err)
		}

		moduleID := "http.handlers.llm_api." + name
		info, err := caddy.GetModule(moduleID)
		if err != nil {
			return fmt.Errorf("handle_llm_api: load %s: %w", moduleID, err)
		}
		mod := info.New()
		api, ok := mod.(llmapi.LLMApiHandler)
		if !ok {
			return fmt.Errorf("handle_llm_api: %s does not implement llmapi.LLMApiHandler", moduleID)
		}
		if err := api.ProvisionLLMApi(app.AuthManager(), prov, h.logger); err != nil {
			return fmt.Errorf("handle_llm_api: provision %s: %w", moduleID, err)
		}
		handlers = append(handlers, api)
	}

	h.router = llmapi.NewRouter(handlers, h.logger)
	return nil
}

// Validate validates the handler configuration.
func (h *LLMAPIHandler) Validate() error {
	seen := make(map[string]struct{}, len(h.LLMAPIs))
	for _, name := range h.LLMAPIs {
		if name == "" {
			return fmt.Errorf("llm_api cannot be empty")
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate llm_api: %s", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (h LLMAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if h.router == nil {
		return next.ServeHTTP(w, r)
	}
	h.router.ServeHTTP(w, r)
	return nil
}

var (
	_ caddy.Module                = (*LLMAPIHandler)(nil)
	_ caddy.Provisioner           = (*LLMAPIHandler)(nil)
	_ caddy.Validator             = (*LLMAPIHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*LLMAPIHandler)(nil)
	_ caddyfile.Unmarshaler       = (*LLMAPIHandler)(nil)
)
