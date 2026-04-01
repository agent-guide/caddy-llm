package manager

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agent-guide/caddy-agent-gateway/llm/cliauth/credential"
)

// scheduledState describes how a credential participates in a model shard.
type scheduledState int

const (
	scheduledStateReady    scheduledState = iota
	scheduledStateCooldown                // quota exceeded, known reset time
	scheduledStateBlocked                 // unavailable, unknown reset or other
	scheduledStateDisabled                // intentionally disabled
)

// authScheduler keeps the incremental per-provider/model scheduling state.
type authScheduler struct {
	mu            sync.Mutex
	strategy      Strategy
	providers     map[string]*providerScheduler
	credProviders map[string]string // credID -> providerKey
	mixedCursors  map[string]int
}

type providerScheduler struct {
	providerKey string
	creds       map[string]*scheduledCredMeta
	modelShards map[string]*modelScheduler
}

type scheduledCredMeta struct {
	cred     *credential.Credential
	priority int
}

type modelScheduler struct {
	modelKey        string
	entries         map[string]*scheduledCred
	priorityOrder   []int
	readyByPriority map[int]*ReadyBucket
	blocked         []*scheduledCred
}

type scheduledCred struct {
	meta        *scheduledCredMeta
	cred        *credential.Credential
	state       scheduledState
	nextRetryAt time.Time
}


// newAuthScheduler constructs an empty scheduler.
func newAuthScheduler(strategy Strategy) *authScheduler {
	return &authScheduler{
		strategy:      strategy,
		providers:     make(map[string]*providerScheduler),
		credProviders: make(map[string]string),
		mixedCursors:  make(map[string]int),
	}
}

// setStrategy updates the active selection strategy.
func (s *authScheduler) setStrategy(strategy Strategy) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategy = strategy
	s.mixedCursors = make(map[string]int)
}

// rebuild recreates the complete scheduler state from a credential snapshot.
func (s *authScheduler) rebuild(creds []*credential.Credential) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = make(map[string]*providerScheduler)
	s.credProviders = make(map[string]string)
	s.mixedCursors = make(map[string]int)
	now := time.Now()
	for _, cred := range creds {
		s.upsertLocked(cred, now)
	}
}

// upsert incrementally synchronizes one credential into the scheduler.
func (s *authScheduler) upsert(cred *credential.Credential) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.upsertLocked(cred, time.Now())
}

// remove deletes one credential from every scheduler shard.
func (s *authScheduler) remove(credID string) {
	if s == nil {
		return
	}
	credID = strings.TrimSpace(credID)
	if credID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(credID)
}

// pick returns the next credential for a provider/model request.
func (s *authScheduler) pick(_ context.Context, provider, model string, tried map[string]struct{}) (*credential.Credential, error) {
	if s == nil {
		return nil, &credential.Error{Code: "credential_not_found", Message: "no credential available"}
	}
	providerKey := strings.ToLower(strings.TrimSpace(provider))
	modelKey := canonicalModelKey(model)

	s.mu.Lock()
	defer s.mu.Unlock()

	ps := s.providers[providerKey]
	if ps == nil {
		return nil, &credential.Error{Code: "credential_not_found", Message: "no credential available"}
	}

	shard := ps.ensureModelLocked(modelKey, time.Now())
	if shard == nil {
		return nil, &credential.Error{Code: "credential_not_found", Message: "no credential available"}
	}

	predicate := func(cred *credential.Credential) bool {
		if cred == nil {
			return false
		}
		if len(tried) > 0 {
			if _, ok := tried[cred.ID]; ok {
				return false
			}
		}
		return true
	}

	if picked := shard.pickReadyLocked(s.strategy, predicate); picked != nil {
		return picked, nil
	}
	return nil, shard.unavailableErrorLocked(provider, model, predicate)
}

func (s *authScheduler) upsertLocked(cred *credential.Credential, now time.Time) {
	if cred == nil {
		return
	}
	credID := strings.TrimSpace(cred.ID)
	providerKey := strings.ToLower(strings.TrimSpace(cred.Provider))
	if credID == "" || providerKey == "" || cred.IsDisabled() {
		s.removeLocked(credID)
		return
	}
	if prev := s.credProviders[credID]; prev != "" && prev != providerKey {
		if prevPS := s.providers[prev]; prevPS != nil {
			prevPS.removeLocked(credID)
		}
	}
	meta := &scheduledCredMeta{
		cred:     cred,
		priority: credentialPriority(cred),
	}
	s.credProviders[credID] = providerKey
	s.ensureProviderLocked(providerKey).upsertLocked(meta, now)
}

