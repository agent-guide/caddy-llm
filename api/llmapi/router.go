package llmapi

import (
	"net/http"

	"go.uber.org/zap"
)

// Router routes incoming requests to the first compatible LLM API handler that matches.
type Router struct {
	handlers []LLMApiHandler
	logger   *zap.Logger
}

// NewRouter creates a new LLM API router.
func NewRouter(handlers []LLMApiHandler, logger *zap.Logger) *Router {
	return &Router{handlers: handlers, logger: logger}
}

// ServeHTTP routes the request based on handler-specific match rules.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, handler := range r.handlers {
		if handler.MatchLLMApi(req) {
			if err := handler.ServeLLMApi(w, req); err != nil && r.logger != nil {
				r.logger.Error("serve llm api", zap.String("api", handler.Name()), zap.Error(err))
			}
			return
		}
	}
	http.NotFound(w, req)
}
