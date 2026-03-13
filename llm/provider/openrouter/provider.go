// Package openrouter implements the OpenRouter provider (OpenAI-compatible API).
package openrouter

import (
	"fmt"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaicompat"
)

func init() {
	provider.RegisterProvider("openrouter", New)
}

type openRouterProvider struct {
	*openaicompat.Base
}

// New creates a new OpenRouter provider.
// OpenRouter requires HTTP-Referer and X-Title headers; set them via NetworkConfig.ExtraHeaders.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("openrouter: api_key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://openrouter.ai/api/v1"
	}
	config.Network.Defaults()
	return &openRouterProvider{Base: openaicompat.NewBase(config)}, nil
}

func (p *openRouterProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming: true,
		Tools:     true,
	}
}

var _ provider.Provider = (*openRouterProvider)(nil)
