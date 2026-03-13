// Package ollama implements the Ollama provider (local deployment, OpenAI-compatible).
package ollama

import (
	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaicompat"
)

func init() {
	provider.RegisterProvider("ollama", New)
}

type ollamaProvider struct {
	*openaicompat.Base
}

// New creates a new Ollama provider. No API key is required for local deployment.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434/v1"
	}
	config.Network.Defaults()
	return &ollamaProvider{Base: openaicompat.NewBase(config)}, nil
}

func (p *ollamaProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming: true,
		Tools:     true,
	}
}

var _ provider.Provider = (*ollamaProvider)(nil)
