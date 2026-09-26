package controlplane

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/provider"
)

// Service coordinates host desired policy and provider synchronization.
type Service struct {
	store         Store
	provider      provider.ChildProfileProvider
	allowlistSync provider.AllowlistSynchronizer
	clock         func() time.Time
	logger        *slog.Logger
}

// ServiceConfig constructs a Service.
type ServiceConfig struct {
	Store         Store
	Provider      provider.ChildProfileProvider
	AllowlistSync provider.AllowlistSynchronizer
	Clock         func() time.Time
	Logger        *slog.Logger
}

// CreateService validates dependencies and returns a Service.
func (cfg ServiceConfig) CreateService() *Service {
	if cfg.Store == nil {
		panic("controlplane.CreateService: store is nil")
	}
	if cfg.Provider == nil {
		panic("controlplane.CreateService: provider is nil")
	}
	if cfg.Clock == nil {
		panic("controlplane.CreateService: clock is nil")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:         cfg.Store,
		provider:      cfg.Provider,
		allowlistSync: cfg.AllowlistSync,
		clock:         cfg.Clock,
		logger:        logger,
	}
}

// NewService is a package-level constructor for discoverability.
func NewService(cfg ServiceConfig) *Service {
	return cfg.CreateService()
}

// CreateChildRequest is the parent input for creating a child profile.
type CreateChildRequest struct {
	Name        string
	AvatarColor string
	Policy      *ChildPolicy
}

// UpdatePolicyRequest is the parent input for updating desired policy.
type UpdatePolicyRequest struct {
	ExpectedVersion int64
	Policy          ChildPolicy
}

// AddAllowlistEntryRequest is the parent input for adding one allowlist entry.
type AddAllowlistEntryRequest struct {
	ExpectedVersion int64
	Entry           AllowlistChannel
}

// UpdateAllowlistEntryRequest updates host-owned metadata for one entry.
type UpdateAllowlistEntryRequest struct {
	Title string
	URL   string
}

// DeleteAllowlistEntryRequest removes one allowlist entry.
type DeleteAllowlistEntryRequest struct {
	ExpectedVersion int64
}

// CreateChildProfile persists desired state then syncs to the provider.
func (s *Service) CreateChildProfile(ctx context.Context, req CreateChildRequest) (ChildProfile, error) {
	const op = "controlplane.Service.CreateChildProfile"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	policy := DefaultChildPolicy()
	if req.Policy != nil {
		policy = *req.Policy
	}
	profile, err := s.store.Create(ctx, strings.TrimSpace(req.Name), req.AvatarColor, policy)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "persist profile")
	}

	prov, err := s.provider.CreateChildProfile(ctx, profile.Name, profile.AvatarColor)
	if err != nil {
		s.logger.Error("provider create failed", "op", op, "child_id", profile.ID, "err", err)
		updated, recErr := s.store.RecordSyncStatus(ctx, profile.ID, SyncFailed, err.Error())
		if recErr != nil {
			return profile, apperr.Wrap(recErr, apperr.CodeFailed, op, "record sync failure after provider create")
		}
		return updated, nil
	}
	profile, err = s.store.SetProviderProfileID(ctx, profile.ID, prov.ID)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "store provider profile id")
	}
	return s.syncProfile(ctx, op, profile)
}

// ListChildProfiles returns host-owned child profiles.
func (s *Service) ListChildProfiles(ctx context.Context) ([]ChildProfile, error) {
	const op = "controlplane.Service.ListChildProfiles"
	if err := requireServiceCtx(ctx, op); err != nil {
		return nil, err
	}
	out, err := s.store.List(ctx)
	if err != nil {
		return nil, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "list profiles")
	}
	return out, nil
}

// GetChildProfile returns one host-owned child profile.
func (s *Service) GetChildProfile(ctx context.Context, id string) (ChildProfile, error) {
	const op = "controlplane.Service.GetChildProfile"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	out, err := s.store.Get(ctx, id)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "get profile")
	}
	allowlist, err := s.store.ListAllowlist(ctx, id)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "get allowlist")
	}
	out.Allowlist = allowlist
	return out, nil
}

// ListChildAllowlist returns the host-owned approved channels for a child.
func (s *Service) ListChildAllowlist(ctx context.Context, id string) ([]AllowlistChannel, int64, error) {
	const op = "controlplane.Service.ListChildAllowlist"
	if err := requireServiceCtx(ctx, op); err != nil {
		return nil, 0, err
	}
	profile, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, 0, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "get profile")
	}
	allowlist, err := s.store.ListAllowlist(ctx, id)
	if err != nil {
		return nil, 0, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "list allowlist")
	}
	return allowlist, profile.AllowlistVersion, nil
}

