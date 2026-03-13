package httputil

import (
	"fmt"
	"io"
	"net/http"

	"github.com/agent-guide/caddy-llm/llm/provider"
)

// CheckResponse returns a StatusError if the HTTP response status is not 2xx.
// It reads up to 4 KB of the body to include in the error message.
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return provider.NewStatusError(resp.StatusCode,
		fmt.Sprintf("upstream %d: %s", resp.StatusCode, string(body)))
}
