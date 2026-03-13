package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/agent-guide/caddy-llm/llm/models"
)

// Converter handles format conversion between Anthropic and gollm formats
type Converter struct {
	provider      string
	modelMappings map[string]string
}

// NewConverter creates a new format converter
func NewConverter(provider string, modelMappings map[string]string) *Converter {
	return &Converter{
		provider:      provider,
		modelMappings: modelMappings,
	}
}

// AnthropicToGollm converts an Anthropic request to a gollm prompt
func (c *Converter) AnthropicToGollm(ctx context.Context, req *models.MessagesRequest) (string, error) {
	var prompt strings.Builder

	// Add system message if present
	if req.System != "" {
		prompt.WriteString(fmt.Sprintf("System: %s\n\n", req.System))
	}

	// Add conversation messages
	for _, msg := range req.Messages {
		prompt.WriteString(fmt.Sprintf("%s: ", msg.Role))
		for _, block := range msg.Content {
			if block.Type == "text" {
				prompt.WriteString(block.Text)
			}
		}
		prompt.WriteString("\n")
	}

	return prompt.String(), nil
}

// GollmToAnthropic converts a gollm response to Anthropic format
func (c *Converter) GollmToAnthropic(response string, originalReq *models.MessagesRequest) (*models.MessagesResponse, error) {
	// Create Anthropic-style response
	resp := &models.MessagesResponse{
		ID:    generateMessageID(),
		Type:  "message",
		Role:  "assistant",
		Model: originalReq.Model,
		Content: []models.ContentBlockResponse{
			{
				Type: "text",
				Text: response,
			},
		},
		StopReason: "end_turn",
		Usage: models.Usage{
			InputTokens:  estimateTokens(response), // Rough estimate
			OutputTokens: len(response) / 4,        // Rough estimate
		},
	}

	return resp, nil
}

// ConvertTools converts Anthropic tools to gollm tools format
func (c *Converter) ConvertTools(tools []models.Tool) ([]map[string]interface{}, error) {
	gollmTools := make([]map[string]interface{}, len(tools))

	for i, tool := range tools {
		gollmTool := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.InputSchema,
		}
		gollmTools[i] = gollmTool
	}

	return gollmTools, nil
}

// ConvertToolChoice converts Anthropic tool choice to gollm format
func (c *Converter) ConvertToolChoice(tc *models.ToolChoice) interface{} {
	if tc == nil {
		return "auto"
	}

	switch tc.Type {
	case "auto":
		return "auto"
	case "any":
		return "any"
	case "tool":
		return map[string]interface{}{
			"type":     "function",
			"function": map[string]string{"name": tc.Name},
		}
	default:
		return "auto"
	}
}

// MapModel maps Anthropic model names to provider-specific models
func (c *Converter) MapModel(model string) string {
	// Parse the model name
	baseModel := models.ParseModelName(model)
	modelType := models.DetectModelType(baseModel)

	// Get the mapped model
	mappedModel := models.GetMappedModel(c.provider, modelType, c.modelMappings)

	// Add provider prefix if needed
	if !strings.Contains(mappedModel, "/") {
		switch c.provider {
		case "openai", "groq", "ollama", "mistral":
			mappedModel = c.provider + "/" + mappedModel
		case "gemini":
			mappedModel = "gemini/" + mappedModel
		case "anthropic":
			mappedModel = "anthropic/" + mappedModel
		}
	}

	return mappedModel
}

// ProcessStreamChunk processes a streaming chunk from gollm and converts to Anthropic SSE format
func (c *Converter) ProcessStreamChunk(chunk string) ([]string, error) {
	var events []string

	// For now, just return the chunk as a text delta
	// In a real implementation, this would parse the gollm streaming format
	// and convert it to proper Anthropic SSE events

	deltaEvent := map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]string{
			"type": "text_delta",
			"text": chunk,
		},
	}
	jsonData, _ := json.Marshal(deltaEvent)
	event := fmt.Sprintf("event: content_block_delta\ndata: %s\n\n", string(jsonData))

	events = append(events, event)
	return events, nil
}

// GenerateStreamStartEvents generates the initial SSE events for a streaming response
func (c *Converter) GenerateStreamStartEvents(req *models.MessagesRequest) []string {
	messageID := generateMessageID()
	events := []string{}

	// message_start event
	messageStartEvent := map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":          messageID,
			"type":        "message",
			"role":        "assistant",
			"content":     []interface{}{},
			"model":       req.Model,
			"stop_reason": nil,
			"usage": map[string]int{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	}
	jsonData, _ := json.Marshal(messageStartEvent)
	events = append(events, fmt.Sprintf("event: message_start\ndata: %s\n\n", string(jsonData)))

	// content_block_start event
	contentBlockStartEvent := map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]string{
			"type": "text",
			"text": "",
		},
	}
	jsonData, _ = json.Marshal(contentBlockStartEvent)
	events = append(events, fmt.Sprintf("event: content_block_start\ndata: %s\n\n", string(jsonData)))

	// ping event
	events = append(events, "event: ping\ndata: {\"type\": \"ping\"}\n\n")

	return events
}

// GenerateStreamEndEvents generates the final SSE events for a streaming response
func (c *Converter) GenerateStreamEndEvents(outputTokens int) []string {
	events := []string{
		// content_block_stop event
		"event: content_block_stop\ndata: {\"type\": \"content_block_stop\", \"index\": 0}\n\n",
	}

	// message_delta event
	messageDeltaEvent := map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]string{
			"stop_reason": "end_turn",
		},
		"usage": map[string]int{
			"output_tokens": outputTokens,
		},
	}
	jsonData, _ := json.Marshal(messageDeltaEvent)
	events = append(events, fmt.Sprintf("event: message_delta\ndata: %s\n\n", string(jsonData)))

	// message_stop event
	events = append(events, "event: message_stop\ndata: {\"type\": \"message_stop\"}\n\n")

	// DONE marker
	events = append(events, "data: [DONE]\n\n")

	return events
}

// Helper functions

func generateMessageID() string {
	// Generate a unique message ID
	// In production, use a proper UUID generator
	return fmt.Sprintf("msg_%x", randInt())
}

func randInt() int64 {
	// Simple random number generator
	// In production, use crypto/rand or similar
	return int64(12345678)
}

func estimateTokens(text string) int {
	// Rough token estimation: ~4 characters per token
	return len(text) / 4
}

// ParseContentBlock parses content from various formats
func ParseContentBlock(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var result string
		for _, item := range v {
			if block, ok := item.(map[string]interface{}); ok {
				if blockType, ok := block["type"].(string); ok && blockType == "text" {
					if text, ok := block["text"].(string); ok {
						result += text + "\n"
					}
				}
			}
		}
		return result
	case []models.ContentBlock:
		var result string
		for _, block := range v {
			if block.Type == "text" {
				result += block.Text + "\n"
			}
		}
		return result
	default:
		return ""
	}
}

// StreamResponseWriter handles writing SSE events
type StreamResponseWriter struct {
	writer io.Writer
}

func NewStreamResponseWriter(w io.Writer) *StreamResponseWriter {
	return &StreamResponseWriter{writer: w}
}

func (s *StreamResponseWriter) WriteEvent(eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))
	_, err = io.WriteString(s.writer, event)
	return err
}

func (s *StreamResponseWriter) WriteRaw(data string) error {
	_, err := io.WriteString(s.writer, data)
	return err
}

func (s *StreamResponseWriter) Flush() {
	if flusher, ok := s.writer.(interface{ Flush() }); ok {
		flusher.Flush()
	}
}
