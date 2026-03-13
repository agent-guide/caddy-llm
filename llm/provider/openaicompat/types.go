// Package openaicompat provides a shared implementation for OpenAI-compatible APIs.
// Used by openai, groq, mistral, openrouter, and ollama providers.
package openaicompat

import "encoding/json"

// --- Chat request types ---

// ChatRequest is the OpenAI Chat Completions request body.
// All OpenAI API extension fields are omitempty so providers that don't
// support them simply ignore the zero values.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`

	// Standard sampling parameters.
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`

	// Token limits.
	// MaxTokens is used by classic models; MaxCompletionTokens by reasoning models (o1/o3).
	MaxTokens           int `json:"max_tokens,omitempty"`
	MaxCompletionTokens int `json:"max_completion_tokens,omitempty"`

	// Streaming.
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`

	// Tool calling.
	Tools      []ChatTool `json:"tools,omitempty"`
	ToolChoice any        `json:"tool_choice,omitempty"`

	// Output format control (structured output / JSON mode).
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Reasoning effort for o1/o3-series models ("low", "medium", "high").
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

// StreamOptions configures per-request streaming behaviour.
// Set IncludeUsage to receive a final SSE chunk with token usage data.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ResponseFormat controls model output format.
// Type may be "text", "json_object", or "json_schema".
type ResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *JSONSchemaSpec `json:"json_schema,omitempty"`
}

// JSONSchemaSpec is used with ResponseFormat{Type: "json_schema"} for structured output.
type JSONSchemaSpec struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict,omitempty"`
}

// ChatMessage is a single message in the conversation.
// Content is json.RawMessage because it may be a plain string or []ContentPart.
type ChatMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	Name       string          `json:"name,omitempty"`
}

// ContentPart is a typed block inside a multi-part message content array.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL holds an image URL or base64 data URI for vision requests.
// Detail controls decode resolution: "auto" (default), "low", or "high".
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// ChatTool represents a tool available to the model.
type ChatTool struct {
	Type     string      `json:"type"` // always "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef describes a callable function.
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall is a tool invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall carries the function name and JSON-encoded arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded string
}

// --- Chat response types ---

// ChatResponse is the non-streaming chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is a single completion candidate.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage holds token counts from a response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// --- Streaming types ---

// StreamResponse is a single SSE chunk in a streaming completion.
// Usage is non-nil only in the final chunk when StreamOptions.IncludeUsage is true.
type StreamResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// StreamChoice is a single delta within a streaming chunk.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        ChatMessage `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// --- Models list ---

// ModelsResponse is the response from GET /v1/models.
type ModelsResponse struct {
	Data []ModelData `json:"data"`
}

// ModelData describes a single model entry.
type ModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// --- Embeddings ---

// EmbedRequest is the request body for POST /v1/embeddings.
type EmbedRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

// EmbedResponse is the response from POST /v1/embeddings.
type EmbedResponse struct {
	Data  []EmbedData `json:"data"`
	Model string      `json:"model"`
	Usage Usage       `json:"usage"`
}

// EmbedData holds a single embedding vector with its index.
type EmbedData struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}
