package route

import (
	"encoding/json"
	"fmt"
	"time"
)

// DecodeRoute decodes a persisted route and fills missing runtime defaults.
func DecodeRoute(data []byte) (any, error) {
	var r Route
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("decode route: %w", err)
	}
	r.Policy.Defaults()
	now := time.Now().UTC()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = now
	}
	return &r, nil
}
