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
	store    Store
	provider provider.ChildProfileProvider
	clock    func() time.Time
	logger   *slog.Logger
}

// ServiceConfig constructs a Service.
type ServiceConfig struct {
	Store    Store
	Provider provider.ChildProfileProvider
	Clock    func() time.Time
	Logger   *slog.Logger
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
		store:    cfg.Store,
		provider: cfg.Provider,
		clock:    cfg.Clock,
		logger:   logger,
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

// ReplaceAllowlistRequest is the parent input for replacing approved channels.
type ReplaceAllowlistRequest struct {
	ExpectedVersion int64
	Channels        []AllowlistChannel
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

// ReplaceChildAllowlist persists desired channels first, then syncs the provider.
func (s *Service) ReplaceChildAllowlist(ctx context.Context, id string, req ReplaceAllowlistRequest) (ChildProfile, error) {
	const op = "controlplane.Service.ReplaceChildAllowlist"
	if err := requireServiceCtx(ctx, op); err != nil {
		return ChildProfile{}, err
	}
	profile, err := s.store.ReplaceAllowlist(ctx, id, req.ExpectedVersion, req.Channels)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "persist allowlist")
	}
	updated, err := s.syncProfile(ctx, op, profile)
	if err != nil {
		return ChildProfile{}, err
	}
	updated.Allowlist, err = s.store.ListAllowlist(ctx, id)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "load saved allowlist")
	}
	return updated, nil
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

func (s *Service) syncProfile(ctx context.Context, op string, profile ChildProfile) (ChildProfile, error) {
	if profile.ProviderProfileID == "" {
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
	}
	allowlist, err := s.store.ListAllowlist(ctx, profile.ID)
	if err != nil {
		return ChildProfile{}, apperr.Wrap(err, codeOf(err, apperr.CodeFailed), op, "load allowlist")
	}
	if err := s.provider.ApplyAllowlist(ctx, profile.ProviderProfileID, toProviderAllowlist(allowlist)); err != nil {
		s.logger.Error("provider allowlist apply failed", "op", op, "child_id", profile.ID, "err", err)
		updated, recErr := s.store.RecordSyncStatus(ctx, profile.ID, SyncFailed, err.Error())
		if recErr != nil {
			return profile, apperr.Wrap(recErr, apperr.CodeFailed, op, "record sync failure")
		}
		return updated, nil
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

func toProviderAllowlist(channels []AllowlistChannel) []provider.Channel {
	out := make([]provider.Channel, 0, len(channels))
	for _, channel := range channels {
		out = append(out, provider.Channel{
			ID:    channel.ChannelID,
			Title: channel.Title,
			URL:   channel.URL,
		})
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
