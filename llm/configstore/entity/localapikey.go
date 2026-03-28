package entity

import (
	"encoding/json"
	"fmt"
	"time"
)

type LocalAPIKey struct {
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

func DecodeLocalAPIKey(data []byte) (any, error) {
	var c LocalAPIKey
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("Decode LocalAPIKey object error: %w", err)
	}
	return &c, nil
}
