package gateway

import (
	"context"
	"net/http/httptest"
	"testing"

	routepkg "github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
	"github.com/cloudwego/eino/schema"
)

type fixedSelector struct {
	target routepkg.RouteTarget
}

func (s fixedSelector) SelectTarget(routepkg.Route, routepkg.ResolveRequest) (*routepkg.RouteTarget, error) {
	target := s.target
	return &target, nil
}

type testProvider struct{}

func (testProvider) Generate(context.Context, *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	return nil, nil
}

func (testProvider) Stream(context.Context, *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (testProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}

func (testProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{}
}

func (testProvider) Config() provider.ProviderConfig {
	return provider.ProviderConfig{}
}

func TestResolverUsesCustomSelector(t *testing.T) {
	route := routepkg.Route{
		ID: "chat-prod",
		Targets: []routepkg.RouteTarget{
			{ProviderRef: "openai", Mode: routepkg.TargetModeWeighted, Weight: 1},
			{ProviderRef: "openrouter", Mode: routepkg.TargetModeWeighted, Weight: 1},
		},
	}
	gw := NewAgentGateway()
	gw.EnsureRoute(route)
	gw.Configure(nil, NewStaticProviderResolver(func(name string) (provider.Provider, bool) {
		if name == "openrouter" {
			return testProvider{}, true
		}
		return nil, false
	}), nil, nil, fixedSelector{target: routepkg.RouteTarget{ProviderRef: "openrouter"}})

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	resolved, err := gw.ResolveProvider(context.Background(), route.ID, routepkg.ResolveRequest{
		HTTPRequest: req,
		Model:       "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("ResolveProvider returned error: %v", err)
	}
	if resolved.ProviderName != "openrouter" {
		t.Fatalf("unexpected provider: got %q want %q", resolved.ProviderName, "openrouter")
	}
}
