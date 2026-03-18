// Package openrouter implements the OpenRouter provider (OpenAI-compatible API).
package openrouter

import (
	"context"
	"fmt"

	einoopenrouter "github.com/cloudwego/eino-ext/components/model/openrouter"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaibase"
)

func init() {
	provider.RegisterProvider("openrouter", New)
}

type openRouterProvider struct {
	config provider.ProviderConfig
	*openaibase.Base
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
	return &openRouterProvider{
		config: config,
		Base:   openaibase.NewBase(config),
	}, nil
}

func (p *openRouterProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
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

func (p *openRouterProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
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

func (p *openRouterProvider) newChatModel(ctx context.Context, req *provider.GenerateRequest) (einomodel.ToolCallingChatModel, []*schema.Message, []einomodel.Option, error) {
	state, err := provider.ResolveChatRequest(ctx, p.config, req)
	if err != nil {
		return nil, nil, nil, err
	}

	cfg := &einoopenrouter.Config{
		APIKey:     state.APIKey,
		BaseURL:    state.BaseURL,
		Model:      state.ModelName,
		HTTPClient: provider.BuildHTTPClient(p.config, nil, state.Credential),
	}

	chatModel, err := einoopenrouter.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return chatModel, state.Messages, state.Options, nil
}

func (p *openRouterProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming: true,
		Tools:     true,
	}
}

var _ provider.Provider = (*openRouterProvider)(nil)
