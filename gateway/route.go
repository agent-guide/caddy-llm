package gateway

import (
	"encoding/json"
	"fmt"
	"time"
)

// DecodeRoute decodes a persisted route and fills missing runtime defaults.
func DecodeRoute(data []byte) (any, error) {
	var route Route
	if err := json.Unmarshal(data, &route); err != nil {
		return nil, fmt.Errorf("decode route: %w", err)
	}
	route.Policy.Defaults()
	now := time.Now().UTC()
	if route.CreatedAt.IsZero() {
		route.CreatedAt = now
	}
	if route.UpdatedAt.IsZero() {
		route.UpdatedAt = now
	}
	return &route, nil
}
