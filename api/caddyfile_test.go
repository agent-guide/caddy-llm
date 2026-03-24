package api

import (
	"testing"

	_ "github.com/agent-guide/caddy-llm/api/llmapi/openai"
	openaiapi "github.com/agent-guide/caddy-llm/api/llmapi/openai"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func TestParseHandleLLMAPI(t *testing.T) {
	d := caddyfile.NewTestDispenser(`
	handle_llm_api openai {
		provider openrouter
	}
	`)

	handler, err := parseHandleLLMAPI(httpcaddyfile.Helper{Dispenser: d})
	if err != nil {
		t.Fatalf("parseHandleLLMAPI() error = %v", err)
	}

	openaiHandler, ok := handler.(*openaiapi.Handler)
	if !ok {
		t.Fatalf("handler type = %T, want *openai.Handler", handler)
	}
	if openaiHandler.Provider != "openrouter" {
		t.Fatalf("provider = %q, want openrouter", openaiHandler.Provider)
	}
}
