package manager

import "fmt"

// Error describes a credential-related failure in a provider-agnostic format.
type Error struct {
	// Code is a short machine-readable identifier.
	Code string `json:"code,omitempty"`
	// Message is a human-readable description of the failure.
	Message string `json:"message"`
	// Retryable indicates whether a retry might fix the issue automatically.
	Retryable bool `json:"retryable"`
	// HTTPStatus optionally records an HTTP-like status code for the error.
	HTTPStatus int `json:"http_status,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

// StatusCode implements optional status accessor for manager decision making.
func (e *Error) StatusCode() int {
	if e == nil {
		return 0
	}
	return e.HTTPStatus
}

// cooldownError is returned when all credentials for a model are in cooldown.
type cooldownError struct {
	model    string
	provider string
	resetIn  string // formatted duration
}

func (e *cooldownError) Error() string {
	if e == nil {
		return ""
	}
	msg := fmt.Sprintf("all credentials for model %s are cooling down", e.model)
	if e.provider != "" {
		msg += fmt.Sprintf(" via provider %s", e.provider)
	}
	if e.resetIn != "" {
		msg += fmt.Sprintf(", retry after %s", e.resetIn)
	}
	return msg
}

func (e *cooldownError) StatusCode() int {
	return 429
}
