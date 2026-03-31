package gateway

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
)

// RouteLoader resolves the latest persisted route definition for a route ID.
type RouteLoader func(ctx context.Context, routeID string) (*Route, error)

// ResolveRequest captures the request attributes required for route resolution.
type ResolveRequest struct {
	HTTPRequest *http.Request
	Model       string
	Stream      bool
}

// ResolvedRoute contains the chosen route, consumer, and provider for a request.
type ResolvedRoute struct {
	Route        Route
	LocalAPIKey  *LocalAPIKey
	ProviderName string
	Provider     provider.Provider
}

// HTTPError describes a request resolution failure with an HTTP status code.
type HTTPError struct {
	status int
	msg    string
}

func (e *HTTPError) Error() string   { return e.msg }
func (e *HTTPError) StatusCode() int { return e.status }

// NewHTTPError constructs an HTTPError with the given status and message.
func NewHTTPError(status int, msg string) error {
	return &HTTPError{status: status, msg: msg}
}

func validateRequestPolicy(route Route, key *LocalAPIKey, req ResolveRequest) error {
	if req.Model != "" {
		if len(route.Policy.AllowedModels) > 0 && !slices.Contains(route.Policy.AllowedModels, req.Model) {
			return &HTTPError{status: http.StatusForbidden, msg: fmt.Sprintf("model %q is not allowed on route %q", req.Model, route.ID)}
		}
		if key != nil && key.PolicyOverride != nil && len(key.PolicyOverride.AllowedModels) > 0 &&
			!slices.Contains(key.PolicyOverride.AllowedModels, req.Model) {
			return &HTTPError{status: http.StatusForbidden, msg: fmt.Sprintf("model %q is not allowed for this local api key", req.Model)}
		}
	}

	if req.Stream {
		if route.Policy.AllowStreaming != nil && !*route.Policy.AllowStreaming {
			return &HTTPError{status: http.StatusForbidden, msg: "streaming is disabled on this route"}
		}
		if key != nil && key.PolicyOverride != nil && key.PolicyOverride.AllowStreaming != nil && !*key.PolicyOverride.AllowStreaming {
			return &HTTPError{status: http.StatusForbidden, msg: "streaming is disabled for this local api key"}
		}
	}

	return nil
}

func matchesConditions(conditions TargetConditions, req ResolveRequest) bool {
	if len(conditions.Models) > 0 && req.Model != "" && !slices.Contains(conditions.Models, req.Model) {
		return false
	}
	if conditions.RequireStreaming != nil && *conditions.RequireStreaming != req.Stream {
		return false
	}
	return true
}

func extractAPIKey(r *http.Request) string {
	if r == nil {
		return ""
	}
	if key := strings.TrimSpace(r.Header.Get("x-api-key")); key != "" {
		return key
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}
