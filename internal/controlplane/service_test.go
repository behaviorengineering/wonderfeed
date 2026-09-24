package controlplane

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/provider"
)

type fakeProvider struct {
	mu         sync.Mutex
	nextID     int
	failCreate bool
	failApply  bool
	profiles   map[string]provider.Profile
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		nextID:   1,
		profiles: map[string]provider.Profile{},
	}
}

func (f *fakeProvider) ListChildProfiles(ctx context.Context) ([]provider.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]provider.Profile, 0, len(f.profiles))
	for _, p := range f.profiles {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeProvider) CreateChildProfile(ctx context.Context, name, avatarColor string) (provider.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failCreate {
		return provider.Profile{}, apperr.New(apperr.CodeUnavailable, "fake.Create", "provider down")
	}
	id := f.nextID
	f.nextID++
	p := provider.Profile{
		ID:          strconv.Itoa(id),
		Name:        name,
		AvatarColor: avatarColor,
		IsChild:     true,
	}
	f.profiles[p.ID] = p
	return p, nil
}

func (f *fakeProvider) ApplyPolicy(ctx context.Context, providerProfileID string, policy provider.PolicyPayload) (provider.ApplyResult, error) {
	if f.failApply {
		return provider.ApplyResult{}, apperr.New(apperr.CodeUnavailable, "fake.Apply", "provider down")
	}
	return provider.ApplyResult{ProviderProfileID: providerProfileID}, nil
}

func testService(t *testing.T, prov provider.ChildProfileProvider) *Service {
	t.Helper()
	now := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)
	return NewService(ServiceConfig{
		Store:    NewMemoryStore(func() time.Time { return now }),
		Provider: prov,
		Clock:    func() time.Time { return now },
		Logger:   slog.Default(),
	})
}

func TestServiceCreateAndUpdateSynced(t *testing.T) {
	t.Parallel()
	svc := testService(t, newFakeProvider())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := svc.CreateChildProfile(ctx, CreateChildRequest{Name: "Ada", AvatarColor: "#abc"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.SyncStatus != SyncSynced || created.ProviderProfileID == "" {
		t.Fatalf("created = %+v", created)
	}

	policy := created.Policy
	policy.DailyMinutes = 20
	updated, err := svc.UpdateChildPolicy(ctx, created.ID, UpdatePolicyRequest{
		ExpectedVersion: created.Policy.Version,
		Policy:          policy,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.SyncStatus != SyncSynced || updated.Policy.DailyMinutes != 20 {
		t.Fatalf("updated = %+v", updated)
	}
}

func TestServiceProviderFailureKeepsDesiredPolicy(t *testing.T) {
	t.Parallel()
	fp := newFakeProvider()
	fp.failApply = true
	svc := testService(t, fp)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := svc.CreateChildProfile(ctx, CreateChildRequest{Name: "Ada"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.SyncStatus != SyncFailed {
		t.Fatalf("expected sync_failed, got %+v", created)
	}
	got, err := svc.GetChildProfile(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Ada" || got.Policy.LocalOnly != true {
		t.Fatalf("desired policy lost: %+v", got)
	}
}

func TestServiceUpdateConflict(t *testing.T) {
	t.Parallel()
	svc := testService(t, newFakeProvider())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	created, err := svc.CreateChildProfile(ctx, CreateChildRequest{Name: "Ada"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.UpdateChildPolicy(ctx, created.ID, UpdatePolicyRequest{
		ExpectedVersion: created.Policy.Version - 1,
		Policy:          created.Policy,
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.CodeConflict {
		t.Fatalf("got %#v", err)
	}
}

func TestMemoryStoreCreateListUpdateSync(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 25, 1, 0, 0, 0, time.UTC)
	s := NewMemoryStore(func() time.Time { return now })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	policy := DefaultChildPolicy()
	created, err := s.Create(ctx, "Ada", "#112233", policy)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.SyncStatus != SyncPending {
		t.Fatalf("created = %+v", created)
	}

	listed, err := s.List(ctx)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list: %v %#v", err, listed)
	}

	policy.DailyMinutes = 45
	updated, err := s.UpdatePolicy(ctx, created.ID, created.Policy.Version, policy)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Policy.Version != 2 || updated.Policy.DailyMinutes != 45 {
		t.Fatalf("updated = %+v", updated)
	}

	_, err = s.UpdatePolicy(ctx, created.ID, created.Policy.Version, policy)
	if err == nil {
		t.Fatal("expected version conflict")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.CodeConflict {
		t.Fatalf("got %#v", err)
	}

	synced, err := s.RecordSyncStatus(ctx, created.ID, SyncSynced, "")
	if err != nil || synced.SyncStatus != SyncSynced {
		t.Fatalf("sync: %v %+v", err, synced)
	}
}
