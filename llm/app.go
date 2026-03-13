package llm

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(App{})
}

// App is the Caddy app module for the LLM gateway.
// It manages providers, MCP clients, memory stores, and configuration.
type App struct {
	// Providers lists the configured LLM providers.
	Providers []caddy.ModuleMap `json:"providers,omitempty" caddy:"namespace=llm.providers"`

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "llm",
		New: func() caddy.Module { return new(App) },
	}
}

// Provision sets up the app.
func (a *App) Provision(ctx caddy.Context) error {
	a.logger = ctx.Logger(a)
	a.logger.Info("LLM Gateway provisioned")
	return nil
}

// Validate validates the app configuration.
func (a *App) Validate() error {
	return nil
}

// Start starts the app.
func (a *App) Start() error {
	a.logger.Info("LLM Gateway started")
	return nil
}

// Stop stops the app.
func (a *App) Stop() error {
	return nil
}

// GetApp retrieves the LLM app from the Caddy context.
func GetApp(ctx caddy.Context) (*App, error) {
	appIface, err := ctx.App("llm")
	if err != nil {
		return nil, err
	}
	app, ok := appIface.(*App)
	if !ok {
		return nil, fmt.Errorf("llm app is not *llm.App")
	}
	return app, nil
}

// Interface guards
var (
	_ caddy.App        = (*App)(nil)
	_ caddy.Provisioner = (*App)(nil)
	_ caddy.Validator  = (*App)(nil)
)
