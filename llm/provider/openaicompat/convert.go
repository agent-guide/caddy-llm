package openaicompat

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/agent-guide/caddy-llm/llm/provider"
)

// BuildChatRequest converts an internal GenerateRequest to OpenAI wire format.
//
// Supported req.Metadata keys:
//   - "response_format": string ("text","json_object") or map[string]any with
//     optional "json_schema" sub-map → sets ChatRequest.ResponseFormat.
//   - "reasoning_effort": string ("low","medium","high") → sets ReasoningEffort
//     and remaps MaxTokens to MaxCompletionTokens (required by o1/o3 models).
func BuildChatRequest(req *provider.GenerateRequest, stream bool) (*ChatRequest, error) {
	msgs, err := buildMessages(req)
	if err != nil {
		return nil, err
	}

	chatReq := &ChatRequest{
		Model:       req.Model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Stream:      stream,
	}

	// Include token usage in the final streaming chunk.
	if stream {
		chatReq.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	if len(req.Tools) > 0 {
		chatReq.Tools = convertTools(req.Tools)
	}
	if req.ToolChoice != nil {
		chatReq.ToolChoice = convertToolChoice(req.ToolChoice)
	}

	// Apply provider-extension metadata.
	if req.Metadata != nil {
		if rf, ok := req.Metadata["response_format"]; ok {
			chatReq.ResponseFormat = parseResponseFormat(rf)
		}
		if re, ok := req.Metadata["reasoning_effort"]; ok {
			if s, ok := re.(string); ok && s != "" {
				chatReq.ReasoningEffort = s
				// Reasoning models use max_completion_tokens instead of max_tokens.
				chatReq.MaxCompletionTokens = req.MaxTokens
				chatReq.MaxTokens = 0
			}
		}
	}

	return chatReq, nil
}

func buildMessages(req *provider.GenerateRequest) ([]ChatMessage, error) {
	var msgs []ChatMessage

	if req.System != "" {
		msgs = append(msgs, ChatMessage{
			Role:    "system",
			Content: marshalString(req.System),
		})
	}

	for _, m := range req.Messages {
		converted, err := convertMessage(m)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, converted...)
	}

	return msgs, nil
}

// convertMessage converts a single internal Message into one or more OpenAI ChatMessages.
// Tool results in user messages become separate "tool" role messages.
func convertMessage(m provider.Message) ([]ChatMessage, error) {
	switch m.Role {
	case "user":
		return convertUserMessage(m), nil
	case "assistant":
		return convertAssistantMessage(m)
	}
	return nil, fmt.Errorf("openaicompat: unsupported message role: %s", m.Role)
}

func convertUserMessage(m provider.Message) []ChatMessage {
	var parts []ContentPart
	var toolMsgs []ChatMessage

	for _, block := range m.Content {
		switch block.Type {
		case "text":
			parts = append(parts, ContentPart{Type: "text", Text: block.Text})
		case "image":
			// block.Text holds the image URL or base64 data URI.
			parts = append(parts, ContentPart{
				Type:     "image_url",
				ImageURL: &ImageURL{URL: block.Text, Detail: "auto"},
			})
		case "tool_result":
			if block.ToolResult != nil {
				toolMsgs = append(toolMsgs, ChatMessage{
					Role:       "tool",
					Content:    marshalString(block.ToolResult.Content),
					ToolCallID: block.ToolResult.ToolUseID,
				})
			}
		}
	}

	var result []ChatMessage
	if len(parts) > 0 {
		result = append(result, ChatMessage{
			Role:    "user",
			Content: marshalParts(parts),
		})
	}
	return append(result, toolMsgs...)
}

func convertAssistantMessage(m provider.Message) ([]ChatMessage, error) {
	var text string
	var toolCalls []ToolCall

	for _, block := range m.Content {
		switch block.Type {
		case "text":
			text += block.Text
		case "tool_use":
			if block.ToolUse != nil {
				argsJSON, err := json.Marshal(block.ToolUse.Input)
				if err != nil {
					return nil, fmt.Errorf("openaicompat: marshal tool input: %w", err)
				}
				toolCalls = append(toolCalls, ToolCall{
					ID:   block.ToolUse.ID,
					Type: "function",
					Function: FunctionCall{
						Name:      block.ToolUse.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
	}

	msg := ChatMessage{Role: "assistant", ToolCalls: toolCalls}
	if text != "" {
		msg.Content = marshalString(text)
	}
	return []ChatMessage{msg}, nil
}

func convertTools(tools []provider.Tool) []ChatTool {
	out := make([]ChatTool, len(tools))
	for i, t := range tools {
		out[i] = ChatTool{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
	}
	return out
}

func convertToolChoice(tc *provider.ToolChoice) any {
	switch tc.Type {
	case "auto":
		return "auto"
	case "any":
		return "required" // OpenAI uses "required" for "any"
	case "tool":
		return map[string]any{
			"type":     "function",
			"function": map[string]string{"name": tc.Name},
		}
	}
	return "auto"
}

// parseResponseFormat converts a req.Metadata["response_format"] value into a ResponseFormat.
// Accepts a plain string ("text", "json_object") or a map with optional json_schema spec.
func parseResponseFormat(v any) *ResponseFormat {
	switch val := v.(type) {
	case string:
		return &ResponseFormat{Type: val}
	case map[string]any:
		rf := &ResponseFormat{}
		if t, ok := val["type"].(string); ok {
			rf.Type = t
		}
		if js, ok := val["json_schema"].(map[string]any); ok {
			spec := &JSONSchemaSpec{}
			if name, ok := js["name"].(string); ok {
				spec.Name = name
			}
			if schema, ok := js["schema"].(map[string]any); ok {
				spec.Schema = schema
			}
			if strict, ok := js["strict"].(bool); ok {
				spec.Strict = strict
			}
			rf.JSONSchema = spec
		}
		return rf
	}
	return nil
}

// ConvertResponse converts an OpenAI ChatResponse to the internal GenerateResponse.
func ConvertResponse(resp *ChatResponse, headers http.Header) *provider.GenerateResponse {
	out := &provider.GenerateResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Headers: headers,
		Usage: provider.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	if len(resp.Choices) > 0 {
		out.StopReason = resp.Choices[0].FinishReason
		out.Content = convertResponseContent(resp.Choices[0].Message)
	}

	return out
}

func convertResponseContent(msg ChatMessage) []provider.ContentBlock {
	var blocks []provider.ContentBlock

	// Text content: try to unmarshal as plain string first.
	if len(msg.Content) > 0 {
		var s string
		if err := json.Unmarshal(msg.Content, &s); err == nil && s != "" {
			blocks = append(blocks, provider.ContentBlock{Type: "text", Text: s})
		}
	}

	// Tool calls from the model.
	for _, tc := range msg.ToolCalls {
		var input map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
		blocks = append(blocks, provider.ContentBlock{
			Type: "tool_use",
			ToolUse: &provider.ToolUse{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			},
		})
	}

	return blocks
}

// --- Content marshaling helpers ---

func marshalString(s string) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

func marshalParts(parts []ContentPart) json.RawMessage {
	// Use plain string for single text parts — better compatibility.
	if len(parts) == 1 && parts[0].Type == "text" {
		return marshalString(parts[0].Text)
	}
	b, _ := json.Marshal(parts)
	return b
}
