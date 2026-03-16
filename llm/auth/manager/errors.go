package manager

import "fmt"

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
