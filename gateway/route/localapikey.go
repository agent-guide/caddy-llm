package route

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"
)

func ValidateLocalAPIKeyForRoute(r Route, key *LocalAPIKey) (*LocalAPIKey, error) {
	if key == nil {
		if r.Policy.Auth.RequireLocalAPIKey {
			return nil, &HTTPError{status: http.StatusUnauthorized, msg: "local api key is required"}
		}
		return nil, nil
	}
	if key.Disabled {
		return nil, &HTTPError{status: http.StatusForbidden, msg: "local api key is disabled"}
	}
	if !key.ExpiresAt.IsZero() && key.ExpiresAt.Before(time.Now()) {
		return nil, &HTTPError{status: http.StatusForbidden, msg: "local api key is expired"}
	}
	if len(key.AllowedRouteIDs) > 0 && !slices.Contains(key.AllowedRouteIDs, r.ID) {
		return nil, &HTTPError{status: http.StatusForbidden, msg: "local api key is not allowed to access this route"}
	}
	return key, nil
}

// DecodeStoredLocalAPIKey decodes local API key records.
func DecodeStoredLocalAPIKey(data []byte) (any, error) {
	var key LocalAPIKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, err
	}
	if key.Key == "" {
		return nil, &json.UnmarshalTypeError{Field: "key"}
	}
	return &key, nil
}
