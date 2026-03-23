package llm

import (
	"testing"

	"github.com/agent-guide/caddy-llm/llm/authmanager/authenticator"
	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/caddyserver/caddy/v2"
)

func TestProvisionAuthenticatorsWithEmptyConfig(t *testing.T) {
	app := &App{authManager: manager.NewManager(nil, nil, nil)}

	if err := app.provisionAuthenticators(caddy.Context{}); err != nil {
		t.Fatalf("provisionAuthenticators() error = %v", err)
	}

	if _, ok := app.authManager.GetAuthenticator("codex"); ok {
		t.Fatal("expected codex authenticator to remain disabled without configuration")
	}
	if _, ok := app.authManager.GetAuthenticator("claude"); ok {
		t.Fatal("expected claude authenticator to remain disabled without configuration")
	}
}

func TestRegisterLoadedAuthenticators(t *testing.T) {
	app := &App{authManager: manager.NewManager(nil, nil, nil)}

	err := app.registerLoadedAuthenticators(map[string]any{
		"gemini": authenticator.NewGeminiAuthenticator(),
	})
	if err != nil {
		t.Fatalf("registerLoadedAuthenticators() error = %v", err)
	}

	if _, ok := app.authManager.GetAuthenticator("gemini"); !ok {
		t.Fatal("expected configured gemini authenticator to be registered")
	}
}

func TestRegisterLoadedAuthenticatorsRejectsInvalidModule(t *testing.T) {
	app := &App{authManager: manager.NewManager(nil, nil, nil)}

	err := app.registerLoadedAuthenticators(map[string]any{
		"invalid": struct{}{},
	})
	if err == nil {
		t.Fatal("expected invalid authenticator module to be rejected")
	}
}
