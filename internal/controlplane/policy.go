// Package controlplane owns Wonderfeed parent child-profile policy.
package controlplane

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/provider"
)

// SyncStatus is the provider synchronization state for a desired policy.
type SyncStatus string

const (
	// SyncPending means the host has desired policy but provider apply has not finished.
	SyncPending SyncStatus = "sync_pending"
	// SyncSynced means the provider accepted the last desired policy.
	SyncSynced SyncStatus = "synced"
	// SyncFailed means the last provider apply failed; host desired policy remains.
	SyncFailed SyncStatus = "sync_failed"
)

// MaxDailyMinutes is the upper bound for a daily watch limit (24 hours).
const MaxDailyMinutes = 24 * 60

// ChildPolicy is the provider-neutral desired policy for one child profile.
type ChildPolicy struct {
	DailyMinutes  int       `json:"daily_minutes"`
	LocalOnly     bool      `json:"local_only"`
	HideShorts    bool      `json:"hide_shorts"`
	HideLive      bool      `json:"hide_live"`
	DownloadsOnly bool      `json:"downloads_only"`
	BedtimeStart  string    `json:"bedtime_start,omitempty"`
	BedtimeEnd    string    `json:"bedtime_end,omitempty"`
	Version       int64     `json:"version"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ChildProfile is a host-owned child identity with desired policy and sync state.
type ChildProfile struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	AvatarColor       string             `json:"avatar_color"`
	ProviderProfileID string             `json:"provider_profile_id,omitempty"`
	AllowlistVersion  int64              `json:"allowlist_version"`
	Allowlist         []AllowlistChannel `json:"allowlist,omitempty"`
	Policy            ChildPolicy        `json:"policy"`
	SyncStatus        SyncStatus         `json:"sync_status"`
	SyncError         string             `json:"sync_error,omitempty"`
	Version           int64              `json:"version"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// AllowlistChannel is a parent-approved provider feed source for a child.
type AllowlistChannel struct {
	Provider   string    `json:"provider"`
	ExternalID string    `json:"external_id"`
	Title      string    `json:"title,omitempty"`
	URL        string    `json:"url,omitempty"`
	AddedAt    time.Time `json:"added_at"`
}

var youtubeChannelID = regexp.MustCompile(`^UC[\w-]{22}$`)

// NormalizeAllowlistEntry validates one provider-scoped allowlist entry.
func NormalizeAllowlistEntry(entry AllowlistChannel) (AllowlistChannel, error) {
	const op = "controlplane.NormalizeAllowlistEntry"
	providerKey := strings.ToLower(strings.TrimSpace(entry.Provider))
	if providerKey == "" {
		providerKey = provider.ProviderYouTube
	}
	externalID := strings.TrimSpace(entry.ExternalID)
	if externalID == "" {
		return AllowlistChannel{}, apperr.New(apperr.CodeInvalid, op, "external_id is required")
	}
	if len(externalID) > 128 || strings.ContainsAny(externalID, " \t\r\n/") {
		return AllowlistChannel{}, apperr.New(apperr.CodeInvalid, op, "external_id is invalid").
			With("external_id", externalID)
	}
	if err := validateProviderExternalID(providerKey, externalID); err != nil {
		return AllowlistChannel{}, err
	}
	channelURL := strings.TrimSpace(entry.URL)
	if channelURL == "" {
		channelURL = defaultChannelURL(providerKey, externalID)
	} else {
		parsed, err := url.Parse(channelURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return AllowlistChannel{}, apperr.New(apperr.CodeInvalid, op, "url must be an HTTP or HTTPS URL").
				With("external_id", externalID)
		}
	}
	return AllowlistChannel{
		Provider:   providerKey,
		ExternalID: externalID,
		Title:      strings.TrimSpace(entry.Title),
		URL:        channelURL,
	}, nil
}

func validateProviderExternalID(providerKey, externalID string) error {
	const op = "controlplane.NormalizeAllowlistEntry"
	switch providerKey {
	case provider.ProviderYouTube:
		if !youtubeChannelID.MatchString(externalID) {
			return apperr.New(apperr.CodeInvalid, op, "youtube external_id must be a UC channel id").
				With("external_id", externalID)
		}
		return nil
	default:
		return apperr.New(apperr.CodeInvalid, op, "unsupported provider").With("provider", providerKey)
	}
}

func defaultChannelURL(providerKey, externalID string) string {
	switch providerKey {
	case provider.ProviderYouTube:
		return "https://www.youtube.com/channel/" + url.PathEscape(externalID)
	default:
		return ""
	}
}

// ToProviderScoped converts a host allowlist entry for provider sync.
func (c AllowlistChannel) ToProviderScoped() provider.ScopedChannel {
	return provider.ScopedChannel{
		Provider:   c.Provider,
		ExternalID: c.ExternalID,
		Title:      c.Title,
		URL:        c.URL,
	}
}

// DefaultChildPolicy returns fail-closed defaults for a new child profile.
func DefaultChildPolicy() ChildPolicy {
	return ChildPolicy{
		DailyMinutes:  0,
		LocalOnly:     true,
		HideShorts:    true,
		HideLive:      true,
		DownloadsOnly: false,
	}
}

// ValidatePolicy checks policy bounds and bedtime format.
func ValidatePolicy(p ChildPolicy) error {
	const op = "controlplane.ValidatePolicy"
	if p.DailyMinutes < 0 || p.DailyMinutes > MaxDailyMinutes {
		return apperr.New(apperr.CodeInvalid, op, "daily_minutes must be between 0 and 1440").
			With("daily_minutes", strconv.Itoa(p.DailyMinutes))
	}
	if err := validateBedtime(p.BedtimeStart, "bedtime_start"); err != nil {
		return err
	}
	if err := validateBedtime(p.BedtimeEnd, "bedtime_end"); err != nil {
		return err
	}
	if (p.BedtimeStart == "") != (p.BedtimeEnd == "") {
		return apperr.New(apperr.CodeInvalid, op, "bedtime_start and bedtime_end must both be set or both empty")
	}
	return nil
}

// ValidateCreateName checks a non-empty child display name.
func ValidateCreateName(name string) error {
	const op = "controlplane.ValidateCreateName"
	if strings.TrimSpace(name) == "" {
		return apperr.New(apperr.CodeInvalid, op, "name is required")
	}
	return nil
}

func validateBedtime(value, field string) error {
	const op = "controlplane.ValidatePolicy"
	if value == "" {
		return nil
	}
	if _, err := time.Parse("15:04", value); err != nil {
		return apperr.New(apperr.CodeInvalid, op, field+" must be HH:MM").With(field, value)
	}
	return nil
}
