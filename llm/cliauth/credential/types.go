package credential

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/agent-guide/caddy-agent-gateway/internal/utils"
)

// Credential encapsulates the runtime state and metadata for a single upstream credential.
type Credential struct {
	// ID uniquely identifies the credential across restarts.
	ID string `json:"id"`
	// Provider is the upstream provider key (e.g. "openai", "anthropic").
	Provider string `json:"provider"`
	// Prefix optionally namespaces models for routing (e.g., "teamA/gpt-4o").
	Prefix string `json:"prefix,omitempty"`
	// Label is an optional human-readable label for logging and display.
	Label string `json:"label,omitempty"`
	// Status is the lifecycle status managed by the Manager.
	// Use StatusDisabled to mark a credential as intentionally disabled.
	Status Status `json:"status"`
	// StatusMessage holds a short description for the current status.
	StatusMessage string `json:"status_message,omitempty"`
	// Unavailable flags transient provider unavailability (e.g. quota exceeded).
	Unavailable bool `json:"unavailable"`
	// ProxyURL overrides the global proxy setting for this credential if provided.
	ProxyURL string `json:"proxy_url,omitempty"`
	// Attributes stores provider-specific configuration metadata (immutable, e.g. api_key, base_url).
	Attributes map[string]string `json:"attributes,omitempty"`
	// Metadata stores runtime mutable provider state (e.g. tokens, cookies).
	Metadata map[string]any `json:"metadata,omitempty"`
	// Quota captures recent quota information for load balancers.
	Quota QuotaState `json:"quota"`
	// LastError stores the last failure encountered while executing or refreshing.
	LastError *Error `json:"last_error,omitempty"`
	// CreatedAt is the creation timestamp in UTC.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the last modification timestamp in UTC.
	UpdatedAt time.Time `json:"updated_at"`
	// LastRefreshedAt records the last successful refresh time in UTC.
	LastRefreshedAt time.Time `json:"last_refreshed_at"`
	// NextRefreshAfter is the earliest time a refresh should retrigger.
	NextRefreshAfter time.Time `json:"next_refresh_after"`
	// NextRetryAfter is the earliest time a retry should retrigger (credential level).
	NextRetryAfter time.Time `json:"next_retry_after"`
	// ModelStates tracks per-model runtime availability data.
	ModelStates map[string]*ModelState `json:"model_states,omitempty"`
}

// QuotaState contains quota limiter tracking data for a credential.
type QuotaState struct {
	// Exceeded indicates the credential recently hit a quota error.
	Exceeded bool `json:"exceeded"`
	// Reason provides an optional human-readable description.
	Reason string `json:"reason,omitempty"`
	// NextRecoverAt is when the credential may become available again.
	NextRecoverAt time.Time `json:"next_recover_at"`
	// BackoffLevel stores the progressive cooldown exponent used for rate limits.
	BackoffLevel int `json:"backoff_level,omitempty"`
}

// ModelState captures the execution state for a specific model under a credential.
type ModelState struct {
	// Status reflects the lifecycle status for this model.
	Status Status `json:"status"`
	// StatusMessage provides an optional short description of the status.
	StatusMessage string `json:"status_message,omitempty"`
	// Unavailable mirrors whether the model is temporarily blocked for retries.
	Unavailable bool `json:"unavailable"`
	// NextRetryAfter defines the per-model retry time.
	NextRetryAfter time.Time `json:"next_retry_after"`
	// LastError records the latest error observed for this model.
	LastError *Error `json:"last_error,omitempty"`
	// Quota retains quota information if this model hit rate limits.
	Quota QuotaState `json:"quota"`
	// UpdatedAt tracks the last update timestamp for this model state.
	UpdatedAt time.Time `json:"updated_at"`
}

func DecodeCredential(data []byte) (any, error) {
	var c Credential
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("Decode credential object error: %w", err)
	}
	return &c, nil
}

// Clone shallow copies the Credential, duplicating maps to avoid accidental mutation.
func (c *Credential) Clone() *Credential {
	if c == nil {
		return nil
	}
	copy := *c
	if len(c.Attributes) > 0 {
		copy.Attributes = make(map[string]string, len(c.Attributes))
		for k, v := range c.Attributes {
			copy.Attributes[k] = v
		}
	}
	if len(c.Metadata) > 0 {
		copy.Metadata = make(map[string]any, len(c.Metadata))
		for k, v := range c.Metadata {
			copy.Metadata[k] = v
		}
	}
	if len(c.ModelStates) > 0 {
		copy.ModelStates = make(map[string]*ModelState, len(c.ModelStates))
		for k, v := range c.ModelStates {
			copy.ModelStates[k] = v.Clone()
		}
	}
	return &copy
}

// Clone duplicates a ModelState including nested error details.
func (m *ModelState) Clone() *ModelState {
	if m == nil {
		return nil
	}
	copy := *m
	if m.LastError != nil {
		copy.LastError = &Error{
			Code:       m.LastError.Code,
			Message:    m.LastError.Message,
			Retryable:  m.LastError.Retryable,
			HTTPStatus: m.LastError.HTTPStatus,
		}
	}
	return &copy
}

// IsDisabled reports whether the credential has been intentionally disabled.
func (c *Credential) IsDisabled() bool {
	return c != nil && c.Status == StatusDisabled
}

// APIKey returns the api_key attribute value, or empty string if not set.
func (c *Credential) APIKey() string {
	if c == nil || c.Attributes == nil {
		return ""
	}
	return c.Attributes["api_key"]
}

// BaseURL returns the base_url attribute value, or empty string if not set.
func (c *Credential) BaseURL() string {
	if c == nil || c.Attributes == nil {
		return ""
	}
	return c.Attributes["base_url"]
}

// Priority returns the scheduling priority for this credential (higher = preferred).
func (c *Credential) Priority() int {
	if c == nil || c.Attributes == nil {
		return 0
	}
	raw := strings.TrimSpace(c.Attributes["priority"])
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return parsed
}

// ExpirationTime attempts to extract the credential expiration timestamp from metadata.
func (c *Credential) ExpirationTime() (time.Time, bool) {
	if c == nil {
		return time.Time{}, false
	}
	return utils.ExpirationFromMap(c.Metadata)
}


// DisableCoolingOverride returns the per-credential disable_cooling override when present.
func (c *Credential) DisableCoolingOverride() (bool, bool) {
	if c == nil || c.Metadata == nil {
		return false, false
	}
	for _, key := range []string{"disable_cooling", "disable-cooling"} {
		if val, ok := c.Metadata[key]; ok {
			if parsed, okParse := utils.ParseBoolAny(val); okParse {
				return parsed, true
			}
		}
	}
	return false, false
}

// RequestRetryOverride returns the per-credential request_retry override when present.
func (c *Credential) RequestRetryOverride() (int, bool) {
	if c == nil || c.Metadata == nil {
		return 0, false
	}
	for _, key := range []string{"request_retry", "request-retry"} {
		if val, ok := c.Metadata[key]; ok {
			if parsed, okParse := utils.ParseIntAny(val); okParse {
				if parsed < 0 {
					parsed = 0
				}
				return parsed, true
			}
		}
	}
	return 0, false
}
