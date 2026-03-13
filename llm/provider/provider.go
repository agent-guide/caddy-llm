package provider

import (
	"context"
	"net/http"
	"time"
)

// Provider defines the interface for LLM providers.
//
// Design follows CLIProxyAPI's approach: a small, focused interface covering
// the core operations every provider must support. Providers that support
// additional capabilities (embeddings, image generation, etc.) may implement
// optional capability interfaces (EmbeddingProvider, etc.).
type Provider interface {
	// Generate performs a non-streaming completion and returns the full response.
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// Stream performs a streaming completion and returns a StreamResult containing
	// upstream response headers and a channel of chunks.
	Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error)

	// CountTokens returns the token count for the given request without generating
	// a response. Used by /v1/messages/count_tokens and the agent orchestrator.
	CountTokens(ctx context.Context, req *GenerateRequest) (*TokenCountResponse, error)

	// ListModels returns the list of models available from this provider.
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// Capabilities returns what this provider instance supports.
	Capabilities() ProviderCapabilities
}

// EmbeddingProvider is an optional interface for providers that support embeddings.
// The memory module uses this to generate vectors for storage and search.
type EmbeddingProvider interface {
	Provider
	Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error)
}

// StatusError is implemented by errors that carry an HTTP status code.
// Provider implementations should return StatusError so that the handler layer
// can make informed retry and degradation decisions (e.g. 401→disable key,
// 429→backoff, 503→try next provider).
type StatusError interface {
	error
	StatusCode() int
}

// ProviderCapabilities describes what a provider instance supports.
type ProviderCapabilities struct {
	Streaming       bool
	Tools           bool
	Vision          bool
	Audio           bool
	Embeddings      bool
	FineTuning      bool
	ContextWindow   int
	MaxOutputTokens int
}

// ProviderConfig contains configuration for a provider instance.
type ProviderConfig struct {
	// Name is the registered provider name (e.g. "openai", "anthropic").
	Name string `json:"name"`
	// APIKey is the provider API key. May be empty for local providers (Ollama).
	APIKey string `json:"api_key,omitempty"`
	// BaseURL overrides the provider's default API base URL.
	BaseURL string `json:"base_url,omitempty"`
	// DefaultModel is used when the request does not specify a model.
	DefaultModel string `json:"default_model,omitempty"`
	// Network contains HTTP client configuration (timeout, retry, proxy).
	Network NetworkConfig `json:"network"`
	// Options holds provider-specific extra configuration.
	Options map[string]any `json:"options,omitempty"`
}

// NetworkConfig controls HTTP client behavior for a provider.
// Borrowed from Bifrost's NetworkConfig, simplified for our needs.
type NetworkConfig struct {
	// TimeoutSeconds is the per-request HTTP timeout. Default: 120.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	// MaxRetries is the number of automatic retries on transient errors. Default: 3.
	MaxRetries int `json:"max_retries,omitempty"`
	// RetryDelaySeconds is the base delay between retries. Default: 1.
	RetryDelaySeconds int `json:"retry_delay_seconds,omitempty"`
	// ProxyURL is an optional HTTP/HTTPS/SOCKS5 proxy URL.
	ProxyURL string `json:"proxy_url,omitempty"`
	// ExtraHeaders are additional HTTP headers sent with every provider request.
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}

// Defaults fills in zero values with sensible defaults.
func (c *NetworkConfig) Defaults() {
	if c.TimeoutSeconds == 0 {
		c.TimeoutSeconds = 120
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.RetryDelaySeconds == 0 {
		c.RetryDelaySeconds = 1
	}
}

// Timeout returns the configured timeout as a time.Duration.
func (c *NetworkConfig) Timeout() time.Duration {
	if c.TimeoutSeconds == 0 {
		return 120 * time.Second
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// --- Request / Response types ---

// GenerateRequest is the unified internal request format passed to providers.
type GenerateRequest struct {
	Model       string
	Messages    []Message
	System      string // system prompt (Anthropic style; providers that use system in messages will convert)
	Tools       []Tool
	ToolChoice  *ToolChoice
	Temperature *float64
	MaxTokens   int
	TopP        *float64
	TopK        *int
	Stop        []string
	Stream      bool
	Thinking    *ThinkingConfig
	Metadata    map[string]any
}

// GenerateResponse is the unified internal response format returned by providers.
type GenerateResponse struct {
	ID         string
	Model      string
	Content    []ContentBlock
	StopReason string
	Usage      Usage
	// Headers carries upstream HTTP response headers (e.g. x-request-id, ratelimit-*).
	Headers http.Header
}

// StreamChunk is a single chunk emitted during streaming.
// Payload is the raw provider JSON chunk — the llmapi converter handles formatting.
type StreamChunk struct {
	// Payload is the raw provider-format chunk bytes.
	Payload []byte
	// Err is non-nil for terminal stream errors.
	Err error
}

// StreamResult wraps streaming output: upstream headers captured before streaming
// begins, and a channel of chunks.
type StreamResult struct {
	// Headers carries upstream response headers (captured before streaming starts).
	Headers http.Header
	// Chunks is the channel of raw streaming chunks from the provider.
	Chunks <-chan StreamChunk
}

// TokenCountResponse is returned by CountTokens.
type TokenCountResponse struct {
	InputTokens int
}

// EmbedRequest is the request to generate vector embeddings.
type EmbedRequest struct {
	// Model is the embedding model to use. Leave empty to use provider default.
	Model string
	// Texts are the strings to embed.
	Texts []string
}

// EmbedResponse contains the generated embeddings.
type EmbedResponse struct {
	Embeddings [][]float64
	Model      string
	Usage      Usage
}

// --- Message types ---

// Message represents a conversation turn.
type Message struct {
	Role    string
	Content []ContentBlock
}

// ContentBlock is a typed block within a message.
type ContentBlock struct {
	Type       string
	Text       string
	ToolUse    *ToolUse
	ToolResult *ToolResult
}

// ThinkingConfig controls extended thinking mode (Anthropic).
type ThinkingConfig struct {
	Enabled    bool
	BudgetTokens int
}

// Tool represents a tool/function that the model can call.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// ToolChoice controls how the model selects tools.
type ToolChoice struct {
	// Type is "auto", "any", or "tool".
	Type string
	// Name is the specific tool name when Type == "tool".
	Name string
}

// ToolUse is a tool call request from the model.
type ToolUse struct {
	ID    string
	Name  string
	Input map[string]any
}

// ToolResult is the caller's response to a ToolUse.
type ToolResult struct {
	ToolUseID string
	Content   string
	IsError   bool
}

// ModelInfo describes a model available from a provider.
type ModelInfo struct {
	ID           string
	Name         string
	Description  string
	Capabilities ProviderCapabilities
}

// Usage contains token consumption information.
type Usage struct {
	InputTokens              int
	OutputTokens             int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}

// --- Error helpers ---

// httpStatusError is a simple StatusError implementation for use by providers.
type httpStatusError struct {
	code    int
	message string
}

// NewStatusError creates a StatusError with the given HTTP status code.
func NewStatusError(code int, message string) StatusError {
	return &httpStatusError{code: code, message: message}
}

func (e *httpStatusError) Error() string  { return e.message }
func (e *httpStatusError) StatusCode() int { return e.code }
