package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/credential"
	"github.com/agent-guide/caddy-agent-gateway/configstore/intf"
	"github.com/google/uuid"
)

const (
	defaultRefreshCheckInterval = 5 * time.Second
	defaultRefreshMaxConcurrent = 8
	defaultQuotaBackoffBase     = time.Second
	defaultQuotaBackoffMax      = 30 * time.Minute
)

// Result captures the execution outcome used to adjust credential state.
type Result struct {
	// CredentialID references the credential that produced this result.
	CredentialID string
	// Provider is copied for convenience.
	Provider string
	// Model is the upstream model identifier used for the request.
	Model string
	// Success marks whether the execution succeeded.
	Success bool
	// RetryAfter carries a provider-supplied retry hint (e.g. 429 Retry-After).
	RetryAfter *time.Duration
	// Error describes the failure when Success is false.
	Error *credential.Error
}

// Hook captures lifecycle callbacks for observing credential changes.
type Hook interface {
	// OnCredentialRegistered fires when a new credential is registered.
	OnCredentialRegistered(ctx context.Context, cred *credential.Credential)
	// OnCredentialUpdated fires when an existing credential changes state.
	OnCredentialUpdated(ctx context.Context, cred *credential.Credential)
	// OnResult fires when an execution result is recorded.
	OnResult(ctx context.Context, result Result)
}

// NoopHook provides empty hook implementations.
type NoopHook struct{}

func (NoopHook) OnCredentialRegistered(context.Context, *credential.Credential) {}
func (NoopHook) OnCredentialUpdated(context.Context, *credential.Credential)    {}
func (NoopHook) OnResult(context.Context, Result)                               {}

// Refresher is an optional interface that provider-specific credential managers
// can implement to refresh expiring credentials.
type Refresher interface {
	// Refresh attempts to refresh the credential and returns the updated state.
	// Returning nil means the credential should be left unchanged.
	Refresh(ctx context.Context, cred *credential.Credential) (*credential.Credential, error)
}

// authenticatorRefresher wraps an Authenticator to satisfy the Refresher interface,
// routing refresh calls through the authenticator's RefreshLead method.
type authenticatorRefresher struct {
	auth Authenticator
}

func (a *authenticatorRefresher) Refresh(ctx context.Context, cred *credential.Credential) (*credential.Credential, error) {
	return a.auth.RefreshLead(ctx, cred)
}

// Manager orchestrates credential lifecycle: registration, selection, result
// feedback, quota tracking, and optional persistence.
type Manager struct {
	store    intf.CredentialStorer
	selector Selector
	hook     Hook

	mu              sync.RWMutex
	creds           map[string]*credential.Credential // credID -> Credential
	authenticators  map[string]Authenticator          // cli/provider key -> Authenticator
	refresher       Refresher                         // fallback global refresher
	scheduler       *authScheduler
	providerOffsets map[string]int

	// requestRetry is the global retry limit for failed requests.
	requestRetry atomic.Int32

	// quotaCooldownDisabled disables quota cooldown scheduling globally.
	quotaCooldownDisabled atomic.Bool

	// Auto-refresh state.
	refreshCancel    context.CancelFunc
	refreshSemaphore chan struct{}
}

// NewManager constructs a Manager with optional custom selector and hook.
// If selector is nil, RoundRobinSelector is used. If hook is nil, NoopHook is used.
func NewManager(store intf.CredentialStorer, selector Selector, hook Hook) *Manager {
	if selector == nil {
		selector = &RoundRobinSelector{}
	}
	if hook == nil {
		hook = NoopHook{}
	}
	m := &Manager{
		store:            store,
		selector:         selector,
		hook:             hook,
		creds:            make(map[string]*credential.Credential),
		authenticators:   make(map[string]Authenticator),
		providerOffsets:  make(map[string]int),
		refreshSemaphore: make(chan struct{}, defaultRefreshMaxConcurrent),
	}
	m.scheduler = newAuthScheduler(selector)
	return m
}

