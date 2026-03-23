package llm

import (
	"fmt"
	"path/filepath"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/agent-guide/caddy-llm/llm/configstore"
	configstoreIntf "github.com/agent-guide/caddy-llm/llm/configstore/intf"
	configstoresqlite "github.com/agent-guide/caddy-llm/llm/configstore/sqlite"
)

func init() {
	caddy.RegisterModule(App{})
}

// App is the Caddy app module for the LLM gateway.
// It manages providers, MCP clients, memory stores, and configuration.
type App struct {
	// Providers lists the configured LLM providers.
	Providers []caddy.ModuleMap `json:"providers,omitempty" caddy:"namespace=llm.providers"`
	// Authenticators configures CLI credential authenticators under the llm.authenticators namespace.
	AuthenticatorsRaw caddy.ModuleMap `json:"authenticators,omitempty" caddy:"namespace=llm.authenticators"`
	// ConfigStore configures persistent admin/auth state storage.
	ConfigStoreCfg *ConfigStoreConfig `json:"config_store,omitempty"`

	logger       *zap.Logger
	authManager  *manager.Manager
	configStorer configstoreIntf.ConfigStorer
}

type ConfigStoreConfig struct {
	Type   string                                     `json:"type,omitempty"`
	SQLite *configstoresqlite.SQLiteConfigStoreConfig `json:"sqlite,omitempty"`
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

	storeCfg := a.effectiveConfigStoreConfig()
	storer, err := configstore.CreateConfigStore(ctx, a.logger, storeCfg.Type, storeCfg.sqliteConfig())
	if err != nil {
		return fmt.Errorf("init config store: %w", err)
	}
	a.configStorer = storer
	credentialStore := a.configStorer.GetCredentialStore()

	a.authManager = manager.NewManager(credentialStore, nil, nil)
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

func (a *App) effectiveConfigStoreConfig() *ConfigStoreConfig {
	if a.ConfigStoreCfg == nil {
		return &ConfigStoreConfig{
			Type: "sqlite",
			SQLite: &configstoresqlite.SQLiteConfigStoreConfig{
				SQLitePath: filepath.Join(caddy.AppDataDir(), "caddy-llm", "configstore.db"),
			},
		}
	}

	cfg := *a.ConfigStoreCfg
	if cfg.Type == "" {
		cfg.Type = "sqlite"
	}
	if cfg.Type == "sqlite" {
		if cfg.SQLite == nil {
			cfg.SQLite = &configstoresqlite.SQLiteConfigStoreConfig{}
		}
		if cfg.SQLite.SQLitePath == "" {
			cfg.SQLite.SQLitePath = filepath.Join(caddy.AppDataDir(), "caddy-llm", "configstore.db")
		}
	}
	return &cfg
}

func (c *ConfigStoreConfig) sqliteConfig() configstoresqlite.SQLiteConfigStoreConfig {
	if c != nil && c.SQLite != nil {
		return *c.SQLite
	}
	return configstoresqlite.SQLiteConfigStoreConfig{}
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
