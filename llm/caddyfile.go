package llm

import (
	"encoding/json"

	configstoresqlite "github.com/agent-guide/caddy-llm/llm/configstore/sqlite"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func init() {
	httpcaddyfile.RegisterGlobalOption("llm", parseApp)
}

func parseApp(d *caddyfile.Dispenser, existingVal any) (any, error) {
	app := &App{}
	if current, ok := existingVal.(*App); ok && current != nil {
		app = current
	}

	if !d.Next() {
		return nil, d.Err("expected directive name")
	}
	if d.NextArg() {
		return nil, d.ArgErr()
	}

	for d.NextBlock(0) {
		switch d.Val() {
		case "config_store":
			if err := parseConfigStore(d, app); err != nil {
				return nil, err
			}
		case "authenticator":
			if err := parseAuthenticator(d, app); err != nil {
				return nil, err
			}
		default:
			return nil, d.Errf("unknown subdirective: %s", d.Val())
		}
	}

	return httpcaddyfile.App{
		Name:  "llm",
		Value: caddyconfig.JSON(app, nil),
	}, nil
}

func parseConfigStore(d *caddyfile.Dispenser, app *App) error {
	if !d.NextArg() {
		return d.ArgErr()
	}
	storeType := d.Val()

	switch storeType {
	case "sqlite":
		cfg, err := parseSQLiteConfigStore(d)
		if err != nil {
			return err
		}
		app.ConfigStoreCfg = &ConfigStoreConfig{
			Type:   "sqlite",
			SQLite: cfg,
		}
		return nil
	default:
		return d.Errf("unsupported config_store type: %s", storeType)
	}
}

func parseSQLiteConfigStore(d *caddyfile.Dispenser) (*configstoresqlite.SQLiteConfigStoreConfig, error) {
	cfg := &configstoresqlite.SQLiteConfigStoreConfig{}
	for d.NextBlock(1) {
		switch d.Val() {
		case "path":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			cfg.SQLitePath = d.Val()
		default:
			return nil, d.Errf("unknown sqlite config_store subdirective: %s", d.Val())
		}
	}
	return cfg, nil
}

func parseAuthenticator(d *caddyfile.Dispenser, app *App) error {
	if !d.NextArg() {
		return d.ArgErr()
	}
	name := d.Val()
	modID := "llm.authenticators." + name
	unm, err := caddyfile.UnmarshalModule(d, modID)
	if err != nil {
		return err
	}

	if app.AuthenticatorsRaw == nil {
		app.AuthenticatorsRaw = make(map[string]json.RawMessage)
	}
	app.AuthenticatorsRaw[name] = caddyconfig.JSON(unm, nil)
	return nil
}
