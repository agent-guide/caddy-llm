package api

import (
	"errors"
	"net/http"
)

// RouteIDAcceptor is implemented by handlers that accept a route id parsed from
// the shared handle_llm_api directive.
type RouteIDAcceptor interface {
	SetRouteID(string)
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
