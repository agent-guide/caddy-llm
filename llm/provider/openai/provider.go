// Package openai implements the OpenAI provider.
package openai

import (
	"fmt"
	"strings"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaicompat"
)

func init() {
	provider.RegisterProvider("openai", New)
}

type openAIProvider struct {
	*openaicompat.Base
}

// New creates a new OpenAI provider.
//
// Optional config.Options keys:
//   - "organization": string → sent as OpenAI-Organization header.
//   - "project":      string → sent as OpenAI-Project header.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("openai: api_key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	config.Network.Defaults()

	// Inject OpenAI-specific headers via ExtraHeaders so Base.setHeaders picks them up.
	if config.Network.ExtraHeaders == nil {
		config.Network.ExtraHeaders = make(map[string]string)
	}
	if v, ok := config.Options["organization"]; ok {
		if s, ok := v.(string); ok && s != "" {
			config.Network.ExtraHeaders["OpenAI-Organization"] = s
		}
	}
	if v, ok := config.Options["project"]; ok {
		if s, ok := v.(string); ok && s != "" {
			config.Network.ExtraHeaders["OpenAI-Project"] = s
		}
	}

	return &openAIProvider{Base: openaicompat.NewBase(config)}, nil
}

func (p *openAIProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming:       true,
		Tools:           true,
		Vision:          true,
		Embeddings:      true,
		ContextWindow:   128000,
		MaxOutputTokens: 16384,
	}
}

// Interface guards.
var (
	_ provider.Provider          = (*openAIProvider)(nil)
	_ provider.EmbeddingProvider = (*openAIProvider)(nil)
)
