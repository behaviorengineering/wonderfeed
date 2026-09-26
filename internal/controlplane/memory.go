package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// MemoryStore is an in-process Store for unit tests and local smoke paths.
type MemoryStore struct {
	mu       sync.Mutex
	clock    func() time.Time
	profiles map[string]ChildProfile
}

// NewMemoryStore builds a memory store. Panics on nil clock.
func NewMemoryStore(clock func() time.Time) *MemoryStore {
	if clock == nil {
		panic("controlplane.NewMemoryStore: clock is nil")
	}
	return &MemoryStore{
		clock:    clock,
		profiles: map[string]ChildProfile{},
	}
}

var _ Store = (*MemoryStore)(nil)

// Create implements Store.
func (s *MemoryStore) Create(ctx context.Context, name, avatarColor string, policy ChildPolicy) (ChildProfile, error) {
	const op = "controlplane.MemoryStore.Create"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	if err := ValidateCreateName(name); err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate name")
	}
	if err := ValidatePolicy(policy); err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate policy")
	}
	if avatarColor == "" {
		avatarColor = "#7c5cff"
	}
	now := s.clock().UTC()
	id, err := newMemoryProfileID()
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "generate id")
	}
	policy.Version = 1
	policy.UpdatedAt = now
	p := ChildProfile{
		ID:               id,
		Name:             name,
		AvatarColor:      avatarColor,
		AllowlistVersion: 1,
		Policy:           policy,
		SyncStatus:       SyncPending,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles[id] = p
	return p, nil
}

// List implements Store.
func (s *MemoryStore) List(ctx context.Context) ([]ChildProfile, error) {
	const op = "controlplane.MemoryStore.List"
	if err := requireServiceCtx(ctx, op); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ChildProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// Get implements Store.
func (s *MemoryStore) Get(ctx context.Context, id string) (ChildProfile, error) {
	const op = "controlplane.MemoryStore.Get"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	return p, nil
}

// ListAllowlist implements Store.
func (s *MemoryStore) ListAllowlist(ctx context.Context, id string) ([]AllowlistChannel, error) {
	const op = "controlplane.MemoryStore.ListAllowlist"
	if err := requireServiceCtx(ctx, op); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return nil, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	out := append([]AllowlistChannel(nil), p.Allowlist...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].AddedAt.Equal(out[j].AddedAt) {
			if out[i].Provider == out[j].Provider {
				return out[i].ExternalID < out[j].ExternalID
			}
			return out[i].Provider < out[j].Provider
		}
		return out[i].AddedAt.Before(out[j].AddedAt)
	})
	return out, nil
}

// GetAllowlistEntry implements Store.
func (s *MemoryStore) GetAllowlistEntry(ctx context.Context, id, providerKey, externalID string) (AllowlistChannel, error) {
	const op = "controlplane.MemoryStore.GetAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return AllowlistChannel{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return AllowlistChannel{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	for _, entry := range p.Allowlist {
		if entry.Provider == providerKey && entry.ExternalID == externalID {
			return entry, nil
		}
	}
	return AllowlistChannel{}, apperr.New(apperr.CodeNotFound, op, "allowlist entry not found").
		With("provider", providerKey).With("external_id", externalID)
}

// AddAllowlistEntry implements Store with optimistic locking.
func (s *MemoryStore) AddAllowlistEntry(ctx context.Context, id string, expectedVersion int64, entry AllowlistChannel) (ChildProfile, error) {
	const op = "controlplane.MemoryStore.AddAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	normalized, err := NormalizeAllowlistEntry(entry)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate allowlist entry")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	if p.AllowlistVersion != expectedVersion {
		return ChildProfile{}, apperr.New(apperr.CodeConflict, op, "allowlist version conflict").With("id", id)
	}
	for _, existing := range p.Allowlist {
		if existing.Provider == normalized.Provider && existing.ExternalID == normalized.ExternalID {
			return ChildProfile{}, apperr.New(apperr.CodeConflict, op, "allowlist entry already exists").
				With("provider", normalized.Provider).With("external_id", normalized.ExternalID)
		}
	}
	now := s.clock().UTC()
	normalized.AddedAt = now
	p.Allowlist = append(p.Allowlist, normalized)
	p.AllowlistVersion++
	p.SyncStatus = SyncPending
	p.SyncError = ""
	p.UpdatedAt = now
	s.profiles[id] = p
	return p, nil
}