// RegisterAuthenticator registers an Authenticator for a CLI name.
// It also indexes the same Authenticator by its provider name so refresh lookups
// can continue resolving via credential.Provider.
func (m *Manager) RegisterAuthenticator(cliname string, auth Authenticator) {
	if auth == nil {
		return
	}
	cliKey := strings.ToLower(strings.TrimSpace(cliname))
	if cliKey == "" {
		return
	}
	providerKey := strings.ToLower(strings.TrimSpace(auth.Provider()))
	m.mu.Lock()
	m.authenticators[cliKey] = auth
	if providerKey != "" {
		m.authenticators[providerKey] = auth
	}
	m.mu.Unlock()
}

// GetAuthenticator returns the Authenticator registered for the given CLI name.
func (m *Manager) GetAuthenticator(cliname string) (Authenticator, bool) {
	key := strings.ToLower(strings.TrimSpace(cliname))
	m.mu.RLock()
	auth, ok := m.authenticators[key]
	m.mu.RUnlock()
	return auth, ok
}

// SetSelector replaces the credential selection strategy.
func (m *Manager) SetSelector(selector Selector) {
	if m == nil {
		return
	}
	if selector == nil {
		selector = &RoundRobinSelector{}
	}
	m.mu.Lock()
	m.selector = selector
	m.mu.Unlock()
	if m.scheduler != nil {
		m.scheduler.setSelector(selector)
		m.syncScheduler()
	}
}

// SetRefresher registers a fallback Refresher used when no Authenticator is registered
// for a credential's provider.
func (m *Manager) SetRefresher(r Refresher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refresher = r
}

// SetRequestRetry configures the global request retry limit.
func (m *Manager) SetRequestRetry(n int32) {
	m.requestRetry.Store(n)
}

// SetQuotaCooldownDisabled toggles quota cooldown scheduling globally.
func (m *Manager) SetQuotaCooldownDisabled(disable bool) {
	m.quotaCooldownDisabled.Store(disable)
}

// Load reads all credentials from the store and registers them in memory.
// This should be called once during startup.
func (m *Manager) Load(ctx context.Context) error {
	if m.store == nil {
		return nil
	}
	items, err := m.store.ListByProviderName(ctx, "")
	if err != nil {
		return fmt.Errorf("manager: load from store: %w", err)
	}
	for _, item := range items {
		cred, ok := item.(*credential.Credential)
		if !ok || cred == nil {
			return fmt.Errorf("manager: load from store: unexpected credential type %T", item)
		}
		if err := m.Register(WithSkipPersist(ctx), cred); err != nil {
			return fmt.Errorf("manager: register credential %s: %w", cred.ID, err)
		}
	}
	return nil
}

// Register adds a new credential to the manager and optionally persists it.
// If the credential has no ID, one is generated. If a credential with the same
// ID already exists, it is replaced.
func (m *Manager) Register(ctx context.Context, cred *credential.Credential) error {
	if cred == nil {
		return fmt.Errorf("manager: credential is nil")
	}
	if strings.TrimSpace(cred.Provider) == "" {
		return fmt.Errorf("manager: credential has no provider")
	}

	cred = cred.Clone()
	if strings.TrimSpace(cred.ID) == "" {
		cred.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if cred.CreatedAt.IsZero() {
		cred.CreatedAt = now
	}
	cred.UpdatedAt = now
	if cred.Status == "" {
		cred.Status = credential.StatusActive
	}
	cred.EnsureIndex()

	if !shouldSkipPersist(ctx) {
		if err := m.persist(ctx, cred); err != nil {
			return err
		}
	}

	m.mu.Lock()
	m.creds[cred.ID] = cred
	m.mu.Unlock()

	m.scheduler.upsert(cred.Clone())
	m.hook.OnCredentialRegistered(ctx, cred.Clone())
	return nil
}

// Update merges new state into an existing credential and optionally persists.
func (m *Manager) Update(ctx context.Context, cred *credential.Credential) error {
	if cred == nil {
		return fmt.Errorf("manager: credential is nil")
	}

	cred = cred.Clone()
	cred.UpdatedAt = time.Now().UTC()

	if !shouldSkipPersist(ctx) {
		if err := m.persist(ctx, cred); err != nil {
			return err
		}
	}

	m.mu.Lock()
	m.creds[cred.ID] = cred
	m.mu.Unlock()

	m.scheduler.upsert(cred.Clone())
	m.hook.OnCredentialUpdated(ctx, cred.Clone())
	return nil
}

// Deregister removes a credential from memory and optionally deletes it from the store.
func (m *Manager) Deregister(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("manager: id is empty")
	}

	m.mu.Lock()
	_, ok := m.creds[id]
	delete(m.creds, id)
	m.mu.Unlock()

	if !ok {
		return nil
	}

	m.scheduler.remove(id)

	if !shouldSkipPersist(ctx) && m.store != nil {
		if err := m.store.Delete(ctx, id); err != nil {
			return fmt.Errorf("manager: delete from store: %w", err)
		}
	}
	return nil
}

