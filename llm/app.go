package llm

import (
	"context"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"go.uber.org/zap"

	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	configstoreIntf "github.com/agent-guide/caddy-llm/llm/configstore/intf"
	configstoresqlite "github.com/agent-guide/caddy-llm/llm/configstore/sqlite"
	"github.com/agent-guide/caddy-llm/llm/provider"
)

func init() {
	caddy.RegisterModule(App{})
}

// App is the Caddy app module for the LLM gateway.
// It manages providers, MCP clients, memory stores, and configuration.
type App struct {
	// Providers lists the configured LLM providers.
	ProvidersRaw caddy.ModuleMap `json:"providers,omitempty" caddy:"namespace=llm.providers"`
	// Authenticators configures CLI credential authenticators under the llm.authenticators namespace.
	AuthenticatorsRaw caddy.ModuleMap `json:"authenticators,omitempty" caddy:"namespace=llm.authenticators"`
	// ConfigStore configures persistent admin/auth state storage.
	ConfigStoreRaw caddy.ModuleMap `json:"config_store,omitempty" caddy:"namespace=llm.config_stores"`

	logger       *zap.Logger
	authManager  *manager.Manager
	configStorer configstoreIntf.ConfigStorer
	providers    map[string]provider.Provider
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

	if err := a.provisionConfigStore(ctx); err != nil {
		return fmt.Errorf("init config store: %w", err)
	}
	credentialStore := a.configStorer.GetCredentialStore()

	a.authManager = manager.NewManager(credentialStore, nil, nil)
	if err := a.provisionProviders(ctx); err != nil {
		return fmt.Errorf("provision providers: %w", err)
	}
	if err := a.provisionAuthenticators(ctx); err != nil {
		return fmt.Errorf("provision authenticators: %w", err)
	}
	if err := a.authManager.Load(ctx); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	a.logger.Info("LLM Gateway provisioned")
	return nil
}

// AuthManager returns the credential manager shared across the gateway.
func (a *App) AuthManager() *manager.Manager {
	return a.authManager
}

func (a *App) ConfigStore() configstoreIntf.ConfigStorer {
	return a.configStorer
}

// Provider returns a configured provider by name.
func (a *App) Provider(name string) (provider.Provider, bool) {
	if a.providers == nil {
		return nil, false
	}
	prov, ok := a.providers[name]
	return prov, ok
}

// Validate validates the app configuration.
func (a *App) Validate() error {
	return nil
}

// Start starts the app.
func (a *App) Start() error {
	a.authManager.StartRefreshLoop(context.Background())
	a.logger.Info("LLM Gateway started")
	return nil
}

// Stop stops the app.
func (a *App) Stop() error {
	a.authManager.StopRefreshLoop()
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

func (a *App) provisionConfigStore(ctx caddy.Context) error {
	if len(a.ConfigStoreRaw) == 0 {
		a.ConfigStoreRaw = caddy.ModuleMap{
			"sqlite": caddyconfig.JSON(&configstoresqlite.SQLiteConfigStore{}, nil),
		}
	}

	modules, err := ctx.LoadModule(a, "ConfigStoreRaw")
	if err != nil {
		return err
	}

	loaded, ok := modules.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected config store module type %T", modules)
	}
	if len(loaded) != 1 {
		return fmt.Errorf("expected exactly one config store module, got %d", len(loaded))
	}

	for name, mod := range loaded {
		storer, ok := mod.(configstoreIntf.ConfigStorer)
		if !ok {
			return fmt.Errorf("config store module %q does not implement configstore.ConfigStorer", name)
		}
		a.configStorer = storer
		return nil
	}

	return fmt.Errorf("no config store module loaded")
}

func (a *App) provisionAuthenticators(ctx caddy.Context) error {
	if len(a.AuthenticatorsRaw) == 0 {
		return nil
	}

	modules, err := ctx.LoadModule(a, "AuthenticatorsRaw")
	if err != nil {
		return err
	}

	loaded, ok := modules.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected authenticator module type %T", modules)
	}
	return a.registerLoadedAuthenticators(loaded)
}

func (a *App) provisionProviders(ctx caddy.Context) error {
	if len(a.ProvidersRaw) == 0 {
		a.providers = map[string]provider.Provider{}
		return nil
	}

	modules, err := ctx.LoadModule(a, "ProvidersRaw")
	if err != nil {
		return err
	}

	loaded, ok := modules.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected provider module type %T", modules)
	}

	a.providers = make(map[string]provider.Provider, len(loaded))
	for name, mod := range loaded {
		prov, ok := mod.(provider.Provider)
		if !ok {
			return fmt.Errorf("provider module %q does not implement provider.Provider", name)
		}
		a.providers[name] = prov
	}
	return nil
}

func (a *App) registerLoadedAuthenticators(loaded map[string]any) error {
	for name, mod := range loaded {
		auth, ok := mod.(manager.Authenticator)
		if !ok {
			return fmt.Errorf("authenticator module %q does not implement manager.Authenticator", name)
		}
		a.authManager.RegisterAuthenticator(name, auth)
	}
	return nil
}

// Interface guards
var (
	_ caddy.App         = (*App)(nil)
	_ caddy.Provisioner = (*App)(nil)
	_ caddy.Validator   = (*App)(nil)
)
