package api_test

import (
	"strings"
	"testing"

	api "github.com/agent-guide/caddy-agent-gateway/api"
	_ "github.com/agent-guide/caddy-agent-gateway/api/llmapi/openai"
	openaiapi "github.com/agent-guide/caddy-agent-gateway/api/llmapi/openai"
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
