package api_test

import (
	"strings"
	"testing"

	api "github.com/agent-guide/caddy-agent-gateway/api"
	_ "github.com/agent-guide/caddy-agent-gateway/api/llmapi/openai"
	openaiapi "github.com/agent-guide/caddy-agent-gateway/api/llmapi/openai"
	"github.com/agent-guide/caddy-agent-gateway/gateway"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func TestParseHandleLLMAPIRequiresRouteID(t *testing.T) {
	d := caddyfile.NewTestDispenser(`
	handle_llm_api openai
	`)

	_, err := api.ParseHandleLLMAPIForTest(httpcaddyfile.Helper{Dispenser: d})
	if err == nil || !strings.Contains(err.Error(), "route_id is required") {
		t.Fatalf("expected route_id is required error, got %v", err)
	}
}

func TestParseHandleLLMAPI(t *testing.T) {
	d := caddyfile.NewTestDispenser(`
	handle_llm_api openai {
		route_id chat-prod
	}
	`)

	handler, err := api.ParseHandleLLMAPIForTest(httpcaddyfile.Helper{Dispenser: d})
	if err != nil {
		t.Fatalf("parseHandleLLMAPI() error = %v", err)
	}

	openaiHandler, ok := handler.(*openaiapi.Handler)
	if !ok {
		t.Fatalf("handler type = %T, want *openai.Handler", handler)
	}
	if openaiHandler.RouteID != "chat-prod" {
		t.Fatalf("route_id = %q, want chat-prod", openaiHandler.RouteID)
	}
}

func TestParseHandleLLMAPIRejectsUnknownSubdirective(t *testing.T) {
	d := caddyfile.NewTestDispenser(`
	handle_llm_api openai {
		route_id chat-prod
		model gpt-4.1
	}
	`)

	_, err := api.ParseHandleLLMAPIForTest(httpcaddyfile.Helper{Dispenser: d})
	if err == nil || !strings.Contains(err.Error(), "unknown subdirective: model") {
		t.Fatalf("expected unknown subdirective error, got %v", err)
	}
}

func TestParseAgentGatewayRouteRegistersRoute(t *testing.T) {
	gateway.ResetGlobalAgentGateway().Configure(nil, nil, nil, nil, nil)
	state := map[string]any{}

	d := caddyfile.NewTestDispenser(`
	agent_gateway_route openai-chat {
		require_local_api_key
		allowed_model gpt-4.1
		allowed_model gpt-4.1-mini
		target openai 80
		target openrouter 20
	}
	`)

	if err := api.ParseAgentGatewayRouteForTest(httpcaddyfile.Helper{
		Dispenser: d,
		State:     state,
	}); err != nil {
		t.Fatalf("parse agent_gateway_route: %v", err)
	}

	route, ok := gateway.GlobalAgentGateway().Route("openai-chat")
	if !ok {
		t.Fatal("expected openai-chat route to be registered")
	}
	if !route.Policy.Auth.RequireLocalAPIKey {
		t.Fatal("expected require_local_api_key to be true")
	}
	if len(route.Policy.AllowedModels) != 2 {
		t.Fatalf("allowed_models = %#v", route.Policy.AllowedModels)
	}
	if len(route.Targets) != 2 || route.Targets[0].ProviderRef != "openai" || route.Targets[1].ProviderRef != "openrouter" {
		t.Fatalf("targets = %#v", route.Targets)
	}
}

func TestParseAgentGatewayRouteRejectsDuplicateRouteID(t *testing.T) {
	gateway.ResetGlobalAgentGateway()
	state := map[string]any{}

	first := caddyfile.NewTestDispenser(`
	agent_gateway_route openai-chat {
		target openai
	}
	`)
	if err := api.ParseAgentGatewayRouteForTest(httpcaddyfile.Helper{
		Dispenser: first,
		State:     state,
	}); err != nil {
		t.Fatalf("parse first agent_gateway_route: %v", err)
	}

	second := caddyfile.NewTestDispenser(`
	agent_gateway_route openai-chat {
		target openrouter
	}
	`)
	err := api.ParseAgentGatewayRouteForTest(httpcaddyfile.Helper{
		Dispenser: second,
		State:     state,
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate agent_gateway_route "openai-chat"`) {
		t.Fatalf("expected duplicate route error, got %v", err)
	}
}
