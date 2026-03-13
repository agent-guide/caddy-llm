// Package mistral implements the Mistral AI provider (OpenAI-compatible API).
package mistral

import (
	"fmt"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaicompat"
)

func init() {
	provider.RegisterProvider("mistral", New)
}

type mistralProvider struct {
	*openaicompat.Base
}

// New creates a new Mistral provider.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("mistral: api_key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.mistral.ai/v1"
	}
	config.Network.Defaults()
	return &mistralProvider{Base: openaicompat.NewBase(config)}, nil
}

func (p *mistralProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming:       true,
		Tools:           true,
		ContextWindow:   131072,
		MaxOutputTokens: 4096,
	}
}

var _ provider.Provider = (*mistralProvider)(nil)
