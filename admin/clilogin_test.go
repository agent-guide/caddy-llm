package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agent-guide/caddy-llm/llm/authmanager/credential"
	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
)

type testAuthenticator struct {
	provider string
	loginFn  func(context.Context) (*credential.Credential, error)
}

func (a *testAuthenticator) Provider() string {
	return a.provider
}

func (a *testAuthenticator) Login(ctx context.Context) (*credential.Credential, error) {
	if a.loginFn != nil {
		return a.loginFn(ctx)
	}
	return &credential.Credential{Provider: a.provider}, nil
}

func (a *testAuthenticator) RefreshLead(context.Context, *credential.Credential) (*credential.Credential, error) {
	return nil, nil
}

func TestCLILoginResolvesAuthenticatorAndRegistersCredential(t *testing.T) {
	authMgr := manager.NewManager(nil, nil, nil)
	authMgr.RegisterAuthenticator("codex", &testAuthenticator{
		provider: "openai",
		loginFn: func(context.Context) (*credential.Credential, error) {
			return &credential.Credential{
				ID:       "cred-openai-1",
				Provider: "openai",
				Label:    "test@example.com",
			}, nil
		},
	})

	handler := NewHandler(authMgr, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/clilogin/codex", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusAccepted)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cred := authMgr.Get("cred-openai-1"); cred != nil {
			if cred.Provider != "openai" {
				t.Fatalf("unexpected provider: got %q want %q", cred.Provider, "openai")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("credential was not registered")
}

func TestCLILoginReturnsNotFoundForUnknownCliname(t *testing.T) {
	handler := NewHandler(manager.NewManager(nil, nil, nil), nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/clilogin/unknown", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusNotFound)
	}
}
