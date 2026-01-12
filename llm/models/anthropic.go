package models

import "encoding/json"

// Anthropic API request/response models
// Based on: https://docs.anthropic.com/claude/reference/messages

// ContentBlock represents a content block in a message
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string        `json:"role"`
	Content []ContentBlock `json:"content"`
}

// Tool represents a tool/function definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolChoice represents tool choice configuration
type ToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// ThinkingConfig represents thinking mode configuration
type ThinkingConfig struct {
	Enabled bool `json:"enabled"`
}

// MessagesRequest represents an Anthropic messages API request
type MessagesRequest struct {
	Model         string          `json:"model"`
	MaxTokens     int             `json:"max_tokens"`
	Messages      []Message       `json:"messages"`
	System        string          `json:"system,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	Temperature   float64         `json:"temperature,omitempty"`
	TopP          float64         `json:"top_p,omitempty"`
	TopK          int             `json:"top_k,omitempty"`
	Tools         []Tool          `json:"tools,omitempty"`
	ToolChoice    *ToolChoice     `json:"tool_choice,omitempty"`
	Thinking      *ThinkingConfig `json:"thinking,omitempty"`
	Metadata      map[string]any  `json:"metadata,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// ContentBlockResponse represents a content block in the response
type ContentBlockResponse struct {
	Type      string      `json:"type"`
	Text      string      `json:"text,omitempty"`
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
}

// MessagesResponse represents an Anthropic messages API response
type MessagesResponse struct {
	ID          string                `json:"id"`
	Type        string                `json:"type"`
	Role        string                `json:"role"`
	Content     []ContentBlockResponse `json:"content"`
	Model       string                `json:"model"`
	StopReason  string                `json:"stop_reason,omitempty"`
	StopSequence string               `json:"stop_sequence,omitempty"`
	Usage       Usage                 `json:"usage"`
}

// TokenCountRequest represents a token count request
type TokenCountRequest struct {
	Model      string          `json:"model"`
	Messages   []Message       `json:"messages"`
	System     string          `json:"system,omitempty"`
	Tools      []Tool          `json:"tools,omitempty"`
	ToolChoice *ToolChoice     `json:"tool_choice,omitempty"`
	Thinking   *ThinkingConfig `json:"thinking,omitempty"`
}

// TokenCountResponse represents a token count response
type TokenCountResponse struct {
	InputTokens int `json:"input_tokens"`
}

// SSE Events for streaming
type MessageStartEvent struct {
	Type    string       `json:"type"`
	Message MessageSnapshot `json:"message"`
}

type MessageSnapshot struct {
	ID          string                `json:"id"`
	Type        string                `json:"type"`
	Role        string                `json:"role"`
	Content     []ContentBlockResponse `json:"content"`
	Model       string                `json:"model"`
	StopReason  *string               `json:"stop_reason"`
	StopSequence *string              `json:"stop_sequence"`
	Usage       Usage                 `json:"usage"`
}

type ContentBlockStartEvent struct {
	Type          string                `json:"type"`
	Index         int                   `json:"index"`
	ContentBlock  ContentBlockResponse  `json:"content_block"`
}

type ContentBlockDeltaEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type         string          `json:"type"`
		Text         string          `json:"text,omitempty"`
		PartialJSON  string          `json:"partial_json,omitempty"`
	} `json:"delta"`
}

type ContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type MessageDeltaEvent struct {
	Type    string `json:"type"`
	Delta   struct {
		StopReason   string  `json:"stop_reason"`
		StopSequence *string `json:"stop_sequence"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type MessageStopEvent struct {
	Type string `json:"type"`
}

type PingEvent struct {
	Type string `json:"type"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Type  string `json:"type"`
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
