package manager

import (
	"context"
	"testing"

	"github.com/agent-guide/caddy-agent-gateway/llm/cliauth/credential"
)

type stubAuthenticator struct {
	provider string
}

type stubCredentialStore struct {
	createCalls int
	updateCalls int
	lastCreated *credential.Credential
	lastUpdated *credential.Credential
}

func (s *stubCredentialStore) ListByProviderName(context.Context, string) ([]any, error) {
	return nil, nil
}

func (s *stubCredentialStore) Create(_ context.Context, _ string, _ string, obj any) (string, error) {
	s.createCalls++
	cred, _ := obj.(*credential.Credential)
	s.lastCreated = cred
	if cred == nil {
		return "", nil
	}
	return cred.ID, nil
}

func (s *stubCredentialStore) Update(_ context.Context, _ string, obj any) error {
	s.updateCalls++
	cred, _ := obj.(*credential.Credential)
	s.lastUpdated = cred
	return nil
}

func (s *stubCredentialStore) Delete(context.Context, string) error {
	return nil
}

func (s *stubCredentialStore) Get(context.Context, string) (string, any, error) {
	return "", nil, nil
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

func TestRegisterPersistsWithCreate(t *testing.T) {
	store := &stubCredentialStore{}
	mgr := NewManager(store, nil, nil)

	if err := mgr.Register(context.Background(), &credential.Credential{
		ID:       "cred-1",
		Provider: "openai",
	}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if store.createCalls != 1 {
		t.Fatalf("Create called %d times, want 1", store.createCalls)
	}
	if store.updateCalls != 0 {
		t.Fatalf("Update called %d times, want 0", store.updateCalls)
	}
}

func TestUpdatePersistsWithUpdate(t *testing.T) {
	store := &stubCredentialStore{}
	mgr := NewManager(store, nil, nil)

	if err := mgr.Update(context.Background(), &credential.Credential{
		ID:       "cred-1",
		Provider: "openai",
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if store.updateCalls != 1 {
		t.Fatalf("Update called %d times, want 1", store.updateCalls)
	}
	if store.createCalls != 0 {
		t.Fatalf("Create called %d times, want 0", store.createCalls)
	}
}