func (s *authScheduler) removeLocked(credID string) {
	if credID == "" {
		return
	}
	if providerKey := s.credProviders[credID]; providerKey != "" {
		if ps := s.providers[providerKey]; ps != nil {
			ps.removeLocked(credID)
		}
		delete(s.credProviders, credID)
	}
}

func (s *authScheduler) ensureProviderLocked(providerKey string) *providerScheduler {
	ps := s.providers[providerKey]
	if ps == nil {
		ps = &providerScheduler{
			providerKey: providerKey,
			creds:       make(map[string]*scheduledCredMeta),
			modelShards: make(map[string]*modelScheduler),
		}
		s.providers[providerKey] = ps
	}
	return ps
}

func (p *providerScheduler) upsertLocked(meta *scheduledCredMeta, now time.Time) {
	if p == nil || meta == nil || meta.cred == nil {
		return
	}
	p.creds[meta.cred.ID] = meta
	for modelKey, shard := range p.modelShards {
		if shard != nil {
			shard.upsertEntryLocked(meta, modelKey, now)
		}
	}
}

func (p *providerScheduler) removeLocked(credID string) {
	if p == nil || credID == "" {
		return
	}
	delete(p.creds, credID)
	for _, shard := range p.modelShards {
		if shard != nil {
			shard.removeEntryLocked(credID)
		}
	}
}

func (p *providerScheduler) ensureModelLocked(modelKey string, now time.Time) *modelScheduler {
	if p == nil {
		return nil
	}
	modelKey = canonicalModelKey(modelKey)
	if shard, ok := p.modelShards[modelKey]; ok && shard != nil {
		shard.promoteExpiredLocked(now)
		return shard
	}
	shard := &modelScheduler{
		modelKey:        modelKey,
		entries:         make(map[string]*scheduledCred),
		readyByPriority: make(map[int]*ReadyBucket),
	}
	for _, meta := range p.creds {
		if meta != nil {
			shard.upsertEntryLocked(meta, modelKey, now)
		}
	}
	p.modelShards[modelKey] = shard
	return shard
}

func (m *modelScheduler) upsertEntryLocked(meta *scheduledCredMeta, modelKey string, now time.Time) {
	if m == nil || meta == nil || meta.cred == nil {
		return
	}
	entry, ok := m.entries[meta.cred.ID]
	if !ok || entry == nil {
		entry = &scheduledCred{}
		m.entries[meta.cred.ID] = entry
	}
	prevState := entry.state
	prevNext := entry.nextRetryAt
	prevPriority := 0
	if entry.meta != nil {
		prevPriority = entry.meta.priority
	}

	entry.meta = meta
	entry.cred = meta.cred
	entry.nextRetryAt = time.Time{}

	blocked, reason, next := isCredentialBlockedForModel(meta.cred, modelKey, now)
	switch {
	case !blocked:
		entry.state = scheduledStateReady
	case reason == blockReasonCooldown:
		entry.state = scheduledStateCooldown
		entry.nextRetryAt = next
	case reason == blockReasonDisabled:
		entry.state = scheduledStateDisabled
	default:
		entry.state = scheduledStateBlocked
		entry.nextRetryAt = next
	}

	// Rebuild indexes only when something changed.
	if ok && prevState == entry.state && prevNext.Equal(entry.nextRetryAt) && prevPriority == meta.priority {
		return
	}
	m.rebuildIndexesLocked()
}

func (m *modelScheduler) removeEntryLocked(credID string) {
	if m == nil || credID == "" {
		return
	}
	if _, ok := m.entries[credID]; !ok {
		return
	}
	delete(m.entries, credID)
	m.rebuildIndexesLocked()
}

func (m *modelScheduler) promoteExpiredLocked(now time.Time) {
	if m == nil || len(m.blocked) == 0 {
		return
	}
	changed := false
	for _, entry := range m.blocked {
		if entry == nil || entry.cred == nil || entry.nextRetryAt.IsZero() || entry.nextRetryAt.After(now) {
			continue
		}
		blocked, reason, next := isCredentialBlockedForModel(entry.cred, m.modelKey, now)
		switch {
		case !blocked:
			entry.state = scheduledStateReady
			entry.nextRetryAt = time.Time{}
		case reason == blockReasonCooldown:
			entry.state = scheduledStateCooldown
			entry.nextRetryAt = next
		case reason == blockReasonDisabled:
			entry.state = scheduledStateDisabled
			entry.nextRetryAt = time.Time{}
		default:
			entry.state = scheduledStateBlocked
			entry.nextRetryAt = next
		}
		changed = true
	}
	if changed {
		m.rebuildIndexesLocked()
	}
}

