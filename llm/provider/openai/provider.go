// Package openai implements the OpenAI provider.
package openai

import (
	"context"
	"fmt"
	"strings"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaibase"
)

func init() {
	provider.RegisterProvider("openai", New)
}

type openAIProvider struct {
	config provider.ProviderConfig
	*openaibase.Base
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

	return &openAIProvider{
		config: config,
		Base:   openaibase.NewBase(config),
	}, nil
}

func (p *openAIProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	return provider.RetryGenerate(p.config.Network, func() (*provider.GenerateResponse, error) {
		chatModel, messages, opts, err := p.newChatModel(ctx, req)
		if err != nil {
			return nil, err
		}
		msg, err := chatModel.Generate(ctx, messages, opts...)
		if err != nil {
			return nil, provider.WrapEinoError(err)
		}
		return provider.FromEinoMessage(msg), nil
	})
}

func (p *openAIProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
	chatModel, messages, opts, err := p.newChatModel(ctx, req)
	if err != nil {
		return nil, err
	}
	stream, err := chatModel.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, provider.WrapEinoError(err)
	}
	return stream, nil
}

func (p *openAIProvider) newChatModel(ctx context.Context, req *provider.GenerateRequest) (einomodel.ToolCallingChatModel, []*schema.Message, []einomodel.Option, error) {
	state, err := provider.ResolveChatRequest(ctx, p.config, req)
	if err != nil {
		return nil, nil, nil, err
	}

	cfg := &einoopenai.ChatModelConfig{
		APIKey:     state.APIKey,
		BaseURL:    state.BaseURL,
		Model:      state.ModelName,
		HTTPClient: provider.BuildHTTPClient(p.config, nil, state.Credential),
	}

	chatModel, err := einoopenai.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return chatModel, state.Messages, state.Options, nil
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
