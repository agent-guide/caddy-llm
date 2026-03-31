package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agent-guide/caddy-agent-gateway/gateway"
	"github.com/cloudwego/eino/schema"

	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/credential"
	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/manager"
	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
)

type testProvider struct {
	streamErr error
}

func (p *testProvider) Generate(context.Context, *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	return nil, nil
}

func (p *testProvider) Stream(context.Context, *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
	return nil, p.streamErr
}

func (p *testProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}

func (p *testProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{Streaming: true}
}

func (p *testProvider) Config() provider.ProviderConfig {
	return provider.ProviderConfig{}
}

type testStatusError struct {
	msg    string
	status int
}

func (e testStatusError) Error() string   { return e.msg }
func (e testStatusError) StatusCode() int { return e.status }

func newSeededHandler(authMgr *manager.Manager, prov provider.Provider) *Handler {
	handler := NewHandler(prov)
	gateway.ResetGlobalAgentGateway().Configure(nil, gateway.NewStaticProviderResolver(func(name string) (provider.Provider, bool) {
		if name != "anthropic" || prov == nil {
			return nil, false
		}
		return prov, true
	}), nil, authMgr, nil)
	handler.RouteID = "anthropic-test-route"
	gateway.GlobalAgentGateway().EnsureRoute(gateway.Route{
		ID:   handler.RouteID,
		Name: handler.RouteID,
		Targets: []gateway.RouteTarget{{
			ProviderRef: "anthropic",
			Mode:        gateway.TargetModeWeighted,
			Weight:      1,
		}},
	})
	return handler
}

func TestServeLLMApiMarksAnthropicStreamFailures(t *testing.T) {
	authMgr := manager.NewManager(nil, nil, nil)
	if err := authMgr.Register(context.Background(), &credential.Credential{
		ID:       "cred-anthropic-1",
		Provider: "anthropic",
	}); err != nil {
		t.Fatalf("register credential: %v", err)
	}

	handler := newSeededHandler(authMgr, &testProvider{
		streamErr: testStatusError{msg: "rate limit", status: http.StatusTooManyRequests},
	})

	body, err := json.Marshal(MessagesRequest{
		Model:     "claude-sonnet-4-5",
		MaxTokens: 16,
		Stream:    true,
		Messages: []MessageItem{{
			Role: "user",
			Content: []ContentBlock{{
				Type: "text",
				Text: "hello",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	if err := handler.ServeLLMApi(rec, req); err != nil {
		t.Fatalf("ServeLLMApi returned error: %v", err)
	}
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusBadGateway)
	}

	cred := authMgr.Get("cred-anthropic-1")
	if cred == nil {
		t.Fatal("credential not found after request")
	}
	if !cred.Quota.Exceeded {
		t.Fatal("expected quota state to be marked exceeded")
	}
	if !cred.Unavailable {
		t.Fatal("expected credential to be marked unavailable")
	}
	if cred.Status != credential.StatusError {
		t.Fatalf("expected credential status %q, got %q", credential.StatusError, cred.Status)
	}
	if cred.StatusMessage != "quota exceeded" {
		t.Fatalf("expected status message %q, got %q", "quota exceeded", cred.StatusMessage)
	}
	if cred.NextRetryAfter.IsZero() {
		t.Fatal("expected next retry time to be set")
	}
}
