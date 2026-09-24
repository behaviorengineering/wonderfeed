package ytzero

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/provider"
)

// Adapter implements provider.ChildProfileProvider against YT Zero HTTP APIs.
type Adapter struct {
	client *ResilientClient
}

// AdapterConfig constructs an Adapter.
type AdapterConfig struct {
	BaseURL       string
	SessionCookie string
	HTTP          HTTPDoer
}

// CreateAdapter builds an Adapter from config.
func (cfg AdapterConfig) CreateAdapter() *Adapter {
	httpClient := cfg.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base := cfg.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	return &Adapter{client: NewResilientClient(base, httpClient, cfg.SessionCookie)}
}

// NewAdapter is a package-level constructor for discoverability.
func NewAdapter(cfg AdapterConfig) *Adapter {
	return cfg.CreateAdapter()
}

var _ provider.ChildProfileProvider = (*Adapter)(nil)

type profilesResponse struct {
	Profiles []ytProfile `json:"profiles"`
}

type ytProfile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	AvatarColor string `json:"avatar_color"`
	IsChild     bool   `json:"is_child"`
}

type createProfileResponse struct {
	Profile ytProfile `json:"profile"`
}

type patchProfileResponse struct {
	Profile ytProfile `json:"profile"`
}

// ListChildProfiles returns child profiles from YT Zero.
func (a *Adapter) ListChildProfiles(ctx context.Context) ([]provider.Profile, error) {
	const op = "ytzero.Adapter.ListChildProfiles"
	if a == nil || a.client == nil {
		return nil, apperr.New(apperr.CodeInvalid, op, "adapter is nil")
	}
	raw, _, err := a.client.DoJSON(ctx, http.MethodGet, "/api/profiles", nil)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, op, "list profiles")
	}
	var resp profilesResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, apperr.Wrap(err, apperr.CodeFailed, op, "decode profiles")
	}
	out := make([]provider.Profile, 0, len(resp.Profiles))
	for _, p := range resp.Profiles {
		if !p.IsChild {
			continue
		}
		out = append(out, provider.Profile{
			ID:          strconv.Itoa(p.ID),
			Name:        p.Name,
			AvatarColor: p.AvatarColor,
			IsChild:     true,
		})
	}
	return out, nil
}

// CreateChildProfile creates a child profile in YT Zero.
func (a *Adapter) CreateChildProfile(ctx context.Context, name, avatarColor string) (provider.Profile, error) {
	const op = "ytzero.Adapter.CreateChildProfile"
	if a == nil || a.client == nil {
		return provider.Profile{}, apperr.New(apperr.CodeInvalid, op, "adapter is nil")
	}
	body := map[string]any{
		"name":         strings.TrimSpace(name),
		"avatar_color": avatarColor,
		"is_child":     true,
	}
	raw, _, err := a.client.DoJSON(ctx, http.MethodPost, "/api/profiles", body)
	if err != nil {
		return provider.Profile{}, apperr.Wrap(err, apperr.CodeUnavailable, op, "create profile")
	}
	var resp createProfileResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return provider.Profile{}, apperr.Wrap(err, apperr.CodeFailed, op, "decode create response")
	}
	if resp.Profile.ID == 0 {
		return provider.Profile{}, apperr.New(apperr.CodeFailed, op, "provider returned empty profile id")
	}
	return provider.Profile{
		ID:          strconv.Itoa(resp.Profile.ID),
		Name:        resp.Profile.Name,
		AvatarColor: resp.Profile.AvatarColor,
		IsChild:     resp.Profile.IsChild,
	}, nil
}

// ApplyPolicy patches child_config on a YT Zero profile.
func (a *Adapter) ApplyPolicy(ctx context.Context, providerProfileID string, policy provider.PolicyPayload) (provider.ApplyResult, error) {
	const op = "ytzero.Adapter.ApplyPolicy"
	if a == nil || a.client == nil {
		return provider.ApplyResult{}, apperr.New(apperr.CodeInvalid, op, "adapter is nil")
	}
	if strings.TrimSpace(providerProfileID) == "" {
		return provider.ApplyResult{}, apperr.New(apperr.CodeInvalid, op, "provider profile id is required")
	}
	unsupported := []string{}
	if policy.BedtimeStart != "" || policy.BedtimeEnd != "" {
		unsupported = append(unsupported, "bedtime")
	}
	body := map[string]any{
		"child_config": map[string]any{
			"limit_minutes":  policy.DailyMinutes,
			"local_only":     policy.LocalOnly,
			"hide_shorts":    policy.HideShorts,
			"hide_live":      policy.HideLive,
			"downloads_only": policy.DownloadsOnly,
		},
	}
	path := fmt.Sprintf("/api/profiles/%s", providerProfileID)
	raw, _, err := a.client.DoJSON(ctx, http.MethodPatch, path, body)
	if err != nil {
		return provider.ApplyResult{}, apperr.Wrap(err, apperr.CodeUnavailable, op, "patch profile").
			With("provider_profile_id", providerProfileID)
	}
	var resp patchProfileResponse
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &resp); err != nil {
			return provider.ApplyResult{}, apperr.Wrap(err, apperr.CodeFailed, op, "decode patch response")
		}
	}
	return provider.ApplyResult{
		ProviderProfileID: providerProfileID,
		Unsupported:       unsupported,
	}, nil
}
