package gateway

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/cloudwego/eino/schema"
)

type fixedSelector struct {
	target RouteTarget
}

func (s fixedSelector) SelectTarget(Route, ResolveRequest) (*RouteTarget, error) {
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
	route := Route{
		ID: "chat-prod",
		Targets: []RouteTarget{
			{ProviderRef: "openai", Mode: TargetModeWeighted, Weight: 1},
			{ProviderRef: "openrouter", Mode: TargetModeWeighted, Weight: 1},
		},
	}
	gw := NewAgentGateway()
	gw.EnsureRoute(route)
	gw.Configure(nil, NewStaticProviderResolver(func(name string) (provider.Provider, bool) {
		if name == "openrouter" {
			return testProvider{}, true
		}
		return nil, false
	}), nil, nil, fixedSelector{target: RouteTarget{ProviderRef: "openrouter"}})

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	resolved, err := gw.ResolveProvider(context.Background(), route.ID, ResolveRequest{
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

func TestDefaultRouteSelectorUsesPolicyStrategyAndFallback(t *testing.T) {
	selector := DefaultRouteSelector{}
	route := Route{
		ID: "chat-prod",
		Targets: []RouteTarget{
			{ProviderRef: "weighted", Mode: TargetModeWeighted, Weight: 1},
			{ProviderRef: "failover", Mode: TargetModeFailover, Priority: 1},
		},
		Policy: RoutePolicy{
			Selection: SelectionPolicy{Strategy: RouteSelectionStrategyFailover},
			Fallback:  FallbackPolicy{Enabled: true},
		},
	}

	target, err := selector.SelectTarget(route, ResolveRequest{})
	if err != nil {
		t.Fatalf("SelectTarget returned error: %v", err)
	}
	if target.ProviderRef != "failover" {
		t.Fatalf("unexpected target: got %q want %q", target.ProviderRef, "failover")
	}

	route.Policy.Selection.Strategy = RouteSelectionStrategyConditional
	target, err = selector.SelectTarget(route, ResolveRequest{})
	if err != nil {
		t.Fatalf("SelectTarget with fallback returned error: %v", err)
	}
	if target.ProviderRef != "weighted" {
		t.Fatalf("unexpected fallback target: got %q want %q", target.ProviderRef, "weighted")
	}
}
