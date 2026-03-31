package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/credential"
	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/manager"
	configstoresqlite "github.com/agent-guide/caddy-agent-gateway/configstore/sqlite"
	_ "github.com/agent-guide/caddy-agent-gateway/llm/provider/ollama"
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
		provider ollama {
			base_url http://127.0.0.1:11434/v1
			default_model qwen2.5
		}

		config_store sqlite {
			path /tmp/caddy-agent-gateway.db
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

	if len(app.ConfigStoreRaw) != 1 {
		t.Fatalf("config_store count = %d, want 1", len(app.ConfigStoreRaw))
	}
	if len(app.ProvidersRaw) != 1 {
		t.Fatalf("provider count = %d, want 1", len(app.ProvidersRaw))
	}

	var ollama struct {
		BaseURL      string `json:"base_url,omitempty"`
		DefaultModel string `json:"default_model,omitempty"`
	}
	if err := json.Unmarshal(app.ProvidersRaw["ollama"], &ollama); err != nil {
		t.Fatalf("unmarshal ollama provider: %v", err)
	}
	if ollama.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("ollama base_url = %q", ollama.BaseURL)
	}
	if ollama.DefaultModel != "qwen2.5" {
		t.Fatalf("ollama default_model = %q", ollama.DefaultModel)
	}

	var cfg configstoresqlite.SQLiteConfigStore
	if err := json.Unmarshal(app.ConfigStoreRaw["sqlite"], &cfg); err != nil {
		t.Fatalf("unmarshal sqlite config store: %v", err)
	}
	if cfg.SQLitePath != "/tmp/caddy-agent-gateway.db" {
		t.Fatalf("sqlite path = %q, want /tmp/caddy-agent-gateway.db", cfg.SQLitePath)
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