// GetChildAllowlistEntry returns one host-owned allowlist entry.
func (s *Service) GetChildAllowlistEntry(ctx context.Context, id, providerKey, externalID string) (AllowlistChannel, error) {
	const op = "controlplane.Service.GetChildAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return AllowlistChannel{}, err
	}
	if _, err := s.store.Get(ctx, id); err != nil {
		return AllowlistChannel{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "get profile")
	}
	entry, err := s.store.GetAllowlistEntry(ctx, id, strings.ToLower(strings.TrimSpace(providerKey)), strings.TrimSpace(externalID))
	if err != nil {
		return AllowlistChannel{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "get allowlist entry")
	}
	return entry, nil
}

// AddChildAllowlistEntry persists one entry, then syncs provider storage.
func (s *Service) AddChildAllowlistEntry(ctx context.Context, id string, req AddAllowlistEntryRequest) (ChildProfile, error) {
	const op = "controlplane.Service.AddChildAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	profile, err := s.store.AddAllowlistEntry(ctx, id, req.ExpectedVersion, req.Entry)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "persist allowlist entry")
	}
	normalized, err := NormalizeAllowlistEntry(req.Entry)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate allowlist entry")
	}
	return s.syncAllowlistMutation(ctx, op, profile, normalized, true)
}

// UpdateChildAllowlistEntry updates host-owned metadata without provider sync.
func (s *Service) UpdateChildAllowlistEntry(ctx context.Context, id, providerKey, externalID string, req UpdateAllowlistEntryRequest) (AllowlistChannel, error) {
	const op = "controlplane.Service.UpdateChildAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return AllowlistChannel{}, err
	}
	if _, err := s.store.Get(ctx, id); err != nil {
		return AllowlistChannel{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "get profile")
	}
	entry, err := s.store.UpdateAllowlistEntry(ctx, id, strings.ToLower(strings.TrimSpace(providerKey)), strings.TrimSpace(externalID), req.Title, req.URL)
	if err != nil {
		return AllowlistChannel{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "update allowlist entry")
	}
	return entry, nil
}

// DeleteChildAllowlistEntry removes one entry, then syncs provider storage.
func (s *Service) DeleteChildAllowlistEntry(ctx context.Context, id, providerKey, externalID string, req DeleteAllowlistEntryRequest) (ChildProfile, error) {
	const op = "controlplane.Service.DeleteChildAllowlistEntry"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	providerKey = strings.ToLower(strings.TrimSpace(providerKey))
	externalID = strings.TrimSpace(externalID)
	entry, err := s.store.GetAllowlistEntry(ctx, id, providerKey, externalID)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "load allowlist entry")
	}
	profile, err := s.store.DeleteAllowlistEntry(ctx, id, req.ExpectedVersion, providerKey, externalID)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "delete allowlist entry")
	}
	return s.syncAllowlistMutation(ctx, op, profile, entry, false)
}

// UpdateChildPolicy saves desired policy first, then attempts provider sync.
func (s *Service) UpdateChildPolicy(ctx context.Context, id string, req UpdatePolicyRequest) (ChildProfile, error) {
	const op = "controlplane.Service.UpdateChildPolicy"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	profile, err := s.store.UpdatePolicy(ctx, id, req.ExpectedVersion, req.Policy)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "persist policy")
	}
	return s.syncProfile(ctx, op, profile)
}

// ForceSync re-applies the current desired policy to the provider.
func (s *Service) ForceSync(ctx context.Context, id string) (ChildProfile, error) {
	const op = "controlplane.Service.ForceSync"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	if _, err := s.store.Get(ctx, id); err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "get profile")
	}
	pending, err := s.store.RecordSyncStatus(ctx, id, SyncPending, "")
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "mark pending")
	}
	return s.syncProfile(ctx, op, pending)
}

