package llmapi

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/agent-guide/caddy-llm/llm/provider"
)

// LLMApiHandler handles one compatible LLM API surface, such as OpenAI or Anthropic.
type LLMApiHandler interface {
	caddy.Module
	Name() string
	MatchLLMApi(*http.Request) bool
	ServeLLMApi(http.ResponseWriter, *http.Request) error
	ProvisionLLMApi(*manager.Manager, provider.Provider, *zap.Logger) error
}
