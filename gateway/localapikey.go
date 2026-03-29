package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"go.uber.org/zap"
)

type legacyLocalAPIKey struct {
	LocalKey      string    `json:"local_key"`
	UserID        string    `json:"user_id"`
	ProviderName  string    `json:"provider_name,omitempty"`
	APIKey        string    `json:"api_key,omitempty"`
	CredentialIDs []string  `json:"credential_ids,omitempty"`
	Disabled      bool      `json:"disabled"`
	StatusMessage string    `json:"status_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func ValidateLocalAPIKeyForRoute(route Route, key *LocalAPIKey) (*LocalAPIKey, error) {
	if key == nil {
		if route.Policy.Auth.RequireLocalAPIKey {
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
	if len(key.AllowedRouteIDs) > 0 && !slices.Contains(key.AllowedRouteIDs, route.ID) {
		return nil, &HTTPError{status: http.StatusForbidden, msg: "local api key is not allowed to access this route"}
	}
	return key, nil
}

// DecodeLocalAPIKey decodes current and legacy local API key records.
func DecodeLocalAPIKey(data []byte) (any, error) {
	var key LocalAPIKey
	if err := json.Unmarshal(data, &key); err == nil && key.Key != "" {
		return &key, nil
	}

	var legacy legacyLocalAPIKey
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("decode local api key: %w", err)
	}
	if legacy.LocalKey == "" {
		return nil, fmt.Errorf("decode local api key: missing key")
	}

	if legacy.ProviderName != "" || legacy.APIKey != "" || len(legacy.CredentialIDs) > 0 {
		zap.L().Warn("legacy local api key has provider credential fields that are not migrated",
			zap.String("key", legacy.LocalKey),
			zap.String("user_id", legacy.UserID),
			zap.Bool("has_provider_name", legacy.ProviderName != ""),
			zap.Bool("has_api_key", legacy.APIKey != ""),
			zap.Int("credential_ids_count", len(legacy.CredentialIDs)),
		)
	}

	return &LocalAPIKey{
		Key:           legacy.LocalKey,
		UserID:        legacy.UserID,
		Disabled:      legacy.Disabled,
		StatusMessage: legacy.StatusMessage,
		CreatedAt:     legacy.CreatedAt,
		UpdatedAt:     legacy.UpdatedAt,
	}, nil
}