// UpdateAllowlistEntry implements Store.
func (s *MemoryStore) UpdateAllowlistEntry(ctx context.Context, id, providerKey, externalID string, title, channelURL string) (AllowlistChannel, error) {
	const op = "controlplane.MemoryStore.UpdateAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return AllowlistChannel{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return AllowlistChannel{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	for i, entry := range p.Allowlist {
		if entry.Provider != providerKey || entry.ExternalID != externalID {
			continue
		}
		updated := entry
		updated.Title = strings.TrimSpace(title)
		if strings.TrimSpace(channelURL) != "" {
			updated.URL = strings.TrimSpace(channelURL)
		}
		p.Allowlist[i] = updated
		p.UpdatedAt = s.clock().UTC()
		s.profiles[id] = p
		return updated, nil
	}
	return AllowlistChannel{}, apperr.New(apperr.CodeNotFound, op, "allowlist entry not found").
		With("provider", providerKey).With("external_id", externalID)
}

// DeleteAllowlistEntry implements Store with optimistic locking.
func (s *MemoryStore) DeleteAllowlistEntry(ctx context.Context, id string, expectedVersion int64, providerKey, externalID string) (ChildProfile, error) {
	const op = "controlplane.MemoryStore.DeleteAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	if p.AllowlistVersion != expectedVersion {
		return ChildProfile{}, apperr.New(apperr.CodeConflict, op, "allowlist version conflict").With("id", id)
	}
	next := make([]AllowlistChannel, 0, len(p.Allowlist))
	found := false
	for _, entry := range p.Allowlist {
		if entry.Provider == providerKey && entry.ExternalID == externalID {
			found = true
			continue
		}
		next = append(next, entry)
	}
	if !found {
		return ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "allowlist entry not found").
			With("provider", providerKey).With("external_id", externalID)
	}
	p.Allowlist = next
	p.AllowlistVersion++
	p.SyncStatus = SyncPending
	p.SyncError = ""
	p.UpdatedAt = s.clock().UTC()
	s.profiles[id] = p
	return p, nil
}

// UpdatePolicy implements Store.
func (s *MemoryStore) UpdatePolicy(ctx context.Context, id string, expectedVersion int64, policy ChildPolicy) (ChildProfile, error) {
	const op = "controlplane.MemoryStore.UpdatePolicy"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	if err := ValidatePolicy(policy); err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate policy")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	if p.Policy.Version != expectedVersion {
		return ChildProfile{}, apperr.New(apperr.CodeConflict, op, "policy version conflict").With("id", id)
	}
	now := s.clock().UTC()
	policy.Version = expectedVersion + 1
	policy.UpdatedAt = now
	p.Policy = policy
	p.Version++
	p.UpdatedAt = now
	p.SyncStatus = SyncPending
	p.SyncError = ""
	s.profiles[id] = p
	return p, nil
}

// SetProviderProfileID implements Store.
func (s *MemoryStore) SetProviderProfileID(ctx context.Context, id, providerProfileID string) (ChildProfile, error) {
	const op = "controlplane.MemoryStore.SetProviderProfileID"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	p.ProviderProfileID = providerProfileID
	p.UpdatedAt = s.clock().UTC()
	s.profiles[id] = p
	return p, nil
}

// RecordSyncStatus implements Store.
func (s *MemoryStore) RecordSyncStatus(ctx context.Context, id string, status SyncStatus, syncErr string) (ChildProfile, error) {
	const op = "controlplane.MemoryStore.RecordSyncStatus"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	switch status {
	case SyncPending, SyncSynced, SyncFailed:
	default:
		return ChildProfile{}, apperr.New(apperr.CodeInvalid, op, "invalid sync status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	p.SyncStatus = status
	p.SyncError = syncErr
	p.UpdatedAt = s.clock().UTC()
	s.profiles[id] = p
	return p, nil
}

func newMemoryProfileID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexed := hex.EncodeToString(b[:])
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32], nil
}