// Get returns the credential with the given ID, or nil if not found.
func (m *Manager) Get(id string) *credential.Credential {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if cred, ok := m.creds[id]; ok {
		return cred.Clone()
	}
	return nil
}

// List returns all credentials, optionally filtered by provider.
// Pass an empty string to list all.
func (m *Manager) List(provider string) []*credential.Credential {
	provider = strings.ToLower(strings.TrimSpace(provider))
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*credential.Credential, 0, len(m.creds))
	for _, cred := range m.creds {
		if provider != "" && strings.ToLower(cred.Provider) != provider {
			continue
		}
		out = append(out, cred.Clone())
	}
	return out
}

// Pick selects the best available credential for the given provider and model.
// Tried is an optional set of credential IDs that have already been attempted
// and should be skipped.
func (m *Manager) Pick(ctx context.Context, provider, model string, tried map[string]struct{}) (*credential.Credential, error) {
	if m == nil {
		return nil, &credential.Error{Code: "manager_nil", Message: "manager not initialized"}
	}

	// Try scheduler first (O(1) incremental state).
	cred, err := m.scheduler.pick(ctx, provider, model, tried)
	if err == nil {
		return cred, nil
	}

	// Fall back to selector if scheduler has no state (e.g. custom selector).
	if isBuiltinSelector(m.selector) {
		return nil, err
	}

	m.mu.RLock()
	providerKey := strings.ToLower(strings.TrimSpace(provider))
	var candidates []*credential.Credential
	for _, c := range m.creds {
		if strings.ToLower(c.Provider) == providerKey {
			if len(tried) > 0 {
				if _, already := tried[c.ID]; already {
					continue
				}
			}
			candidates = append(candidates, c)
		}
	}
	m.mu.RUnlock()

	return m.selector.Pick(ctx, provider, model, candidates)
}

func isBuiltinSelector(s Selector) bool {
	switch s.(type) {
	case *RoundRobinSelector, *FillFirstSelector:
		return true
	default:
		return false
	}
}

