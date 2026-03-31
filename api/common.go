package api

import (
	"errors"
	"net/http"

	"github.com/agent-guide/caddy-agent-gateway/gateway"
)

// RouteIDAcceptor is implemented by handlers that accept a route id parsed from
// the shared handle_llm_api directive.
type RouteIDAcceptor interface {
	SetRouteID(string)
}

func ResolveRequest(httpReq *http.Request, model string, stream bool, routeID string) (*gateway.ResolvedRoute, error) {
	return gateway.GlobalAgentGateway().ResolveProvider(httpReq.Context(), routeID, gateway.ResolveRequest{
		HTTPRequest: httpReq,
		Model:       model,
		Stream:      stream,
	})
}

func StatusCode(err error) int {
	type statusCoder interface {
		StatusCode() int
	}
	var sc statusCoder
	if errors.As(err, &sc) {
		return sc.StatusCode()
	}
	return http.StatusBadGateway
}
