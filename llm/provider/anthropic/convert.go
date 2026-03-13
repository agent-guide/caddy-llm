package anthropic

import (
	"net/http"

	"github.com/agent-guide/caddy-llm/llm/provider"
)

// BuildRequest converts an internal GenerateRequest to Anthropic wire format.
func BuildRequest(req *provider.GenerateRequest, stream bool) *MessagesRequest {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024 // max_tokens is required by the Anthropic API
	}

	anthReq := &MessagesRequest{
		Model:       req.Model,
		Messages:    convertMessages(req.Messages),
		System:      req.System,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		TopK:        req.TopK,
		Stream:      stream,
	}

	if len(req.Tools) > 0 {
		anthReq.Tools = convertTools(req.Tools)
	}
	if req.ToolChoice != nil {
		anthReq.ToolChoice = convertToolChoice(req.ToolChoice)
	}
	if req.Thinking != nil && req.Thinking.Enabled {
		anthReq.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: req.Thinking.BudgetTokens,
		}
	}

	return anthReq
}

func convertMessages(msgs []provider.Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, Message{
			Role:    m.Role,
			Content: convertContent(m.Content),
		})
	}
	return out
}

func convertContent(blocks []provider.ContentBlock) []ContentBlock {
	out := make([]ContentBlock, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "text":
			out = append(out, ContentBlock{Type: "text", Text: b.Text})
		case "tool_use":
			if b.ToolUse != nil {
				out = append(out, ContentBlock{
					Type:  "tool_use",
					ID:    b.ToolUse.ID,
					Name:  b.ToolUse.Name,
					Input: b.ToolUse.Input,
				})
			}
		case "tool_result":
			if b.ToolResult != nil {
				out = append(out, ContentBlock{
					Type:      "tool_result",
					ToolUseID: b.ToolResult.ToolUseID,
					Content:   b.ToolResult.Content,
					IsError:   b.ToolResult.IsError,
				})
			}
		}
	}
	return out
}

func convertTools(tools []provider.Tool) []Tool {
	out := make([]Tool, len(tools))
	for i, t := range tools {
		out[i] = Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}
	return out
}

func convertToolChoice(tc *provider.ToolChoice) *ToolChoice {
	return &ToolChoice{Type: tc.Type, Name: tc.Name}
}

// ConvertResponse converts an Anthropic MessagesResponse to the internal GenerateResponse.
func ConvertResponse(resp *MessagesResponse, headers http.Header) *provider.GenerateResponse {
	out := &provider.GenerateResponse{
		ID:         resp.ID,
		Model:      resp.Model,
		StopReason: resp.StopReason,
		Headers:    headers,
		Usage: provider.Usage{
			InputTokens:              resp.Usage.InputTokens,
			OutputTokens:             resp.Usage.OutputTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		},
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			out.Content = append(out.Content, provider.ContentBlock{
				Type: "text",
				Text: block.Text,
			})
		case "tool_use":
			out.Content = append(out.Content, provider.ContentBlock{
				Type: "tool_use",
				ToolUse: &provider.ToolUse{
					ID:    block.ID,
					Name:  block.Name,
					Input: block.Input,
				},
			})
		case "thinking":
			// Pass through thinking blocks as text for now.
			if block.Thinking != "" {
				out.Content = append(out.Content, provider.ContentBlock{
					Type: "thinking",
					Text: block.Thinking,
				})
			}
		}
	}

	return out
}
