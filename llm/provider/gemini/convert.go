package gemini

import (
	"net/http"
	"strings"

	"github.com/agent-guide/caddy-llm/llm/provider"
)

// BuildRequest converts an internal GenerateRequest to Gemini wire format.
func BuildRequest(req *provider.GenerateRequest) *GenerateContentRequest {
	gemReq := &GenerateContentRequest{
		Contents: convertMessages(req.Messages),
	}

	if req.System != "" {
		gemReq.SystemInstruction = &SystemContent{
			Parts: []Part{{Text: req.System}},
		}
	}

	if req.Temperature != nil || req.TopP != nil || req.TopK != nil || req.MaxTokens > 0 || len(req.Stop) > 0 {
		gemReq.GenerationConfig = &GenerationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			TopK:            req.TopK,
			MaxOutputTokens: req.MaxTokens,
			StopSequences:   req.Stop,
		}
	}

	if len(req.Tools) > 0 {
		gemReq.Tools = []GeminiTool{{FunctionDeclarations: convertTools(req.Tools)}}
	}

	if req.ToolChoice != nil {
		gemReq.ToolConfig = convertToolChoice(req.ToolChoice)
	}

	return gemReq
}

func convertMessages(msgs []provider.Message) []Content {
	out := make([]Content, 0, len(msgs))
	for _, m := range msgs {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		out = append(out, Content{
			Role:  role,
			Parts: convertParts(m.Content),
		})
	}
	return out
}

func convertParts(blocks []provider.ContentBlock) []Part {
	out := make([]Part, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "text":
			out = append(out, Part{Text: b.Text})
		case "tool_use":
			if b.ToolUse != nil {
				out = append(out, Part{
					FunctionCall: &FunctionCall{
						Name: b.ToolUse.Name,
						Args: b.ToolUse.Input,
					},
				})
			}
		case "tool_result":
			if b.ToolResult != nil {
				out = append(out, Part{
					FunctionResponse: &FunctionResponse{
						Name:     b.ToolResult.ToolUseID,
						Response: map[string]any{"content": b.ToolResult.Content},
					},
				})
			}
		}
	}
	return out
}

func convertTools(tools []provider.Tool) []FunctionDeclaration {
	out := make([]FunctionDeclaration, len(tools))
	for i, t := range tools {
		out[i] = FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		}
	}
	return out
}

func convertToolChoice(tc *provider.ToolChoice) *ToolConfig {
	cfg := &FunctionCallingConfig{}
	switch tc.Type {
	case "auto":
		cfg.Mode = "AUTO"
	case "any":
		cfg.Mode = "ANY"
	case "tool":
		cfg.Mode = "ANY"
		cfg.AllowedFunctionNames = []string{tc.Name}
	default:
		cfg.Mode = "AUTO"
	}
	return &ToolConfig{FunctionCallingConfig: cfg}
}

// ConvertResponse converts a Gemini GenerateContentResponse to the internal GenerateResponse.
func ConvertResponse(resp *GenerateContentResponse, model string, headers http.Header) *provider.GenerateResponse {
	out := &provider.GenerateResponse{
		Model:   model,
		Headers: headers,
		Usage: provider.Usage{
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
		},
	}

	if len(resp.Candidates) > 0 {
		cand := resp.Candidates[0]
		out.StopReason = strings.ToLower(cand.FinishReason)
		for _, part := range cand.Content.Parts {
			switch {
			case part.Text != "":
				out.Content = append(out.Content, provider.ContentBlock{
					Type: "text",
					Text: part.Text,
				})
			case part.FunctionCall != nil:
				out.Content = append(out.Content, provider.ContentBlock{
					Type: "tool_use",
					ToolUse: &provider.ToolUse{
						Name:  part.FunctionCall.Name,
						Input: part.FunctionCall.Args,
					},
				})
			}
		}
	}

	return out
}