func (m *modelScheduler) pickReadyLocked(strategy Strategy, predicate func(*credential.Credential) bool) *credential.Credential {
	if m == nil {
		return nil
	}
	m.promoteExpiredLocked(time.Now())

	// Find the highest priority bucket with a matching ready credential.
	bestPriority := 0
	found := false
	for _, p := range m.priorityOrder {
		bucket := m.readyByPriority[p]
		if bucket == nil {
			continue
		}
		if hasMatch(bucket, predicate) {
			if !found || p > bestPriority {
				bestPriority = p
				found = true
			}
		}
	}
	if !found {
		return nil
	}

	return strategy.PickFromBucket(m.readyByPriority[bestPriority], predicate)
}

func hasMatch(bucket *ReadyBucket, predicate func(*credential.Credential) bool) bool {
	for _, cred := range bucket.creds {
		if predicate == nil || predicate(cred) {
			return true
		}
	}
	return false
}

func (m *modelScheduler) unavailableErrorLocked(provider, model string, predicate func(*credential.Credential) bool) error {
	now := time.Now()
	total := 0
	cooldownCount := 0
	earliest := time.Time{}
	for _, entry := range m.entries {
		if predicate != nil && !predicate(entry.cred) {
			continue
		}
		total++
		if entry.state != scheduledStateCooldown {
			continue
		}
		cooldownCount++
		if !entry.nextRetryAt.IsZero() && (earliest.IsZero() || entry.nextRetryAt.Before(earliest)) {
			earliest = entry.nextRetryAt
		}
	}
	if total == 0 {
		return &credential.Error{Code: "credential_not_found", Message: "no credential available"}
	}
	if cooldownCount == total && !earliest.IsZero() {
		resetIn := earliest.Sub(now)
		if resetIn < 0 {
			resetIn = 0
		}
		return &cooldownError{model: model, provider: provider, resetIn: formatDuration(resetIn)}
	}
	return &credential.Error{Code: "credential_unavailable", Message: "no credential available"}
}

func formatDuration(d time.Duration) string {
	secs := int(math.Ceil(d.Seconds()))
	if secs <= 0 {
		return "0s"
	}
	return strconv.Itoa(secs) + "s"
}

func (m *modelScheduler) rebuildIndexesLocked() {
	m.readyByPriority = make(map[int]*ReadyBucket)
	m.priorityOrder = m.priorityOrder[:0]
	m.blocked = m.blocked[:0]

	byPriority := make(map[int][]*scheduledCred)
	for _, entry := range m.entries {
		if entry == nil || entry.cred == nil {
			continue
		}
		switch entry.state {
		case scheduledStateReady:
			p := entry.meta.priority
			byPriority[p] = append(byPriority[p], entry)
		case scheduledStateCooldown, scheduledStateBlocked:
			m.blocked = append(m.blocked, entry)
		}
	}

	for priority, entries := range byPriority {
		sort.Slice(entries, func(i, j int) bool { return entries[i].cred.ID < entries[j].cred.ID })
		creds := make([]*credential.Credential, len(entries))
		for i, e := range entries {
			creds[i] = e.cred
		}
		m.readyByPriority[priority] = &ReadyBucket{creds: creds}
		m.priorityOrder = append(m.priorityOrder, priority)
	}
	sort.Slice(m.priorityOrder, func(i, j int) bool { return m.priorityOrder[i] > m.priorityOrder[j] })
	sort.Slice(m.blocked, func(i, j int) bool {
		l, r := m.blocked[i], m.blocked[j]
		if l == nil || r == nil {
			return l != nil
		}
		if l.nextRetryAt.Equal(r.nextRetryAt) {
			return l.cred.ID < r.cred.ID
		}
		if l.nextRetryAt.IsZero() {
			return false
		}
		if r.nextRetryAt.IsZero() {
			return true
		}
		return l.nextRetryAt.Before(r.nextRetryAt)
	})
}