func (s *Service) syncAllowlistMutation(ctx context.Context, op string, profile ChildProfile, entry AllowlistChannel, add bool) (ChildProfile, error) {
	profile, err := s.ensureProviderProfile(ctx, op, profile)
	if err != nil {
		return profile, err
	}
	if s.allowlistSync == nil {
		return s.store.RecordSyncStatus(ctx, profile.ID, SyncSynced, "")
	}
	scoped := entry.ToProviderScoped()
	var syncErr error
	if add {
		syncErr = s.allowlistSync.AddMembership(ctx, profile.ProviderProfileID, scoped)
	} else {
		syncErr = s.allowlistSync.RemoveMembership(ctx, profile.ProviderProfileID, scoped)
	}
	if syncErr != nil {
		s.logger.Error("provider allowlist mutation failed", "op", op, "child_id", profile.ID, "err", syncErr)
		updated, recErr := s.store.RecordSyncStatus(ctx, profile.ID, SyncFailed, syncErr.Error())
		if recErr != nil {
			return profile, apperr.Wrap(recErr, apperr.CodeFailed, op, "record sync failure")
		}
		updated.Allowlist, err = s.store.ListAllowlist(ctx, profile.ID)
		if err != nil {
			return updated, apperr.Wrap(err, apperr.CodeFailed, op, "load allowlist")
		}
		return updated, nil
	}
	updated, err := s.store.RecordSyncStatus(ctx, profile.ID, SyncSynced, "")
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "record sync success")
	}
	updated.Allowlist, err = s.store.ListAllowlist(ctx, profile.ID)
	if err != nil {
		return updated, apperr.Wrap(err, apperr.CodeFailed, op, "load allowlist")
	}
	return updated, nil
}

func (s *Service) syncProfile(ctx context.Context, op string, profile ChildProfile) (ChildProfile, error) {
	profile, err := s.ensureProviderProfile(ctx, op, profile)
	if err != nil {
		return profile, err
	}
	if s.allowlistSync != nil {
		allowlist, err := s.store.ListAllowlist(ctx, profile.ID)
		if err != nil {
			return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "load allowlist")
		}
		if err := s.allowlistSync.ReconcileAll(ctx, profile.ProviderProfileID, toProviderScopedAllowlist(allowlist)); err != nil {
			s.logger.Error("provider allowlist reconcile failed", "op", op, "child_id", profile.ID, "err", err)
			updated, recErr := s.store.RecordSyncStatus(ctx, profile.ID, SyncFailed, err.Error())
			if recErr != nil {
				return profile, apperr.Wrap(recErr, apperr.CodeFailed, op, "record sync failure")
			}
			return updated, nil
		}
	}
	apply, err := s.provider.ApplyPolicy(ctx, profile.ProviderProfileID, toProviderPolicy(profile.Policy))
	if err != nil {
		s.logger.Error("provider apply failed", "op", op, "child_id", profile.ID, "err", err)
		updated, recErr := s.store.RecordSyncStatus(ctx, profile.ID, SyncFailed, err.Error())
		if recErr != nil {
			return profile, apperr.Wrap(recErr, apperr.CodeFailed, op, "record sync failure")
		}
		return updated, nil
	}
	if len(apply.Unsupported) > 0 {
		s.logger.Info("provider applied with unsupported fields", "op", op, "child_id", profile.ID, "unsupported", apply.Unsupported)
	}
	updated, err := s.store.RecordSyncStatus(ctx, profile.ID, SyncSynced, "")
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "record sync success")
	}
	return updated, nil
}

func (s *Service) ensureProviderProfile(ctx context.Context, op string, profile ChildProfile) (ChildProfile, error) {
	if profile.ProviderProfileID != "" {
		return profile, nil
	}
	prov, err := s.provider.CreateChildProfile(ctx, profile.Name, profile.AvatarColor)
	if err != nil {
		s.logger.Error("provider create failed during sync", "op", op, "child_id", profile.ID, "err", err)
		updated, recErr := s.store.RecordSyncStatus(ctx, profile.ID, SyncFailed, err.Error())
		if recErr != nil {
			return profile, apperr.Wrap(recErr, apperr.CodeFailed, op, "record sync failure")
		}
		return updated, nil
	}
	profile, err = s.store.SetProviderProfileID(ctx, profile.ID, prov.ID)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "store provider profile id")
	}
	return profile, nil
}

func requireServiceCtx(ctx context.Context, op string) error {
	if ctx == nil {
		return apperr.New(apperr.CodeInvalid, op, "context is required")
	}
	if _, ok := ctx.Deadline(); !ok {
		return apperr.New(apperr.CodeInvalid, op, "outbound: missing deadline")
	}
	return nil
}

func toProviderPolicy(p ChildPolicy) provider.PolicyPayload {
	return provider.PolicyPayload{
		DailyMinutes:  p.DailyMinutes,
		LocalOnly:     p.LocalOnly,
		HideShorts:    p.HideShorts,
		HideLive:      p.HideLive,
		DownloadsOnly: p.DownloadsOnly,
		BedtimeStart:  p.BedtimeStart,
		BedtimeEnd:    p.BedtimeEnd,
	}
}

func toProviderScopedAllowlist(channels []AllowlistChannel) []provider.ScopedChannel {
	out := make([]provider.ScopedChannel, 0, len(channels))
	for _, channel := range channels {
		out = append(out, channel.ToProviderScoped())
	}
	return out
}

func codeOf(err error, fallback apperr.Code) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return fallback
}
