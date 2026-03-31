package manager

import (
	"context"
	"testing"

	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/credential"
)

type stubAuthenticator struct {
	provider string
}

func (a *stubAuthenticator) Provider() string {
	return a.provider
}

func (a *stubAuthenticator) Login(context.Context) (*credential.Credential, error) {
	return nil, nil
}

func (a *stubAuthenticator) RefreshLead(context.Context, *credential.Credential) (*credential.Credential, error) {
	return nil, nil
}

func TestRegisterAuthenticatorIndexesProviderKey(t *testing.T) {
	mgr := NewManager(nil, nil, nil)
	auth := &stubAuthenticator{provider: "openai"}

	mgr.RegisterAuthenticator("codex", auth)

	if got, ok := mgr.GetAuthenticator("codex"); !ok || got != auth {
		t.Fatalf("GetAuthenticator(codex) = (%v, %v), want registered authenticator", got, ok)
	}

	refresher := mgr.resolveRefresher("openai")
	if refresher == nil {
		t.Fatal("resolveRefresher(openai) returned nil")
	}

	wrapped, ok := refresher.(*authenticatorRefresher)
	if !ok {
		t.Fatalf("resolveRefresher(openai) returned %T, want *authenticatorRefresher", refresher)
	}
	if wrapped.auth != auth {
		t.Fatal("resolveRefresher(openai) did not return the registered authenticator")
	}
}
