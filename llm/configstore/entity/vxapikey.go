package entity

import (
	"encoding/json"
	"fmt"
	"time"
)

type VXApiKey struct {
	Key           string    `json:"key"`
	UserID        string    `json:"user_id"`
	Disabled      bool      `json:"disabled"`
	StatusMessage string    `json:"status_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func DecodeVXApiKey(data []byte) (any, error) {
	var c VXApiKey
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("Decode VXApiKey object error: %w", err)
	}
	return &c, nil
}