// MarkResult records the outcome of a provider request and adjusts credential state
// (quota tracking, retry scheduling, etc.).
func (m *Manager) MarkResult(ctx context.Context, result Result) {
	if m == nil {
		return
	}
	credID := strings.TrimSpace(result.CredentialID)
	if credID == "" {
		m.hook.OnResult(ctx, result)
		return
	}

	m.mu.Lock()
	cred, ok := m.creds[credID]
	if !ok || cred == nil {
		m.mu.Unlock()
		m.hook.OnResult(ctx, result)
		return
	}
	cred = cred.Clone()
	m.mu.Unlock()

	now := time.Now().UTC()
	changed := false

	if result.Success {
		// Clear errors and availability flags on success.
		if cred.Unavailable || cred.LastError != nil || cred.Quota.Exceeded {
			cred.Unavailable = false
			cred.LastError = nil
			cred.NextRetryAfter = time.Time{}
			cred.Quota = credential.QuotaState{}
			cred.Status = credential.StatusActive
			cred.StatusMessage = ""
			changed = true
		}
		if result.Model != "" {
			if state, ok := cred.ModelStates[result.Model]; ok && state != nil {
				if state.Unavailable || state.LastError != nil {
					state.Unavailable = false
					state.LastError = nil
					state.NextRetryAfter = time.Time{}
					state.Quota = credential.QuotaState{}
					state.Status = credential.StatusActive
					state.StatusMessage = ""
					state.UpdatedAt = now
					changed = true
				}
			}
		}
	} else if result.Error != nil {
		rerr := result.Error
		changed = true
		isQuota := rerr.HTTPStatus == 429

		if isQuota && !m.quotaCooldownDisabledForCred(cred) {
			// Apply progressive backoff for quota errors.
			backoffLevel := cred.Quota.BackoffLevel
			backoff := computeBackoff(backoffLevel, defaultQuotaBackoffBase, defaultQuotaBackoffMax)
			if result.RetryAfter != nil && *result.RetryAfter > backoff {
				backoff = *result.RetryAfter
			}
			cred.Quota = credential.QuotaState{
				Exceeded:      true,
				Reason:        rerr.Message,
				NextRecoverAt: now.Add(backoff),
				BackoffLevel:  backoffLevel + 1,
			}
			cred.Unavailable = true
			cred.NextRetryAfter = now.Add(backoff)
			cred.Status = credential.StatusError
			cred.StatusMessage = "quota exceeded"
		} else if rerr.HTTPStatus == 401 || rerr.HTTPStatus == 403 {
			// Auth errors disable the credential.
			cred.Status = credential.StatusDisabled
			cred.StatusMessage = rerr.Message
			cred.Disabled = true
		} else {
			cred.LastError = rerr
			if result.RetryAfter != nil {
				cred.NextRetryAfter = now.Add(*result.RetryAfter)
				cred.Unavailable = true
			}
			cred.Status = credential.StatusError
			cred.StatusMessage = rerr.Message
		}

		// Apply model-level state if model is specified.
		if result.Model != "" {
			if cred.ModelStates == nil {
				cred.ModelStates = make(map[string]*credential.ModelState)
			}
			state := cred.ModelStates[result.Model]
			if state == nil {
				state = &credential.ModelState{}
				cred.ModelStates[result.Model] = state
			}
			state.UpdatedAt = now
			if isQuota && !m.quotaCooldownDisabledForCred(cred) {
				state.Quota = cred.Quota
				state.Unavailable = true
				state.NextRetryAfter = cred.NextRetryAfter
				state.Status = credential.StatusError
				state.StatusMessage = "quota exceeded"
			} else {
				state.LastError = rerr
				state.Status = credential.StatusError
				state.StatusMessage = rerr.Message
				if result.RetryAfter != nil {
					state.Unavailable = true
					state.NextRetryAfter = now.Add(*result.RetryAfter)
				}
			}
		}
	}

	if changed {
		cred.UpdatedAt = now
		_ = m.Update(ctx, cred)
	}

	m.hook.OnResult(ctx, result)
}

func (m *Manager) quotaCooldownDisabledForCred(cred *credential.Credential) bool {
	if cred != nil {
		if override, ok := cred.DisableCoolingOverride(); ok {
			return override
		}
	}
	return m.quotaCooldownDisabled.Load()
}

// StartRefreshLoop starts a background goroutine that periodically checks for
// expiring credentials and calls the Authenticator's RefreshLead (or fallback Refresher).
// Call StopRefreshLoop to shut it down.
func (m *Manager) StartRefreshLoop(ctx context.Context) {
	m.mu.Lock()
	if m.refreshCancel != nil {
		m.mu.Unlock()
		return
	}
	loopCtx, cancel := context.WithCancel(ctx)
	m.refreshCancel = cancel
	m.mu.Unlock()

	go m.refreshLoop(loopCtx)
}

// StopRefreshLoop stops the background refresh goroutine.
func (m *Manager) StopRefreshLoop() {
	m.mu.Lock()
	cancel := m.refreshCancel
	m.refreshCancel = nil
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *Manager) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultRefreshCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.runRefreshCycle(ctx)
		}
	}
}

