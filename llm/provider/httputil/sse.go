// Package httputil provides shared HTTP utilities for provider implementations.
package httputil

import (
	"bufio"
	"bytes"
	"io"
)

const sseMaxTokenSize = 10 * 1024 * 1024 // 10 MB

var (
	dataPrefix = []byte("data: ")
	doneMarker = []byte("[DONE]")
)

// NewSSEScanner returns a bufio.Scanner configured for reading SSE streams.
// The 10 MB buffer accommodates large JSON chunks (e.g. base64 image data).
func NewSSEScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, sseMaxTokenSize), sseMaxTokenSize)
	return scanner
}

// ParseSSELine extracts the JSON payload from a "data: ..." SSE line.
// Returns (payload, isDone, ok):
//   - ok=false  for non-data lines (empty lines, comments, event:/id: headers).
//   - isDone=true when the payload is the "[DONE]" sentinel.
//   - payload is a sub-slice of the scanner buffer; copy before reuse.
func ParseSSELine(line []byte) (payload []byte, isDone bool, ok bool) {
	if !bytes.HasPrefix(line, dataPrefix) {
		return nil, false, false
	}
	payload = line[len(dataPrefix):]
	if bytes.Equal(payload, doneMarker) {
		return nil, true, true
	}
	return payload, false, true
}
