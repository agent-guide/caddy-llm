package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/agent-guide/caddy-llm/llm/authmanager/credential"
	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func init() {
	caddy.RegisterModule(testAuthenticatorModule{})
}

type testAuthenticatorModule struct {
	Foo string `json:"foo,omitempty"`
}

func (testAuthenticatorModule) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "llm.authenticators.test",
		New: func() caddy.Module { return new(testAuthenticatorModule) },
	}
}

func (m *testAuthenticatorModule) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "foo":
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.Foo = d.Val()
			default:
				return d.Errf("unknown subdirective: %s", d.Val())
			}
		}
	}
	return nil
}

func (testAuthenticatorModule) Provider() string { return "test" }

func (testAuthenticatorModule) Login(context.Context) (*credential.Credential, error) {
	return nil, nil
}

func (testAuthenticatorModule) RefreshLead(context.Context, *credential.Credential) (*credential.Credential, error) {
	return nil, nil
}

var _ manager.Authenticator = (*testAuthenticatorModule)(nil)

func TestParseAppFromCaddyfile(t *testing.T) {
	d := caddyfile.NewTestDispenser(`
	llm {
		config_store sqlite {
			path /tmp/caddy-llm.db
		}

		authenticator test {
			foo bar
		}
	}
	`)

	val, err := parseApp(d, nil)
	if err != nil {
		t.Fatalf("parseApp() error = %v", err)
	}

	appVal, ok := val.(httpcaddyfile.App)
	if !ok {
		t.Fatalf("parseApp() type = %T, want httpcaddyfile.App", val)
	}
	if appVal.Name != "llm" {
		t.Fatalf("app name = %q, want llm", appVal.Name)
	}

	var app App
	if err := json.Unmarshal(appVal.Value, &app); err != nil {
		t.Fatalf("unmarshal app json: %v", err)
	}

	if app.ConfigStoreCfg == nil || app.ConfigStoreCfg.Type != "sqlite" {
		t.Fatalf("config_store type = %#v, want sqlite", app.ConfigStoreCfg)
	}
	if app.ConfigStoreCfg.SQLite == nil || app.ConfigStoreCfg.SQLite.SQLitePath != "/tmp/caddy-llm.db" {
		t.Fatalf("sqlite path = %#v, want /tmp/caddy-llm.db", app.ConfigStoreCfg.SQLite)
	}
	if len(app.AuthenticatorsRaw) != 1 {
		t.Fatalf("authenticator count = %d, want 1", len(app.AuthenticatorsRaw))
	}

	var codex struct {
		Foo string `json:"foo,omitempty"`
	}
	if err := json.Unmarshal(app.AuthenticatorsRaw["test"], &codex); err != nil {
		t.Fatalf("unmarshal test authenticator: %v", err)
	}
	if codex.Foo != "bar" {
		t.Fatalf("unexpected test authenticator config: %+v", codex)
	}
}

func TestParseAppRejectsUnknownConfigStore(t *testing.T) {
	d := caddyfile.NewTestDispenser(`
	llm {
		config_store memory
	}
	`)

	if _, err := parseApp(d, nil); err == nil {
		t.Fatal("expected unsupported config_store type to fail")
	}
}
