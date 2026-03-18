// Package openaibase provides shared OpenAI-compatible wire types
// still used for model listing and embeddings.
package openaibase

// Usage holds token counts from a response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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