func (m *Manager) runRefreshCycle(ctx context.Context) {
	now := time.Now()
	candidates := m.snapshotForRefresh(now)
	for _, cred := range candidates {
		// Resolve the refresher: prefer registered Authenticator, fall back to global Refresher.
		refresher := m.resolveRefresher(cred.Provider)
		if refresher == nil {
			continue
		}

		select {
		case m.refreshSemaphore <- struct{}{}:
		default:
			// Semaphore full; skip this cycle.
			return
		}
		go func(c *credential.Credential, r Refresher) {
			defer func() { <-m.refreshSemaphore }()
			m.refreshOne(ctx, c, r, now)
		}(cred, refresher)
	}
}

// resolveRefresher returns the Refresher for the given provider.
// Prefers a registered Authenticator; falls back to the global Refresher.
func (m *Manager) resolveRefresher(provider string) Refresher {
	key := strings.ToLower(strings.TrimSpace(provider))
	m.mu.RLock()
	auth, ok := m.authenticators[key]
	fallback := m.refresher
	m.mu.RUnlock()
	if ok && auth != nil {
		return &authenticatorRefresher{auth: auth}
	}
	return fallback
}

func (m *Manager) snapshotForRefresh(now time.Time) []*credential.Credential {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var candidates []*credential.Credential
	for _, cred := range m.creds {
		if cred.Disabled || cred.Status == credential.StatusDisabled {
			continue
		}
		if !needsRefresh(cred, now) {
			continue
		}
		candidates = append(candidates, cred.Clone())
	}
	return candidates
}

func needsRefresh(cred *credential.Credential, now time.Time) bool {
	if cred.Status == credential.StatusRefreshing {
		return false
	}
	if !cred.NextRefreshAfter.IsZero() && now.Before(cred.NextRefreshAfter) {
		return false
	}
	if exp, ok := cred.ExpirationTime(); ok {
		// Refresh 5 minutes before expiration.
		return now.After(exp.Add(-5 * time.Minute))
	}
	return false
}

func (m *Manager) refreshOne(ctx context.Context, cred *credential.Credential, refresher Refresher, now time.Time) {
	// Mark as refreshing.
	refreshing := cred.Clone()
	refreshing.Status = credential.StatusRefreshing
	refreshing.UpdatedAt = now
	_ = m.Update(ctx, refreshing)

	updated, err := refresher.Refresh(ctx, cred)
	if err != nil {
		// Refresh failed: mark error and schedule retry.
		failed := cred.Clone()
		failed.Status = credential.StatusError
		failed.StatusMessage = err.Error()
		failed.NextRefreshAfter = time.Now().Add(5 * time.Minute)
		_ = m.Update(ctx, failed)
		return
	}
	if updated == nil {
		// Refresher returned nil: leave credential unchanged.
		restored := cred.Clone()
		restored.Status = credential.StatusActive
		_ = m.Update(ctx, restored)
		return
	}

	updated.LastRefreshedAt = time.Now().UTC()
	if updated.Status == "" || updated.Status == credential.StatusRefreshing {
		updated.Status = credential.StatusActive
	}
	_ = m.Update(ctx, updated)
}

// snapshotAuths returns a snapshot of all credentials for scheduler rebuild.
func (m *Manager) snapshotAuths() []*credential.Credential {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*credential.Credential, 0, len(m.creds))
	for _, cred := range m.creds {
		out = append(out, cred.Clone())
	}
	return out
}

func (m *Manager) syncScheduler() {
	if m == nil || m.scheduler == nil {
		return
	}
	m.scheduler.rebuild(m.snapshotAuths())
}

func (m *Manager) persist(ctx context.Context, cred *credential.Credential) error {
	if m.store == nil {
		return nil
	}
	if _, err := m.store.Save(ctx, cred.ID, cred.Provider, cred); err != nil {
		return fmt.Errorf("manager: persist credential %s: %w", cred.ID, err)
	}
	return nil
}

// computeBackoff returns the progressive backoff duration for the given level.
func computeBackoff(level int, base, maxBackoff time.Duration) time.Duration {
	if level <= 0 {
		return base
	}
	d := base
	for i := 0; i < level && d < maxBackoff; i++ {
		d *= 2
	}
	if d > maxBackoff {
		d = maxBackoff
	}
	return d
}
