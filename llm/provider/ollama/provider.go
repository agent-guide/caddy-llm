// Package ollama implements the Ollama provider (local deployment, OpenAI-compatible).
package ollama

import (
	"context"
	"strings"

	einoollama "github.com/cloudwego/eino-ext/components/model/ollama"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/openaibase"
)

func init() {
	provider.RegisterProvider("ollama", New)
}

type ollamaProvider struct {
	config provider.ProviderConfig
	*openaibase.Base
}

// New creates a new Ollama provider. No API key is required for local deployment.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434/v1"
	}
	config.Network.Defaults()
	return &ollamaProvider{
		config: config,
		Base:   openaibase.NewBase(config),
	}, nil
}

func (p *ollamaProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
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

func (p *ollamaProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
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

func (p *ollamaProvider) newChatModel(ctx context.Context, req *provider.GenerateRequest) (einomodel.ToolCallingChatModel, []*schema.Message, []einomodel.Option, error) {
	state, err := provider.ResolveChatRequest(ctx, p.config, req)
	if err != nil {
		return nil, nil, nil, err
	}

	baseURL := strings.TrimSuffix(state.BaseURL, "/v1")
	cfg := &einoollama.ChatModelConfig{
		BaseURL:    baseURL,
		Model:      state.ModelName,
		HTTPClient: provider.BuildHTTPClient(p.config, nil, state.Credential),
	}

	chatModel, err := einoollama.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return chatModel, state.Messages, state.Options, nil
}

func (p *ollamaProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming: true,
		Tools:     true,
	}
}

var _ provider.Provider = (*ollamaProvider)(nil)
