// Package groq implements the Groq provider (OpenAI-compatible API).
package groq

import (
	"fmt"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaicompat"
)

func init() {
	provider.RegisterProvider("groq", New)
}

type groqProvider struct {
	*openaicompat.Base
}

// New creates a new Groq provider.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("groq: api_key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.groq.com/openai/v1"
	}
	config.Network.Defaults()
	return &groqProvider{Base: openaicompat.NewBase(config)}, nil
}

func (p *groqProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming:       true,
		Tools:           true,
		ContextWindow:   128000,
		MaxOutputTokens: 8192,
	}
}

var _ provider.Provider = (*groqProvider)(nil)
