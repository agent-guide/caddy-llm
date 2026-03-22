package manager

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agent-guide/caddy-llm/llm/authmanager/credential"
)

// Selector chooses a credential candidate for a request.
type Selector interface {
	Pick(ctx context.Context, provider, model string, creds []*credential.Credential) (*credential.Credential, error)
}

// RoundRobinSelector provides a provider-scoped round-robin selection strategy.
type RoundRobinSelector struct {
	mu      sync.Mutex
	cursors map[string]int
	maxKeys int
}

// FillFirstSelector selects the first available credential (deterministic ordering).
// This "burns" one credential before moving to the next, useful for staggering
// rolling-window subscription caps.
type FillFirstSelector struct{}

type blockReason int

const (
	blockReasonNone     blockReason = iota
	blockReasonCooldown             // quota exhausted with known reset time
	blockReasonDisabled             // intentionally disabled
	blockReasonOther                // temporarily unavailable but not cooldown
)

// isCredentialBlockedForModel reports whether a credential is blocked for the given model.
// Returns (blocked, reason, nextRetry).
func isCredentialBlockedForModel(cred *credential.Credential, model string, now time.Time) (bool, blockReason, time.Time) {
	if cred == nil {
		return true, blockReasonOther, time.Time{}
	}
	if cred.Disabled || cred.Status == credential.StatusDisabled {
		return true, blockReasonDisabled, time.Time{}
	}

	// Check per-model state first.
	if model != "" && len(cred.ModelStates) > 0 {
		state, ok := cred.ModelStates[model]
		if !ok {
			// Try without any suffix/variant part.
			baseModel := canonicalModelKey(model)
			if baseModel != "" && baseModel != model {
				state, ok = cred.ModelStates[baseModel]
			}
		}
		if ok && state != nil {
			if state.Status == credential.StatusDisabled {
				return true, blockReasonDisabled, time.Time{}
			}
			if state.Unavailable && !state.NextRetryAfter.IsZero() && state.NextRetryAfter.After(now) {
				next := state.NextRetryAfter
				if !state.Quota.NextRecoverAt.IsZero() && state.Quota.NextRecoverAt.After(next) {
					next = state.Quota.NextRecoverAt
				}
				if state.Quota.Exceeded {
					return true, blockReasonCooldown, next
				}
				return true, blockReasonOther, next
			}
			return false, blockReasonNone, time.Time{}
		}
		// No model state entry; fall through to credential-level check.
		return false, blockReasonNone, time.Time{}
	}

	// Credential-level availability check.
	if cred.Unavailable && cred.NextRetryAfter.After(now) {
		next := cred.NextRetryAfter
		if !cred.Quota.NextRecoverAt.IsZero() && cred.Quota.NextRecoverAt.After(next) {
			next = cred.Quota.NextRecoverAt
		}
		if cred.Quota.Exceeded {
			return true, blockReasonCooldown, next
		}
		return true, blockReasonOther, next
	}
	return false, blockReasonNone, time.Time{}
}

// canonicalModelKey strips variant suffixes for consistent model key lookup.
func canonicalModelKey(model string) string {
	model = strings.TrimSpace(model)
	if idx := strings.LastIndexByte(model, ':'); idx > 0 {
		return strings.TrimSpace(model[:idx])
	}
	return model
}

// credentialPriority returns the scheduling priority for a credential.
func credentialPriority(cred *credential.Credential) int {
	if cred == nil {
		return 0
	}
	return cred.Priority()
}

// getAvailableCredentials filters the candidate list to those currently available,
// grouping by priority and returning the highest-priority group.
func getAvailableCredentials(creds []*credential.Credential, provider, model string, now time.Time) ([]*credential.Credential, error) {
	if len(creds) == 0 {
		return nil, &credential.Error{Code: "credential_not_found", Message: "no credentials configured"}
	}

	type group struct {
		creds         []*credential.Credential
		cooldownCount int
		earliest      time.Time
	}
	byPriority := make(map[int]*group)
	totalCooldown := 0
	globalEarliest := time.Time{}

	for _, cred := range creds {
		blocked, reason, next := isCredentialBlockedForModel(cred, model, now)
		if !blocked {
			priority := credentialPriority(cred)
			g := byPriority[priority]
			if g == nil {
				g = &group{}
				byPriority[priority] = g
			}
			g.creds = append(g.creds, cred)
			continue
		}
		if reason == blockReasonCooldown {
			totalCooldown++
			if !next.IsZero() && (globalEarliest.IsZero() || next.Before(globalEarliest)) {
				globalEarliest = next
			}
		}
	}

	if len(byPriority) == 0 {
		if totalCooldown == len(creds) && !globalEarliest.IsZero() {
			resetIn := globalEarliest.Sub(now)
			if resetIn < 0 {
				resetIn = 0
			}
			return nil, &cooldownError{
				model:    model,
				provider: provider,
				resetIn:  formatDuration(resetIn),
			}
		}
		return nil, &credential.Error{Code: "credential_unavailable", Message: "no credentials available"}
	}

	// Pick the highest priority.
	bestPriority := 0
	found := false
	for p := range byPriority {
		if !found || p > bestPriority {
			bestPriority = p
			found = true
		}
	}

	available := byPriority[bestPriority].creds
	if len(available) > 1 {
		sort.Slice(available, func(i, j int) bool { return available[i].ID < available[j].ID })
	}
	return available, nil
}

func formatDuration(d time.Duration) string {
	secs := int(math.Ceil(d.Seconds()))
	if secs <= 0 {
		return "0s"
	}
	return strconv.Itoa(secs) + "s"
}

// Pick selects the next available credential using round-robin per provider+model.
func (s *RoundRobinSelector) Pick(ctx context.Context, provider, model string, creds []*credential.Credential) (*credential.Credential, error) {
	now := time.Now()
	available, err := getAvailableCredentials(creds, provider, model, now)
	if err != nil {
		return nil, err
	}

	key := provider + ":" + canonicalModelKey(model)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cursors == nil {
		s.cursors = make(map[string]int)
	}

	limit := s.maxKeys
	if limit <= 0 {
		limit = 4096
	}
	if _, ok := s.cursors[key]; !ok && len(s.cursors) >= limit {
		s.cursors = make(map[string]int)
	}

	index := s.cursors[key]
	if index >= 2_147_483_640 {
		index = 0
	}
	s.cursors[key] = index + 1

	return available[index%len(available)], nil
}

// Pick selects the first available credential deterministically.
func (s *FillFirstSelector) Pick(_ context.Context, provider, model string, creds []*credential.Credential) (*credential.Credential, error) {
	now := time.Now()
	available, err := getAvailableCredentials(creds, provider, model, now)
	if err != nil {
		return nil, err
	}
	return available[0], nil
}
