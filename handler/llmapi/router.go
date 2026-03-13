package llmapi

import (
	"net/http"
	"strings"
)

// Router routes incoming requests to the correct API format handler
// (OpenAI, Anthropic, Gemini).
type Router struct {
	openai    http.Handler
	anthropic http.Handler
	gemini    http.Handler
}

// NewRouter creates a new LLM API router.
func NewRouter() *Router {
	return &Router{}
}

// ServeHTTP routes the request based on the path.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	switch {
	case strings.HasPrefix(path, "/v1/messages"):
		// Anthropic API
		if r.anthropic != nil {
			r.anthropic.ServeHTTP(w, req)
		}
	case strings.HasPrefix(path, "/v1/chat/completions") ||
		strings.HasPrefix(path, "/v1/models") ||
		strings.HasPrefix(path, "/v1/embeddings"):
		// OpenAI API
		if r.openai != nil {
			r.openai.ServeHTTP(w, req)
		}
	default:
		http.NotFound(w, req)
	}
}
