package openai

import "github.com/agent-guide/caddy-llm/llm/provider"

// Converter converts between OpenAI API format and internal format.
type Converter struct{}

// ChatCompletionRequest is the OpenAI chat completion request format.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

// ChatMessage is a single message in an OpenAI chat request.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse is the OpenAI chat completion response format.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is a single choice in an OpenAI chat completion response.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage is token usage information in an OpenAI response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToInternal converts an OpenAI ChatCompletionRequest to the internal GenerateRequest.
func (c *Converter) ToInternal(req *ChatCompletionRequest) *provider.GenerateRequest {
	msgs := make([]provider.Message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = provider.Message{
			Role: m.Role,
			Content: []provider.ContentBlock{
				{Type: "text", Text: m.Content},
			},
		}
	}
	genReq := &provider.GenerateRequest{
		Model:     req.Model,
		Messages:  msgs,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
		Stop:      req.Stop,
	}
	if req.Temperature != 0 {
		t := req.Temperature
		genReq.Temperature = &t
	}
	return genReq
}

// FromInternal converts an internal GenerateResponse to a ChatCompletionResponse.
func (c *Converter) FromInternal(resp *provider.GenerateResponse) *ChatCompletionResponse {
	var content string
	for _, b := range resp.Content {
		if b.Type == "text" {
			content += b.Text
		}
	}
	return &ChatCompletionResponse{
		ID:     resp.ID,
		Object: "chat.completion",
		Model:  resp.Model,
		Choices: []Choice{{
			Index:        0,
			Message:      ChatMessage{Role: "assistant", Content: content},
			FinishReason: resp.StopReason,
		}},
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}
