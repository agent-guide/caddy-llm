package anthropic

import (
	"github.com/agent-guide/caddy-llm/llm/provider"
)

// Converter converts between Anthropic API format and internal format.
type Converter struct{}

// ToInternal converts an Anthropic MessagesRequest to the internal GenerateRequest.
func (c *Converter) ToInternal(req *MessagesRequest) *provider.GenerateRequest {
	msgs := make([]provider.Message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = provider.Message{
			Role:    m.Role,
			Content: convertContentBlocks(m.Content),
		}
	}

	genReq := &provider.GenerateRequest{
		Model:     req.Model,
		Messages:  msgs,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}
	if req.Temperature != 0 {
		t := req.Temperature
		genReq.Temperature = &t
	}
	if req.TopP != 0 {
		p := req.TopP
		genReq.TopP = &p
	}
	return genReq
}

// FromInternal converts an internal GenerateResponse to an Anthropic MessagesResponse.
func (c *Converter) FromInternal(resp *provider.GenerateResponse) *MessagesResponse {
	content := make([]ContentBlockResponse, len(resp.Content))
	for i, b := range resp.Content {
		content[i] = ContentBlockResponse{Type: b.Type, Text: b.Text}
	}
	return &MessagesResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      resp.Model,
		Content:    content,
		StopReason: resp.StopReason,
		Usage: UsageResponse{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}
}

func convertContentBlocks(blocks []ContentBlock) []provider.ContentBlock {
	out := make([]provider.ContentBlock, len(blocks))
	for i, b := range blocks {
		out[i] = provider.ContentBlock{Type: b.Type, Text: b.Text}
	}
	return out
}

// --- Anthropic API types ---

type MessagesRequest struct {
	Model         string        `json:"model"`
	MaxTokens     int           `json:"max_tokens"`
	Messages      []MessageItem `json:"messages"`
	System        string        `json:"system,omitempty"`
	Temperature   float64       `json:"temperature,omitempty"`
	TopP          float64       `json:"top_p,omitempty"`
	Stream        bool          `json:"stream,omitempty"`
	StopSequences []string      `json:"stop_sequences,omitempty"`
}

type MessageItem struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type MessagesResponse struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Role       string                 `json:"role"`
	Content    []ContentBlockResponse `json:"content"`
	Model      string                 `json:"model"`
	StopReason string                 `json:"stop_reason,omitempty"`
	Usage      UsageResponse          `json:"usage"`
}

type ContentBlockResponse struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type UsageResponse struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
