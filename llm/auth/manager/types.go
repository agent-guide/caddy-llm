package manager

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// Credential encapsulates the runtime state and metadata for a single upstream credential.
type Credential struct {
	// ID uniquely identifies the credential across restarts.
	ID string `json:"id"`
	// Index is a stable runtime identifier derived from credential metadata (not persisted).
	Index string `json:"-"`
	// Provider is the upstream provider key (e.g. "openai", "anthropic").
	Provider string `json:"provider"`
	// Prefix optionally namespaces models for routing (e.g., "teamA/gpt-4o").
	Prefix string `json:"prefix,omitempty"`
	// Label is an optional human-readable label for logging and display.
	Label string `json:"label,omitempty"`
	// Status is the lifecycle status managed by the Manager.
	Status Status `json:"status"`
	// StatusMessage holds a short description for the current status.
	StatusMessage string `json:"status_message,omitempty"`
	// Disabled indicates the credential is intentionally disabled by the operator.
	Disabled bool `json:"disabled"`
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

	indexAssigned bool
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

// EnsureIndex returns a stable index derived from api_key attribute or ID.
func (c *Credential) EnsureIndex() string {
	if c == nil {
		return ""
	}
	if c.indexAssigned && c.Index != "" {
		return c.Index
	}

	seed := ""
	if c.Attributes != nil {
		if apiKey := strings.TrimSpace(c.Attributes["api_key"]); apiKey != "" {
			seed = "api_key:" + apiKey
		}
	}
	if seed == "" {
		if id := strings.TrimSpace(c.ID); id != "" {
			seed = "id:" + id
		} else {
			return ""
		}
	}

	sum := sha256.Sum256([]byte(seed))
	idx := hex.EncodeToString(sum[:8])
	c.Index = idx
	c.indexAssigned = true
	return idx
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
	return expirationFromMap(c.Metadata)
}

var expireKeys = [...]string{"expired", "expire", "expires_at", "expiresAt", "expiry", "expires"}

func expirationFromMap(meta map[string]any) (time.Time, bool) {
	if meta == nil {
		return time.Time{}, false
	}
	for _, key := range expireKeys {
		if v, ok := meta[key]; ok {
			if ts, ok1 := parseTimeValue(v); ok1 {
				return ts, true
			}
		}
	}
	// Check nested "token" object for OAuth-style tokens.
	for _, nestedKey := range []string{"token", "Token"} {
		if nested, ok := meta[nestedKey]; ok {
			switch val := nested.(type) {
			case map[string]any:
				if ts, ok1 := expirationFromMap(val); ok1 {
					return ts, true
				}
			case map[string]string:
				temp := make(map[string]any, len(val))
				for k, v := range val {
					temp[k] = v
				}
				if ts, ok1 := expirationFromMap(temp); ok1 {
					return ts, true
				}
			}
		}
	}
	return time.Time{}, false
}

func parseTimeValue(v any) (time.Time, bool) {
	switch value := v.(type) {
	case string:
		s := strings.TrimSpace(value)
		if s == "" {
			return time.Time{}, false
		}
		for _, layout := range []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
		} {
			if ts, err := time.Parse(layout, s); err == nil {
				return ts, true
			}
		}
		if unix, err := strconv.ParseInt(s, 10, 64); err == nil {
			return normaliseUnix(unix), true
		}
	case float64:
		return normaliseUnix(int64(value)), true
	case int64:
		return normaliseUnix(value), true
	case json.Number:
		if i, err := value.Int64(); err == nil {
			return normaliseUnix(i), true
		}
	}
	return time.Time{}, false
}

func normaliseUnix(raw int64) time.Time {
	if raw <= 0 {
		return time.Time{}
	}
	if raw > 1_000_000_000_000 {
		return time.UnixMilli(raw)
	}
	return time.Unix(raw, 0)
}

func parseBoolAny(val any) (bool, bool) {
	switch typed := val.(type) {
	case bool:
		return typed, true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return false, false
		}
		parsed, err := strconv.ParseBool(trimmed)
		if err != nil {
			return false, false
		}
		return parsed, true
	case float64:
		return typed != 0, true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return false, false
		}
		return parsed != 0, true
	default:
		return false, false
	}
}

func parseIntAny(val any) (int, bool) {
	switch typed := val.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

// DisableCoolingOverride returns the per-credential disable_cooling override when present.
func (c *Credential) DisableCoolingOverride() (bool, bool) {
	if c == nil || c.Metadata == nil {
		return false, false
	}
	for _, key := range []string{"disable_cooling", "disable-cooling"} {
		if val, ok := c.Metadata[key]; ok {
			if parsed, okParse := parseBoolAny(val); okParse {
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
			if parsed, okParse := parseIntAny(val); okParse {
				if parsed < 0 {
					parsed = 0
				}
				return parsed, true
			}
		}
	}
	return 0, false
}
